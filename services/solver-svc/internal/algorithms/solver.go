// Package algorithms provides implementations of various network flow algorithms
// including max-flow algorithms (Ford-Fulkerson, Edmonds-Karp, Dinic, Push-Relabel)
// and min-cost max-flow algorithms (Successive Shortest Path, Capacity Scaling).
//
// # Thread Safety
//
// Individual algorithm functions are NOT thread-safe. Each goroutine should work
// with its own copy of the graph. Use ResidualGraph.Clone() or the SolverPool
// for concurrent operations.
//
// # Determinism
//
// All algorithms are designed to produce deterministic results when given the same
// input graph. This is achieved by iterating over nodes and edges in sorted order.
//
// # Context Support
//
// All algorithms support context cancellation for timeout and graceful shutdown.
// The XxxWithContext variants should be preferred for production use.
//
// # Example Usage
//
//	g := graph.NewResidualGraph()
//	g.AddEdgeWithReverse(1, 2, 10, 0)
//	g.AddEdgeWithReverse(2, 3, 5, 0)
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	result := algorithms.Solve(ctx, g, 1, 3, commonv1.Algorithm_ALGORITHM_DINIC, nil)
//	if result.Error != nil {
//	    log.Printf("Error: %v", result.Error)
//	} else {
//	    log.Printf("Max flow: %f", result.MaxFlow)
//	}
package algorithms

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/services/solver-svc/internal/converter"
	"logistics/services/solver-svc/internal/graph"
)

// =============================================================================
// Error Definitions
// =============================================================================

// Standard errors returned by solver operations.
// These errors can be checked using errors.Is() for robust error handling.
var (
	// ErrNilGraph indicates that a nil graph was passed to a solver function.
	ErrNilGraph = errors.New("graph is nil")

	// ErrSourceNotFound indicates that the source node does not exist in the graph.
	ErrSourceNotFound = errors.New("source node not in graph")

	// ErrSinkNotFound indicates that the sink node does not exist in the graph.
	ErrSinkNotFound = errors.New("sink node not in graph")

	// ErrSourceEqualSink indicates that source and sink are the same node.
	ErrSourceEqualSink = errors.New("source equals sink")

	// ErrContextCanceled indicates that the operation was cancelled via context.
	ErrContextCanceled = errors.New("context canceled")

	// ErrTimeout indicates that the operation exceeded the configured timeout.
	ErrTimeout = errors.New("operation timeout")
)

// =============================================================================
// Solver Options
// =============================================================================

// SolverOptions configures the behavior of flow algorithms.
//
// Zero values are safe to use - DefaultSolverOptions() will be applied.
// Options can be chained using the builder pattern:
//
//	opts := DefaultSolverOptions().
//	    WithTimeout(10 * time.Second).
//	    WithPool(customPool)
type SolverOptions struct {
	// Epsilon is the tolerance for floating-point comparisons.
	// Values smaller than Epsilon are considered zero.
	// Default: graph.Epsilon (1e-9)
	Epsilon float64

	// MaxIterations limits the number of augmenting path iterations.
	// Zero or negative means unlimited.
	// Default: 0 (unlimited)
	MaxIterations int

	// Timeout sets the maximum duration for the algorithm.
	// Zero means no timeout (relies on context).
	// Default: 30 seconds
	Timeout time.Duration

	// ReturnPaths indicates whether to collect and return individual flow paths.
	// Enabling this increases memory usage proportional to the number of paths.
	// Default: false
	ReturnPaths bool

	// NegativeEdgeFallbackThreshold sets the number of negative reduced cost edges
	// encountered before falling back from Dijkstra to Bellman-Ford.
	// Default: 3
	NegativeEdgeFallbackThreshold int

	// Pool is the graph pool for memory reuse.
	// If nil, the global pool is used.
	Pool *graph.GraphPool

	// Resources holds pooled resources for this request.
	// Typically managed internally.
	Resources *graph.PooledResources
}

// DefaultSolverOptions returns options with sensible defaults for most use cases.
//
// Default values:
//   - Epsilon: 1e-9
//   - MaxIterations: unlimited
//   - Timeout: 30 seconds
//   - ReturnPaths: false
//   - NegativeEdgeFallbackThreshold: 3
func DefaultSolverOptions() *SolverOptions {
	return &SolverOptions{
		Epsilon:                       graph.Epsilon,
		MaxIterations:                 0,
		Timeout:                       30 * time.Second,
		ReturnPaths:                   false,
		NegativeEdgeFallbackThreshold: 3,
		Pool:                          graph.GetPool(),
	}
}

