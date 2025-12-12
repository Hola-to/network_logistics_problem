// Package service provides the gRPC service implementation for the solver microservice.
//
// The SolverService handles network flow optimization requests, supporting multiple
// algorithms including Ford-Fulkerson, Edmonds-Karp, Dinic, Push-Relabel, and
// Min-Cost Max-Flow variants.
//
// # Thread Safety
//
// The service is designed for concurrent use. Each request operates on its own
// copy of the graph, and all shared state is protected by appropriate synchronization.
//
// # Resource Management
//
// The service uses object pooling for graphs and related data structures to minimize
// GC pressure under high load. Resources are automatically released after each request.
//
// # Graceful Shutdown
//
// The service supports graceful shutdown via the Shutdown() method, which waits for
// all in-flight requests to complete before returning.
package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/cache"
	"logistics/pkg/domain"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/telemetry"
	"logistics/services/solver-svc/internal/algorithms"
	"logistics/services/solver-svc/internal/converter"
	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Constants and Limits
// =============================================================================

const (
	// MaxGraphNodes is the maximum number of nodes allowed in a graph.
	MaxGraphNodes = 1_000_000

	// MaxGraphEdges is the maximum number of edges allowed in a graph.
	MaxGraphEdges = 10_000_000

	// MinEpsilon is the minimum allowed epsilon value for floating-point comparisons.
	MinEpsilon = 1e-15

	// MaxEpsilon is the maximum allowed epsilon value.
	MaxEpsilon = 1e-3

	// MinTimeoutSeconds is the minimum allowed timeout in seconds.
	MinTimeoutSeconds = 0.1

	// MaxTimeoutSeconds is the maximum allowed timeout (1 hour).
	MaxTimeoutSeconds = 3600.0

	// MinIterations is the minimum iteration limit when specified.
	MinIterations = 10

	// CacheOperationTimeout is the timeout for cache operations.
	CacheOperationTimeout = 5 * time.Second

	// StreamProgressInterval is the minimum interval between progress updates.
	StreamProgressInterval = 200 * time.Millisecond

	// ContextCheckInterval is how often to check for context cancellation in loops.
	ContextCheckInterval = 10
)

// =============================================================================
// Configuration
// =============================================================================

// ServiceConfig holds the configuration for the SolverService.
type ServiceConfig struct {
	// MaxConcurrentSolves limits the number of simultaneous solve operations.
	// Requests beyond this limit will wait or timeout.
	MaxConcurrentSolves int

	// DefaultTimeout is applied when no timeout is specified in the request.
	DefaultTimeout time.Duration

	// MemStatsInterval controls how often memory statistics are refreshed.
	// More frequent updates have higher overhead but provide more accurate data.
	MemStatsInterval time.Duration

	// ShutdownTimeout is the maximum time to wait for in-flight requests during shutdown.
	ShutdownTimeout time.Duration

	// EnableMemoryTracking enables per-request memory usage tracking.
	// This adds some overhead but provides useful metrics.
	EnableMemoryTracking bool
}

// DefaultServiceConfig returns a ServiceConfig with sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxConcurrentSolves:  runtime.NumCPU() * 2,
		DefaultTimeout:       30 * time.Second,
		MemStatsInterval:     time.Second,
		ShutdownTimeout:      30 * time.Second,
		EnableMemoryTracking: true,
	}
}

// =============================================================================
// Statistics
// =============================================================================

// serviceStats holds atomic counters for service metrics.
type serviceStats struct {
	requestsTotal   atomic.Int64
	requestsActive  atomic.Int64
	requestsSuccess atomic.Int64
	requestsFailed  atomic.Int64
	cacheHits       atomic.Int64
	cacheMisses     atomic.Int64
}

// Stats is a snapshot of service statistics.
type Stats struct {
	RequestsTotal   int64
	RequestsActive  int64
	RequestsSuccess int64
	RequestsFailed  int64
	CacheHits       int64
	CacheMisses     int64
}

// =============================================================================
// Memory Stats Cache
// =============================================================================

// memStatsCache provides cached access to runtime memory statistics.
type memStatsCache struct {
	mu       sync.RWMutex
	stats    runtime.MemStats
	lastRead time.Time
	interval time.Duration
}

// newMemStatsCache creates a new memory stats cache with the specified refresh interval.
func newMemStatsCache(interval time.Duration) *memStatsCache {
	return &memStatsCache{
		interval: interval,
	}
}

