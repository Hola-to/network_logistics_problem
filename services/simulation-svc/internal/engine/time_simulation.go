package engine

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TimeSimulationEngine движок временной симуляции
type TimeSimulationEngine struct {
	solverClient *client.SolverClient
	rng          *rand.Rand
}

// NewTimeSimulationEngine создаёт новый движок
func NewTimeSimulationEngine(solverClient *client.SolverClient) *TimeSimulationEngine {
	return &TimeSimulationEngine{
		solverClient: solverClient,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RunTimeSimulation запускает временную симуляцию
func (e *TimeSimulationEngine) RunTimeSimulation(
	ctx context.Context,
	req *simulationv1.RunTimeSimulationRequest,
) (*simulationv1.RunTimeSimulationResponse, error) {
	config := e.normalizeConfig(req.TimeConfig)
	steps := e.getSteps(config)
	startTime := e.getStartTime(config)
	stepDuration := e.getStepDuration(config.TimeStep)

	results := make([]*simulationv1.TimeStepResult, 0, steps)
	stats := newTimeSimulationStats()
	criticalTracker := newCriticalPeriodTracker()

	for i := 0; i < steps; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		currentTime := startTime.Add(time.Duration(i) * stepDuration)
		stepResult, err := e.processTimeStep(ctx, req, i, currentTime)
		if err != nil {
			return nil, err
		}

		results = append(results, stepResult)
		stats.update(stepResult)
		criticalTracker.track(i, currentTime, stepResult, stats, stepDuration)
	}

	criticalTracker.finalize(steps, startTime, stepDuration)

	return &simulationv1.RunTimeSimulationResponse{
		Success:         true,
		StepResults:     results,
		Stats:           stats.finalize(steps),
		CriticalPeriods: criticalTracker.periods,
	}, nil
}

// Helper types and methods
type timeSimulationStats struct {
	minFlow, maxFlow, sumFlow, sumFlowSq float64
	minCost, maxCost, sumCost            float64
	stepsWithBottlenecks                 int32
}

func newTimeSimulationStats() *timeSimulationStats {
	return &timeSimulationStats{
		minFlow: math.MaxFloat64,
		minCost: math.MaxFloat64,
	}
}

func (s *timeSimulationStats) update(step *simulationv1.TimeStepResult) {
	if step.MaxFlow < s.minFlow {
		s.minFlow = step.MaxFlow
	}
	if step.MaxFlow > s.maxFlow {
		s.maxFlow = step.MaxFlow
	}
	s.sumFlow += step.MaxFlow
	s.sumFlowSq += step.MaxFlow * step.MaxFlow

	if step.TotalCost < s.minCost {
		s.minCost = step.TotalCost
	}
	if step.TotalCost > s.maxCost {
		s.maxCost = step.TotalCost
	}
	s.sumCost += step.TotalCost

	if len(step.Bottlenecks) > 0 {
		s.stepsWithBottlenecks++
	}
}

func (s *timeSimulationStats) finalize(steps int) *simulationv1.TimeSimulationStats {
	n := float64(steps)
	avgFlow := s.sumFlow / n
	variance := s.sumFlowSq/n - avgFlow*avgFlow
	stdDevFlow := math.Sqrt(math.Max(0, variance))

	return &simulationv1.TimeSimulationStats{
		MinFlow:              s.minFlow,
		MaxFlow:              s.maxFlow,
		AvgFlow:              avgFlow,
		StdDevFlow:           stdDevFlow,
		MinCost:              s.minCost,
		MaxCost:              s.maxCost,
		AvgCost:              s.sumCost / n,
		TotalSteps:           int32(steps),
		StepsWithBottlenecks: s.stepsWithBottlenecks,
	}
}

func (e *TimeSimulationEngine) normalizeConfig(config *simulationv1.TimeSimulationConfig) *simulationv1.TimeSimulationConfig {
	if config == nil {
		return &simulationv1.TimeSimulationConfig{
			NumSteps: 24,
			TimeStep: simulationv1.TimeStep_TIME_STEP_HOUR,
		}
	}
	return config
}

func (e *TimeSimulationEngine) getSteps(config *simulationv1.TimeSimulationConfig) int {
	steps := int(config.NumSteps)
	if steps <= 0 {
		return 24
	}
	return steps
}

func (e *TimeSimulationEngine) getStartTime(config *simulationv1.TimeSimulationConfig) time.Time {
	if config.StartTime != nil {
		return config.StartTime.AsTime()
	}
	return time.Now()
}

func (e *TimeSimulationEngine) processTimeStep(
	ctx context.Context,
	req *simulationv1.RunTimeSimulationRequest,
	step int,
	currentTime time.Time,
) (*simulationv1.TimeStepResult, error) {
	modGraph := e.applyTimePatterns(req.Graph, step, currentTime, req.EdgePatterns, req.NodePatterns)

	solveRes, err := e.solverClient.Solve(ctx, modGraph, req.Algorithm, nil)
	if err != nil {
		return nil, err
	}

	bottlenecks := e.findBottlenecks(solveRes.Graph)

	return &simulationv1.TimeStepResult{
		Step:               int32(step),
		Timestamp:          timestamppb.New(currentTime),
		MaxFlow:            solveRes.MaxFlow,
		TotalCost:          solveRes.TotalCost,
		AverageUtilization: solveRes.AverageUtilization,
		SaturatedEdges:     solveRes.SaturatedEdges,
		Bottlenecks:        bottlenecks,
	}, nil
}

func (e *TimeSimulationEngine) findBottlenecks(graph *commonv1.Graph) []*commonv1.EdgeKey {
	if graph == nil {
		return nil
	}
	var bottlenecks []*commonv1.EdgeKey
	for _, edge := range graph.Edges {
		if edge.Capacity > 0 && edge.CurrentFlow/edge.Capacity >= 0.95 {
			bottlenecks = append(bottlenecks, &commonv1.EdgeKey{From: edge.From, To: edge.To})
		}
	}
	return bottlenecks
}

type criticalPeriodTracker struct {
	periods []*simulationv1.CriticalPeriod
	current *simulationv1.CriticalPeriod
}

func newCriticalPeriodTracker() *criticalPeriodTracker {
	return &criticalPeriodTracker{
		periods: make([]*simulationv1.CriticalPeriod, 0),
	}
}

func (t *criticalPeriodTracker) track(
	step int,
	currentTime time.Time,
	stepResult *simulationv1.TimeStepResult,
	stats *timeSimulationStats,
	stepDuration time.Duration,
) {
	isCritical := t.isCriticalStep(stepResult, stats.maxFlow)

	if isCritical {
		if t.current == nil {
			t.current = &simulationv1.CriticalPeriod{
				StartStep:   int32(step),
				StartTime:   timestamppb.New(currentTime),
				Type:        t.determineCriticalType(stepResult),
				Description: "Начало критического периода",
			}
		}
	} else if t.current != nil {
		t.closePeriod(step-1, currentTime.Add(-stepDuration))
	}
}

func (t *criticalPeriodTracker) isCriticalStep(step *simulationv1.TimeStepResult, maxFlow float64) bool {
	if maxFlow == 0 {
		return false
	}
	threshold := maxFlow * 0.8
	return step.MaxFlow < threshold || len(step.Bottlenecks) > 2
}

func (t *criticalPeriodTracker) determineCriticalType(step *simulationv1.TimeStepResult) simulationv1.CriticalPeriodType {
	if len(step.Bottlenecks) > 3 {
		return simulationv1.CriticalPeriodType_CRITICAL_PERIOD_TYPE_CONGESTION
	}
	if step.AverageUtilization > 0.9 {
		return simulationv1.CriticalPeriodType_CRITICAL_PERIOD_TYPE_HIGH_DEMAND
	}
	return simulationv1.CriticalPeriodType_CRITICAL_PERIOD_TYPE_LOW_CAPACITY
}

func (t *criticalPeriodTracker) closePeriod(endStep int, endTime time.Time) {
	t.current.EndStep = int32(endStep)
	t.current.EndTime = timestamppb.New(endTime)
	t.current.Severity = t.calculateSeverity()
	t.periods = append(t.periods, t.current)
	t.current = nil
}

func (t *criticalPeriodTracker) calculateSeverity() float64 {
	if t.current == nil {
		return 0
	}
	duration := t.current.EndStep - t.current.StartStep + 1
	return math.Min(float64(duration)/10.0, 1.0)
}

func (t *criticalPeriodTracker) finalize(steps int, startTime time.Time, stepDuration time.Duration) {
	if t.current != nil {
		t.current.EndStep = int32(steps - 1)
		t.current.EndTime = timestamppb.New(startTime.Add(time.Duration(steps-1) * stepDuration))
		t.current.Severity = t.calculateSeverity()
		t.periods = append(t.periods, t.current)
		t.current = nil
	}
}

// SimulatePeakLoad симулирует пиковую нагрузку
func (e *TimeSimulationEngine) SimulatePeakLoad(
	ctx context.Context,
	req *simulationv1.SimulatePeakLoadRequest,
) (*simulationv1.SimulatePeakLoadResponse, error) {
	normalResult, err := e.solverClient.Solve(ctx, req.Graph, req.Algorithm, nil)
	if err != nil {
		return nil, err
	}

	peakGraph := e.buildPeakGraph(req)
	peakResult, err := e.solverClient.Solve(ctx, peakGraph, req.Algorithm, nil)
	if err != nil {
		return nil, err
	}

	comparison := CompareResults(normalResult, peakResult)
	overloadedEdges := e.findOverloadedEdges(peakResult.Graph, req.DemandMultiplier)
	recommendations := e.generatePeakLoadRecommendations(overloadedEdges, comparison)

	return &simulationv1.SimulatePeakLoadResponse{
		Success:         true,
		NormalResult:    ToScenarioResult(normalResult, "Normal"),
		PeakResult:      ToScenarioResult(peakResult, "Peak Load"),
		Comparison:      comparison,
		OverloadedEdges: overloadedEdges,
		Recommendations: recommendations,
	}, nil
}

func (e *TimeSimulationEngine) buildPeakGraph(req *simulationv1.SimulatePeakLoadRequest) *commonv1.Graph {
	peakGraph := CloneGraph(req.Graph)

	if req.DemandMultiplier > 0 {
		for _, node := range peakGraph.Nodes {
			if len(req.AffectedNodes) == 0 || containsNode(req.AffectedNodes, node.Id) {
				node.Demand *= req.DemandMultiplier
			}
		}
	}

	if req.CapacityReduction > 0 && req.CapacityReduction < 1 {
		for _, edge := range peakGraph.Edges {
			if len(req.AffectedEdges) == 0 || containsEdge(req.AffectedEdges, edge.From, edge.To) {
				edge.Capacity *= req.CapacityReduction
			}
		}
	}

	return peakGraph
}

func (e *TimeSimulationEngine) findOverloadedEdges(graph *commonv1.Graph, demandMultiplier float64) []*simulationv1.OverloadedEdge {
	if graph == nil {
		return nil
	}

	overloadedEdges := make([]*simulationv1.OverloadedEdge, 0, len(graph.Edges)/10+1)
	for _, edge := range graph.Edges {
		if edge.CurrentFlow < edge.Capacity*0.95 {
			continue
		}

		requiredCapacity := edge.CurrentFlow * demandMultiplier
		shortage := requiredCapacity - edge.Capacity
		if shortage <= 0 {
			continue
		}

		shortagePercent := 0.0
		if edge.Capacity > 0 {
			shortagePercent = (shortage / edge.Capacity) * 100
		}

		overloadedEdges = append(overloadedEdges, &simulationv1.OverloadedEdge{
			Edge:              &commonv1.EdgeKey{From: edge.From, To: edge.To},
			RequiredCapacity:  requiredCapacity,
			AvailableCapacity: edge.Capacity,
			Shortage:          shortage,
			ShortagePercent:   shortagePercent,
		})
	}

	sort.Slice(overloadedEdges, func(i, j int) bool {
		return overloadedEdges[i].Shortage > overloadedEdges[j].Shortage
	})

	return overloadedEdges
}

func (e *TimeSimulationEngine) generatePeakLoadRecommendations(overloaded []*simulationv1.OverloadedEdge, comparison *simulationv1.ScenarioComparison) []string {
	var recs []string

	if len(overloaded) > 0 {
		recs = append(recs, fmt.Sprintf("Обнаружено %d перегруженных рёбер", len(overloaded)))
	}

	if comparison.FlowChangePercent < -10 {
		recs = append(recs, "Значительное снижение пропускной способности при пиковой нагрузке")
		recs = append(recs, "Рекомендуется увеличить capacity критических рёбер")
	}

	if len(overloaded) > 3 {
		recs = append(recs, "Множественные узкие места - рекомендуется добавить альтернативные маршруты")
	}

	return recs
}

// RunWhatIfWithTime запускает what-if анализ с временными параметрами
func (e *TimeSimulationEngine) RunWhatIfWithTime(
	ctx context.Context,
	baseGraph *commonv1.Graph,
	modifications []*simulationv1.Modification,
	timeConfig *simulationv1.TimeSimulationConfig,
	edgePatterns []*simulationv1.EdgeTimePattern,
	nodePatterns []*simulationv1.NodeTimePattern,
	algorithm commonv1.Algorithm,
) (*TimeBasedWhatIfResult, error) {
	if timeConfig == nil {
		timeConfig = &simulationv1.TimeSimulationConfig{
			NumSteps: 24,
			TimeStep: simulationv1.TimeStep_TIME_STEP_HOUR,
		}
	}

	steps := int(timeConfig.NumSteps)
	startTime := time.Now()
	if timeConfig.StartTime != nil {
		startTime = timeConfig.StartTime.AsTime()
	}
	stepDuration := e.getStepDuration(timeConfig.TimeStep)

	baselineResults := make([]*simulationv1.TimeStepResult, 0, steps)
	modifiedResults := make([]*simulationv1.TimeStepResult, 0, steps)
	modifiedBaseGraph := ApplyModifications(baseGraph, modifications)

	for i := 0; i < steps; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		currentTime := startTime.Add(time.Duration(i) * stepDuration)

		baselineRes, err := e.solveWithPatterns(ctx, baseGraph, i, currentTime, edgePatterns, nodePatterns, algorithm)
		if err != nil {
			return nil, err
		}
		baselineResults = append(baselineResults, baselineRes)

		modifiedRes, err := e.solveWithPatterns(ctx, modifiedBaseGraph, i, currentTime, edgePatterns, nodePatterns, algorithm)
		if err != nil {
			return nil, err
		}
		modifiedResults = append(modifiedResults, modifiedRes)
	}

	return e.buildWhatIfResult(baselineResults, modifiedResults, steps), nil
}

