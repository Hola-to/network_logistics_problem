// Package main is the entry point for the solver-svc microservice.
//
// solver-svc provides network flow optimization algorithms as a gRPC service.
// It implements various max-flow and min-cost flow algorithms for logistics
// network optimization problems.
//
// # Service Overview
//
// The solver service exposes the following capabilities via gRPC:
//   - Maximum flow computation (Ford-Fulkerson, Edmonds-Karp, Dinic, Push-Relabel)
//   - Minimum cost maximum flow (Successive Shortest Path, Capacity Scaling)
//   - Algorithm recommendation based on graph characteristics
//   - Batch processing for multiple flow problems
//   - Result caching for repeated queries
//
// # Architecture
//
// The service follows a clean architecture pattern with clear separation of concerns:
//
//	┌─────────────────────────────────────────────────────────────┐
//	│                     gRPC Transport Layer                    │
//	│  Interceptors: logging, metrics, tracing, rate-limit, audit│
//	├─────────────────────────────────────────────────────────────┤
//	│                      Service Layer                          │
//	│  (internal/service/solver.go - SolverService)               │
//	│  - Request validation                                       │
//	│  - Caching logic                                            │
//	│  - Algorithm dispatch                                       │
//	├─────────────────────────────────────────────────────────────┤
//	│                      Algorithm Layer                        │
//	│  (internal/algorithms/*.go)                                 │
//	│  - Ford-Fulkerson, Edmonds-Karp, Dinic                     │
//	│  - Push-Relabel (FIFO, Highest Label, Lowest Label)        │
//	│  - Min-Cost Flow (SSP, Capacity Scaling)                   │
//	│  - Bellman-Ford, Dijkstra (shortest paths)                 │
//	├─────────────────────────────────────────────────────────────┤
//	│                       Graph Layer                           │
//	│  (internal/graph/*.go)                                      │
//	│  - ResidualGraph: core data structure                       │
//	│  - GraphPool: memory pooling                                │
//	│  - BFS, path reconstruction utilities                       │
//	├─────────────────────────────────────────────────────────────┤
//	│                      Converter Layer                        │
//	│  (internal/converter/*.go)                                  │
//	│  - Proto ↔ internal type conversion                         │
//	│  - Result formatting                                        │
//	└─────────────────────────────────────────────────────────────┘
//
// # Configuration
//
// Configuration is loaded with the following priority (highest to lowest):
//  1. Environment variables (prefix: LOGISTICS_)
//  2. Config files (config.yaml, config/config.yaml, /etc/logistics/config.yaml)
//  3. Default values
//
// Key configuration options (environment variable format):
//
//	# Application
//	LOGISTICS_APP_NAME           - Service name (default: solver-svc)
//	LOGISTICS_APP_VERSION        - Service version (default: 1.0.0)
//	LOGISTICS_APP_ENVIRONMENT    - Environment: development, staging, production
//
//	# gRPC Server
//	LOGISTICS_GRPC_PORT              - gRPC server port (default: 50054)
//	LOGISTICS_GRPC_MAX_RECV_MSG_SIZE - Max receive message size in bytes (default: 16MB)
//	LOGISTICS_GRPC_MAX_SEND_MSG_SIZE - Max send message size in bytes (default: 16MB)
//	LOGISTICS_GRPC_MAX_CONCURRENT_CONN - Max concurrent connections (default: 1000)
//
//	# Logging
//	LOGISTICS_LOG_LEVEL    - Log level: debug, info, warn, error (default: info)
//	LOGISTICS_LOG_FORMAT   - Log format: json, text (default: json)
//	LOGISTICS_LOG_OUTPUT   - Output: stdout, stderr, file (default: stdout)
//	LOGISTICS_LOG_FILE_PATH - Log file path when output=file
//
//	# Caching
//	LOGISTICS_CACHE_ENABLED     - Enable result caching (default: false)
//	LOGISTICS_CACHE_DRIVER      - Cache backend: memory, redis (default: memory)
//	LOGISTICS_CACHE_HOST        - Redis host (default: localhost)
//	LOGISTICS_CACHE_PORT        - Redis port (default: 6379)
//	LOGISTICS_CACHE_DEFAULT_TTL - Cache TTL duration (default: 5m)
//
//	# Tracing (OpenTelemetry)
//	LOGISTICS_TRACING_ENABLED     - Enable distributed tracing (default: false)
//	LOGISTICS_TRACING_ENDPOINT    - OTLP endpoint (default: localhost:4317)
//	LOGISTICS_TRACING_SAMPLE_RATE - Sampling rate 0.0-1.0 (default: 0.1)
//
//	# Metrics (Prometheus)
//	LOGISTICS_METRICS_ENABLED   - Enable Prometheus metrics (default: true)
//	LOGISTICS_METRICS_PORT      - Metrics HTTP port (default: 9090)
//	LOGISTICS_METRICS_PATH      - Metrics endpoint path (default: /metrics)
//	LOGISTICS_METRICS_NAMESPACE - Metrics namespace (default: logistics)
//
//	# Rate Limiting
//	LOGISTICS_RATE_LIMIT_ENABLED  - Enable rate limiting (default: true)
//	LOGISTICS_RATE_LIMIT_REQUESTS - Requests per window (default: 100)
//	LOGISTICS_RATE_LIMIT_WINDOW   - Time window (default: 1m)
//	LOGISTICS_RATE_LIMIT_STRATEGY - Strategy: sliding_window, token_bucket
//
//	# Audit Logging
//	LOGISTICS_AUDIT_ENABLED      - Enable audit logging (default: true)
//	LOGISTICS_AUDIT_BACKEND      - Backend: stdout, file, grpc (default: stdout)
//	LOGISTICS_AUDIT_FILE_PATH    - Audit log file path
//	LOGISTICS_AUDIT_BUFFER_SIZE  - Buffer size for async logging (default: 1000)
//
// # Interceptor Chain
//
// The gRPC server uses a chain of interceptors (applied in order):
//  1. Recovery - Catches panics and returns proper gRPC errors
//  2. Logging - Structured request/response logging
//  3. Metrics - Prometheus metrics collection (latency, counts, errors)
//  4. Tracing - OpenTelemetry distributed tracing (if enabled)
//  5. RateLimit - Per-client rate limiting (if enabled)
//  6. Audit - Audit logging for compliance (if enabled)
//
// # Health Checks
//
// The service implements the standard gRPC health check protocol:
//
//	grpc.health.v1.Health/Check  - Returns SERVING when ready
//	grpc.health.v1.Health/Watch  - Streams health status changes
//
// Health check endpoints are automatically excluded from:
//   - Rate limiting
//   - Audit logging
//   - Request/response logging (at info level)
//
// # Graceful Shutdown
//
// The service handles SIGINT and SIGTERM signals for graceful shutdown:
//  1. Sets health status to NOT_SERVING (stops receiving new requests)
//  2. Waits for in-flight requests to complete (up to 30 seconds)
//  3. Flushes telemetry, metrics, and audit buffers
//  4. Closes rate limiter and cache connections
//  5. Stops the gRPC server
//
// # Performance Considerations
//
// The service is designed for high throughput:
//
//	Memory Management:
//	  - Graph pooling via sync.Pool reduces GC pressure
//	  - Pooled resources for algorithm temporary data
//	  - Efficient slice and map reuse
//
//	Concurrency:
//	  - Configurable max concurrent connections
//	  - Non-blocking context cancellation in algorithms
//	  - Batch processing support for multiple graphs
//
//	Caching:
//	  - Optional result caching (memory or Redis)
//	  - Cache key based on graph hash + algorithm + options
//	  - Configurable TTL per result type
//
//	Algorithm Selection:
//	  - Automatic algorithm recommendation based on graph characteristics
//	  - Capacity Scaling for large capacity values (>1e6)
//	  - Push-Relabel for dense graphs
//	  - Dinic for sparse graphs and bipartite matching
//
// # Observability
//
// Metrics (Prometheus):
//
//	logistics_solver_requests_total          - Total requests by method and status
//	logistics_solver_request_duration_seconds - Request latency histogram
//	logistics_solver_graph_nodes_total       - Nodes processed histogram
//	logistics_solver_graph_edges_total       - Edges processed histogram
//	logistics_solver_algorithm_usage_total   - Algorithm usage counter
//	logistics_solver_cache_hits_total        - Cache hit counter
//	logistics_solver_cache_misses_total      - Cache miss counter
//
// Tracing (OpenTelemetry):
//
//	Spans are created for:
//	  - Each gRPC method invocation
//	  - Graph conversion operations
//	  - Algorithm execution
//	  - Cache operations
//
// Logging (Structured JSON):
//
//	Each request logs:
//	  - request_id: Unique identifier for correlation
//	  - method: gRPC method name
//	  - duration_ms: Request duration in milliseconds
//	  - status: gRPC status code
//	  - graph_nodes: Number of nodes (if applicable)
//	  - graph_edges: Number of edges (if applicable)
//	  - algorithm: Algorithm used
//	  - max_flow: Result max flow value
//
// # Development Mode
//
// When LOGISTICS_APP_ENVIRONMENT=development:
//   - gRPC reflection is enabled (for grpcurl/grpcui)
//   - Debug logging is available
//   - Swagger UI is served (if enabled)
//   - More verbose error messages
//
// # Docker Deployment
//
// Build:
//
//	docker build -f Dockerfile -t solver-svc:latest .
//
// Run:
//
//	docker run -p 50054:50054 -p 9090:9090 \
//	  -e LOGISTICS_LOG_LEVEL=info \
//	  -e LOGISTICS_CACHE_ENABLED=true \
//	  solver-svc:latest
//
// Docker Compose:
//
//	docker-compose up solver-svc
//
// # Kubernetes Deployment
//
// The service is designed for Kubernetes deployment with:
//   - Liveness probe: gRPC health check
//   - Readiness probe: gRPC health check
//   - Resource limits: Configurable via helm values
//   - Horizontal Pod Autoscaler: Based on CPU/memory or custom metrics
//
// Example probe configuration:
//
//	livenessProbe:
//	  grpc:
//	    port: 50054
//	  initialDelaySeconds: 5
//	  periodSeconds: 10
//
//	readinessProbe:
//	  grpc:
//	    port: 50054
//	  initialDelaySeconds: 5
//	  periodSeconds: 5
//
// # Local Development
//
// With hot reload using Air:
//
//	air  # Uses .air.toml configuration
//
// Manual run:
//
//	go run cmd/main.go
//
// With custom config:
//
//	CONFIG_PATH=./config/local.yaml go run cmd/main.go
//
// # API Usage Examples
//
// Using grpcurl to solve a max-flow problem:
//
//	grpcurl -plaintext -d '{
//	  "graph": {
//	    "nodes": [
//	      {"id": 1, "type": "NODE_TYPE_WAREHOUSE"},
//	      {"id": 2, "type": "NODE_TYPE_HUB"},
//	      {"id": 3, "type": "NODE_TYPE_DELIVERY_POINT"}
//	    ],
//	    "edges": [
//	      {"from": 1, "to": 2, "capacity": 10, "cost": 1.5},
//	      {"from": 2, "to": 3, "capacity": 5, "cost": 2.0}
//	    ],
//	    "source_id": 1,
//	    "sink_id": 3
//	  },
//	  "algorithm": "ALGORITHM_DINIC",
//	  "options": {
//	    "return_paths": true,
//	    "timeout_ms": 5000
//	  }
//	}' localhost:50054 logistics.optimization.v1.SolverService/SolveFlow
//
// Response:
//
//	{
//	  "result": {
//	    "max_flow": 5.0,
//	    "total_cost": 17.5,
//	    "status": "FLOW_STATUS_OPTIMAL",
//	    "paths": [
//	      {"node_ids": [1, 2, 3], "flow": 5.0, "cost": 17.5}
//	    ]
//	  },
//	  "metadata": {
//	    "algorithm_used": "ALGORITHM_DINIC",
//	    "iterations": 1,
//	    "duration_ms": 0.5
//	  }
//	}
//
// # Dependencies
//
// External services (optional):
//
//	Redis:
//	  - For distributed caching (LOGISTICS_CACHE_DRIVER=redis)
//	  - For distributed rate limiting (LOGISTICS_RATE_LIMIT_BACKEND=redis)
//
//	Jaeger/OTLP Collector:
//	  - For distributed tracing (LOGISTICS_TRACING_ENABLED=true)
//
// Internal dependencies (via gRPC):
//
//	None - solver-svc is a leaf service with no downstream dependencies.
//
// # Error Handling
//
// The service returns standard gRPC status codes:
//
//	OK (0)                 - Success
//	INVALID_ARGUMENT (3)   - Invalid graph structure or parameters
//	DEADLINE_EXCEEDED (4)  - Algorithm timeout
//	NOT_FOUND (5)          - Source or sink node not in graph
//	RESOURCE_EXHAUSTED (8) - Rate limit exceeded
//	INTERNAL (13)          - Internal algorithm error
//	UNAVAILABLE (14)       - Service shutting down
//
// Error details include:
//   - Error code and message
//   - Request ID for correlation
//   - Partial results (if available)
//
// # Security Considerations
//
// The service supports:
//   - TLS encryption (LOGISTICS_GRPC_TLS_ENABLED=true)
//   - Per-client rate limiting
//   - Audit logging for compliance
//   - Input validation (graph size limits, parameter bounds)
//
// Recommended production settings:
//
//	LOGISTICS_GRPC_TLS_ENABLED=true
//	LOGISTICS_RATE_LIMIT_ENABLED=true
//	LOGISTICS_AUDIT_ENABLED=true
//	LOGISTICS_LOG_LEVEL=info
package main

