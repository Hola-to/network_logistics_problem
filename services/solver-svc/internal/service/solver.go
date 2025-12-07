package service

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

type SolverService struct {
	optimizationv1.UnimplementedSolverServiceServer
	version     string
	metrics     *metrics.Metrics
	solverCache *cache.SolverCache
}

func NewSolverService(version string, solverCache *cache.SolverCache) *SolverService {
	return &SolverService{
		version:     version,
		metrics:     metrics.Get(),
		solverCache: solverCache,
	}
}

func (s *SolverService) Solve(ctx context.Context, req *optimizationv1.SolveRequest) (*optimizationv1.SolveResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SolverService.Solve",
		trace.WithAttributes(
			attribute.String("algorithm", req.Algorithm.String()),
		),
	)
	defer span.End()

	// Проверяем кэш
	if s.solverCache != nil {
		cached, found, err := s.solverCache.Get(ctx, req.Graph, req.Algorithm)
		if err == nil && found {
			telemetry.AddEvent(ctx, "cache_hit",
				attribute.Float64("max_flow", cached.MaxFlow),
			)
			span.SetAttributes(attribute.Bool("cache_hit", true))

			return &optimizationv1.SolveResponse{
				Success: true,
				Result:  cached.ToFlowResult(),
				Metrics: &optimizationv1.SolveMetrics{
					ComputationTimeMs: 0,
				},
			}, nil
		}
	}

	span.SetAttributes(attribute.Bool("cache_hit", false))

	// Валидация
	if err := s.validateRequest(req); err != nil {
		telemetry.SetError(ctx, err)
		return nil, err
	}

	start := time.Now()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Решение
	rg := converter.ToResidualGraph(req.Graph)
	opts := s.buildOptions(req.Options)
	result := algorithms.Solve(rg, req.Graph.SourceId, req.Graph.SinkId, req.Algorithm, opts)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	elapsed := time.Since(start)

	// Сохраняем в кэш
	if s.solverCache != nil && result.Error == nil {
		response := &optimizationv1.SolveResponse{
			Result: &commonv1.FlowResult{
				MaxFlow:           result.MaxFlow,
				TotalCost:         result.TotalCost,
				Edges:             converter.ToFlowEdges(rg),
				Status:            result.Status,
				Iterations:        int32(result.Iterations),
				ComputationTimeMs: float64(elapsed.Milliseconds()),
			},
		}
		if err := s.solverCache.SetFromResponse(ctx, req.Graph, req.Algorithm, response, 0); err != nil {
			logger.Log.Warn("Failed to cache solve result", "error", err)
		}
	}

	// Записываем метрики
	if s.metrics != nil {
		s.metrics.RecordSolveOperation(
			req.Algorithm.String(),
			result.Error == nil,
			elapsed,
			result.MaxFlow,
		)
	}

	return &optimizationv1.SolveResponse{
		Success: result.Error == nil,
		Result: &commonv1.FlowResult{
			MaxFlow:           result.MaxFlow,
			TotalCost:         result.TotalCost,
			Edges:             converter.ToFlowEdges(rg),
			Status:            result.Status,
			Iterations:        int32(result.Iterations),
			ComputationTimeMs: float64(elapsed.Milliseconds()),
		},
		SolvedGraph: converter.UpdateGraphWithFlow(req.Graph, rg),
		Metrics: &optimizationv1.SolveMetrics{
			ComputationTimeMs:    float64(elapsed.Milliseconds()),
			Iterations:           int32(result.Iterations),
			AugmentingPathsFound: int32(len(result.Paths)),
			MemoryUsedBytes:      int64(memAfter.TotalAlloc - memBefore.TotalAlloc),
		},
	}, nil
}