func (e *TimeSimulationEngine) solveWithPatterns(
	ctx context.Context,
	graph *commonv1.Graph,
	step int,
	currentTime time.Time,
	edgePatterns []*simulationv1.EdgeTimePattern,
	nodePatterns []*simulationv1.NodeTimePattern,
	algorithm commonv1.Algorithm,
) (*simulationv1.TimeStepResult, error) {
	timeGraph := e.applyTimePatterns(graph, step, currentTime, edgePatterns, nodePatterns)
	res, err := e.solverClient.Solve(ctx, timeGraph, algorithm, nil)
	if err != nil {
		return nil, err
	}

	return &simulationv1.TimeStepResult{
		Step:               int32(step),
		Timestamp:          timestamppb.New(currentTime),
		MaxFlow:            res.MaxFlow,
		TotalCost:          res.TotalCost,
		AverageUtilization: res.AverageUtilization,
		SaturatedEdges:     res.SaturatedEdges,
	}, nil
}

func (e *TimeSimulationEngine) buildWhatIfResult(
	baselineResults, modifiedResults []*simulationv1.TimeStepResult,
	steps int,
) *TimeBasedWhatIfResult {
	baselineStats := e.aggregateTimeResults(baselineResults)
	modifiedStats := e.aggregateTimeResults(modifiedResults)

	timeComparisons := make([]*TimeStepComparison, 0, steps)
	for i := 0; i < steps; i++ {
		timeComparisons = append(timeComparisons, &TimeStepComparison{
			Step:              int32(i),
			Timestamp:         baselineResults[i].Timestamp,
			BaselineFlow:      baselineResults[i].MaxFlow,
			ModifiedFlow:      modifiedResults[i].MaxFlow,
			FlowChange:        modifiedResults[i].MaxFlow - baselineResults[i].MaxFlow,
			FlowChangePercent: calculateChangePercent(baselineResults[i].MaxFlow, modifiedResults[i].MaxFlow),
		})
	}

	return &TimeBasedWhatIfResult{
		BaselineResults: baselineResults,
		ModifiedResults: modifiedResults,
		BaselineStats:   baselineStats,
		ModifiedStats:   modifiedStats,
		TimeComparisons: timeComparisons,
		ImpactPeriods:   e.findHighImpactPeriods(timeComparisons),
	}
}

