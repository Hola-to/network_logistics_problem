// services/simulation-svc/internal/service/simulation.go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/client"
	"logistics/pkg/logger"
	"logistics/pkg/telemetry"
	"logistics/services/simulation-svc/internal/engine"
	"logistics/services/simulation-svc/internal/repository"
)

var startTime = time.Now()

// SimulationService реализация gRPC сервиса симуляций
type SimulationService struct {
	simulationv1.UnimplementedSimulationServiceServer
	repo         repository.SimulationRepository
	solverClient engine.SolverClientInterface // Изменено на интерфейс
	version      string

	// Движки
	solverEngine      *engine.SolverEngine
	resilienceEngine  *engine.ResilienceEngine
	sensitivityEngine *engine.SensitivityEngine
	timeEngine        *engine.TimeSimulationEngine
}

// NewSimulationService создаёт новый сервис
func NewSimulationService(
	repo repository.SimulationRepository,
	solverClient *client.SolverClient,
	version string,
) *SimulationService {
	var solverInterface engine.SolverClientInterface
	if solverClient != nil {
		solverInterface = engine.NewSolverClientAdapter(solverClient)
	}

	return &SimulationService{
		repo:              repo,
		solverClient:      solverInterface,
		version:           version,
		solverEngine:      engine.NewSolverEngine(solverClient),
		resilienceEngine:  engine.NewResilienceEngine(solverInterface),
		sensitivityEngine: engine.NewSensitivityEngine(solverInterface),
		timeEngine:        engine.NewTimeSimulationEngine(solverInterface),
	}
}

// NewSimulationServiceWithInterface создаёт сервис с интерфейсом (для тестов)
func NewSimulationServiceWithInterface(
	repo repository.SimulationRepository,
	solverClient engine.SolverClientInterface,
	version string,
) *SimulationService {
	return &SimulationService{
		repo:              repo,
		solverClient:      solverClient,
		version:           version,
		solverEngine:      engine.NewSolverEngineWithInterface(solverClient),
		resilienceEngine:  engine.NewResilienceEngine(solverClient),
		sensitivityEngine: engine.NewSensitivityEngine(solverClient),
		timeEngine:        engine.NewTimeSimulationEngine(solverClient),
	}
}

// ============ WHAT-IF ANALYSIS ============

// RunWhatIf запускает what-if сценарий
func (s *SimulationService) RunWhatIf(
	ctx context.Context,
	req *simulationv1.RunWhatIfRequest,
) (*simulationv1.RunWhatIfResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.RunWhatIf",
		trace.WithAttributes(
			attribute.Int("modifications_count", len(req.Modifications)),
		),
	)
	defer span.End()

	start := time.Now()

	// Валидация
	if req.BaselineGraph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "baseline_graph is required"),
		)
	}

	// Базовый результат
	baselineResult, err := s.solverEngine.Solve(ctx, req.BaselineGraph, req.Algorithm)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to solve baseline"),
		)
	}

	// Применяем модификации
	modifiedGraph := engine.ApplyModifications(req.BaselineGraph, req.Modifications)

	// Решаем модифицированный граф
	modifiedResult, err := s.solverEngine.Solve(ctx, modifiedGraph, req.Algorithm)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to solve modified graph"),
		)
	}

	// Сравниваем результаты
	comparison := engine.CompareResults(baselineResult, modifiedResult)

	// Находим изменения в bottlenecks
	var bottleneckChanges []*simulationv1.BottleneckChange
	if req.Options != nil && req.Options.FindNewBottlenecks {
		bottleneckChanges = s.findBottleneckChanges(baselineResult, modifiedResult)
	}

	telemetry.AddEvent(ctx, "what_if_completed",
		attribute.Float64("baseline_flow", baselineResult.MaxFlow),
		attribute.Float64("modified_flow", modifiedResult.MaxFlow),
		attribute.Float64("change_percent", comparison.FlowChangePercent),
	)

	response := &simulationv1.RunWhatIfResponse{
		Success:           true,
		Baseline:          engine.ToScenarioResult(baselineResult, "Baseline"),
		Modified:          engine.ToScenarioResult(modifiedResult, "Modified"),
		Comparison:        comparison,
		BottleneckChanges: bottleneckChanges,
		Metadata: &simulationv1.SimulationMetadata{
			ComputationTimeMs: float64(time.Since(start).Milliseconds()),
			AlgorithmUsed:     req.Algorithm.String(),
			CompletedAt:       timestamppb.Now(),
		},
	}

	if req.Options != nil && req.Options.ReturnModifiedGraph {
		response.ModifiedGraph = modifiedGraph
	}

	return response, nil
}