// WithPool sets the graph pool and returns the options for chaining.
func (o *SolverOptions) WithPool(pool *graph.GraphPool) *SolverOptions {
	o.Pool = pool
	return o
}

// WithTimeout sets the timeout and returns the options for chaining.
func (o *SolverOptions) WithTimeout(timeout time.Duration) *SolverOptions {
	o.Timeout = timeout
	return o
}

// WithReturnPaths enables path collection and returns the options for chaining.
func (o *SolverOptions) WithReturnPaths(returnPaths bool) *SolverOptions {
	o.ReturnPaths = returnPaths
	return o
}

// WithMaxIterations sets the iteration limit and returns the options for chaining.
func (o *SolverOptions) WithMaxIterations(max int) *SolverOptions {
	o.MaxIterations = max
	return o
}

// =============================================================================
// Solver Result
// =============================================================================

// SolverResult contains the complete result of a flow computation.
//
// Check Status and Error first to determine if the result is valid:
//
//	result := Solve(ctx, g, source, sink, algo, opts)
//	if result.Status != commonv1.FlowStatus_FLOW_STATUS_OPTIMAL {
//	    log.Printf("Failed: %v", result.Error)
//	    return
//	}
//	log.Printf("Max flow: %f, Cost: %f", result.MaxFlow, result.TotalCost)
type SolverResult struct {
	// MaxFlow is the maximum flow value found.
	MaxFlow float64

	// TotalCost is the total cost of the flow (for min-cost algorithms).
	TotalCost float64

	// Iterations is the number of augmenting path iterations performed.
	Iterations int

	// Paths contains individual flow paths if ReturnPaths was enabled.
	// Each path includes the sequence of node IDs and the flow amount.
	Paths []converter.PathWithFlow

	// Status indicates the outcome of the computation.
	Status commonv1.FlowStatus

	// Error contains any error that occurred during computation.
	// nil if Status is FLOW_STATUS_OPTIMAL.
	Error error

	// Duration is the wall-clock time taken by the algorithm.
	Duration time.Duration
}

// =============================================================================
// Validation
// =============================================================================

// validateGraph performs basic validation of the graph and source/sink nodes.
//
// Returns nil if the graph is valid, or a descriptive error otherwise.
// The error wraps one of the standard errors (ErrNilGraph, ErrSourceNotFound, etc.)
// for easy checking with errors.Is().
func validateGraph(g *graph.ResidualGraph, source, sink int64) error {
	if g == nil {
		return ErrNilGraph
	}
	if !g.Nodes[source] {
		return fmt.Errorf("%w: %d", ErrSourceNotFound, source)
	}
	if !g.Nodes[sink] {
		return fmt.Errorf("%w: %d", ErrSinkNotFound, sink)
	}
	if source == sink {
		return ErrSourceEqualSink
	}
	return nil
}

// =============================================================================
// Main Solver Entry Point
// =============================================================================

