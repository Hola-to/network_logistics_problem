package domain

import (
	"testing"
)

func TestCalculateGraphStatistics(t *testing.T) {
	g := NewGraph()
	g.SourceID = 1
	g.SinkID = 4

	g.AddNode(&Node{ID: 1, Type: NodeTypeSource})
	g.AddNode(&Node{ID: 2, Type: NodeTypeWarehouse})
	g.AddNode(&Node{ID: 3, Type: NodeTypeDeliveryPoint})
	g.AddNode(&Node{ID: 4, Type: NodeTypeSink})

	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10, Length: 100})
	g.AddEdge(&Edge{From: 2, To: 3, Capacity: 5, Length: 50})
	g.AddEdge(&Edge{From: 3, To: 4, Capacity: 5, Length: 75})

	stats := CalculateGraphStatistics(g)

	if stats.NodeCount != 4 {
		t.Errorf("NodeCount = %d, want 4", stats.NodeCount)
	}
	if stats.EdgeCount != 3 {
		t.Errorf("EdgeCount = %d, want 3", stats.EdgeCount)
	}
	if stats.WarehouseCount != 1 {
		t.Errorf("WarehouseCount = %d, want 1", stats.WarehouseCount)
	}
	if stats.DeliveryPointCount != 1 {
		t.Errorf("DeliveryPointCount = %d, want 1", stats.DeliveryPointCount)
	}
	if !FloatEquals(stats.TotalCapacity, 20) {
		t.Errorf("TotalCapacity = %v, want 20", stats.TotalCapacity)
	}
	if !stats.IsConnected {
		t.Error("IsConnected should be true")
	}
}

func TestCalculateFlowStatistics(t *testing.T) {
	g := NewGraph()
	g.SourceID = 1
	g.SinkID = 3

	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})

	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10, CurrentFlow: 10}) // saturated
	g.AddEdge(&Edge{From: 2, To: 3, Capacity: 10, CurrentFlow: 5})  // 50% utilization
	g.AddEdge(&Edge{From: 1, To: 3, Capacity: 5, CurrentFlow: 0})   // zero flow

	stats := CalculateFlowStatistics(g)

	if !FloatEquals(stats.TotalFlow, 10) {
		t.Errorf("TotalFlow = %v, want 10", stats.TotalFlow)
	}
	if stats.SaturatedEdges != 1 {
		t.Errorf("SaturatedEdges = %d, want 1", stats.SaturatedEdges)
	}
	if stats.ZeroFlowEdges != 1 {
		t.Errorf("ZeroFlowEdges = %d, want 1", stats.ZeroFlowEdges)
	}
	if stats.ActiveEdges != 2 {
		t.Errorf("ActiveEdges = %d, want 2", stats.ActiveEdges)
	}
}

func TestCalculateEfficiency(t *testing.T) {
	tests := []struct {
		name          string
		utilization   float64
		expectedGrade EfficiencyGrade
	}{
		{"grade A", 0.85, GradeA},
		{"grade B", 0.65, GradeB},
		{"grade C", 0.45, GradeC},
		{"grade D", 0.25, GradeD},
		{"grade F", 0.1, GradeF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGraph()
			g.SourceID = 1
			g.SinkID = 2

			g.AddNode(&Node{ID: 1})
			g.AddNode(&Node{ID: 2})
			g.AddEdge(&Edge{From: 1, To: 2, Capacity: 100, CurrentFlow: tt.utilization * 100})

			report := CalculateEfficiency(g)

			if report.Grade != tt.expectedGrade {
				t.Errorf("Grade = %v, want %v", report.Grade, tt.expectedGrade)
			}
		})
	}
}

func TestFindBottlenecks(t *testing.T) {
	g := NewGraph()
	g.SourceID = 1
	g.SinkID = 4

	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: 2})
	g.AddNode(&Node{ID: 3})
	g.AddNode(&Node{ID: 4})

	g.AddEdge(&Edge{From: 1, To: 2, Capacity: 10, CurrentFlow: 10})  // 100% - critical
	g.AddEdge(&Edge{From: 2, To: 3, Capacity: 10, CurrentFlow: 9.6}) // 96% - high
	g.AddEdge(&Edge{From: 3, To: 4, Capacity: 10, CurrentFlow: 9.1}) // 91% - medium
	g.AddEdge(&Edge{From: 1, To: 4, Capacity: 10, CurrentFlow: 5})   // 50% - not a bottleneck

	bottlenecks := FindBottlenecks(g, DefaultBottleneckThreshold)

	if len(bottlenecks) != 3 {
		t.Errorf("bottlenecks count = %d, want 3", len(bottlenecks))
	}

	// Check severities
	severityCounts := make(map[BottleneckSeverity]int)
	for _, b := range bottlenecks {
		severityCounts[b.Severity]++
	}

	if severityCounts[SeverityCritical] != 1 {
		t.Errorf("critical bottlenecks = %d, want 1", severityCounts[SeverityCritical])
	}
}

func TestBottleneckSeverity_String(t *testing.T) {
	tests := []struct {
		severity BottleneckSeverity
		expected string
	}{
		{SeverityLow, "low"},
		{SeverityMedium, "medium"},
		{SeverityHigh, "high"},
		{SeverityCritical, "critical"},
		{BottleneckSeverity(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.severity.String(); got != tt.expected {
			t.Errorf("Severity.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestFindBottlenecks_VirtualNodesExcluded(t *testing.T) {
	g := NewGraph()
	g.SourceID = SuperSourceID
	g.SinkID = SuperSinkID

	g.AddNode(&Node{ID: SuperSourceID})
	g.AddNode(&Node{ID: 1})
	g.AddNode(&Node{ID: SuperSinkID})

	// Virtual node edges should be excluded
	g.AddEdge(&Edge{From: SuperSourceID, To: 1, Capacity: 10, CurrentFlow: 10})
	g.AddEdge(&Edge{From: 1, To: SuperSinkID, Capacity: 10, CurrentFlow: 10})

	bottlenecks := FindBottlenecks(g, DefaultBottleneckThreshold)

	if len(bottlenecks) != 0 {
		t.Errorf("bottlenecks with virtual nodes = %d, want 0", len(bottlenecks))
	}
}