// CompareScenarios сравнивает несколько сценариев
func (s *SimulationService) CompareScenarios(
	ctx context.Context,
	req *simulationv1.CompareScenariosRequest,
) (*simulationv1.CompareScenariosResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.CompareScenarios",
		trace.WithAttributes(
			attribute.Int("scenarios_count", len(req.Scenarios)),
		),
	)
	defer span.End()

	start := time.Now()

	if req.BaselineGraph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "baseline_graph is required"),
		)
	}

	// Базовый результат
	baselineResult, err := s.solverEngine.Solve(ctx, req.BaselineGraph, req.Algorithm)
	if err != nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to solve baseline"),
		)
	}

	rankedScenarios := make([]*simulationv1.ScenarioResultWithRank, 0, len(req.Scenarios))
	bestScenario := ""
	bestFlow := baselineResult.MaxFlow

	for i, scenario := range req.Scenarios {
		modifiedGraph := engine.ApplyModifications(req.BaselineGraph, scenario.Modifications)
		result, err := s.solverEngine.Solve(ctx, modifiedGraph, req.Algorithm)
		if err != nil {
			continue
		}

		comparison := engine.CompareResults(baselineResult, result)

		// Расчёт ROI если запрошено
		var roi float64
		if req.Options != nil && req.Options.CalculateRoi && req.Options.ModificationCostPerUnit > 0 {
			totalModCost := s.calculateModificationCost(scenario.Modifications, req.Options.ModificationCostPerUnit)
			if totalModCost > 0 {
				flowGain := result.MaxFlow - baselineResult.MaxFlow
				roi = flowGain / totalModCost
			}
		}

		ranked := &simulationv1.ScenarioResultWithRank{
			Result:     engine.ToScenarioResult(result, scenario.Name),
			Rank:       int32(i + 1),
			Score:      result.MaxFlow,
			Roi:        roi,
			VsBaseline: comparison,
		}
		rankedScenarios = append(rankedScenarios, ranked)

		if result.MaxFlow > bestFlow {
			bestFlow = result.MaxFlow
			bestScenario = scenario.Name
		}
	}

	// Сортировка по потоку
	s.sortScenariosByFlow(rankedScenarios)

	// Обновляем ранги после сортировки
	for i := range rankedScenarios {
		rankedScenarios[i].Rank = int32(i + 1)
	}

	return &simulationv1.CompareScenariosResponse{
		Baseline:        engine.ToScenarioResult(baselineResult, "Baseline"),
		RankedScenarios: rankedScenarios,
		BestScenario:    bestScenario,
		Recommendation:  s.generateRecommendation(rankedScenarios, bestScenario),
		Metadata: &simulationv1.SimulationMetadata{
			ComputationTimeMs: float64(time.Since(start).Milliseconds()),
			Iterations:        int32(len(req.Scenarios)),
			AlgorithmUsed:     req.Algorithm.String(),
			CompletedAt:       timestamppb.Now(),
		},
	}, nil
}

// ============ MONTE CARLO ============

// RunMonteCarlo запускает Monte Carlo симуляцию
func (s *SimulationService) RunMonteCarlo(
	ctx context.Context,
	req *simulationv1.RunMonteCarloRequest,
) (*simulationv1.RunMonteCarloResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.RunMonteCarlo",
		trace.WithAttributes(
			attribute.Int("iterations", int(req.Config.GetNumIterations())),
		),
	)
	defer span.End()

	start := time.Now()

	if req.Graph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "graph is required"),
		)
	}

	config := req.Config
	if config == nil {
		config = &simulationv1.MonteCarloConfig{
			NumIterations:   1000,
			ConfidenceLevel: 0.95,
		}
	}

	// Используем интерфейс вместо конкретного клиента
	mcEngine := engine.NewMonteCarloEngine(config, s.solverClient)
	result, err := mcEngine.Run(ctx, req.Graph, req.Uncertainties, req.Algorithm, nil)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "monte carlo simulation failed"),
		)
	}

	result.Metadata = &simulationv1.SimulationMetadata{
		ComputationTimeMs: float64(time.Since(start).Milliseconds()),
		Iterations:        config.NumIterations,
		AlgorithmUsed:     req.Algorithm.String(),
		CompletedAt:       timestamppb.Now(),
	}

	return result, nil
}

// RunMonteCarloStream streaming версия Monte Carlo
func (s *SimulationService) RunMonteCarloStream(
	req *simulationv1.RunMonteCarloRequest,
	stream simulationv1.SimulationService_RunMonteCarloStreamServer,
) error {
	ctx := stream.Context()
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.RunMonteCarloStream")
	defer span.End()

	if req.Graph == nil {
		return pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "graph is required"),
		)
	}

	config := req.Config
	if config == nil {
		config = &simulationv1.MonteCarloConfig{
			NumIterations:   1000,
			ConfidenceLevel: 0.95,
		}
	}

	progressChan := make(chan *simulationv1.MonteCarloProgress, 100)
	errChan := make(chan error, 1)

	// Запускаем в горутине - используем интерфейс
	go func() {
		mcEngine := engine.NewMonteCarloEngine(config, s.solverClient)
		_, err := mcEngine.Run(ctx, req.Graph, req.Uncertainties, req.Algorithm, progressChan)
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}()

	// Стримим прогресс
	for progress := range progressChan {
		if err := stream.Send(progress); err != nil {
			return err
		}
	}

	// Проверяем ошибки
	if err, ok := <-errChan; ok && err != nil {
		return pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "monte carlo simulation failed"),
		)
	}

	return nil
}

// ============ SENSITIVITY ANALYSIS ============

