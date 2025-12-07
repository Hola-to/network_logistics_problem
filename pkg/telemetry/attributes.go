package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
)

// Стандартные ключи атрибутов
const (
	// Граф
	AttrGraphNodes    = "graph.nodes"
	AttrGraphEdges    = "graph.edges"
	AttrGraphSourceID = "graph.source_id"
	AttrGraphSinkID   = "graph.sink_id"

	// Алгоритм
	AttrAlgorithm  = "algorithm.name"
	AttrIterations = "algorithm.iterations"
	AttrMaxFlow    = "algorithm.max_flow"
	AttrTotalCost  = "algorithm.total_cost"
	AttrPathsFound = "algorithm.paths_found"

	// Валидация
	AttrValidationLevel  = "validation.level"
	AttrValidationErrors = "validation.errors"
	AttrValidationPassed = "validation.passed"

	// Аналитика
	AttrBottlenecksCount = "analytics.bottlenecks_count"
	AttrUtilization      = "analytics.utilization"
)

// GraphAttributes возвращает атрибуты графа
func GraphAttributes(nodes, edges int, sourceID, sinkID int64) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int(AttrGraphNodes, nodes),
		attribute.Int(AttrGraphEdges, edges),
		attribute.Int64(AttrGraphSourceID, sourceID),
		attribute.Int64(AttrGraphSinkID, sinkID),
	}
}

// AlgorithmAttributes возвращает атрибуты алгоритма
func AlgorithmAttributes(name string, iterations int, maxFlow, totalCost float64) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrAlgorithm, name),
		attribute.Int(AttrIterations, iterations),
		attribute.Float64(AttrMaxFlow, maxFlow),
		attribute.Float64(AttrTotalCost, totalCost),
	}
}

// ValidationAttributes возвращает атрибуты валидации
func ValidationAttributes(level string, errorsCount int, passed bool) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrValidationLevel, level),
		attribute.Int(AttrValidationErrors, errorsCount),
		attribute.Bool(AttrValidationPassed, passed),
	}
}
