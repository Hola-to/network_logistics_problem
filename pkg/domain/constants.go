package domain

import "math"

// Математические константы
const (
	Epsilon          = 1e-9
	Infinity         = math.MaxFloat64
	NegativeInfinity = -math.MaxFloat64
)

// Пороговые значения для виртуальных узлов
const (
	VirtualNodeThreshold int64 = 0
	SuperSourceID        int64 = -1
	SuperSinkID          int64 = -2
)

// Утилизация и bottleneck пороги
const (
	DefaultBottleneckThreshold   = 0.9
	CriticalUtilizationThreshold = 0.99
	HighUtilizationThreshold     = 0.95
	MediumUtilizationThreshold   = 0.90
	LowUtilizationThreshold      = 0.80
)

// IsVirtualNode проверяет, является ли узел виртуальным
func IsVirtualNode(nodeID int64) bool {
	return nodeID < VirtualNodeThreshold
}

// FloatEquals сравнивает два float64 с учётом Epsilon
func FloatEquals(a, b float64) bool {
	return math.Abs(a-b) < Epsilon
}

// FloatLess проверяет a < b с учётом Epsilon
func FloatLess(a, b float64) bool {
	return a < b-Epsilon
}

// FloatGreater проверяет a > b с учётом Epsilon
func FloatGreater(a, b float64) bool {
	return a > b+Epsilon
}

// IsZero проверяет, равно ли значение нулю
func IsZero(v float64) bool {
	return math.Abs(v) < Epsilon
}

// IsPositive проверяет, положительно ли значение
func IsPositive(v float64) bool {
	return v > Epsilon
}

// Min возвращает минимум двух float64
func Min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Max возвращает максимум двух float64
func Max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