// Solve is the primary entry point for solving flow problems.
//
// It dispatches to the appropriate algorithm based on the algorithm parameter
// and handles context management, timeout, and error wrapping.
//
// # Parameters
//
//   - ctx: Context for cancellation and timeout. Must not be nil.
//   - g: The residual graph. Will be modified by the algorithm.
//   - source: The source node ID. Must exist in the graph.
//   - sink: The sink node ID. Must exist in the graph and differ from source.
//   - algorithm: The algorithm to use. ALGORITHM_UNSPECIFIED defaults to Dinic.
//   - options: Solver options. nil uses DefaultSolverOptions().
//
// # Algorithm Selection
//
//   - ALGORITHM_FORD_FULKERSON: Classic DFS-based algorithm. O(E × max_flow).
//   - ALGORITHM_EDMONDS_KARP: BFS-based. O(VE²). Good for general graphs.
//   - ALGORITHM_DINIC: Level graph + blocking flow. O(V²E). Best for most cases.
//   - ALGORITHM_PUSH_RELABEL: Preflow-push. O(V³) or O(V²√E). Best for dense graphs.
//   - ALGORITHM_MIN_COST: SSP or Capacity Scaling. Finds minimum cost max flow.
//
// # Thread Safety
//
// This function is NOT thread-safe. The graph g will be modified.
// For concurrent use, clone the graph first or use SolverPool.
//
// # Example
//
//	ctx := context.Background()
//	g := buildGraph()
//	result := Solve(ctx, g, 1, 100, commonv1.Algorithm_ALGORITHM_DINIC, nil)
//	if result.Error != nil {
//	    return fmt.Errorf("solve failed: %w", result.Error)
//	}
//	fmt.Printf("Max flow: %.2f\n", result.MaxFlow)
func Solve(ctx context.Context, g *graph.ResidualGraph, source, sink int64, algorithm commonv1.Algorithm, options *SolverOptions) *SolverResult {
	start := time.Now()

	if options == nil {
		options = DefaultSolverOptions()
	}

	// Validate input
	if err := validateGraph(g, source, sink); err != nil {
		return &SolverResult{
			Status:   commonv1.FlowStatus_FLOW_STATUS_ERROR,
			Error:    err,
			Duration: time.Since(start),
		}
	}

	// Create context with timeout if specified
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	result := solveInternal(ctx, g, source, sink, algorithm, options)
	result.Duration = time.Since(start)

	return result
}

// solveInternal dispatches to the appropriate algorithm implementation.
func solveInternal(ctx context.Context, g *graph.ResidualGraph, source, sink int64, algorithm commonv1.Algorithm, options *SolverOptions) *SolverResult {
	switch algorithm {
	case commonv1.Algorithm_ALGORITHM_EDMONDS_KARP:
		return solveEdmondsKarp(ctx, g, source, sink, options)

	case commonv1.Algorithm_ALGORITHM_DINIC:
		return solveDinic(ctx, g, source, sink, options)

	case commonv1.Algorithm_ALGORITHM_PUSH_RELABEL:
		return solvePushRelabel(ctx, g, source, sink, options)

	case commonv1.Algorithm_ALGORITHM_MIN_COST:
		return solveMinCost(ctx, g, source, sink, options)

	case commonv1.Algorithm_ALGORITHM_FORD_FULKERSON:
		return solveFordFulkerson(ctx, g, source, sink, options)

	default:
		// Default to Dinic as it has the best general performance
		return solveDinic(ctx, g, source, sink, options)
	}
}

// solveEdmondsKarp runs the Edmonds-Karp algorithm and wraps the result.
func solveEdmondsKarp(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *SolverResult {
	result := EdmondsKarpWithContext(ctx, g, source, sink, options)
	if result.Canceled {
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_ERROR,
			Error:      ErrContextCanceled,
		}
	}
	return &SolverResult{
		MaxFlow:    result.MaxFlow,
		TotalCost:  g.GetTotalCost(),
		Iterations: result.Iterations,
		Paths:      result.Paths,
		Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
	}
}

// solveDinic runs the Dinic algorithm and wraps the result.
func solveDinic(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *SolverResult {
	result := DinicWithContext(ctx, g, source, sink, options)
	if result.Canceled {
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_ERROR,
			Error:      ErrContextCanceled,
		}
	}
	return &SolverResult{
		MaxFlow:    result.MaxFlow,
		TotalCost:  g.GetTotalCost(),
		Iterations: result.Iterations,
		Paths:      result.Paths,
		Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
	}
}

// solvePushRelabel runs the Push-Relabel algorithm and wraps the result.
func solvePushRelabel(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *SolverResult {
	result := PushRelabelWithContext(ctx, g, source, sink, options)
	if result.Canceled {
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			Iterations: result.Iterations,
			Status:     commonv1.FlowStatus_FLOW_STATUS_ERROR,
			Error:      ErrContextCanceled,
		}
	}
	return &SolverResult{
		MaxFlow:    result.MaxFlow,
		TotalCost:  g.GetTotalCost(),
		Iterations: result.Iterations,
		Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
	}
}