// AnalyzeSensitivity выполняет анализ чувствительности
func (s *SimulationService) AnalyzeSensitivity(
	ctx context.Context,
	req *simulationv1.AnalyzeSensitivityRequest,
) (*simulationv1.AnalyzeSensitivityResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.AnalyzeSensitivity",
		trace.WithAttributes(
			attribute.Int("parameters_count", len(req.Parameters)),
		),
	)
	defer span.End()

	start := time.Now()

	if req.Graph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "graph is required"),
		)
	}

	result, err := s.sensitivityEngine.AnalyzeSensitivity(ctx, req.Graph, req.Parameters, req.Config, req.Algorithm)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "sensitivity analysis failed"),
		)
	}

	result.Metadata = &simulationv1.SimulationMetadata{
		ComputationTimeMs: float64(time.Since(start).Milliseconds()),
		AlgorithmUsed:     req.Algorithm.String(),
		CompletedAt:       timestamppb.Now(),
	}

	return result, nil
}

// ============ CRITICAL ELEMENTS ============

// criticalElementsAnalyzer вспомогательная структура для анализа критических элементов
type criticalElementsAnalyzer struct {
	service    *SimulationService
	graph      *commonv1.Graph
	algorithm  commonv1.Algorithm
	config     *simulationv1.CriticalElementsConfig
	baseResult *client.SolveResult
}

func newCriticalElementsAnalyzer(
	s *SimulationService,
	graph *commonv1.Graph,
	algorithm commonv1.Algorithm,
	config *simulationv1.CriticalElementsConfig,
	baseResult *client.SolveResult,
) *criticalElementsAnalyzer {
	if config == nil {
		config = &simulationv1.CriticalElementsConfig{
			AnalyzeEdges:     true,
			AnalyzeNodes:     true,
			TopN:             10,
			FailureThreshold: 0.1,
		}
	}
	return &criticalElementsAnalyzer{
		service:    s,
		graph:      graph,
		algorithm:  algorithm,
		config:     config,
		baseResult: baseResult,
	}
}

// FindCriticalElements находит критические элементы
func (s *SimulationService) FindCriticalElements(
	ctx context.Context,
	req *simulationv1.FindCriticalElementsRequest,
) (*simulationv1.FindCriticalElementsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.FindCriticalElements")
	defer span.End()

	if req.Graph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "graph is required"),
		)
	}

	start := time.Now()

	baseResult, err := s.solverEngine.Solve(ctx, req.Graph, req.Algorithm)
	if err != nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to solve baseline"),
		)
	}

	analyzer := newCriticalElementsAnalyzer(s, req.Graph, req.Algorithm, req.Config, baseResult)

	criticalEdges, edgeSPOFs := analyzer.analyzeEdges(ctx)
	criticalNodes := analyzer.analyzeNodes(ctx)

	analyzer.sortAndLimit(&criticalEdges, &criticalNodes)

	resilienceScore := s.calculateResilienceScore(
		criticalEdges, criticalNodes,
		len(req.Graph.Edges), len(req.Graph.Nodes),
	)

	return &simulationv1.FindCriticalElementsResponse{
		Success:               true,
		CriticalEdges:         criticalEdges,
		CriticalNodes:         criticalNodes,
		SinglePointsOfFailure: edgeSPOFs,
		ResilienceScore:       resilienceScore,
		Metadata: &simulationv1.SimulationMetadata{
			ComputationTimeMs: float64(time.Since(start).Milliseconds()),
			AlgorithmUsed:     req.Algorithm.String(),
			CompletedAt:       timestamppb.Now(),
		},
	}, nil
}

func (a *criticalElementsAnalyzer) analyzeEdges(ctx context.Context) ([]*simulationv1.CriticalEdge, []*commonv1.EdgeKey) {
	if !a.config.AnalyzeEdges {
		return nil, nil
	}

	criticalEdges := make([]*simulationv1.CriticalEdge, 0)
	var spofs []*commonv1.EdgeKey

	for _, edge := range a.graph.Edges {
		result := a.analyzeEdge(ctx, edge)
		if result == nil {
			continue
		}

		criticalEdges = append(criticalEdges, result)
		if result.IsSinglePointOfFailure {
			spofs = append(spofs, &commonv1.EdgeKey{From: edge.From, To: edge.To})
		}
	}

	return criticalEdges, spofs
}

func (a *criticalElementsAnalyzer) analyzeEdge(ctx context.Context, edge *commonv1.Edge) *simulationv1.CriticalEdge {
	modGraph := a.service.removeEdgeFromGraph(a.graph, edge.From, edge.To)
	modResult, err := a.service.solverEngine.Solve(ctx, modGraph, a.algorithm)
	if err != nil {
		return nil
	}

	flowImpact := a.baseResult.MaxFlow - modResult.MaxFlow
	flowImpactPercent := 0.0
	if a.baseResult.MaxFlow > 0 {
		flowImpactPercent = flowImpact / a.baseResult.MaxFlow
	}

	isSPOF := modResult.MaxFlow == 0 && a.baseResult.MaxFlow > 0

	if flowImpactPercent < a.config.FailureThreshold && !isSPOF {
		return nil
	}

	costImpact := 0.0
	if modResult.TotalCost > 0 && a.baseResult.TotalCost > 0 {
		costImpact = modResult.TotalCost - a.baseResult.TotalCost
	}

	return &simulationv1.CriticalEdge{
		Edge:                   &commonv1.EdgeKey{From: edge.From, To: edge.To},
		CriticalityScore:       flowImpactPercent,
		FlowImpactIfRemoved:    flowImpact,
		CostImpactIfRemoved:    costImpact,
		IsSinglePointOfFailure: isSPOF,
	}
}