// get returns the current memory allocation, using cached value if fresh enough.
func (m *memStatsCache) get() uint64 {
	m.mu.RLock()
	if time.Since(m.lastRead) < m.interval {
		alloc := m.stats.Alloc
		m.mu.RUnlock()
		return alloc
	}
	m.mu.RUnlock()

	return m.refresh()
}

// refresh forces a memory stats read and returns the current allocation.
func (m *memStatsCache) refresh() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(m.lastRead) < m.interval {
		return m.stats.Alloc
	}

	runtime.ReadMemStats(&m.stats)
	m.lastRead = time.Now()
	return m.stats.Alloc
}

// =============================================================================
// SolverService
// =============================================================================

// SolverService implements the gRPC SolverService for network flow optimization.
//
// The service is safe for concurrent use from multiple goroutines.
type SolverService struct {
	optimizationv1.UnimplementedSolverServiceServer

	version     string
	metrics     *metrics.Metrics
	solverCache *cache.SolverCache
	config      *ServiceConfig

	graphPool  *graph.GraphPool
	solverPool *algorithms.SolverPool

	stats         serviceStats
	memStatsCache *memStatsCache

	// Shutdown coordination
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
	wg           sync.WaitGroup
}

// NewSolverService creates a new SolverService with default configuration.
func NewSolverService(version string, solverCache *cache.SolverCache) *SolverService {
	return NewSolverServiceWithConfig(version, solverCache, DefaultServiceConfig())
}

// NewSolverServiceWithConfig creates a new SolverService with custom configuration.
func NewSolverServiceWithConfig(version string, solverCache *cache.SolverCache, config *ServiceConfig) *SolverService {
	if config == nil {
		config = DefaultServiceConfig()
	}

	return &SolverService{
		version:       version,
		metrics:       metrics.Get(),
		solverCache:   solverCache,
		config:        config,
		graphPool:     graph.GetPool(),
		solverPool:    algorithms.NewSolverPool(config.MaxConcurrentSolves),
		memStatsCache: newMemStatsCache(config.MemStatsInterval),
		shutdownCh:    make(chan struct{}),
	}
}

// =============================================================================
// Main Solve Method
// =============================================================================

// Solve handles a synchronous flow optimization request.
//
// The method:
//  1. Checks cache for existing result
//  2. Validates the request
//  3. Acquires a solver slot (with backpressure)
//  4. Runs the requested algorithm
//  5. Caches the result asynchronously
//  6. Returns the solution
//
// Thread-safe: can be called concurrently from multiple goroutines.
func (s *SolverService) Solve(ctx context.Context, req *optimizationv1.SolveRequest) (*optimizationv1.SolveResponse, error) {
	// Track request lifecycle
	if err := s.trackRequest(); err != nil {
		return nil, err
	}
	defer s.untrackRequest()

	// Start tracing span
	ctx, span := telemetry.StartSpan(ctx, "SolverService.Solve",
		trace.WithAttributes(
			attribute.String("algorithm", req.Algorithm.String()),
		),
	)
	defer span.End()

	// Check cache first
	if cached, found := s.checkCache(ctx, req, span); found {
		return cached, nil
	}

	// Validate request
	if err := s.validateSolveRequest(req); err != nil {
		s.stats.requestsFailed.Add(1)
		telemetry.SetError(ctx, err)
		return nil, err
	}

	// Build options and create context with timeout
	opts := s.buildSolverOptions(req.Options)
	ctx, cancel := s.createTimeoutContext(ctx, opts)
	defer cancel()

	// Execute solve operation
	return s.executeSolve(ctx, req, opts, span)
}

// trackRequest registers a new request and checks shutdown status.
func (s *SolverService) trackRequest() error {
	select {
	case <-s.shutdownCh:
		return status.Error(codes.Unavailable, "service is shutting down")
	default:
	}

	s.wg.Add(1)
	s.stats.requestsTotal.Add(1)
	s.stats.requestsActive.Add(1)
	return nil
}

// untrackRequest decrements the active request counter.
func (s *SolverService) untrackRequest() {
	s.stats.requestsActive.Add(-1)
	s.wg.Done()
}

