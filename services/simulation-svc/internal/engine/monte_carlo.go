// services/simulation-svc/internal/engine/monte_carlo.go
package engine

import (
	"context"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/client"
)

// MonteCarloEngine движок Monte Carlo симуляции
type MonteCarloEngine struct {
	config       *simulationv1.MonteCarloConfig
	rng          *rand.Rand
	solverClient *client.SolverClient
}

// NewMonteCarloEngine создаёт новый движок
func NewMonteCarloEngine(config *simulationv1.MonteCarloConfig, solverClient *client.SolverClient) *MonteCarloEngine {
	seed := config.RandomSeed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &MonteCarloEngine{
		config:       config,
		rng:          rand.New(rand.NewSource(seed)),
		solverClient: solverClient,
	}
}

// MonteCarloResult результат одной итерации
type MonteCarloResult struct {
	Iteration int
	Flow      float64
	Cost      float64
	Error     error
}

// Run запускает Monte Carlo симуляцию
func (e *MonteCarloEngine) Run(
	ctx context.Context,
	baseGraph *commonv1.Graph,
	uncertainties []*simulationv1.UncertaintySpec,
	algorithm commonv1.Algorithm,
	progressChan chan<- *simulationv1.MonteCarloProgress,
) (*simulationv1.RunMonteCarloResponse, error) {
	numIterations := int(e.config.NumIterations)
	if numIterations <= 0 {
		numIterations = 1000
	}

	results := make([]MonteCarloResult, 0, numIterations)
	var mu sync.Mutex

	// Определяем количество воркеров
	numWorkers := runtime.NumCPU()
	if e.config.MaxWorkers > 0 && int(e.config.MaxWorkers) < numWorkers {
		numWorkers = int(e.config.MaxWorkers)
	}
	if !e.config.Parallel {
		numWorkers = 1
	}

	// Канал задач
	tasks := make(chan int, numIterations)
	var wg sync.WaitGroup

	// Запускаем воркеры
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// Локальный RNG для потокобезопасности
			localRng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for iteration := range tasks {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Применяем случайные модификации
				modifiedGraph := e.applyUncertainties(baseGraph, uncertainties, localRng)

				// Решаем через клиент
				result, err := e.solverClient.Solve(ctx, modifiedGraph, algorithm, nil)

				mcResult := MonteCarloResult{
					Iteration: iteration,
					Error:     err,
				}

				if result != nil {
					mcResult.Flow = result.MaxFlow
					mcResult.Cost = result.TotalCost
				}

				mu.Lock()
				results = append(results, mcResult)

				// Отправляем прогресс
				if progressChan != nil && len(results)%10 == 0 {
					stats := e.calculateRunningStats(results)
					select {
					case progressChan <- &simulationv1.MonteCarloProgress{
						Iteration:       int32(len(results)),
						TotalIterations: int32(numIterations),
						ProgressPercent: float64(len(results)) / float64(numIterations) * 100,
						CurrentMeanFlow: stats.Mean,
						CurrentStdDev:   stats.StdDev,
						Status:          "running",
					}:
					default:
					}
				}
				mu.Unlock()
			}
		}(w)
	}

	// Отправляем задачи
	for i := 0; i < numIterations; i++ {
		tasks <- i
	}
	close(tasks)

	// Ждём завершения
	wg.Wait()

	// Анализируем результаты
	return e.analyzeResults(results)
}

func (e *MonteCarloEngine) applyUncertainties(
	baseGraph *commonv1.Graph,
	uncertainties []*simulationv1.UncertaintySpec,
	rng *rand.Rand,
) *commonv1.Graph {
	graph := CloneGraph(baseGraph)

	for _, u := range uncertainties {
		multiplier := e.sampleDistribution(u.Distribution, rng)

		switch u.Type {
		case simulationv1.UncertaintyType_UNCERTAINTY_TYPE_EDGE:
			e.applyEdgeUncertainty(graph, u.Edge, u.Target, multiplier)

		case simulationv1.UncertaintyType_UNCERTAINTY_TYPE_NODE:
			e.applyNodeUncertainty(graph, u.NodeId, u.Target, multiplier)

		case simulationv1.UncertaintyType_UNCERTAINTY_TYPE_GLOBAL:
			e.applyGlobalUncertainty(graph, u.Target, multiplier)
		}
	}

	return graph
}

