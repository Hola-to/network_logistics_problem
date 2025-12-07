package analysis

import (
	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
)

// Значения по умолчанию
var defaultFixedCosts = &analyticsv1.FixedCostConfig{
	WarehouseCost:       1000.0,
	DeliveryPointCost:   100.0,
	IntersectionCost:    0.0,
	BaseOperationCost:   0.0,
	PerEdgeCost:         0.0,
	PerUnitHandlingCost: 0.0,
	RoadTypeBaseCosts:   map[string]float64{},
}

// CalculateCost вычисляет стоимость потока
func CalculateCost(graph *commonv1.Graph, options *analyticsv1.CostOptions) *analyticsv1.CalculateCostResponse {
	// Нормализуем опции
	opts := normalizeOptions(options)
	fixedCfg := getFixedCostConfig(opts)

	// Инициализация счётчиков
	var (
		transportCost  float64
		handlingCost   float64
		roadBaseCost   float64
		totalFlow      float64
		activeEdges    int32
		costByRoadType = make(map[string]float64)
		costByNodeType = make(map[string]float64)
	)

	// Расчёт транспортных затрат
	for _, edge := range graph.Edges {
		if IsVirtualNode(edge.From) || IsVirtualNode(edge.To) {
			continue
		}
		if edge.CurrentFlow <= Epsilon {
			continue
		}

		activeEdges++
		totalFlow += edge.CurrentFlow

		// Базовая стоимость: flow * cost
		edgeCost := edge.CurrentFlow * edge.Cost

		// Применяем множитель по типу дороги
		roadTypeName := edge.RoadType.String()
		if opts.CostMultipliers != nil {
			if multiplier, ok := opts.CostMultipliers[roadTypeName]; ok {
				edgeCost *= multiplier
			}
		}

		transportCost += edgeCost
		costByRoadType[roadTypeName] += edgeCost

		// Базовая стоимость дороги (за длину)
		if fixedCfg.RoadTypeBaseCosts != nil {
			if baseCost, ok := fixedCfg.RoadTypeBaseCosts[roadTypeName]; ok {
				roadBaseCost += edge.Length * baseCost
			}
		}
	}

	// Расчёт стоимости обработки
	handlingCost = totalFlow * fixedCfg.PerUnitHandlingCost

	// Стоимость за каждое активное ребро
	perEdgeCost := float64(activeEdges) * fixedCfg.PerEdgeCost

	// Расчёт фиксированных затрат по узлам
	fixedCost, nodeStats := calculateNodeFixedCosts(graph, fixedCfg, costByNodeType)

	// Базовая операционная стоимость
	fixedCost += fixedCfg.BaseOperationCost

	// Добавляем стоимость за рёбра
	fixedCost += perEdgeCost

	// Определяем режим расчёта
	mode := opts.Mode
	if mode == analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_UNSPECIFIED {
		if opts.IncludeFixedCosts {
			mode = analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_WITH_FIXED
		} else {
			mode = analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE
		}
	}

	// Собираем итоговую стоимость в зависимости от режима
	var subtotal float64
	switch mode {
	case analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE:
		subtotal = transportCost
		fixedCost = 0
		handlingCost = 0
		roadBaseCost = 0

	case analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_WITH_FIXED:
		subtotal = transportCost + fixedCost

	case analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_FULL:
		subtotal = transportCost + fixedCost + handlingCost + roadBaseCost
	}

	// Применяем скидку и наценку
	discountAmount := subtotal * (opts.DiscountPercent / 100.0)
	afterDiscount := subtotal - discountAmount
	markupAmount := afterDiscount * (opts.MarkupPercent / 100.0)
	totalCost := afterDiscount + markupAmount

	// Валюта
	currency := "RUB"
	if opts.Currency != "" {
		currency = opts.Currency
	}

	return &analyticsv1.CalculateCostResponse{
		TotalCost: totalCost,
		Currency:  currency,
		Breakdown: &analyticsv1.CostBreakdown{
			TransportCost:        transportCost,
			FixedCost:            fixedCost,
			HandlingCost:         handlingCost,
			RoadBaseCost:         roadBaseCost,
			DiscountAmount:       discountAmount,
			MarkupAmount:         markupAmount,
			CostByRoadType:       costByRoadType,
			CostByNodeType:       costByNodeType,
			ActiveWarehouses:     nodeStats.activeWarehouses,
			ActiveDeliveryPoints: nodeStats.activeDeliveryPoints,
			ActiveEdges:          activeEdges,
			TotalFlow:            totalFlow,
		},
	}
}

// nodeStats статистика по узлам
type nodeStats struct {
	activeWarehouses     int32
	activeDeliveryPoints int32
}

// calculateNodeFixedCosts вычисляет фиксированные затраты по узлам
func calculateNodeFixedCosts(
	graph *commonv1.Graph,
	cfg *analyticsv1.FixedCostConfig,
	costByNodeType map[string]float64,
) (float64, nodeStats) {

	var (
		totalCost float64
		stats     nodeStats
	)

	// Определяем активные узлы (через которые идёт поток)
	activeNodes := getActiveNodes(graph)

	for _, node := range graph.Nodes {
		// Можно считать только активные узлы или все
		// Здесь считаем все узлы определённых типов

		nodeTypeName := node.Type.String()
		var nodeCost float64

		switch node.Type {
		case commonv1.NodeType_NODE_TYPE_WAREHOUSE:
			nodeCost = cfg.WarehouseCost
			if activeNodes[node.Id] {
				stats.activeWarehouses++
			}

		case commonv1.NodeType_NODE_TYPE_DELIVERY_POINT:
			nodeCost = cfg.DeliveryPointCost
			if activeNodes[node.Id] {
				stats.activeDeliveryPoints++
			}

		case commonv1.NodeType_NODE_TYPE_INTERSECTION:
			nodeCost = cfg.IntersectionCost
		}

		// Считаем только если узел активен (или если считаем все)
		if activeNodes[node.Id] && nodeCost > 0 {
			totalCost += nodeCost
			costByNodeType[nodeTypeName] += nodeCost
		}
	}

	return totalCost, stats
}