// solveFordFulkerson runs the Ford-Fulkerson algorithm and wraps the result.
func solveFordFulkerson(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *SolverResult {
	result := FordFulkersonWithContext(ctx, g, source, sink, options)
	if result.Canceled {
		return &SolverResult{
			MaxFlow:    result.MaxFlow,
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_ERROR,
			Error:      ErrContextCanceled,
		}
	}
	return &SolverResult{
		MaxFlow:    result.MaxFlow,
		TotalCost:  g.GetTotalCost(),
		Iterations: result.Iterations,
		Paths:      result.Paths,
		Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
	}
}

// solveMinCost runs the min-cost max-flow algorithm and wraps the result.
//
// This function finds the maximum flow with minimum total cost. The algorithm
// automatically terminates when no augmenting path exists, so no pre-computation
// of the maximum flow value is needed.
func solveMinCost(ctx context.Context, g *graph.ResidualGraph, source, sink int64, options *SolverOptions) *SolverResult {
	// Pass Infinity as required flow — the algorithm will find the maximum
	// possible flow and stop when sink becomes unreachable
	result := MinCostMaxFlowWithContext(ctx, g, source, sink, math.MaxFloat64, options)

	if result.Canceled {
		return &SolverResult{
			MaxFlow:    result.Flow,
			TotalCost:  result.Cost,
			Iterations: result.Iterations,
			Paths:      result.Paths,
			Status:     commonv1.FlowStatus_FLOW_STATUS_ERROR,
			Error:      ErrContextCanceled,
		}
	}

	return &SolverResult{
		MaxFlow:    result.Flow,
		TotalCost:  result.Cost,
		Iterations: result.Iterations,
		Paths:      result.Paths,
		Status:     commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
	}
}

// =============================================================================
// Solver Pool
// =============================================================================

// SolverPool manages concurrent solver executions with resource pooling.
//
// It provides:
//   - Concurrency limiting to prevent resource exhaustion
//   - Graph pooling for memory reuse
//   - Automatic graph cloning for thread safety
//
// # Example
//
//	pool := NewSolverPool(runtime.NumCPU())
//
//	// Concurrent solving
//	var wg sync.WaitGroup
//	for _, task := range tasks {
//	    wg.Add(1)
//	    go func(t Task) {
//	        defer wg.Done()
//	        result := pool.SolvePooled(ctx, t.Graph, t.Source, t.Sink, t.Algo, nil)
//	        handleResult(result)
//	    }(task)
//	}
//	wg.Wait()
type SolverPool struct {
	graphPool *graph.GraphPool
	workers   chan struct{} // Semaphore for concurrency limiting
}

// NewSolverPool creates a new solver pool with the specified maximum concurrency.
//
// maxConcurrency limits the number of simultaneous solver executions.
// If maxConcurrency <= 0, it defaults to 10.
//
// The pool uses the global graph pool for memory reuse.
func NewSolverPool(maxConcurrency int) *SolverPool {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	return &SolverPool{
		graphPool: graph.GetPool(),
		workers:   make(chan struct{}, maxConcurrency),
	}
}

// Acquire obtains a worker slot from the pool.
//
// Blocks until a slot is available or the context is cancelled.
// Returns nil on success, or ctx.Err() if the context was cancelled.
//
// Call Release() when the work is complete.
func (sp *SolverPool) Acquire(ctx context.Context) error {
	select {
	case sp.workers <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release returns a worker slot to the pool.
//
// Must be called exactly once after each successful Acquire().
func (sp *SolverPool) Release() {
	<-sp.workers
}

// SolvePooled solves a flow problem using pooled resources.
//
// This method is thread-safe and will:
//  1. Acquire a worker slot (blocking if at capacity)
//  2. Clone the graph from the pool
//  3. Run the algorithm on the cloned graph
//  4. Release resources back to the pool
//
// The original graph g is NOT modified.
//
// # Parameters
//
// Same as Solve(), but the graph is cloned internally.
//
// # Returns
//
// SolverResult with the computation results. On context cancellation during
// slot acquisition, returns an error result.
func (sp *SolverPool) SolvePooled(ctx context.Context, g *graph.ResidualGraph, source, sink int64, algorithm commonv1.Algorithm, options *SolverOptions) *SolverResult {
	if err := sp.Acquire(ctx); err != nil {
		return &SolverResult{
			Status: commonv1.FlowStatus_FLOW_STATUS_ERROR,
			Error:  err,
		}
	}
	defer sp.Release()

	// Clone graph from pool for thread safety
	cloned := g.CloneToPooled(sp.graphPool)
	defer sp.graphPool.ReleaseGraph(cloned)

	if options == nil {
		options = DefaultSolverOptions()
	}
	options.Pool = sp.graphPool

	return Solve(ctx, cloned, source, sink, algorithm, options)
}

// BatchSolve solves multiple flow problems in parallel.
//
// Tasks are executed concurrently up to the pool's concurrency limit.
// Results are returned in the same order as the input tasks.
//
// The method blocks until all tasks are complete or the context is cancelled.
//
// # Example
//
//	tasks := []BatchTask{
//	    {TaskID: "task1", Graph: g1, Source: 1, Sink: 10, Algorithm: algo},
//	    {TaskID: "task2", Graph: g2, Source: 1, Sink: 20, Algorithm: algo},
//	}
//	results := pool.BatchSolve(ctx, tasks)
//	for _, r := range results {
//	    fmt.Printf("Task %s: flow=%f\n", r.TaskID, r.Result.MaxFlow)
//	}
func (sp *SolverPool) BatchSolve(ctx context.Context, tasks []BatchTask) []BatchResult {
	results := make([]BatchResult, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t BatchTask) {
			defer wg.Done()
			result := sp.SolvePooled(ctx, t.Graph, t.Source, t.Sink, t.Algorithm, t.Options)
			results[idx] = BatchResult{
				TaskID: t.TaskID,
				Result: result,
			}
		}(i, task)
	}

	wg.Wait()
	return results
}

// BatchTask represents a single task for batch processing.
type BatchTask struct {
	// TaskID is a user-defined identifier for correlating results.
	TaskID string

	// Graph is the input graph. Will be cloned internally.
	Graph *graph.ResidualGraph

	// Source is the source node ID.
	Source int64

	// Sink is the sink node ID.
	Sink int64

	// Algorithm specifies which algorithm to use.
	Algorithm commonv1.Algorithm

	// Options for the solver. nil uses defaults.
	Options *SolverOptions
}

// BatchResult contains the result of a batch task.
type BatchResult struct {
	// TaskID matches the input BatchTask.TaskID.
	TaskID string

	// Result is the solver result for this task.
	Result *SolverResult
}

// =============================================================================
// Algorithm Information
// =============================================================================

// AlgorithmInfo provides metadata about a flow algorithm.
//
// Use GetAlgorithmInfo() or GetAllAlgorithms() to retrieve this information
// for displaying to users or for algorithm selection logic.
type AlgorithmInfo struct {
	// Algorithm is the algorithm enum value.
	Algorithm commonv1.Algorithm

	// Name is the human-readable name.
	Name string

	// Description is a brief description of the algorithm.
	Description string

	// TimeComplexity is the Big-O time complexity.
	TimeComplexity string

	// SpaceComplexity is the Big-O space complexity.
	SpaceComplexity string

	// SupportsMinCost indicates if the algorithm minimizes cost.
	SupportsMinCost bool

	// SupportsNegativeCosts indicates if the algorithm handles negative edge costs.
	SupportsNegativeCosts bool

	// BestFor lists scenarios where this algorithm excels.
	BestFor []string

	// Caveats lists potential issues or limitations.
	Caveats []string
}

// GetAlgorithmInfo returns detailed information about a specific algorithm.
//
// Returns nil for unknown algorithms.
func GetAlgorithmInfo(algo commonv1.Algorithm) *AlgorithmInfo {
	infos := map[commonv1.Algorithm]*AlgorithmInfo{
		commonv1.Algorithm_ALGORITHM_FORD_FULKERSON: {
			Algorithm:       commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
			Name:            "Ford-Fulkerson",
			Description:     "Classic augmenting path algorithm using DFS",
			TimeComplexity:  "O(E × max_flow)",
			SpaceComplexity: "O(V + E)",
			BestFor:         []string{"small_graphs", "integer_capacities", "educational"},
			Caveats: []string{
				"May be very slow for large max_flow values",
				"May not terminate for irrational capacities",
				"Consider using Edmonds-Karp or Dinic instead",
			},
		},
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP: {
			Algorithm:       commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			Name:            "Edmonds-Karp",
			Description:     "Ford-Fulkerson with BFS for shortest augmenting paths",
			TimeComplexity:  "O(V × E²)",
			SpaceComplexity: "O(V + E)",
			BestFor:         []string{"general_graphs", "small_to_medium_size"},
			Caveats:         []string{"Slower than Dinic for large graphs"},
		},
		commonv1.Algorithm_ALGORITHM_DINIC: {
			Algorithm:       commonv1.Algorithm_ALGORITHM_DINIC,
			Name:            "Dinic",
			Description:     "Level graphs with blocking flow optimization",
			TimeComplexity:  "O(V² × E)",
			SpaceComplexity: "O(V + E)",
			BestFor:         []string{"large_graphs", "unit_capacity_graphs", "bipartite_matching"},
			Caveats:         []string{},
		},
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL: {
			Algorithm:       commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
			Name:            "Push-Relabel (FIFO with Highest Label option)",
			Description:     "Preflow-push with FIFO/Highest Label vertex selection and gap heuristic",
			TimeComplexity:  "O(V³) for FIFO, O(V²√E) for Highest Label",
			SpaceComplexity: "O(V + E)",
			BestFor:         []string{"dense_graphs", "very_large_graphs"},
			Caveats:         []string{"More complex implementation", "Does not naturally produce paths"},
		},
		commonv1.Algorithm_ALGORITHM_MIN_COST: {
			Algorithm:             commonv1.Algorithm_ALGORITHM_MIN_COST,
			Name:                  "Min-Cost Max-Flow (SSP + Capacity Scaling)",
			Description:           "Successive Shortest Paths with potentials; auto-switches to Capacity Scaling for large capacities",
			TimeComplexity:        "O(V × E + V × E × log(V) × F) for SSP; O(E² log U) for Capacity Scaling",
			SpaceComplexity:       "O(V + E)",
			SupportsMinCost:       true,
			SupportsNegativeCosts: true,
			BestFor:               []string{"cost_optimization", "transportation_problems", "assignment_problems"},
			Caveats: []string{
				"Slower than pure max-flow algorithms",
				"Uses Dijkstra with fallback to Bellman-Ford for negative edges",
			},
		},
	}

	return infos[algo]
}

// GetAllAlgorithms returns information about all available algorithms.
//
// The returned slice is in a stable order suitable for display.
func GetAllAlgorithms() []*AlgorithmInfo {
	algorithms := []commonv1.Algorithm{
		commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
		commonv1.Algorithm_ALGORITHM_MIN_COST,
	}

	var infos []*AlgorithmInfo
	for _, algo := range algorithms {
		if info := GetAlgorithmInfo(algo); info != nil {
			infos = append(infos, info)
		}
	}
	return infos
}

// RecommendAlgorithm suggests the best algorithm based on graph characteristics.
//
// # Parameters
//
//   - nodeCount: Number of nodes in the graph.
//   - edgeCount: Number of edges in the graph.
//   - needMinCost: Whether minimum cost is required.
//   - hasNegativeCosts: Whether the graph has negative edge costs.
//
// # Returns
//
// The recommended algorithm enum value.
//
// # Recommendation Logic
//
//   - If min-cost is needed or graph has negative costs: MIN_COST
//   - If graph is dense (>50% edges) and large (>100 nodes): PUSH_RELABEL
//   - If graph is large (>100 nodes): DINIC
//   - Otherwise: EDMONDS_KARP
func RecommendAlgorithm(nodeCount, edgeCount int, needMinCost bool, hasNegativeCosts bool) commonv1.Algorithm {
	// Min-cost requirement takes priority
	if needMinCost || hasNegativeCosts {
		return commonv1.Algorithm_ALGORITHM_MIN_COST
	}

	// Calculate graph density
	maxEdges := nodeCount * (nodeCount - 1)
	if maxEdges == 0 {
		return commonv1.Algorithm_ALGORITHM_EDMONDS_KARP
	}

	density := float64(edgeCount) / float64(maxEdges)

	// Dense graphs with many nodes benefit from Push-Relabel
	if density > 0.5 && nodeCount > 100 {
		return commonv1.Algorithm_ALGORITHM_PUSH_RELABEL
	}

	// Large graphs benefit from Dinic
	if nodeCount > 100 {
		return commonv1.Algorithm_ALGORITHM_DINIC
	}

	// Default to Edmonds-Karp for small graphs
	return commonv1.Algorithm_ALGORITHM_EDMONDS_KARP
}
