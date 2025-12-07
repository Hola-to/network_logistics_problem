// services/report-svc/internal/generator/json.go
package generator

import (
	"context"
	"encoding/json"
	"time"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

// JSONGenerator генератор JSON отчётов
type JSONGenerator struct {
	BaseGenerator
}

// NewJSONGenerator создаёт новый генератор
func NewJSONGenerator() *JSONGenerator {
	return &JSONGenerator{}
}

// Format возвращает формат генератора
func (g *JSONGenerator) Format() reportv1.ReportFormat {
	return reportv1.ReportFormat_REPORT_FORMAT_JSON
}

// JSONReport структура JSON отчёта
type JSONReport struct {
	Metadata   JSONMetadata      `json:"metadata"`
	Graph      *JSONGraph        `json:"graph,omitempty"`
	FlowResult *JSONFlowResult   `json:"flowResult,omitempty"`
	Analytics  *JSONAnalytics    `json:"analytics,omitempty"`
	Simulation *JSONSimulation   `json:"simulation,omitempty"`
	Comparison []*JSONComparison `json:"comparison,omitempty"`
}

type JSONMetadata struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description,omitempty"`
	GeneratedAt string `json:"generatedAt"`
	ReportType  string `json:"reportType"`
	Version     string `json:"version"`
}

type JSONGraph struct {
	NodeCount int64 `json:"nodeCount"`
	EdgeCount int64 `json:"edgeCount"`
	SourceID  int64 `json:"sourceId"`
	SinkID    int64 `json:"sinkId"`
}

type JSONFlowResult struct {
	MaxFlow           float64         `json:"maxFlow"`
	TotalCost         float64         `json:"totalCost"`
	Status            string          `json:"status"`
	Iterations        int32           `json:"iterations"`
	ComputationTimeMs float64         `json:"computationTimeMs"`
	Edges             []*JSONFlowEdge `json:"edges,omitempty"`
}

type JSONFlowEdge struct {
	From        int64   `json:"from"`
	To          int64   `json:"to"`
	Flow        float64 `json:"flow"`
	Capacity    float64 `json:"capacity"`
	Cost        float64 `json:"cost"`
	Utilization float64 `json:"utilization"`
}

type JSONAnalytics struct {
	TotalCost       float64               `json:"totalCost"`
	Currency        string                `json:"currency"`
	CostBreakdown   *JSONCostBreakdown    `json:"costBreakdown,omitempty"`
	Bottlenecks     []*JSONBottleneck     `json:"bottlenecks,omitempty"`
	Recommendations []*JSONRecommendation `json:"recommendations,omitempty"`
	Efficiency      *JSONEfficiency       `json:"efficiency,omitempty"`
}

type JSONCostBreakdown struct {
	TransportCost  float64            `json:"transportCost"`
	FixedCost      float64            `json:"fixedCost"`
	HandlingCost   float64            `json:"handlingCost"`
	CostByRoadType map[string]float64 `json:"costByRoadType,omitempty"`
	CostByNodeType map[string]float64 `json:"costByNodeType,omitempty"`
}

type JSONBottleneck struct {
	From        int64   `json:"from"`
	To          int64   `json:"to"`
	Utilization float64 `json:"utilization"`
	ImpactScore float64 `json:"impactScore"`
	Severity    string  `json:"severity"`
}

type JSONRecommendation struct {
	Type                 string  `json:"type"`
	Description          string  `json:"description"`
	EstimatedImprovement float64 `json:"estimatedImprovement"`
	EstimatedCost        float64 `json:"estimatedCost"`
}

type JSONEfficiency struct {
	OverallEfficiency   float64 `json:"overallEfficiency"`
	CapacityUtilization float64 `json:"capacityUtilization"`
	UnusedEdges         int32   `json:"unusedEdges"`
	SaturatedEdges      int32   `json:"saturatedEdges"`
	Grade               string  `json:"grade"`
}