// SolveStream решает задачу с потоковой передачей прогресса
func (s *SolverService) SolveStream(req *optimizationv1.SolveRequestForBigGraphs, stream optimizationv1.SolverService_SolveStreamServer) error {
	ctx := stream.Context()

	ctx, span := telemetry.StartSpan(ctx, "SolverService.SolveStream",
		trace.WithAttributes(
			attribute.String("algorithm", req.Algorithm.String()),
			attribute.Int("nodes", len(req.Graph.Nodes)),
			attribute.Int("edges", len(req.Graph.Edges)),
		),
	)
	defer span.End()

	start := time.Now()
	lastSendTime := time.Now()

	var memStats runtime.MemStats

	// Конвертация
	rg := converter.ToResidualGraph(req.Graph)
	opts := s.buildOptions(req.Options)

	source := req.Graph.SourceId
	sink := req.Graph.SinkId

	maxFlow := 0.0
	iteration := 0

	for {
		// Проверка отмены
		select {
		case <-ctx.Done():
			telemetry.AddEvent(ctx, "cancelled")
			return ctx.Err()
		default:
		}

		// Проверка лимитов
		if opts.MaxIterations > 0 && iteration >= opts.MaxIterations {
			telemetry.AddEvent(ctx, "iteration_limit_reached",
				attribute.Int("limit", opts.MaxIterations),
			)
			break
		}

		if opts.Timeout > 0 && time.Since(start) > opts.Timeout {
			telemetry.AddEvent(ctx, "timeout_reached")
			break
		}

		// Шаг алгоритма
		bfsResult := graph.BFS(rg, source, sink)
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

		// Обновление графа
		graph.AugmentPath(rg, path, pathFlow)
		maxFlow += pathFlow
		iteration++

		// Отправка прогресса с throttling
		if time.Since(lastSendTime) > 200*time.Millisecond || iteration == 1 {
			runtime.ReadMemStats(&memStats)

			progress := &optimizationv1.SolveProgress{
				Iteration:         int32(iteration),
				CurrentFlow:       maxFlow,
				Status:            "running",
				ProgressPercent:   0.0,
				LastPath:          s.convertPathToProto(path, pathFlow),
				ComputationTimeMs: float64(time.Since(start).Milliseconds()),
				MemoryUsedBytes:   int64(memStats.Alloc),
			}

			if err := stream.Send(progress); err != nil {
				telemetry.SetError(ctx, err)
				return err
			}
			lastSendTime = time.Now()
		}
	}

	// Финальное сообщение
	runtime.ReadMemStats(&memStats)

	telemetry.AddEvent(ctx, "stream_completed",
		attribute.Float64("max_flow", maxFlow),
		attribute.Int("total_iterations", iteration),
	)

	return stream.Send(&optimizationv1.SolveProgress{
		Iteration:         int32(iteration),
		CurrentFlow:       maxFlow,
		ProgressPercent:   100.0,
		Status:            "completed",
		ComputationTimeMs: float64(time.Since(start).Milliseconds()),
		MemoryUsedBytes:   int64(memStats.Alloc),
	})
}

// GetAlgorithms возвращает метаданные доступных алгоритмов
func (s *SolverService) GetAlgorithms(_ context.Context, _ *emptypb.Empty) (*optimizationv1.GetAlgorithmsResponse, error) {
	_, span := telemetry.StartSpan(context.Background(), "SolverService.GetAlgorithms")
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

// validateRequest валидирует входящий запрос
func (s *SolverService) validateRequest(req *optimizationv1.SolveRequest) error {
	if req.Graph == nil {
		return pkgerrors.ErrNilGraph
	}

	if len(req.Graph.Nodes) == 0 {
		return pkgerrors.ErrEmptyGraph
	}

	// Проверяем source и sink
	nodeExists := make(map[int64]bool)
	for _, node := range req.Graph.Nodes {
		nodeExists[node.Id] = true
	}

	if !nodeExists[req.Graph.SourceId] {
		return pkgerrors.ErrInvalidSource
	}

	if !nodeExists[req.Graph.SinkId] {
		return pkgerrors.ErrInvalidSink
	}

	if req.Graph.SourceId == req.Graph.SinkId {
		return pkgerrors.ErrSourceEqualsSink
	}

	return nil
}

func (s *SolverService) buildOptions(opts *optimizationv1.SolveOptions) *algorithms.SolverOptions {
	result := algorithms.DefaultSolverOptions()

	if opts == nil {
		return result
	}

	if opts.Epsilon > 0 {
		result.Epsilon = opts.Epsilon
	}

	if opts.MaxIterations > 0 {
		result.MaxIterations = int(opts.MaxIterations)
	}

	if opts.TimeoutSeconds > 0 {
		result.Timeout = time.Duration(opts.TimeoutSeconds * float64(time.Second))
	}

	result.ReturnPaths = opts.ReturnPaths

	return result
}

func (s *SolverService) convertPathToProto(nodeIDs []int64, flow float64) *commonv1.Path {
	return &commonv1.Path{
		NodeIds: nodeIDs,
		Flow:    flow,
	}
}