import (
	"context"
	"log"
	"time"

	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/pkg/cache"
	"logistics/pkg/config"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/pkg/server"
	"logistics/pkg/telemetry"
	"logistics/services/solver-svc/internal/service"
)

func main() {
	// =========================================================================
	// Configuration Loading
	// =========================================================================
	//
	// LoadWithServiceDefaults loads configuration with the following priority:
	//   1. Environment variables (LOGISTICS_* prefix)
	//   2. Config files (config.yaml in standard locations)
	//   3. Default values from pkg/config/loader.go
	//
	// The service name and default port are applied if not explicitly configured.
	// This allows sharing a common config.yaml while overriding per-service.
	cfg, err := config.LoadWithServiceDefaults("solver-svc", 50054)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// =========================================================================
	// Logger Initialization
	// =========================================================================
	//
	// The logger is configured based on the loaded configuration.
	// Supported outputs:
	//   - stdout/stderr: Direct console output
	//   - file: File output with automatic rotation (via lumberjack)
	//
	// Log rotation settings (when output=file):
	//   - MaxSize: Maximum size in MB before rotation
	//   - MaxBackups: Number of old files to retain
	//   - MaxAge: Maximum days to retain old files
	//   - Compress: Whether to gzip rotated files
	logger.InitWithConfig(logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		FilePath:   cfg.Log.FilePath,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	})

	ctx := context.Background()

	// =========================================================================
	// Telemetry Initialization (OpenTelemetry)
	// =========================================================================
	//
	// When enabled, initializes the OpenTelemetry trace provider.
	// Traces are exported to the configured OTLP endpoint (e.g., Jaeger).
	//
	// The trace provider is stored in the server for proper shutdown.
	// Shutdown flushes pending spans before the process exits.
	//
	// Note: The server.Run() also initializes telemetry if not already done.
	// This early initialization allows tracing of cache and other setup.
	if cfg.Tracing.Enabled {
		tp, err := telemetry.Init(ctx, telemetry.Config{
			Enabled:     cfg.Tracing.Enabled,
			Endpoint:    cfg.Tracing.Endpoint,
			ServiceName: cfg.App.Name,
			Version:     cfg.App.Version,
			Environment: cfg.App.Environment,
			SampleRate:  cfg.Tracing.SampleRate,
		})
		if err != nil {
			logger.Log.Warn("Failed to init telemetry", "error", err)
		} else {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tp.Shutdown(shutdownCtx); err != nil {
					logger.Log.Warn("Failed to shutdown telemetry", "error", err)
				}
			}()
			logger.Log.Info("Telemetry initialized", "endpoint", cfg.Tracing.Endpoint)
		}
	}

	// =========================================================================
	// Metrics Initialization (Prometheus)
	// =========================================================================
	//
	// Initializes Prometheus metrics with the configured namespace.
	// The metrics server is started separately by server.Run() on the metrics port.
	//
	// Available metrics:
	//   - grpc_server_started_total: Total gRPC requests started
	//   - grpc_server_handled_total: Total gRPC requests completed
	//   - grpc_server_handling_seconds: Request duration histogram
	//   - Custom solver metrics (defined in internal/service)
	metrics.InitMetrics(cfg.Metrics.Namespace, cfg.App.Name)

	// =========================================================================
	// Cache Initialization
	// =========================================================================
	//
	// The solver cache stores computation results to avoid redundant calculations.
	// Cache key is computed from:
	//   - Graph structure hash (nodes, edges, capacities, costs)
	//   - Algorithm type
	//   - Solver options
	//
	// Supported backends:
	//   - memory: In-process LRU cache (fast, not shared between instances)
	//   - redis: Distributed cache (shared, requires Redis server)
	//
	// Cache entries expire after DefaultTTL. The cache is optional and the
	// service continues to function if cache initialization fails.
	var solverCache *cache.SolverCache
	if cfg.Cache.Enabled {
		// Create cache options from configuration
		// This maps config fields to cache.Options struct
		cacheOpts := cache.FromConfig(&cfg.Cache)

		// Create the base cache (memory or Redis)
		baseCache, err := cache.New(cacheOpts)
		if err != nil {
			logger.Log.Warn("Failed to create cache, continuing without cache", "error", err)
		} else {
			// Wrap with solver-specific cache that handles serialization
			solverCache = cache.NewSolverCache(baseCache, cfg.Cache.DefaultTTL)
			logger.Log.Info("Solver cache initialized",
				"driver", cfg.Cache.Driver,
				"ttl", cfg.Cache.DefaultTTL,
			)
		}
	}

	// =========================================================================
	// gRPC Server Creation
	// =========================================================================
	//
	// server.New creates a gRPC server with:
	//   - Keep-alive settings for long-running connections
	//   - Message size limits (default 16MB)
	//   - Concurrent stream limits
	//   - Interceptor chain (logging, metrics, tracing, rate-limit, audit)
	//   - Health check service (grpc.health.v1.Health)
	//   - Reflection service (development mode only)
	//
	// The server handles graceful shutdown on SIGINT/SIGTERM.
	srv := server.New(cfg)

	// =========================================================================
	// Service Registration
	// =========================================================================
	//
	// Create the solver service with:
	//   - Version string for metadata responses
	//   - Optional cache for result caching
	//
	// The service implements the SolverService gRPC interface defined in:
	//   proto/logistics/optimization/v1/solver.proto
	//
	// Available RPC methods:
	//   - SolveFlow: Compute max-flow or min-cost flow
	//   - GetAlgorithmInfo: Get algorithm metadata
	//   - RecommendAlgorithm: Get algorithm recommendation for a graph
	//   - BatchSolve: Solve multiple flow problems
	solverService := service.NewSolverService(cfg.App.Version, solverCache)
	optimizationv1.RegisterSolverServiceServer(srv.GetEngine(), solverService)

	// =========================================================================
	// Server Startup
	// =========================================================================
	//
	// Logs startup information for operational visibility.
	// This log entry is useful for:
	//   - Confirming successful startup
	//   - Verifying configuration
	//   - Correlating with deployment events
	logger.Info("Starting solver service",
		"port", cfg.GRPC.Port,
		"environment", cfg.App.Environment,
		"version", cfg.App.Version,
		"cache_enabled", solverCache != nil,
	)

	// =========================================================================
	// Run Server (Blocking)
	// =========================================================================
	//
	// srv.Run() performs the following:
	//   1. Starts the metrics HTTP server (if enabled)
	//   2. Starts the Swagger UI server (if enabled, development mode)
	//   3. Binds to the gRPC port
	//   4. Sets health status to SERVING
	//   5. Logs an audit event for service start
	//   6. Blocks until shutdown signal received
	//   7. Performs graceful shutdown (see waitForShutdown in server.go)
	//
	// Returns nil on clean shutdown, error if server fails to start.
	if err := srv.Run(); err != nil {
		logger.Fatal("server failed", "error", err)
	}
}