func (e *MonteCarloEngine) sampleDistribution(dist *simulationv1.Distribution, rng *rand.Rand) float64 {
	if dist == nil {
		return 1.0
	}

	switch dist.Type {
	case simulationv1.DistributionType_DISTRIBUTION_TYPE_NORMAL:
		return rng.NormFloat64()*dist.Param2 + dist.Param1

	case simulationv1.DistributionType_DISTRIBUTION_TYPE_UNIFORM:
		return dist.Param1 + rng.Float64()*(dist.Param2-dist.Param1)

	case simulationv1.DistributionType_DISTRIBUTION_TYPE_TRIANGULAR:
		u := rng.Float64()
		min, max, mode := dist.Param1, dist.Param2, dist.Param3
		fc := (mode - min) / (max - min)
		if u < fc {
			return min + math.Sqrt(u*(max-min)*(mode-min))
		}
		return max - math.Sqrt((1-u)*(max-min)*(max-mode))

	case simulationv1.DistributionType_DISTRIBUTION_TYPE_LOGNORMAL:
		return math.Exp(rng.NormFloat64()*dist.Param2 + dist.Param1)

	case simulationv1.DistributionType_DISTRIBUTION_TYPE_EXPONENTIAL:
		return rng.ExpFloat64() * dist.Param1

	default:
		return 1.0
	}
}

func (e *MonteCarloEngine) applyEdgeUncertainty(
	graph *commonv1.Graph,
	edgeKey *commonv1.EdgeKey,
	target simulationv1.ModificationTarget,
	multiplier float64,
) {
	for _, edge := range graph.Edges {
		if edge.From == edgeKey.From && edge.To == edgeKey.To {
			switch target {
			case simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY:
				edge.Capacity *= multiplier
			case simulationv1.ModificationTarget_MODIFICATION_TARGET_COST:
				edge.Cost *= multiplier
			}
			return
		}
	}
}

func (e *MonteCarloEngine) applyNodeUncertainty(
	graph *commonv1.Graph,
	nodeID int64,
	target simulationv1.ModificationTarget,
	multiplier float64,
) {
	for _, node := range graph.Nodes {
		if node.Id == nodeID {
			switch target {
			case simulationv1.ModificationTarget_MODIFICATION_TARGET_SUPPLY:
				node.Supply *= multiplier
			case simulationv1.ModificationTarget_MODIFICATION_TARGET_DEMAND:
				node.Demand *= multiplier
			}
			return
		}
	}
}

func (e *MonteCarloEngine) applyGlobalUncertainty(
	graph *commonv1.Graph,
	target simulationv1.ModificationTarget,
	multiplier float64,
) {
	for _, edge := range graph.Edges {
		switch target {
		case simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY:
			edge.Capacity *= multiplier
		case simulationv1.ModificationTarget_MODIFICATION_TARGET_COST:
			edge.Cost *= multiplier
		}
	}
}

type runningStats struct {
	Mean   float64
	StdDev float64
}

func (e *MonteCarloEngine) calculateRunningStats(results []MonteCarloResult) runningStats {
	if len(results) == 0 {
		return runningStats{}
	}

	var sum, sumSq float64
	for _, r := range results {
		sum += r.Flow
		sumSq += r.Flow * r.Flow
	}

	n := float64(len(results))
	mean := sum / n
	variance := sumSq/n - mean*mean
	stdDev := math.Sqrt(math.Max(0, variance))

	return runningStats{Mean: mean, StdDev: stdDev}
}

func (e *MonteCarloEngine) analyzeResults(results []MonteCarloResult) (*simulationv1.RunMonteCarloResponse, error) {
	n := len(results)
	if n == 0 {
		return &simulationv1.RunMonteCarloResponse{Success: false}, nil
	}

	// Собираем значения
	flows := make([]float64, n)
	costs := make([]float64, n)
	for i, r := range results {
		flows[i] = r.Flow
		costs[i] = r.Cost
	}

	// Статистика потока
	flowStats := calculateStats(flows, e.config.ConfidenceLevel)
	costStats := calculateStats(costs, e.config.ConfidenceLevel)

	// Гистограммы
	flowHist := buildHistogram(flows, 20)
	costHist := buildHistogram(costs, 20)

	// Percentiles
	flowPercentiles := calculatePercentiles(flows)
	costPercentiles := calculatePercentiles(costs)

	// Анализ рисков
	riskAnalysis := analyzeRisks(flows, flowStats.Mean)

	return &simulationv1.RunMonteCarloResponse{
		Success:         true,
		FlowStats:       flowStats,
		CostStats:       costStats,
		FlowHistogram:   flowHist,
		CostHistogram:   costHist,
		FlowPercentiles: flowPercentiles,
		CostPercentiles: costPercentiles,
		RiskAnalysis:    riskAnalysis,
	}, nil
}

