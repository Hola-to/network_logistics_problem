// services/report-svc/internal/generator/html.go
package generator

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

// HTMLGenerator генератор HTML отчётов
type HTMLGenerator struct {
	BaseGenerator
}

// NewHTMLGenerator создаёт новый генератор
func NewHTMLGenerator() *HTMLGenerator {
	return &HTMLGenerator{}
}

// Format возвращает формат генератора
func (g *HTMLGenerator) Format() reportv1.ReportFormat {
	return reportv1.ReportFormat_REPORT_FORMAT_HTML
}

// Generate генерирует HTML отчёт
func (g *HTMLGenerator) Generate(ctx context.Context, data *ReportData) ([]byte, error) {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"formatFloat":   func(v float64, p int) string { return fmt.Sprintf("%.*f", p, v) },
		"formatPercent": func(v float64) string { return fmt.Sprintf("%.1f%%", v*100) },
		"now":           func() string { return time.Now().Format("2006-01-02 15:04:05") },
		"gtZero":        func(v float64) bool { return v > 0 }, // Добавлен хелпер для сравнения с 0
	}).Parse(htmlTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Конвертируем edges для шаблона
	var flowEdges []*EdgeFlowData
	if data.FlowResult != nil && len(data.FlowResult.Edges) > 0 {
		flowEdges = ConvertFlowEdges(data.FlowResult.Edges)
	}
	if data.FlowEdges != nil {
		flowEdges = data.FlowEdges
	}

	templateData := map[string]any{
		"Title":          g.GetTitle(data),
		"Author":         g.GetAuthor(data),
		"Description":    g.GetDescription(data),
		"Type":           data.Type.String(),
		"Graph":          data.Graph,
		"FlowResult":     data.FlowResult,
		"FlowEdges":      flowEdges,
		"AnalyticsData":  data.AnalyticsData,
		"SimulationData": data.SimulationData,
		"ComparisonData": data.ComparisonData,
		"IncludeRawData": g.ShouldIncludeRawData(data),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        * { box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; border-bottom: 1px solid #ecf0f1; padding-bottom: 8px; margin-top: 30px; }
        h3 { color: #7f8c8d; }
        .meta { color: #7f8c8d; font-size: 0.9em; margin-bottom: 20px; }
        .metric-box {
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 15px 25px;
            border-radius: 8px;
            margin: 5px;
            min-width: 150px;
        }
        .metric-box .label { font-size: 0.85em; opacity: 0.9; }
        .metric-box .value { font-size: 1.8em; font-weight: bold; }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 15px 0;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ecf0f1;
        }
        th {
            background: #3498db;
            color: white;
            font-weight: 500;
        }
        tr:nth-child(even) { background: #f8f9fa; }
        tr:hover { background: #e8f4f8; }
        .status-optimal { color: #27ae60; font-weight: bold; }
        .status-warning { color: #f39c12; font-weight: bold; }
        .status-error { color: #e74c3c; font-weight: bold; }
        .severity-high { background: #ffe6e6; color: #c0392b; }
        .severity-medium { background: #fff3e0; color: #e67e22; }
        .severity-low { background: #e8f5e9; color: #27ae60; }
        .recommendation {
            background: #f0f7ff;
            border-left: 4px solid #3498db;
            padding: 15px;
            margin: 10px 0;
            border-radius: 0 8px 8px 0;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #ecf0f1;
            color: #7f8c8d;
            font-size: 0.85em;
            text-align: center;
        }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
        .card {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 8px;
            border: 1px solid #e9ecef;
        }
        .card-label { font-size: 0.85em; color: #6c757d; }
        .card-value { font-size: 1.2em; font-weight: 600; color: #495057; }
    </style>
</head>
<body>
<div class="container">
    <h1>{{.Title}}</h1>
    <div class="meta">
        <p><strong>Author:</strong> {{.Author}} | <strong>Generated:</strong> {{now}}</p>
        {{if .Description}}<p>{{.Description}}</p>{{end}}
    </div>

    {{if .FlowResult}}
    <h2>Optimization Results</h2>
    <div>
        <div class="metric-box">
            <div class="label">Maximum Flow</div>
            <div class="value">{{formatFloat .FlowResult.MaxFlow 4}}</div>
        </div>
        <div class="metric-box">
            <div class="label">Total Cost</div>
            <div class="value">{{formatFloat .FlowResult.TotalCost 2}}</div>
        </div>
    </div>

    <div class="grid" style="margin-top: 20px;">
        <div class="card">
            <div class="card-label">Status</div>
            <div class="card-value">{{.FlowResult.Status}}</div>
        </div>
        <div class="card">
            <div class="card-label">Iterations</div>
            <div class="card-value">{{.FlowResult.Iterations}}</div>
        </div>
        <div class="card">
            <div class="card-label">Computation Time</div>
            <div class="card-value">{{formatFloat .FlowResult.ComputationTimeMs 2}} ms</div>
        </div>
    </div>

    {{if and .FlowEdges .IncludeRawData}}
    <h3>Edge Flows</h3>
    <table>
        <thead>
            <tr><th>From</th><th>To</th><th>Flow</th><th>Capacity</th><th>Utilization</th></tr>
        </thead>
        <tbody>
        {{range .FlowEdges}}
            {{if gtZero .Flow}}
            <tr>
                <td>{{.From}}</td>
                <td>{{.To}}</td>
                <td>{{formatFloat .Flow 4}}</td>
                <td>{{formatFloat .Capacity 4}}</td>
                <td>{{formatPercent .Utilization}}</td>
            </tr>
            {{end}}
        {{end}}
        </tbody>
    </table>
    {{end}}
    {{end}}

    {{if .Graph}}
    <h2>Network Information</h2>
    <div class="grid">
        <div class="card">
            <div class="card-label">Nodes</div>
            <div class="card-value">{{len .Graph.Nodes}}</div>
        </div>
        <div class="card">
            <div class="card-label">Edges</div>
            <div class="card-value">{{len .Graph.Edges}}</div>
        </div>
        <div class="card">
            <div class="card-label">Source</div>
            <div class="card-value">{{.Graph.SourceId}}</div>
        </div>
        <div class="card">
            <div class="card-label">Sink</div>
            <div class="card-value">{{.Graph.SinkId}}</div>
        </div>
    </div>
    {{end}}

    {{if .AnalyticsData}}
    <h2>Analytics</h2>
    <div class="metric-box">
        <div class="label">Total Cost</div>
        <div class="value">{{formatFloat .AnalyticsData.TotalCost 2}} {{.AnalyticsData.Currency}}</div>
    </div>

    {{if .AnalyticsData.Bottlenecks}}
    <h3>Bottlenecks</h3>
    <table>
        <thead>
            <tr><th>From</th><th>To</th><th>Utilization</th><th>Impact</th><th>Severity</th></tr>
        </thead>
        <tbody>
        {{range .AnalyticsData.Bottlenecks}}
            <tr>
                <td>{{.From}}</td>
                <td>{{.To}}</td>
                <td>{{formatPercent .Utilization}}</td>
                <td>{{formatFloat .ImpactScore 2}}</td>
                <td>{{.Severity}}</td>
            </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .AnalyticsData.Recommendations}}
    <h3>Recommendations</h3>
    {{range .AnalyticsData.Recommendations}}
    <div class="recommendation">
        <strong>{{.Type}}</strong>
        <p>{{.Description}}</p>
        {{if gtZero .EstimatedImprovement}}<p>Expected improvement: {{formatPercent .EstimatedImprovement}}</p>{{end}}
    </div>
    {{end}}
    {{end}}

    {{if .AnalyticsData.Efficiency}}
    <h3>Efficiency</h3>
    <div class="grid">
        <div class="card">
            <div class="card-label">Overall Efficiency</div>
            <div class="card-value">{{formatPercent .AnalyticsData.Efficiency.OverallEfficiency}}</div>
        </div>
        <div class="card">
            <div class="card-label">Capacity Utilization</div>
            <div class="card-value">{{formatPercent .AnalyticsData.Efficiency.CapacityUtilization}}</div>
        </div>
        <div class="card">
            <div class="card-label">Grade</div>
            <div class="card-value">{{.AnalyticsData.Efficiency.Grade}}</div>
        </div>
    </div>
    {{end}}
    {{end}}

    {{if .SimulationData}}
    <h2>Simulation: {{.SimulationData.SimulationType}}</h2>
    <div class="grid">
        <div class="card">
            <div class="card-label">Baseline Flow</div>
            <div class="card-value">{{formatFloat .SimulationData.BaselineFlow 4}}</div>
        </div>
        <div class="card">
            <div class="card-label">Baseline Cost</div>
            <div class="card-value">{{formatFloat .SimulationData.BaselineCost 2}}</div>
        </div>
    </div>

    {{if .SimulationData.Scenarios}}
    <h3>Scenarios</h3>
    <table>
        <thead>
            <tr><th>Name</th><th>Max Flow</th><th>Cost</th><th>Change</th><th>Impact</th></tr>
        </thead>
        <tbody>
        {{range .SimulationData.Scenarios}}
            <tr>
                <td>{{.Name}}</td>
                <td>{{formatFloat .MaxFlow 4}}</td>
                <td>{{formatFloat .TotalCost 2}}</td>
                <td>{{formatFloat .FlowChangePercent 1}}%</td>
                <td>{{.ImpactLevel}}</td>
            </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .SimulationData.MonteCarlo}}
    <h3>Monte Carlo Results</h3>
    <div class="grid">
        <div class="card">
            <div class="card-label">Mean Flow</div>
            <div class="card-value">{{formatFloat .SimulationData.MonteCarlo.MeanFlow 4}}</div>
        </div>
        <div class="card">
            <div class="card-label">Std Dev</div>
            <div class="card-value">{{formatFloat .SimulationData.MonteCarlo.StdDev 4}}</div>
        </div>
        <div class="card">
            <div class="card-label">Range</div>
            <div class="card-value">{{formatFloat .SimulationData.MonteCarlo.MinFlow 2}} - {{formatFloat .SimulationData.MonteCarlo.MaxFlow 2}}</div>
        </div>
    </div>
    {{end}}

    {{if .SimulationData.Resilience}}
    <h3>Resilience Analysis</h3>
    <div class="grid">
        <div class="card">
            <div class="card-label">Overall Score</div>
            <div class="card-value">{{formatFloat .SimulationData.Resilience.OverallScore 2}}</div>
        </div>
        <div class="card">
            <div class="card-label">Single Points of Failure</div>
            <div class="card-value">{{.SimulationData.Resilience.SinglePointsOfFailure}}</div>
        </div>
        <div class="card">
            <div class="card-label">N-1 Feasible</div>
            <div class="card-value">{{.SimulationData.Resilience.NMinusOneFeasible}}</div>
        </div>
    </div>
    {{end}}
    {{end}}

    {{if .ComparisonData}}
    <h2>Scenario Comparison</h2>
    <table>
        <thead>
            <tr><th>Scenario</th><th>Max Flow</th><th>Total Cost</th><th>Efficiency</th></tr>
        </thead>
        <tbody>
        {{range .ComparisonData}}
            <tr>
                <td>{{.Name}}</td>
                <td>{{formatFloat .MaxFlow 4}}</td>
                <td>{{formatFloat .TotalCost 2}}</td>
                <td>{{formatPercent .Efficiency}}</td>
            </tr>
        {{end}}
        </tbody>
    </table>
    {{end}}

    <div class="footer">
        <p>Generated by Logistics Platform | {{now}}</p>
    </div>
</div>
</body>
</html>`