func (a *criticalElementsAnalyzer) analyzeNodes(ctx context.Context) []*simulationv1.CriticalNode {
	if !a.config.AnalyzeNodes {
		return nil
	}

	var criticalNodes []*simulationv1.CriticalNode

	for _, node := range a.graph.Nodes {
		if node.Id == a.graph.SourceId || node.Id == a.graph.SinkId {
			continue
		}

		result := a.analyzeNode(ctx, node)
		if result != nil {
			criticalNodes = append(criticalNodes, result)
		}
	}

	return criticalNodes
}

func (a *criticalElementsAnalyzer) analyzeNode(ctx context.Context, node *commonv1.Node) *simulationv1.CriticalNode {
	modGraph := a.service.removeNodeFromGraph(a.graph, node.Id)
	modResult, err := a.service.solverEngine.Solve(ctx, modGraph, a.algorithm)
	if err != nil {
		return nil
	}

	flowImpact := a.baseResult.MaxFlow - modResult.MaxFlow
	flowImpactPercent := 0.0
	if a.baseResult.MaxFlow > 0 {
		flowImpactPercent = flowImpact / a.baseResult.MaxFlow
	}

	isSPOF := modResult.MaxFlow == 0 && a.baseResult.MaxFlow > 0

	if flowImpactPercent < a.config.FailureThreshold && !isSPOF {
		return nil
	}

	affectedEdges := a.service.countAffectedEdges(a.graph, node.Id)

	return &simulationv1.CriticalNode{
		NodeId:                 node.Id,
		CriticalityScore:       flowImpactPercent,
		FlowImpactIfRemoved:    flowImpact,
		AffectedEdges:          int32(affectedEdges),
		IsSinglePointOfFailure: isSPOF,
	}
}

func (a *criticalElementsAnalyzer) sortAndLimit(edges *[]*simulationv1.CriticalEdge, nodes *[]*simulationv1.CriticalNode) {
	a.service.sortCriticalEdges(*edges)
	a.service.sortCriticalNodes(*nodes)

	if a.config.TopN > 0 {
		if len(*edges) > int(a.config.TopN) {
			*edges = (*edges)[:a.config.TopN]
		}
		if len(*nodes) > int(a.config.TopN) {
			*nodes = (*nodes)[:a.config.TopN]
		}
	}

	for i := range *edges {
		(*edges)[i].Rank = int32(i + 1)
	}
	for i := range *nodes {
		(*nodes)[i].Rank = int32(i + 1)
	}
}

// ============ FAILURE & RESILIENCE ============

// SimulateFailures симулирует отказы
func (s *SimulationService) SimulateFailures(
	ctx context.Context,
	req *simulationv1.SimulateFailuresRequest,
) (*simulationv1.SimulateFailuresResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.SimulateFailures")
	defer span.End()

	if req.Graph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "graph is required"),
		)
	}

	start := time.Now()

	// Базовый результат
	baseResult, err := s.solverEngine.Solve(ctx, req.Graph, req.Algorithm)
	if err != nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to solve baseline"),
		)
	}

	scenarios := req.FailureScenarios

	// Если заданы случайные отказы, генерируем сценарии
	if req.RandomConfig != nil && len(scenarios) == 0 {
		scenarios = s.generateRandomFailureScenarios(req.Graph, req.RandomConfig)
	}

	scenarioResults := make([]*simulationv1.FailureScenarioResult, 0, len(scenarios))

	for _, scenario := range scenarios {
		modGraph := engine.CloneGraph(req.Graph)

		// Удаляем failed edges
		for _, edgeKey := range scenario.FailedEdges {
			modGraph = s.removeEdgeFromGraph(modGraph, edgeKey.From, edgeKey.To)
		}

		// Удаляем failed nodes
		for _, nodeID := range scenario.FailedNodes {
			modGraph = s.removeNodeFromGraph(modGraph, nodeID)
		}

		modResult, err := s.solverEngine.Solve(ctx, modGraph, req.Algorithm)
		if err != nil {
			continue
		}

		comparison := engine.CompareResults(baseResult, modResult)
		isDisconnected := modResult.MaxFlow == 0 && baseResult.MaxFlow > 0

		scenarioResults = append(scenarioResults, &simulationv1.FailureScenarioResult{
			ScenarioName:        scenario.Name,
			Probability:         scenario.Probability,
			Result:              engine.ToScenarioResult(modResult, scenario.Name),
			VsBaseline:          comparison,
			NetworkDisconnected: isDisconnected,
		})
	}

	// Статистика
	stats := s.calculateFailureStats(scenarioResults, baseResult.MaxFlow)

	// Рекомендации
	recommendations := s.generateResilienceRecommendations(scenarioResults, req.Graph)

	return &simulationv1.SimulateFailuresResponse{
		Success:         true,
		Baseline:        engine.ToScenarioResult(baseResult, "Baseline"),
		ScenarioResults: scenarioResults,
		Stats:           stats,
		Recommendations: recommendations,
		Metadata: &simulationv1.SimulationMetadata{
			ComputationTimeMs: float64(time.Since(start).Milliseconds()),
			Iterations:        int32(len(scenarioResults)),
			AlgorithmUsed:     req.Algorithm.String(),
			CompletedAt:       timestamppb.Now(),
		},
	}, nil
}