// getActiveNodes возвращает узлы, через которые проходит поток
func getActiveNodes(graph *commonv1.Graph) map[int64]bool {
	active := make(map[int64]bool)

	for _, edge := range graph.Edges {
		if edge.CurrentFlow > Epsilon {
			active[edge.From] = true
			active[edge.To] = true
		}
	}

	return active
}

// normalizeOptions нормализует опции (устанавливает значения по умолчанию)
func normalizeOptions(opts *analyticsv1.CostOptions) *analyticsv1.CostOptions {
	if opts == nil {
		return &analyticsv1.CostOptions{
			Currency:          "RUB",
			IncludeFixedCosts: false,
			Mode:              analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE,
		}
	}
	return opts
}

// getFixedCostConfig возвращает конфигурацию фиксированных затрат
func getFixedCostConfig(opts *analyticsv1.CostOptions) *analyticsv1.FixedCostConfig {
	if opts.FixedCosts != nil {
		return opts.FixedCosts
	}
	return defaultFixedCosts
}

// CalculateCostSimple упрощённый расчёт (только transport cost)
func CalculateCostSimple(graph *commonv1.Graph) float64 {
	var totalCost float64

	for _, edge := range graph.Edges {
		if IsVirtualNode(edge.From) || IsVirtualNode(edge.To) {
			continue
		}
		if edge.CurrentFlow > Epsilon {
			totalCost += edge.CurrentFlow * edge.Cost
		}
	}

	return totalCost
}

// CalculateCostWithDefaults расчёт с дефолтными настройками
func CalculateCostWithDefaults(graph *commonv1.Graph, currency string, includeFixed bool) *analyticsv1.CalculateCostResponse {
	mode := analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_SIMPLE
	if includeFixed {
		mode = analyticsv1.CostCalculationMode_COST_CALCULATION_MODE_WITH_FIXED
	}

	return CalculateCost(graph, &analyticsv1.CostOptions{
		Currency:          currency,
		IncludeFixedCosts: includeFixed,
		Mode:              mode,
		FixedCosts:        defaultFixedCosts,
	})
}

// CreateCostOptions хелпер для создания опций
func CreateCostOptions() *CostOptionsBuilder {
	return &CostOptionsBuilder{
		opts: &analyticsv1.CostOptions{
			Currency:        "RUB",
			CostMultipliers: make(map[string]float64),
			FixedCosts: &analyticsv1.FixedCostConfig{
				RoadTypeBaseCosts: make(map[string]float64),
			},
		},
	}
}

// CostOptionsBuilder билдер для опций
type CostOptionsBuilder struct {
	opts *analyticsv1.CostOptions
}

func (b *CostOptionsBuilder) Currency(c string) *CostOptionsBuilder {
	b.opts.Currency = c
	return b
}

func (b *CostOptionsBuilder) IncludeFixed(v bool) *CostOptionsBuilder {
	b.opts.IncludeFixedCosts = v
	return b
}

func (b *CostOptionsBuilder) Mode(m analyticsv1.CostCalculationMode) *CostOptionsBuilder {
	b.opts.Mode = m
	return b
}

func (b *CostOptionsBuilder) WarehouseCost(v float64) *CostOptionsBuilder {
	b.opts.FixedCosts.WarehouseCost = v
	return b
}

func (b *CostOptionsBuilder) DeliveryPointCost(v float64) *CostOptionsBuilder {
	b.opts.FixedCosts.DeliveryPointCost = v
	return b
}

func (b *CostOptionsBuilder) PerUnitHandlingCost(v float64) *CostOptionsBuilder {
	b.opts.FixedCosts.PerUnitHandlingCost = v
	return b
}

func (b *CostOptionsBuilder) BaseOperationCost(v float64) *CostOptionsBuilder {
	b.opts.FixedCosts.BaseOperationCost = v
	return b
}

func (b *CostOptionsBuilder) PerEdgeCost(v float64) *CostOptionsBuilder {
	b.opts.FixedCosts.PerEdgeCost = v
	return b
}

func (b *CostOptionsBuilder) RoadMultiplier(roadType string, multiplier float64) *CostOptionsBuilder {
	b.opts.CostMultipliers[roadType] = multiplier
	return b
}

func (b *CostOptionsBuilder) RoadBaseCost(roadType string, cost float64) *CostOptionsBuilder {
	b.opts.FixedCosts.RoadTypeBaseCosts[roadType] = cost
	return b
}

func (b *CostOptionsBuilder) Discount(percent float64) *CostOptionsBuilder {
	b.opts.DiscountPercent = percent
	return b
}

func (b *CostOptionsBuilder) Markup(percent float64) *CostOptionsBuilder {
	b.opts.MarkupPercent = percent
	return b
}

func (b *CostOptionsBuilder) Build() *analyticsv1.CostOptions {
	return b.opts
}