// checkCache attempts to retrieve a cached result.
func (s *SolverService) checkCache(ctx context.Context, req *optimizationv1.SolveRequest, span trace.Span) (*optimizationv1.SolveResponse, bool) {
	if s.solverCache == nil {
		return nil, false
	}

	cached, found, err := s.solverCache.Get(ctx, req.Graph, req.Algorithm)
	if err != nil || !found {
		s.stats.cacheMisses.Add(1)
		span.SetAttributes(attribute.Bool("cache_hit", false))
		return nil, false
	}

	s.stats.cacheHits.Add(1)
	span.SetAttributes(attribute.Bool("cache_hit", true))
	telemetry.AddEvent(ctx, "cache_hit",
		attribute.Float64("max_flow", cached.MaxFlow),
	)

	return &optimizationv1.SolveResponse{
		Success: true,
		Result:  cached.ToFlowResult(),
		Metrics: &optimizationv1.SolveMetrics{
			ComputationTimeMs: 0,
		},
	}, true
}

// createTimeoutContext creates a context with the appropriate timeout.
func (s *SolverService) createTimeoutContext(ctx context.Context, opts *algorithms.SolverOptions) (context.Context, context.CancelFunc) {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = s.config.DefaultTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// executeSolve runs the actual solve operation.
func (s *SolverService) executeSolve(ctx context.Context, req *optimizationv1.SolveRequest, opts *algorithms.SolverOptions, span trace.Span) (*optimizationv1.SolveResponse, error) {
	start := time.Now()

	// Track memory before
	var memBefore uint64
	if s.config.EnableMemoryTracking {
		memBefore = s.memStatsCache.refresh()
	}

	// Acquire solver slot
	if err := s.solverPool.Acquire(ctx); err != nil {
		s.stats.requestsFailed.Add(1)
		if ctx.Err() == context.DeadlineExceeded {
			return nil, status.Error(codes.DeadlineExceeded, "timeout waiting for solver slot")
		}
		return nil, status.Error(codes.ResourceExhausted, "too many concurrent requests")
	}
	defer s.solverPool.Release()

	// Convert and solve
	rg := converter.ToResidualGraph(req.Graph)
	result := algorithms.Solve(ctx, rg, req.Graph.SourceId, req.Graph.SinkId, req.Algorithm, opts)

	elapsed := time.Since(start)

	// Calculate memory used
	var memUsed int64
	if s.config.EnableMemoryTracking {
		memAfter := s.memStatsCache.refresh()
		if memAfter > memBefore {
			memUsed = int64(memAfter - memBefore)
		}
	}

	// Handle errors
	if result.Error != nil {
		return s.handleSolveError(ctx, result, elapsed)
	}

	// Build successful response
	return s.buildSuccessResponse(ctx, req, rg, result, opts, elapsed, memUsed, span)
}

// handleSolveError processes a failed solve result.
func (s *SolverService) handleSolveError(ctx context.Context, result *algorithms.SolverResult, elapsed time.Duration) (*optimizationv1.SolveResponse, error) {
	s.stats.requestsFailed.Add(1)
	telemetry.SetError(ctx, result.Error)

	if result.Status == commonv1.FlowStatus_FLOW_STATUS_ERROR {
		return nil, status.Error(codes.DeadlineExceeded, "computation timeout")
	}

	return &optimizationv1.SolveResponse{
		Success:      false,
		ErrorMessage: result.Error.Error(),
		Metrics: &optimizationv1.SolveMetrics{
			ComputationTimeMs: float64(elapsed.Milliseconds()),
		},
	}, nil
}

// buildSuccessResponse constructs the response for a successful solve.
func (s *SolverService) buildSuccessResponse(
	ctx context.Context,
	req *optimizationv1.SolveRequest,
	rg *graph.ResidualGraph,
	result *algorithms.SolverResult,
	opts *algorithms.SolverOptions,
	elapsed time.Duration,
	memUsed int64,
	span trace.Span,
) (*optimizationv1.SolveResponse, error) {
	s.stats.requestsSuccess.Add(1)

	flowResult := &commonv1.FlowResult{
		MaxFlow:           result.MaxFlow,
		TotalCost:         result.TotalCost,
		Edges:             converter.ToFlowEdges(rg),
		Status:            result.Status,
		Iterations:        int32(result.Iterations),
		ComputationTimeMs: float64(elapsed.Milliseconds()),
	}

	if opts.ReturnPaths && len(result.Paths) > 0 {
		flowResult.Paths = converter.ToPaths(result.Paths, rg)
	}

	// Cache result asynchronously
	s.cacheResultAsync(req.Graph, req.Algorithm, flowResult)

	// Record metrics
	if s.metrics != nil {
		s.metrics.RecordSolveOperation(
			req.Algorithm.String(),
			true,
			elapsed,
			result.MaxFlow,
		)
	}

	span.SetAttributes(
		attribute.Float64("max_flow", result.MaxFlow),
		attribute.Int("iterations", result.Iterations),
	)

	return &optimizationv1.SolveResponse{
		Success:     true,
		Result:      flowResult,
		SolvedGraph: converter.UpdateGraphWithFlow(req.Graph, rg),
		Metrics: &optimizationv1.SolveMetrics{
			ComputationTimeMs:    float64(elapsed.Milliseconds()),
			Iterations:           int32(result.Iterations),
			AugmentingPathsFound: int32(len(result.Paths)),
			MemoryUsedBytes:      memUsed,
		},
	}, nil
}

// cacheResultAsync asynchronously caches the solve result.
// The goroutine is properly tracked for graceful shutdown.
func (s *SolverService) cacheResultAsync(g *commonv1.Graph, algorithm commonv1.Algorithm, result *commonv1.FlowResult) {
	if s.solverCache == nil {
		return
	}

	// Check shutdown before spawning goroutine
	select {
	case <-s.shutdownCh:
		return
	default:
	}

	// Track this goroutine in WaitGroup
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), CacheOperationTimeout)
		defer cancel()

		// Check shutdown again inside goroutine
		select {
		case <-s.shutdownCh:
			return
		default:
		}

		response := &optimizationv1.SolveResponse{Result: result}
		if err := s.solverCache.SetFromResponse(ctx, g, algorithm, response, 0); err != nil {
			logger.Log.Warn("Failed to cache solve result", "error", err)
		}
	}()
}