// AnalyzeResilience анализирует устойчивость сети
func (s *SimulationService) AnalyzeResilience(
	ctx context.Context,
	req *simulationv1.AnalyzeResilienceRequest,
) (*simulationv1.AnalyzeResilienceResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.AnalyzeResilience")
	defer span.End()

	if req.Graph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "graph is required"),
		)
	}

	start := time.Now()

	result, err := s.resilienceEngine.AnalyzeResilience(ctx, req.Graph, req.Config, req.Algorithm)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "resilience analysis failed"),
		)
	}

	result.Metadata = &simulationv1.SimulationMetadata{
		ComputationTimeMs: float64(time.Since(start).Milliseconds()),
		AlgorithmUsed:     req.Algorithm.String(),
		CompletedAt:       timestamppb.Now(),
	}

	return result, nil
}

// ============ TIME SIMULATION ============

// RunTimeSimulation запускает временную симуляцию
func (s *SimulationService) RunTimeSimulation(
	ctx context.Context,
	req *simulationv1.RunTimeSimulationRequest,
) (*simulationv1.RunTimeSimulationResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.RunTimeSimulation")
	defer span.End()

	if req.Graph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "graph is required"),
		)
	}

	start := time.Now()

	result, err := s.timeEngine.RunTimeSimulation(ctx, req)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "time simulation failed"),
		)
	}

	result.Metadata = &simulationv1.SimulationMetadata{
		ComputationTimeMs: float64(time.Since(start).Milliseconds()),
		AlgorithmUsed:     req.Algorithm.String(),
		CompletedAt:       timestamppb.Now(),
	}

	return result, nil
}

// SimulatePeakLoad симулирует пиковую нагрузку
func (s *SimulationService) SimulatePeakLoad(
	ctx context.Context,
	req *simulationv1.SimulatePeakLoadRequest,
) (*simulationv1.SimulatePeakLoadResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.SimulatePeakLoad")
	defer span.End()

	if req.Graph == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "graph is required"),
		)
	}

	start := time.Now()

	result, err := s.timeEngine.SimulatePeakLoad(ctx, req)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "peak load simulation failed"),
		)
	}

	result.Metadata = &simulationv1.SimulationMetadata{
		ComputationTimeMs: float64(time.Since(start).Milliseconds()),
		AlgorithmUsed:     req.Algorithm.String(),
		CompletedAt:       timestamppb.Now(),
	}

	return result, nil
}

// ============ MANAGEMENT ============

// SaveSimulation сохраняет симуляцию
func (s *SimulationService) SaveSimulation(
	ctx context.Context,
	req *simulationv1.SaveSimulationRequest,
) (*simulationv1.SaveSimulationResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.SaveSimulation")
	defer span.End()

	if req.UserId == "" {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeInvalidArgument, "user_id is required"),
		)
	}

	var graphData []byte
	if req.Graph != nil {
		var err error
		graphData, err = protojson.Marshal(req.Graph)
		if err != nil {
			logger.Log.Warn("Failed to marshal graph data", "error", err)
			// Продолжаем без графа - это некритичная ошибка
			graphData = nil
		}
	}

	// Конвертируем теги
	tags := make([]string, 0, len(req.Tags))
	for k, v := range req.Tags {
		tags = append(tags, k+":"+v)
	}

	sim := &repository.Simulation{
		UserID:         req.UserId,
		Name:           req.Name,
		Description:    req.Description,
		SimulationType: req.Type.String(),
		GraphData:      graphData,
		RequestData:    req.RequestData,
		ResponseData:   req.ResponseData,
		Tags:           tags,
	}

	if req.Graph != nil {
		sim.NodeCount = len(req.Graph.Nodes)
		sim.EdgeCount = len(req.Graph.Edges)
	}

	if err := s.repo.Create(ctx, sim); err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to save simulation"),
		)
	}

	return &simulationv1.SaveSimulationResponse{
		SimulationId: sim.ID,
		CreatedAt:    timestamppb.New(sim.CreatedAt),
	}, nil
}

// GetSimulation получает симуляцию
func (s *SimulationService) GetSimulation(
	ctx context.Context,
	req *simulationv1.GetSimulationRequest,
) (*simulationv1.GetSimulationResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.GetSimulation")
	defer span.End()

	sim, err := s.repo.GetByUserAndID(ctx, req.UserId, req.SimulationId)
	if err != nil {
		if errors.Is(err, repository.ErrSimulationNotFound) {
			return nil, pkgerrors.ToGRPC(
				pkgerrors.New(pkgerrors.CodeNotFound, "simulation not found"),
			)
		}
		if errors.Is(err, repository.ErrAccessDenied) {
			return nil, pkgerrors.ToGRPC(
				pkgerrors.New(pkgerrors.CodePermissionDenied, "access denied"),
			)
		}
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get simulation"),
		)
	}

	// Конвертируем теги
	tags := make(map[string]string)
	for _, tag := range sim.Tags {
		parts := splitOnce(tag, ":")
		if len(parts) == 2 {
			tags[parts[0]] = parts[1]
		}
	}

	return &simulationv1.GetSimulationResponse{
		Record: &simulationv1.SimulationRecord{
			Id:           sim.ID,
			UserId:       sim.UserID,
			Name:         sim.Name,
			Description:  sim.Description,
			Type:         parseSimulationType(sim.SimulationType),
			CreatedAt:    timestamppb.New(sim.CreatedAt),
			RequestData:  sim.RequestData,
			ResponseData: sim.ResponseData,
			Tags:         tags,
		},
	}, nil
}