func calculateStats(values []float64, confidenceLevel float64) *simulationv1.MonteCarloStats {
	n := float64(len(values))
	if n == 0 {
		return &simulationv1.MonteCarloStats{}
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Базовая статистика
	var sum, sumSq float64
	for _, v := range values {
		sum += v
		sumSq += v * v
	}
	mean := sum / n
	variance := sumSq/n - mean*mean
	stdDev := math.Sqrt(math.Max(0, variance))

	// Моменты высших порядков
	var sumCube, sumQuad float64
	for _, v := range values {
		d := v - mean
		sumCube += d * d * d
		sumQuad += d * d * d * d
	}

	skewness := 0.0
	kurtosis := 0.0
	if stdDev > 0 {
		skewness = (sumCube / n) / stdDev * stdDev * stdDev
		kurtosis = (sumQuad/n)/math.Pow(stdDev, 4) - 3
	}

	// Confidence interval
	zScore := 1.96 // 95% CI
	if confidenceLevel > 0 && confidenceLevel < 1 {
		zScore = normalInverse((1 + confidenceLevel) / 2)
	}
	marginOfError := zScore * stdDev / math.Sqrt(n)

	return &simulationv1.MonteCarloStats{
		Mean:                   mean,
		StdDev:                 stdDev,
		Min:                    sorted[0],
		Max:                    sorted[len(sorted)-1],
		Median:                 sorted[len(sorted)/2],
		Variance:               variance,
		Skewness:               skewness,
		Kurtosis:               kurtosis,
		ConfidenceIntervalLow:  mean - marginOfError,
		ConfidenceIntervalHigh: mean + marginOfError,
	}
}

func buildHistogram(values []float64, numBuckets int) []*simulationv1.HistogramBucket {
	if len(values) == 0 {
		return nil
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	min, max := sorted[0], sorted[len(sorted)-1]
	bucketWidth := (max - min) / float64(numBuckets)
	if bucketWidth == 0 {
		bucketWidth = 1
	}

	buckets := make([]*simulationv1.HistogramBucket, numBuckets)
	for i := range buckets {
		buckets[i] = &simulationv1.HistogramBucket{
			LowerBound: min + float64(i)*bucketWidth,
			UpperBound: min + float64(i+1)*bucketWidth,
		}
	}

	for _, v := range values {
		idx := int((v - min) / bucketWidth)
		if idx >= numBuckets {
			idx = numBuckets - 1
		}
		if idx < 0 {
			idx = 0
		}
		buckets[idx].Count++
	}

	n := float64(len(values))
	for _, b := range buckets {
		b.Frequency = float64(b.Count) / n
	}

	return buckets
}

func calculatePercentiles(values []float64) map[string]float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	percentile := func(p float64) float64 {
		idx := int(p / 100 * float64(len(sorted)-1))
		return sorted[idx]
	}

	return map[string]float64{
		"p5":  percentile(5),
		"p10": percentile(10),
		"p25": percentile(25),
		"p50": percentile(50),
		"p75": percentile(75),
		"p90": percentile(90),
		"p95": percentile(95),
		"p99": percentile(99),
	}
}

func analyzeRisks(flows []float64, meanFlow float64) *simulationv1.RiskAnalysis {
	sorted := make([]float64, len(flows))
	copy(sorted, flows)
	sort.Float64s(sorted)

	varIndex := int(0.05 * float64(len(sorted)))
	var valueAtRisk float64
	if varIndex < len(sorted) {
		valueAtRisk = meanFlow - sorted[varIndex]
	}

	var sumBelow float64
	countBelow := 0
	threshold := sorted[varIndex]
	for _, f := range sorted {
		if f <= threshold {
			sumBelow += f
			countBelow++
		}
	}
	var expectedShortfall float64
	if countBelow > 0 {
		expectedShortfall = meanFlow - sumBelow/float64(countBelow)
	}

	return &simulationv1.RiskAnalysis{
		ValueAtRisk:       valueAtRisk,
		ExpectedShortfall: expectedShortfall,
		WorstCaseFlow:     sorted[0],
		BestCaseFlow:      sorted[len(sorted)-1],
	}
}

func normalInverse(p float64) float64 {
	a := []float64{-3.969683028665376e+01, 2.209460984245205e+02,
		-2.759285104469687e+02, 1.383577518672690e+02,
		-3.066479806614716e+01, 2.506628277459239e+00}
	b := []float64{-5.447609879822406e+01, 1.615858368580409e+02,
		-1.556989798598866e+02, 6.680131188771972e+01, -1.328068155288572e+01}
	c := []float64{-7.784894002430293e-03, -3.223964580411365e-01,
		-2.400758277161838e+00, -2.549732539343734e+00,
		4.374664141464968e+00, 2.938163982698783e+00}
	d := []float64{7.784695709041462e-03, 3.224671290700398e-01,
		2.445134137142996e+00, 3.754408661907416e+00}

	pLow := 0.02425
	pHigh := 1 - pLow

	var q, r float64
	if p < pLow {
		q = math.Sqrt(-2 * math.Log(p))
		return (((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	} else if p <= pHigh {
		q = p - 0.5
		r = q * q
		return (((((a[0]*r+a[1])*r+a[2])*r+a[3])*r+a[4])*r + a[5]) * q /
			(((((b[0]*r+b[1])*r+b[2])*r+b[3])*r+b[4])*r + 1)
	} else {
		q = math.Sqrt(-2 * math.Log(1-p))
		return -(((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	}
}