// =============================================================================
// Streaming Solve
// =============================================================================

// SolveStream handles a streaming flow optimization request with progress updates.
//
// Progress updates are sent periodically as the algorithm runs, allowing clients
// to monitor long-running computations and display real-time status.
func (s *SolverService) SolveStream(req *optimizationv1.SolveRequestForBigGraphs, stream optimizationv1.SolverService_SolveStreamServer) error {
	// Track request lifecycle
	if err := s.trackRequest(); err != nil {
		return err
	}
	defer s.untrackRequest()

	ctx := stream.Context()
	ctx, span := telemetry.StartSpan(ctx, "SolverService.SolveStream",
		trace.WithAttributes(
			attribute.String("algorithm", req.Algorithm.String()),
			attribute.Int("nodes", len(req.Graph.Nodes)),
			attribute.Int("edges", len(req.Graph.Edges)),
		),
	)
	defer span.End()

	// Validate request
	if err := s.validateStreamRequest(req); err != nil {
		s.stats.requestsFailed.Add(1)
		telemetry.SetError(ctx, err)
		return err
	}

	// Build options and create context
	opts := s.buildSolverOptions(req.Options)
	ctx, cancel := s.createTimeoutContext(ctx, opts)
	defer cancel()

	// Acquire solver slot
	if err := s.solverPool.Acquire(ctx); err != nil {
		s.stats.requestsFailed.Add(1)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return status.Error(codes.ResourceExhausted, "too many concurrent requests")
	}
	defer s.solverPool.Release()

	// Execute streaming solve
	return s.executeStreamSolve(ctx, req, opts, stream)
}

// executeStreamSolve runs the streaming solve operation.
func (s *SolverService) executeStreamSolve(
	ctx context.Context,
	req *optimizationv1.SolveRequestForBigGraphs,
	opts *algorithms.SolverOptions,
	stream optimizationv1.SolverService_SolveStreamServer,
) error {
	start := time.Now()
	progress := newProgressTracker(stream, start, s.memStatsCache)

	// Check context before starting
	if ctx.Err() != nil {
		s.stats.requestsFailed.Add(1)
		return ctx.Err()
	}

	// Send initial progress
	if err := progress.sendStatus("starting", 0, 0); err != nil {
		s.stats.requestsFailed.Add(1)
		return err
	}

	rg := converter.ToResidualGraph(req.Graph)
	source := req.Graph.SourceId
	sink := req.Graph.SinkId

	// Run algorithm with progress callback
	var err error
	switch req.Algorithm {
	case commonv1.Algorithm_ALGORITHM_DINIC:
		err = s.streamDinic(ctx, rg, source, sink, opts, progress)
	case commonv1.Algorithm_ALGORITHM_PUSH_RELABEL:
		err = s.streamPushRelabel(ctx, rg, source, sink, opts, progress)
	case commonv1.Algorithm_ALGORITHM_MIN_COST:
		err = s.streamMinCostFlow(ctx, rg, source, sink, opts, progress)
	default:
		err = s.streamEdmondsKarp(ctx, rg, source, sink, opts, progress)
	}

	if err != nil {
		s.stats.requestsFailed.Add(1)
		// Return context errors directly (Canceled or DeadlineExceeded)
		// This allows callers to distinguish between cancellation and timeout
		return err
	}

	s.stats.requestsSuccess.Add(1)

	// Send final progress
	maxFlow := rg.GetTotalFlow(source)
	telemetry.AddEvent(ctx, "stream_completed", attribute.Float64("max_flow", maxFlow))

	return progress.sendCompleted(maxFlow)
}