// ListSimulations возвращает список симуляций
func (s *SimulationService) ListSimulations(
	ctx context.Context,
	req *simulationv1.ListSimulationsRequest,
) (*simulationv1.ListSimulationsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "SimulationService.ListSimulations")
	defer span.End()

	opts := &repository.ListOptions{
		Limit:  20,
		Offset: 0,
	}
	if req.Pagination != nil {
		if req.Pagination.PageSize > 0 {
			opts.Limit = int(req.Pagination.PageSize)
		}
		if req.Pagination.Page > 0 {
			opts.Offset = int((req.Pagination.Page - 1) * req.Pagination.PageSize)
		}
	}

	simType := ""
	if req.Type != simulationv1.SimulationType_SIMULATION_TYPE_UNSPECIFIED {
		simType = req.Type.String()
	}

	sims, total, err := s.repo.List(ctx, req.UserId, simType, opts)
	if err != nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to list simulations"),
		)
	}

	summaries := make([]*simulationv1.SimulationSummary, 0, len(sims))
	for _, sim := range sims {
		tags := make(map[string]string)
		for _, tag := range sim.Tags {
			parts := splitOnce(tag, ":")
			if len(parts) == 2 {
				tags[parts[0]] = parts[1]
			}
		}

		summaries = append(summaries, &simulationv1.SimulationSummary{
			Id:        sim.ID,
			Name:      sim.Name,
			Type:      parseSimulationType(sim.SimulationType),
			CreatedAt: timestamppb.New(sim.CreatedAt),
			Tags:      tags,
		})
	}

	pageSize := int32(opts.Limit)
	currentPage := int32(1)
	if opts.Limit > 0 {
		currentPage = int32(opts.Offset/opts.Limit) + 1
	}
	totalPages := int32(1)
	if opts.Limit > 0 {
		totalPages = int32((total + int64(opts.Limit) - 1) / int64(opts.Limit))
	}

	return &simulationv1.ListSimulationsResponse{
		Simulations: summaries,
		Pagination: &commonv1.PaginationResponse{
			CurrentPage: currentPage,
			PageSize:    pageSize,
			TotalPages:  totalPages,
			TotalItems:  total,
			HasNext:     int64(opts.Offset+opts.Limit) < total,
			HasPrevious: opts.Offset > 0,
		},
	}, nil
}

// Health возвращает статус сервиса
func (s *SimulationService) Health(
	ctx context.Context,
	req *simulationv1.HealthRequest,
) (*simulationv1.HealthResponse, error) {
	return &simulationv1.HealthResponse{
		Status:        "SERVING",
		Version:       s.version,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
	}, nil
}

// ============ HELPER FUNCTIONS ============

// ============ BOTTLENECK CHANGES ANALYSIS ============

// bottleneckChangeAnalyzer анализатор изменений bottleneck
type bottleneckChangeAnalyzer struct {
	baseEdges map[string]*commonv1.Edge
	modEdges  map[string]*commonv1.Edge
	changes   []*simulationv1.BottleneckChange
}

func newBottleneckChangeAnalyzer(baseResult, modResult *client.SolveResult) *bottleneckChangeAnalyzer {
	a := &bottleneckChangeAnalyzer{
		baseEdges: make(map[string]*commonv1.Edge),
		modEdges:  make(map[string]*commonv1.Edge),
		changes:   make([]*simulationv1.BottleneckChange, 0),
	}

	if baseResult.Graph != nil {
		for _, e := range baseResult.Graph.Edges {
			a.baseEdges[edgeKey(e.From, e.To)] = e
		}
	}

	if modResult.Graph != nil {
		for _, e := range modResult.Graph.Edges {
			a.modEdges[edgeKey(e.From, e.To)] = e
		}
	}

	return a
}

func (s *SimulationService) findBottleneckChanges(
	baseResult, modResult *client.SolveResult,
) []*simulationv1.BottleneckChange {
	analyzer := newBottleneckChangeAnalyzer(baseResult, modResult)
	analyzer.analyzeExistingEdges()
	analyzer.analyzeNewEdges()
	return analyzer.changes
}

func (a *bottleneckChangeAnalyzer) analyzeExistingEdges() {
	for key, baseEdge := range a.baseEdges {
		baseUtil := a.calculateUtilization(baseEdge)
		modEdge, exists := a.modEdges[key]

		if !exists {
			a.handleRemovedEdge(baseEdge, baseUtil)
			continue
		}

		modUtil := a.calculateUtilization(modEdge)
		a.compareUtilizations(baseEdge, baseUtil, modUtil)
	}
}

func (a *bottleneckChangeAnalyzer) analyzeNewEdges() {
	for key, modEdge := range a.modEdges {
		if _, exists := a.baseEdges[key]; exists {
			continue
		}

		modUtil := a.calculateUtilization(modEdge)
		if modUtil >= 0.9 {
			a.addChange(modEdge, simulationv1.BottleneckChangeType_BOTTLENECK_CHANGE_TYPE_NEW, 0, modUtil)
		}
	}
}