// TimeBasedWhatIfResult результат what-if с временной привязкой
type TimeBasedWhatIfResult struct {
	BaselineResults []*simulationv1.TimeStepResult
	ModifiedResults []*simulationv1.TimeStepResult
	BaselineStats   *simulationv1.TimeSimulationStats
	ModifiedStats   *simulationv1.TimeSimulationStats
	TimeComparisons []*TimeStepComparison
	ImpactPeriods   []*ImpactPeriod
}

// TimeStepComparison сравнение на одном шаге
type TimeStepComparison struct {
	Step              int32
	Timestamp         *timestamppb.Timestamp
	BaselineFlow      float64
	ModifiedFlow      float64
	FlowChange        float64
	FlowChangePercent float64
}

// ImpactPeriod период с значительным влиянием
type ImpactPeriod struct {
	StartStep         int32
	EndStep           int32
	StartTime         *timestamppb.Timestamp
	EndTime           *timestamppb.Timestamp
	AverageImpact     float64
	MaxImpact         float64
	ImpactDescription string
}

// Вспомогательные методы

func (e *TimeSimulationEngine) getStepDuration(step simulationv1.TimeStep) time.Duration {
	switch step {
	case simulationv1.TimeStep_TIME_STEP_MINUTE:
		return time.Minute
	case simulationv1.TimeStep_TIME_STEP_HOUR:
		return time.Hour
	case simulationv1.TimeStep_TIME_STEP_DAY:
		return 24 * time.Hour
	case simulationv1.TimeStep_TIME_STEP_WEEK:
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func (e *TimeSimulationEngine) applyTimePatterns(
	base *commonv1.Graph,
	step int,
	currentTime time.Time,
	ePatterns []*simulationv1.EdgeTimePattern,
	nPatterns []*simulationv1.NodeTimePattern,
) *commonv1.Graph {
	g := CloneGraph(base)

	for _, p := range ePatterns {
		mult := e.getMultiplier(p.Pattern, step, currentTime)
		for _, edge := range g.Edges {
			if edge.From == p.Edge.From && edge.To == p.Edge.To {
				edge.Capacity *= mult
				break
			}
		}
	}

	for _, p := range nPatterns {
		mult := e.getMultiplier(p.Pattern, step, currentTime)
		for _, node := range g.Nodes {
			if node.Id == p.NodeId {
				switch p.Target {
				case simulationv1.PatternTarget_PATTERN_TARGET_DEMAND:
					node.Demand *= mult
				case simulationv1.PatternTarget_PATTERN_TARGET_SUPPLY:
					node.Supply *= mult
				}
				break
			}
		}
	}

	return g
}

func (e *TimeSimulationEngine) getMultiplier(p *simulationv1.TimePattern, step int, t time.Time) float64 {
	if p == nil {
		return 1.0
	}

	switch p.Type {
	case simulationv1.PatternType_PATTERN_TYPE_CONSTANT:
		return 1.0
	case simulationv1.PatternType_PATTERN_TYPE_HOURLY:
		if len(p.HourlyMultipliers) == 24 {
			return p.HourlyMultipliers[t.Hour()]
		}
	case simulationv1.PatternType_PATTERN_TYPE_DAILY:
		if len(p.DailyMultipliers) == 7 {
			return p.DailyMultipliers[int(t.Weekday())]
		}
	case simulationv1.PatternType_PATTERN_TYPE_CUSTOM:
		for _, point := range p.CustomPoints {
			if int(point.Step) == step {
				return point.Multiplier
			}
		}
	case simulationv1.PatternType_PATTERN_TYPE_RANDOM_NORMAL:
		return e.randomNormalMultiplier(p)
	case simulationv1.PatternType_PATTERN_TYPE_RANDOM_UNIFORM:
		return e.randomUniformMultiplier(p)
	}

	return 1.0
}

func (e *TimeSimulationEngine) randomNormalMultiplier(p *simulationv1.TimePattern) float64 {
	value := e.rng.NormFloat64()*p.StdDev + p.Mean
	if p.MinValue > 0 && value < p.MinValue {
		value = p.MinValue
	}
	if p.MaxValue > 0 && value > p.MaxValue {
		value = p.MaxValue
	}
	return value
}

func (e *TimeSimulationEngine) randomUniformMultiplier(p *simulationv1.TimePattern) float64 {
	minVal := p.Mean - p.StdDev
	maxVal := p.Mean + p.StdDev
	if p.MinValue > 0 {
		minVal = p.MinValue
	}
	if p.MaxValue > 0 {
		maxVal = p.MaxValue
	}
	return minVal + e.rng.Float64()*(maxVal-minVal)
}

func (e *TimeSimulationEngine) aggregateTimeResults(results []*simulationv1.TimeStepResult) *simulationv1.TimeSimulationStats {
	if len(results) == 0 {
		return &simulationv1.TimeSimulationStats{}
	}

	var minFlow, maxFlow, sumFlow float64 = math.MaxFloat64, 0, 0
	var minCost, maxCost, sumCost float64 = math.MaxFloat64, 0, 0
	var stepsWithBottlenecks int32

	for _, r := range results {
		if r.MaxFlow < minFlow {
			minFlow = r.MaxFlow
		}
		if r.MaxFlow > maxFlow {
			maxFlow = r.MaxFlow
		}
		sumFlow += r.MaxFlow

		if r.TotalCost < minCost {
			minCost = r.TotalCost
		}
		if r.TotalCost > maxCost {
			maxCost = r.TotalCost
		}
		sumCost += r.TotalCost

		if len(r.Bottlenecks) > 0 {
			stepsWithBottlenecks++
		}
	}

	n := float64(len(results))

	return &simulationv1.TimeSimulationStats{
		MinFlow:              minFlow,
		MaxFlow:              maxFlow,
		AvgFlow:              sumFlow / n,
		MinCost:              minCost,
		MaxCost:              maxCost,
		AvgCost:              sumCost / n,
		TotalSteps:           int32(len(results)),
		StepsWithBottlenecks: stepsWithBottlenecks,
	}
}

func (e *TimeSimulationEngine) findHighImpactPeriods(comparisons []*TimeStepComparison) []*ImpactPeriod {
	var periods []*ImpactPeriod
	var current *ImpactPeriod
	threshold := 5.0

	for _, c := range comparisons {
		absChange := math.Abs(c.FlowChangePercent)

		if absChange >= threshold {
			if current == nil {
				current = &ImpactPeriod{
					StartStep: c.Step,
					StartTime: c.Timestamp,
					MaxImpact: absChange,
				}
			} else if absChange > current.MaxImpact {
				current.MaxImpact = absChange
			}
		} else if current != nil {
			current.EndStep = c.Step - 1
			if int(c.Step) > 0 && int(c.Step-1) < len(comparisons) {
				current.EndTime = comparisons[c.Step-1].Timestamp
			}
			current.ImpactDescription = e.describeImpact(current.MaxImpact)
			periods = append(periods, current)
			current = nil
		}
	}

	if current != nil && len(comparisons) > 0 {
		last := comparisons[len(comparisons)-1]
		current.EndStep = last.Step
		current.EndTime = last.Timestamp
		current.ImpactDescription = e.describeImpact(current.MaxImpact)
		periods = append(periods, current)
	}

	return periods
}

func (e *TimeSimulationEngine) describeImpact(maxImpact float64) string {
	switch {
	case maxImpact >= 30:
		return "Критическое влияние на пропускную способность"
	case maxImpact >= 15:
		return "Значительное влияние на пропускную способность"
	case maxImpact >= 5:
		return "Умеренное влияние на пропускную способность"
	default:
		return "Незначительное влияние"
	}
}

func containsNode(nodes []int64, id int64) bool {
	for _, n := range nodes {
		if n == id {
			return true
		}
	}
	return false
}

func containsEdge(edges []*commonv1.EdgeKey, from, to int64) bool {
	for _, e := range edges {
		if e.From == from && e.To == to {
			return true
		}
	}
	return false
}

func calculateChangePercent(baseline, modified float64) float64 {
	if baseline == 0 {
		if modified == 0 {
			return 0
		}
		return 100
	}
	return ((modified - baseline) / baseline) * 100
}