// =============================================================================
// Progress Tracker
// =============================================================================

// progressTracker manages progress updates for streaming operations.
type progressTracker struct {
	stream        optimizationv1.SolverService_SolveStreamServer
	start         time.Time
	lastSendTime  time.Time
	memStatsCache *memStatsCache
}

// newProgressTracker creates a new progress tracker.
func newProgressTracker(stream optimizationv1.SolverService_SolveStreamServer, start time.Time, memCache *memStatsCache) *progressTracker {
	return &progressTracker{
		stream:        stream,
		start:         start,
		lastSendTime:  start,
		memStatsCache: memCache,
	}
}

// sendProgress sends a progress update if enough time has passed.
func (p *progressTracker) sendProgress(iteration int, currentFlow float64, path []int64, pathFlow float64) error {
	if time.Since(p.lastSendTime) < StreamProgressInterval && iteration > 1 {
		return nil
	}

	return p.sendProgressForced(iteration, currentFlow, "running", path, pathFlow)
}

// sendProgressWithCost sends a progress update including cost information.
func (p *progressTracker) sendProgressWithCost(iteration int, currentFlow, currentCost float64, path []int64, pathFlow float64) error {
	if time.Since(p.lastSendTime) < StreamProgressInterval && iteration > 1 {
		return nil
	}

	status := fmt.Sprintf("running (cost: %.2f)", currentCost)
	return p.sendProgressForced(iteration, currentFlow, status, path, pathFlow)
}

// sendProgressForced sends a progress update regardless of throttling.
func (p *progressTracker) sendProgressForced(iteration int, currentFlow float64, statusMsg string, path []int64, pathFlow float64) error {
	progress := &optimizationv1.SolveProgress{
		Iteration:         int32(iteration),
		CurrentFlow:       currentFlow,
		Status:            statusMsg,
		ComputationTimeMs: float64(time.Since(p.start).Milliseconds()),
		MemoryUsedBytes:   int64(p.memStatsCache.get()),
	}

	if len(path) > 0 {
		progress.LastPath = &commonv1.Path{
			NodeIds: path,
			Flow:    pathFlow,
		}
	}

	if err := p.stream.Send(progress); err != nil {
		return err
	}

	p.lastSendTime = time.Now()
	return nil
}

// sendStatus sends a status-only progress update.
func (p *progressTracker) sendStatus(statusMsg string, iteration int, currentFlow float64) error {
	return p.stream.Send(&optimizationv1.SolveProgress{
		Iteration:         int32(iteration),
		CurrentFlow:       currentFlow,
		Status:            statusMsg,
		ProgressPercent:   0,
		ComputationTimeMs: float64(time.Since(p.start).Milliseconds()),
		MemoryUsedBytes:   int64(p.memStatsCache.get()),
	})
}

// sendCompleted sends the final completion progress.
func (p *progressTracker) sendCompleted(maxFlow float64) error {
	return p.stream.Send(&optimizationv1.SolveProgress{
		CurrentFlow:       maxFlow,
		ProgressPercent:   100.0,
		Status:            "completed",
		ComputationTimeMs: float64(time.Since(p.start).Milliseconds()),
		MemoryUsedBytes:   int64(p.memStatsCache.get()),
	})
}

// =============================================================================
// Streaming Algorithm Implementations
// =============================================================================

// streamEdmondsKarp runs Edmonds-Karp with progress updates.
func (s *SolverService) streamEdmondsKarp(
	ctx context.Context,
	rg *graph.ResidualGraph,
	source, sink int64,
	opts *algorithms.SolverOptions,
	progress *progressTracker,
) error {
	maxFlow := 0.0
	iteration := 0

	for {
		// Check context periodically
		if iteration%ContextCheckInterval == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		// Check iteration limit
		if opts.MaxIterations > 0 && iteration >= opts.MaxIterations {
			break
		}

		// Find augmenting path
		bfsResult := graph.BFSDeterministic(rg, source, sink)
		if !bfsResult.Found {
			break
		}

		path := domain.ReconstructPath(bfsResult.Parent, source, sink)
		if len(path) == 0 {
			break
		}

		pathFlow := graph.FindMinCapacityOnPath(rg, path)
		if pathFlow <= opts.Epsilon {
			break
		}

		graph.AugmentPath(rg, path, pathFlow)
		maxFlow += pathFlow
		iteration++

		// Send progress
		if err := progress.sendProgress(iteration, maxFlow, path, pathFlow); err != nil {
			return err
		}
	}

	return nil
}