func (a *bottleneckChangeAnalyzer) calculateUtilization(edge *commonv1.Edge) float64 {
	if edge.Capacity <= 0 {
		return 0
	}
	return edge.CurrentFlow / edge.Capacity
}

func (a *bottleneckChangeAnalyzer) handleRemovedEdge(edge *commonv1.Edge, util float64) {
	if util >= 0.9 {
		a.addChange(edge, simulationv1.BottleneckChangeType_BOTTLENECK_CHANGE_TYPE_RESOLVED, util, 0)
	}
}

func (a *bottleneckChangeAnalyzer) compareUtilizations(edge *commonv1.Edge, baseUtil, modUtil float64) {
	wasBottleneck := baseUtil >= 0.9
	isBottleneck := modUtil >= 0.9

	switch {
	case !wasBottleneck && isBottleneck:
		a.addChange(edge, simulationv1.BottleneckChangeType_BOTTLENECK_CHANGE_TYPE_NEW, baseUtil, modUtil)
	case wasBottleneck && !isBottleneck:
		a.addChange(edge, simulationv1.BottleneckChangeType_BOTTLENECK_CHANGE_TYPE_RESOLVED, baseUtil, modUtil)
	case wasBottleneck && isBottleneck:
		a.handleBottleneckChange(edge, baseUtil, modUtil)
	}
}

func (a *bottleneckChangeAnalyzer) handleBottleneckChange(edge *commonv1.Edge, baseUtil, modUtil float64) {
	const threshold = 0.05

	if modUtil > baseUtil+threshold {
		a.addChange(edge, simulationv1.BottleneckChangeType_BOTTLENECK_CHANGE_TYPE_WORSENED, baseUtil, modUtil)
	} else if modUtil < baseUtil-threshold {
		a.addChange(edge, simulationv1.BottleneckChangeType_BOTTLENECK_CHANGE_TYPE_IMPROVED, baseUtil, modUtil)
	}
}

func (a *bottleneckChangeAnalyzer) addChange(
	edge *commonv1.Edge,
	changeType simulationv1.BottleneckChangeType,
	oldUtil, newUtil float64,
) {
	a.changes = append(a.changes, &simulationv1.BottleneckChange{
		Edge:           &commonv1.EdgeKey{From: edge.From, To: edge.To},
		ChangeType:     changeType,
		OldUtilization: oldUtil,
		NewUtilization: newUtil,
	})
}

func (s *SimulationService) sortScenariosByFlow(scenarios []*simulationv1.ScenarioResultWithRank) {
	for i := 0; i < len(scenarios); i++ {
		for j := i + 1; j < len(scenarios); j++ {
			if scenarios[j].Result.MaxFlow > scenarios[i].Result.MaxFlow {
				scenarios[i], scenarios[j] = scenarios[j], scenarios[i]
			}
		}
	}
}

func (s *SimulationService) generateRecommendation(scenarios []*simulationv1.ScenarioResultWithRank, best string) string {
	if best == "" {
		return "Все сценарии показали худшие результаты, чем базовый"
	}
	return "Рекомендуется сценарий: " + best
}

func (s *SimulationService) calculateModificationCost(mods []*simulationv1.Modification, costPerUnit float64) float64 {
	var total float64
	for _, mod := range mods {
		if mod.Type == simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE {
			if v, ok := mod.Change.(*simulationv1.Modification_Delta); ok && v.Delta > 0 {
				total += v.Delta * costPerUnit
			}
		}
	}
	return total
}

func (s *SimulationService) removeEdgeFromGraph(g *commonv1.Graph, from, to int64) *commonv1.Graph {
	clone := engine.CloneGraph(g)
	newEdges := make([]*commonv1.Edge, 0, len(clone.Edges)-1)
	for _, e := range clone.Edges {
		if e.From != from || e.To != to {
			newEdges = append(newEdges, e)
		}
	}
	clone.Edges = newEdges
	return clone
}

func (s *SimulationService) removeNodeFromGraph(g *commonv1.Graph, nodeID int64) *commonv1.Graph {
	clone := engine.CloneGraph(g)

	newNodes := make([]*commonv1.Node, 0, len(clone.Nodes)-1)
	for _, n := range clone.Nodes {
		if n.Id != nodeID {
			newNodes = append(newNodes, n)
		}
	}
	clone.Nodes = newNodes

	newEdges := make([]*commonv1.Edge, 0)
	for _, e := range clone.Edges {
		if e.From != nodeID && e.To != nodeID {
			newEdges = append(newEdges, e)
		}
	}
	clone.Edges = newEdges

	return clone
}

func (s *SimulationService) countAffectedEdges(g *commonv1.Graph, nodeID int64) int {
	count := 0
	for _, e := range g.Edges {
		if e.From == nodeID || e.To == nodeID {
			count++
		}
	}
	return count
}

func (s *SimulationService) sortCriticalEdges(edges []*simulationv1.CriticalEdge) {
	for i := 0; i < len(edges); i++ {
		for j := i + 1; j < len(edges); j++ {
			if edges[j].CriticalityScore > edges[i].CriticalityScore {
				edges[i], edges[j] = edges[j], edges[i]
			}
		}
	}
}

func (s *SimulationService) sortCriticalNodes(nodes []*simulationv1.CriticalNode) {
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[j].CriticalityScore > nodes[i].CriticalityScore {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}
}

