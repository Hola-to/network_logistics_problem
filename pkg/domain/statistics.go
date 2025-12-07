package domain

// GraphStatistics статистика графа
type GraphStatistics struct {
	NodeCount          int64
	EdgeCount          int64
	WarehouseCount     int64
	DeliveryPointCount int64
	TotalCapacity      float64
	AverageEdgeLength  float64
	IsConnected        bool
	Density            float64
	AverageDegree      float64
	MaxDegree          int
	MinDegree          int
}

// FlowStatistics статистика потока
type FlowStatistics struct {
	TotalFlow          float64
	TotalCost          float64
	AverageUtilization float64
	SaturatedEdges     int64
	ZeroFlowEdges      int64
	ActiveEdges        int64
	Bottlenecks        []EdgeKey
}

// CalculateGraphStatistics вычисляет статистику графа
func CalculateGraphStatistics(g *Graph) *GraphStatistics {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := &GraphStatistics{
		NodeCount: int64(len(g.Nodes)),
		EdgeCount: int64(len(g.Edges)),
		MinDegree: int(^uint(0) >> 1), // MaxInt
	}

	var totalLength float64
	degree := make(map[int64]int)

	// Подсчёт узлов по типам
	for _, node := range g.Nodes {
		switch node.Type {
		case NodeTypeWarehouse:
			stats.WarehouseCount++
		case NodeTypeDeliveryPoint:
			stats.DeliveryPointCount++
		}
	}

	// Подсчёт рёбер и степеней
	for _, edge := range g.Edges {
		if IsVirtualNode(edge.From) || IsVirtualNode(edge.To) {
			continue
		}

		stats.TotalCapacity += edge.Capacity
		totalLength += edge.Length

		degree[edge.From]++
		degree[edge.To]++
	}

	// Статистика степеней
	if len(degree) > 0 {
		totalDegree := 0
		for _, d := range degree {
			totalDegree += d
			if d > stats.MaxDegree {
				stats.MaxDegree = d
			}
			if d < stats.MinDegree {
				stats.MinDegree = d
			}
		}
		stats.AverageDegree = float64(totalDegree) / float64(len(degree))
	}

	if stats.MinDegree == int(^uint(0)>>1) {
		stats.MinDegree = 0
	}

	// Средняя длина ребра
	if stats.EdgeCount > 0 {
		stats.AverageEdgeLength = totalLength / float64(stats.EdgeCount)
	}

	// Плотность графа
	if stats.NodeCount > 1 {
		maxEdges := stats.NodeCount * (stats.NodeCount - 1)
		stats.Density = float64(stats.EdgeCount) / float64(maxEdges)
	}

	// Проверка связности
	stats.IsConnected = IsConnected(g)

	return stats
}

// CalculateFlowStatistics вычисляет статистику потока
func CalculateFlowStatistics(g *Graph) *FlowStatistics {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := &FlowStatistics{
		Bottlenecks: make([]EdgeKey, 0),
	}

	var totalUtilization float64
	var activeEdgesCount int64

	for _, edge := range g.Edges {
		if IsVirtualNode(edge.From) || IsVirtualNode(edge.To) {
			continue
		}

		if !edge.HasFlow() {
			stats.ZeroFlowEdges++
			continue
		}

		// Считаем исходящий поток из источника как общий поток
		if edge.From == g.SourceID {
			stats.TotalFlow += edge.CurrentFlow
		}

		activeEdgesCount++
		stats.ActiveEdges++
		stats.TotalCost += edge.CurrentFlow * edge.Cost

		utilization := edge.Utilization()
		totalUtilization += utilization

		if edge.IsSaturated() {
			stats.SaturatedEdges++
			stats.Bottlenecks = append(stats.Bottlenecks, edge.Key())
		}
	}

	if activeEdgesCount > 0 {
		stats.AverageUtilization = totalUtilization / float64(activeEdgesCount)
	}

	return stats
}

// EfficiencyGrade определяет оценку эффективности
type EfficiencyGrade string

const (
	GradeA EfficiencyGrade = "A"
	GradeB EfficiencyGrade = "B"
	GradeC EfficiencyGrade = "C"
	GradeD EfficiencyGrade = "D"
	GradeF EfficiencyGrade = "F"
)

// EfficiencyReport отчёт об эффективности
type EfficiencyReport struct {
	OverallEfficiency   float64
	CapacityUtilization float64
	UnusedEdgesCount    int32
	SaturatedEdgesCount int32
	Grade               EfficiencyGrade
}

// CalculateEfficiency вычисляет эффективность использования сети
func CalculateEfficiency(g *Graph) *EfficiencyReport {
	flowStats := CalculateFlowStatistics(g)

	report := &EfficiencyReport{
		OverallEfficiency:   flowStats.AverageUtilization,
		CapacityUtilization: flowStats.AverageUtilization,
		UnusedEdgesCount:    int32(flowStats.ZeroFlowEdges),
		SaturatedEdgesCount: int32(flowStats.SaturatedEdges),
	}

	// Определяем оценку
	switch {
	case flowStats.AverageUtilization >= 0.8:
		report.Grade = GradeA
	case flowStats.AverageUtilization >= 0.6:
		report.Grade = GradeB
	case flowStats.AverageUtilization >= 0.4:
		report.Grade = GradeC
	case flowStats.AverageUtilization >= 0.2:
		report.Grade = GradeD
	default:
		report.Grade = GradeF
	}

	return report
}

// BottleneckInfo информация об узком месте
type BottleneckInfo struct {
	Edge        EdgeKey
	Utilization float64
	ImpactScore float64
	Severity    BottleneckSeverity
}

// BottleneckSeverity уровень критичности узкого места
type BottleneckSeverity int

const (
	SeverityLow BottleneckSeverity = iota + 1
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// String возвращает строковое представление уровня критичности
func (s BottleneckSeverity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// FindBottlenecks находит узкие места в сети
func FindBottlenecks(g *Graph, threshold float64) []*BottleneckInfo {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var bottlenecks []*BottleneckInfo

	// Вычисляем общий поток для расчёта impact score
	var totalFlow float64
	for _, edge := range g.Edges {
		if !IsVirtualNode(edge.From) && !IsVirtualNode(edge.To) && edge.HasFlow() {
			totalFlow += edge.CurrentFlow
		}
	}

	for _, edge := range g.Edges {
		if IsVirtualNode(edge.From) || IsVirtualNode(edge.To) {
			continue
		}

		if !edge.HasFlow() {
			continue
		}

		utilization := edge.Utilization()

		if utilization >= threshold {
			var severity BottleneckSeverity
			switch {
			case utilization >= CriticalUtilizationThreshold:
				severity = SeverityCritical
			case utilization >= HighUtilizationThreshold:
				severity = SeverityHigh
			case utilization >= MediumUtilizationThreshold:
				severity = SeverityMedium
			default:
				severity = SeverityLow
			}

			impactScore := 0.0
			if totalFlow > Epsilon {
				impactScore = edge.CurrentFlow / totalFlow
			}

			bottlenecks = append(bottlenecks, &BottleneckInfo{
				Edge:        edge.Key(),
				Utilization: utilization,
				ImpactScore: impactScore,
				Severity:    severity,
			})
		}
	}

	return bottlenecks
}