// streamDinic runs Dinic's algorithm with progress updates.
func (s *SolverService) streamDinic(
	ctx context.Context,
	rg *graph.ResidualGraph,
	source, sink int64,
	opts *algorithms.SolverOptions,
	progress *progressTracker,
) error {
	maxFlow := 0.0
	iteration := 0

	for {
		// Check context
		if iteration%ContextCheckInterval == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		// Check iteration limit
		if opts.MaxIterations > 0 && iteration >= opts.MaxIterations {
			break
		}

		// Build level graph
		level := graph.BFSLevel(rg, source)
		if _, exists := level[sink]; !exists {
			break
		}

		// Find blocking flow
		currentArc := make(map[int64]int)
		phaseFlow := 0.0

		for {
			flow, path := s.dinicDFSIterative(rg, source, sink, level, currentArc, opts.Epsilon)
			if flow <= opts.Epsilon {
				break
			}

			maxFlow += flow
			phaseFlow += flow
			iteration++

			// Send progress
			if err := progress.sendProgress(iteration, maxFlow, path, flow); err != nil {
				return err
			}

			// Check context after each path
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		if phaseFlow <= opts.Epsilon {
			break
		}
	}

	return nil
}

// dinicDFSIterative performs iterative DFS for Dinic's algorithm.
func (s *SolverService) dinicDFSIterative(
	rg *graph.ResidualGraph,
	source, sink int64,
	level map[int64]int,
	currentArc map[int64]int,
	epsilon float64,
) (float64, []int64) {
	path := make([]int64, 0, 64)
	minCap := make([]float64, 0, 64)

	path = append(path, source)
	minCap = append(minCap, graph.Infinity)

	for len(path) > 0 {
		u := path[len(path)-1]

		if u == sink {
			bottleneck := minCap[len(minCap)-1]

			// Augment flow
			for i := 0; i < len(path)-1; i++ {
				rg.UpdateFlow(path[i], path[i+1], bottleneck)
			}

			result := make([]int64, len(path))
			copy(result, path)
			return bottleneck, result
		}

		edges := rg.GetNeighborsList(u)
		startIdx := currentArc[u]

		advanced := false
		for i := startIdx; i < len(edges); i++ {
			edge := edges[i]
			v := edge.To

			if level[v] != level[u]+1 || edge.Capacity <= epsilon {
				continue
			}

			currentArc[u] = i

			newMinCap := minCap[len(minCap)-1]
			if edge.Capacity < newMinCap {
				newMinCap = edge.Capacity
			}

			path = append(path, v)
			minCap = append(minCap, newMinCap)

			advanced = true
			break
		}

		if !advanced {
			currentArc[u] = len(edges)
			delete(level, u)
			path = path[:len(path)-1]
			minCap = minCap[:len(minCap)-1]
		}
	}

	return 0, nil
}

// streamPushRelabel runs Push-Relabel with periodic progress updates.
func (s *SolverService) streamPushRelabel(
	ctx context.Context,
	rg *graph.ResidualGraph,
	source, sink int64,
	opts *algorithms.SolverOptions,
	progress *progressTracker,
) error {
	// Run algorithm in goroutine with periodic progress updates
	resultCh := make(chan *algorithms.PushRelabelResult, 1)

	go func() {
		result := algorithms.PushRelabelWithContext(ctx, rg, source, sink, opts)
		resultCh <- result
	}()

	progressTicker := time.NewTicker(StreamProgressInterval)
	defer progressTicker.Stop()

	iteration := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case result := <-resultCh:
			if result.Canceled {
				return ctx.Err()
			}
			return nil

		case <-progressTicker.C:
			iteration++
			currentFlow := rg.GetTotalFlow(source)
			if err := progress.sendStatus("running (push-relabel)", iteration, currentFlow); err != nil {
				return err
			}
		}
	}
}