func (s *SimulationService) calculateResilienceScore(
	edges []*simulationv1.CriticalEdge,
	nodes []*simulationv1.CriticalNode,
	totalEdges, totalNodes int,
) float64 {
	if totalEdges == 0 && totalNodes == 0 {
		return 1.0
	}

	criticalRatio := float64(len(edges)+len(nodes)) / float64(totalEdges+totalNodes)
	return 1.0 - criticalRatio
}

func (s *SimulationService) calculateFailureStats(results []*simulationv1.FailureScenarioResult, baseFlow float64) *simulationv1.FailureStats {
	if len(results) == 0 {
		return &simulationv1.FailureStats{}
	}

	var totalLoss, maxLoss, totalProbability float64
	var disconnections int

	for _, r := range results {
		prob := r.Probability
		if prob <= 0 {
			prob = 1.0 / float64(len(results))
		}
		totalProbability += prob

		loss := baseFlow - r.Result.MaxFlow
		if loss > 0 {
			totalLoss += loss * prob
			if loss > maxLoss {
				maxLoss = loss
			}
		}
		if r.NetworkDisconnected {
			disconnections++
		}
	}

	return &simulationv1.FailureStats{
		ExpectedFlowLoss:           totalLoss,
		MaxFlowLoss:                maxLoss,
		ProbabilityOfDisconnection: float64(disconnections) / float64(len(results)),
	}
}

func (s *SimulationService) generateResilienceRecommendations(
	results []*simulationv1.FailureScenarioResult,
	g *commonv1.Graph,
) []*simulationv1.ResilienceRecommendation {
	var recs []*simulationv1.ResilienceRecommendation

	for _, r := range results {
		if r.NetworkDisconnected {
			recs = append(recs, &simulationv1.ResilienceRecommendation{
				Type:                 simulationv1.RecommendationType_RECOMMENDATION_TYPE_ADD_REDUNDANCY,
				Description:          fmt.Sprintf("Добавьте резервные пути для сценария: %s", r.ScenarioName),
				EstimatedImprovement: 0.5, // Примерная оценка
			})
		}
	}

	// Проверяем общий уровень резервирования
	if len(g.Nodes) > 0 && float64(len(g.Edges))/float64(len(g.Nodes)) < 2.0 {
		recs = append(recs, &simulationv1.ResilienceRecommendation{
			Type:                 simulationv1.RecommendationType_RECOMMENDATION_TYPE_ADD_BACKUP_ROUTE,
			Description:          "Низкий уровень резервирования сети. Рекомендуется добавить дополнительные связи.",
			EstimatedImprovement: 0.3,
		})
	}

	return recs
}

func (s *SimulationService) generateRandomFailureScenarios(
	g *commonv1.Graph,
	config *simulationv1.RandomFailureConfig,
) []*simulationv1.FailureScenario {
	var scenarios []*simulationv1.FailureScenario

	numScenarios := int(config.NumScenarios)
	if numScenarios <= 0 {
		numScenarios = 10
	}

	edgeProb := config.EdgeFailureProbability
	if edgeProb <= 0 {
		edgeProb = 0.1
	}

	maxFailures := int(config.MaxSimultaneousFailures)
	if maxFailures <= 0 {
		maxFailures = 3
	}

	// Простая генерация: для каждого сценария случайно выбираем рёбра
	for i := 0; i < numScenarios; i++ {
		scenario := &simulationv1.FailureScenario{
			Name:        fmt.Sprintf("Random Scenario %d", i+1),
			Probability: 1.0 / float64(numScenarios),
		}

		failedCount := 0
		for _, edge := range g.Edges {
			if failedCount >= maxFailures {
				break
			}
			// Простой детерминистический выбор на основе индекса
			// (в реальной реализации использовать rand)
			if (int(edge.From)+int(edge.To)+i)%10 < int(edgeProb*10) {
				scenario.FailedEdges = append(scenario.FailedEdges, &commonv1.EdgeKey{
					From: edge.From,
					To:   edge.To,
				})
				failedCount++
			}
		}

		scenarios = append(scenarios, scenario)
	}

	return scenarios
}

func parseSimulationType(s string) simulationv1.SimulationType {
	switch s {
	case "SIMULATION_TYPE_WHAT_IF":
		return simulationv1.SimulationType_SIMULATION_TYPE_WHAT_IF
	case "SIMULATION_TYPE_TIME":
		return simulationv1.SimulationType_SIMULATION_TYPE_TIME
	case "SIMULATION_TYPE_MONTE_CARLO":
		return simulationv1.SimulationType_SIMULATION_TYPE_MONTE_CARLO
	case "SIMULATION_TYPE_SENSITIVITY":
		return simulationv1.SimulationType_SIMULATION_TYPE_SENSITIVITY
	case "SIMULATION_TYPE_FAILURE":
		return simulationv1.SimulationType_SIMULATION_TYPE_FAILURE
	case "SIMULATION_TYPE_RESILIENCE":
		return simulationv1.SimulationType_SIMULATION_TYPE_RESILIENCE
	default:
		return simulationv1.SimulationType_SIMULATION_TYPE_UNSPECIFIED
	}
}

func splitOnce(s, sep string) []string {
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s}
}

func edgeKey(from, to int64) string {
	return fmt.Sprintf("%d->%d", from, to)
}