type JSONSimulation struct {
	Type         string           `json:"type"`
	BaselineFlow float64          `json:"baselineFlow"`
	BaselineCost float64          `json:"baselineCost"`
	Scenarios    []*JSONScenario  `json:"scenarios,omitempty"`
	MonteCarlo   *JSONMonteCarlo  `json:"monteCarlo,omitempty"`
	Sensitivity  []*JSONSensParam `json:"sensitivity,omitempty"`
	Resilience   *JSONResilience  `json:"resilience,omitempty"`
}

type JSONScenario struct {
	Name              string  `json:"name"`
	MaxFlow           float64 `json:"maxFlow"`
	TotalCost         float64 `json:"totalCost"`
	FlowChangePercent float64 `json:"flowChangePercent"`
	ImpactLevel       string  `json:"impactLevel"`
}

type JSONMonteCarlo struct {
	Iterations      int32   `json:"iterations"`
	MeanFlow        float64 `json:"meanFlow"`
	StdDev          float64 `json:"stdDev"`
	MinFlow         float64 `json:"minFlow"`
	MaxFlow         float64 `json:"maxFlow"`
	P5              float64 `json:"p5"`
	P50             float64 `json:"p50"`
	P95             float64 `json:"p95"`
	ConfidenceLevel float64 `json:"confidenceLevel"`
	CiLow           float64 `json:"ciLow"`
	CiHigh          float64 `json:"ciHigh"`
}

type JSONSensParam struct {
	ParameterId      string  `json:"parameterId"`
	Elasticity       float64 `json:"elasticity"`
	SensitivityIndex float64 `json:"sensitivityIndex"`
	Level            string  `json:"level"`
}

type JSONResilience struct {
	OverallScore           float64 `json:"overallScore"`
	SinglePointsOfFailure  int32   `json:"singlePointsOfFailure"`
	WorstCaseFlowReduction float64 `json:"worstCaseFlowReduction"`
	NMinusOneFeasible      bool    `json:"nMinusOneFeasible"`
}