// streamMinCostFlow runs Min-Cost Flow with progress updates.
func (s *SolverService) streamMinCostFlow(
	ctx context.Context,
	rg *graph.ResidualGraph,
	source, sink int64,
	opts *algorithms.SolverOptions,
	progress *progressTracker,
) error {
	// First, find max flow to determine required flow
	cloned := rg.Clone()
	ekResult := algorithms.EdmondsKarpWithContext(ctx, cloned, source, sink, opts)
	if ekResult.Canceled {
		return ctx.Err()
	}
	requiredFlow := ekResult.MaxFlow

	// Initialize potentials
	nodes := rg.GetSortedNodes()
	potentials := make(map[int64]float64, len(nodes))
	for _, node := range nodes {
		potentials[node] = 0
	}

	initResult := algorithms.BellmanFordWithContext(ctx, rg, source)
	if initResult.Canceled {
		return ctx.Err()
	}

	if !initResult.HasNegativeCycle {
		for _, node := range nodes {
			if initResult.Distances[node] < graph.Infinity-graph.Epsilon {
				potentials[node] = initResult.Distances[node]
			}
		}
	}

	totalFlow := 0.0
	totalCost := 0.0
	iteration := 0
	firstIteration := true

	for totalFlow < requiredFlow-opts.Epsilon {
		// Check context
		if iteration%ContextCheckInterval == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		// Check iteration limit
		if opts.MaxIterations > 0 && iteration >= opts.MaxIterations {
			break
		}

		var distances map[int64]float64
		var parent map[int64]int64
		var shouldUpdatePotentials bool

		if firstIteration {
			distances = initResult.Distances
			parent = initResult.Parent
			shouldUpdatePotentials = false
			firstIteration = false
		} else {
			dijkstraResult := algorithms.DijkstraWithPotentialsContext(ctx, rg, source, potentials)
			if dijkstraResult.Canceled {
				return ctx.Err()
			}
			distances = dijkstraResult.Distances
			parent = dijkstraResult.Parent
			shouldUpdatePotentials = true
		}

		if distances[sink] >= graph.Infinity-graph.Epsilon {
			break
		}

		// Update potentials
		if shouldUpdatePotentials {
			for _, node := range nodes {
				if distances[node] < graph.Infinity-graph.Epsilon {
					potentials[node] += distances[node]
				}
			}
		}

		path := graph.ReconstructPath(parent, source, sink)
		if len(path) == 0 {
			break
		}

		pathFlow := requiredFlow - totalFlow
		pathFlow = min(pathFlow, graph.FindMinCapacityOnPath(rg, path))
		if pathFlow <= opts.Epsilon {
			break
		}

		// Compute path cost
		pathCost := 0.0
		for i := 0; i < len(path)-1; i++ {
			edge := rg.GetEdge(path[i], path[i+1])
			if edge != nil {
				pathCost += edge.Cost * pathFlow
			}
		}

		graph.AugmentPath(rg, path, pathFlow)

		totalFlow += pathFlow
		totalCost += pathCost
		iteration++

		// Send progress with cost
		if err := progress.sendProgressWithCost(iteration, totalFlow, totalCost, path, pathFlow); err != nil {
			return err
		}
	}

	return nil
}

// =============================================================================
// Algorithm Information
// =============================================================================

// GetAlgorithms returns metadata about all available algorithms.
func (s *SolverService) GetAlgorithms(ctx context.Context, _ *emptypb.Empty) (*optimizationv1.GetAlgorithmsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SolverService.GetAlgorithms")
	defer span.End()

	infos := algorithms.GetAllAlgorithms()
	protoInfos := make([]*optimizationv1.AlgorithmInfo, 0, len(infos))

	for _, info := range infos {
		protoInfos = append(protoInfos, &optimizationv1.AlgorithmInfo{
			Algorithm:             info.Algorithm,
			Name:                  info.Name,
			Description:           info.Description,
			TimeComplexity:        info.TimeComplexity,
			SpaceComplexity:       info.SpaceComplexity,
			SupportsMinCost:       info.SupportsMinCost,
			SupportsNegativeCosts: info.SupportsNegativeCosts,
			BestFor:               info.BestFor,
		})
	}

	span.SetAttributes(attribute.Int("algorithms_count", len(protoInfos)))

	return &optimizationv1.GetAlgorithmsResponse{
		Algorithms: protoInfos,
	}, nil
}

// =============================================================================
// Health and Statistics
// =============================================================================

// GetStats returns current service statistics.
func (s *SolverService) GetStats() Stats {
	return Stats{
		RequestsTotal:   s.stats.requestsTotal.Load(),
		RequestsActive:  s.stats.requestsActive.Load(),
		RequestsSuccess: s.stats.requestsSuccess.Load(),
		RequestsFailed:  s.stats.requestsFailed.Load(),
		CacheHits:       s.stats.cacheHits.Load(),
		CacheMisses:     s.stats.cacheMisses.Load(),
	}
}

// IsHealthy returns true if the service is operational.
func (s *SolverService) IsHealthy() bool {
	select {
	case <-s.shutdownCh:
		return false
	default:
		return true
	}
}

// IsReady returns true if the service can accept new requests.
// Returns false if at 90% capacity or shutting down.
func (s *SolverService) IsReady() bool {
	if !s.IsHealthy() {
		return false
	}

	active := s.stats.requestsActive.Load()
	maxConcurrent := int64(s.config.MaxConcurrentSolves)

	return active < (maxConcurrent * 9 / 10)
}

// =============================================================================
// Shutdown
// =============================================================================

// Shutdown gracefully stops the service, waiting for in-flight requests.
//
// The method:
//  1. Closes the shutdown channel to prevent new requests
//  2. Waits for all in-flight requests to complete
//  3. Returns when all requests are done or context is cancelled
//
// Thread-safe: can be called from any goroutine, but only the first call has effect.
func (s *SolverService) Shutdown(ctx context.Context) error {
	var err error

	s.shutdownOnce.Do(func() {
		close(s.shutdownCh)

		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			logger.Log.Info("All requests completed gracefully")
		case <-ctx.Done():
			err = ctx.Err()
			logger.Log.Warn("Shutdown timeout, some requests may be interrupted",
				"active_requests", s.stats.requestsActive.Load(),
			)
		}
	})

	return err
}

// =============================================================================
// Validation
// =============================================================================

// validateSolveRequest validates a synchronous solve request.
func (s *SolverService) validateSolveRequest(req *optimizationv1.SolveRequest) error {
	return s.validateGraph(req.Graph)
}

// validateStreamRequest validates a streaming solve request.
func (s *SolverService) validateStreamRequest(req *optimizationv1.SolveRequestForBigGraphs) error {
	return s.validateGraph(req.Graph)
}

// validateGraph performs comprehensive graph validation.
func (s *SolverService) validateGraph(g *commonv1.Graph) error {
	if g == nil {
		return pkgerrors.ErrNilGraph
	}

	if len(g.Nodes) == 0 {
		return pkgerrors.ErrEmptyGraph
	}

	if len(g.Nodes) > MaxGraphNodes {
		return status.Errorf(codes.InvalidArgument,
			"graph has too many nodes: %d > %d", len(g.Nodes), MaxGraphNodes)
	}

	if len(g.Edges) > MaxGraphEdges {
		return status.Errorf(codes.InvalidArgument,
			"graph has too many edges: %d > %d", len(g.Edges), MaxGraphEdges)
	}

	// Build node set
	nodeExists := make(map[int64]bool, len(g.Nodes))
	for _, node := range g.Nodes {
		nodeExists[node.Id] = true
	}

	if !nodeExists[g.SourceId] {
		return pkgerrors.ErrInvalidSource
	}

	if !nodeExists[g.SinkId] {
		return pkgerrors.ErrInvalidSink
	}

	if g.SourceId == g.SinkId {
		return pkgerrors.ErrSourceEqualsSink
	}

	return nil
}

// =============================================================================
// Options Building
// =============================================================================

// buildSolverOptions creates algorithm options from request options.
func (s *SolverService) buildSolverOptions(opts *optimizationv1.SolveOptions) *algorithms.SolverOptions {
	result := algorithms.DefaultSolverOptions()

	if opts == nil {
		return result
	}

	// Epsilon with bounds validation
	if opts.Epsilon > 0 {
		epsilon := opts.Epsilon
		if epsilon < MinEpsilon {
			epsilon = MinEpsilon
		} else if epsilon > MaxEpsilon {
			epsilon = MaxEpsilon
		}
		result.Epsilon = epsilon
	}

	// MaxIterations with minimum
	if opts.MaxIterations > 0 {
		maxIter := int(opts.MaxIterations)
		if maxIter < MinIterations {
			maxIter = MinIterations
		}
		result.MaxIterations = maxIter
	}

	// Timeout with bounds validation
	if opts.TimeoutSeconds > 0 {
		timeout := opts.TimeoutSeconds
		if timeout < MinTimeoutSeconds {
			timeout = MinTimeoutSeconds
		} else if timeout > MaxTimeoutSeconds {
			timeout = MaxTimeoutSeconds
		}
		result.Timeout = time.Duration(timeout * float64(time.Second))
	}

	result.ReturnPaths = opts.ReturnPaths

	return result
}