type JSONComparison struct {
	Name       string             `json:"name"`
	MaxFlow    float64            `json:"maxFlow"`
	TotalCost  float64            `json:"totalCost"`
	Efficiency float64            `json:"efficiency"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

// Generate генерирует JSON отчёт
func (g *JSONGenerator) Generate(ctx context.Context, data *ReportData) ([]byte, error) {
	report := JSONReport{
		Metadata: JSONMetadata{
			Title:       g.GetTitle(data),
			Author:      g.GetAuthor(data),
			Description: g.GetDescription(data),
			GeneratedAt: time.Now().Format(time.RFC3339),
			ReportType:  data.Type.String(),
			Version:     "1.0",
		},
	}

	// Graph
	if data.Graph != nil {
		report.Graph = &JSONGraph{
			NodeCount: int64(len(data.Graph.Nodes)),
			EdgeCount: int64(len(data.Graph.Edges)),
			SourceID:  data.Graph.SourceId,
			SinkID:    data.Graph.SinkId,
		}
	}

	// Flow Result
	if data.FlowResult != nil {
		fr := &JSONFlowResult{
			MaxFlow:           data.FlowResult.MaxFlow,
			TotalCost:         data.FlowResult.TotalCost,
			Status:            data.FlowResult.Status.String(),
			Iterations:        data.FlowResult.Iterations,
			ComputationTimeMs: data.FlowResult.ComputationTimeMs,
		}

		if g.ShouldIncludeRawData(data) {
			// Используем FlowEdges если есть
			edges := data.FlowEdges
			if edges == nil {
				edges = ConvertFlowEdges(data.FlowResult.Edges)
			}
			for _, e := range edges {
				if e.Flow > 0.001 {
					fr.Edges = append(fr.Edges, &JSONFlowEdge{
						From:        e.From,
						To:          e.To,
						Flow:        e.Flow,
						Capacity:    e.Capacity,
						Cost:        e.Cost,
						Utilization: e.Utilization,
					})
				}
			}
		}
		report.FlowResult = fr
	}

	// Analytics
	if data.AnalyticsData != nil {
		ad := data.AnalyticsData
		analytics := &JSONAnalytics{
			TotalCost: ad.TotalCost,
			Currency:  ad.Currency,
		}

		if ad.CostBreakdown != nil {
			analytics.CostBreakdown = &JSONCostBreakdown{
				TransportCost:  ad.CostBreakdown.TransportCost,
				FixedCost:      ad.CostBreakdown.FixedCost,
				HandlingCost:   ad.CostBreakdown.HandlingCost,
				CostByRoadType: ad.CostBreakdown.CostByRoadType,
				CostByNodeType: ad.CostBreakdown.CostByNodeType,
			}
		}

		for _, bn := range ad.Bottlenecks {
			analytics.Bottlenecks = append(analytics.Bottlenecks, &JSONBottleneck{
				From:        bn.From,
				To:          bn.To,
				Utilization: bn.Utilization,
				ImpactScore: bn.ImpactScore,
				Severity:    bn.Severity,
			})
		}

		if g.ShouldIncludeRecommendations(data) {
			for _, rec := range ad.Recommendations {
				analytics.Recommendations = append(analytics.Recommendations, &JSONRecommendation{
					Type:                 rec.Type,
					Description:          rec.Description,
					EstimatedImprovement: rec.EstimatedImprovement,
					EstimatedCost:        rec.EstimatedCost,
				})
			}
		}

		if ad.Efficiency != nil {
			analytics.Efficiency = &JSONEfficiency{
				OverallEfficiency:   ad.Efficiency.OverallEfficiency,
				CapacityUtilization: ad.Efficiency.CapacityUtilization,
				UnusedEdges:         ad.Efficiency.UnusedEdges,
				SaturatedEdges:      ad.Efficiency.SaturatedEdges,
				Grade:               ad.Efficiency.Grade,
			}
		}
		report.Analytics = analytics
	}

	// Simulation
	if data.SimulationData != nil {
		sd := data.SimulationData
		sim := &JSONSimulation{
			Type:         sd.SimulationType,
			BaselineFlow: sd.BaselineFlow,
			BaselineCost: sd.BaselineCost,
		}

		for _, sc := range sd.Scenarios {
			sim.Scenarios = append(sim.Scenarios, &JSONScenario{
				Name:              sc.Name,
				MaxFlow:           sc.MaxFlow,
				TotalCost:         sc.TotalCost,
				FlowChangePercent: sc.FlowChangePercent,
				ImpactLevel:       sc.ImpactLevel,
			})
		}

		if sd.MonteCarlo != nil {
			mc := sd.MonteCarlo
			sim.MonteCarlo = &JSONMonteCarlo{
				Iterations:      mc.Iterations,
				MeanFlow:        mc.MeanFlow,
				StdDev:          mc.StdDev,
				MinFlow:         mc.MinFlow,
				MaxFlow:         mc.MaxFlow,
				P5:              mc.P5,
				P50:             mc.P50,
				P95:             mc.P95,
				ConfidenceLevel: mc.ConfidenceLevel,
				CiLow:           mc.CiLow,
				CiHigh:          mc.CiHigh,
			}
		}

		for _, sp := range sd.Sensitivity {
			sim.Sensitivity = append(sim.Sensitivity, &JSONSensParam{
				ParameterId:      sp.ParameterId,
				Elasticity:       sp.Elasticity,
				SensitivityIndex: sp.SensitivityIndex,
				Level:            sp.Level,
			})
		}

		if sd.Resilience != nil {
			r := sd.Resilience
			sim.Resilience = &JSONResilience{
				OverallScore:           r.OverallScore,
				SinglePointsOfFailure:  r.SinglePointsOfFailure,
				WorstCaseFlowReduction: r.WorstCaseFlowReduction,
				NMinusOneFeasible:      r.NMinusOneFeasible,
			}
		}
		report.Simulation = sim
	}

	// Comparison
	for _, item := range data.ComparisonData {
		report.Comparison = append(report.Comparison, &JSONComparison{
			Name:       item.Name,
			MaxFlow:    item.MaxFlow,
			TotalCost:  item.TotalCost,
			Efficiency: item.Efficiency,
			Metrics:    item.Metrics,
		})
	}

	return json.MarshalIndent(report, "", "  ")
}
