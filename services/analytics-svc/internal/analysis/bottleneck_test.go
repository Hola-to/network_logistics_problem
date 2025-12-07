package analysis

import (
	"testing"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestFindBottlenecks(t *testing.T) {
	tests := []struct {
		name             string
		graph            *commonv1.Graph
		threshold        float64
		topN             int32
		expectedCount    int
		expectedSeverity analyticsv1.BottleneckSeverity
	}{
		{
			name: "finds saturated edge",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 100, CurrentFlow: 100, Cost: 1},
					{From: 2, To: 3, Capacity: 100, CurrentFlow: 50, Cost: 1},
				},
			},
			threshold:        0.9,
			topN:             10,
			expectedCount:    1,
			expectedSeverity: analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL,
		},
		{
			name: "no bottlenecks below threshold",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 100, CurrentFlow: 50, Cost: 1},
					{From: 2, To: 3, Capacity: 100, CurrentFlow: 40, Cost: 1},
				},
			},
			threshold:     0.9,
			topN:          10,
			expectedCount: 0,
		},
		{
			name: "skips virtual nodes",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: -1, To: 2, Capacity: 100, CurrentFlow: 100, Cost: 1},
					{From: 2, To: -2, Capacity: 100, CurrentFlow: 100, Cost: 1},
				},
			},
			threshold:     0.9,
			topN:          10,
			expectedCount: 0,
		},
		{
			name: "skips zero flow edges",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 100, CurrentFlow: 0, Cost: 1},
				},
			},
			threshold:     0.9,
			topN:          10,
			expectedCount: 0,
		},
		{
			name: "limits results with topN",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 100, CurrentFlow: 95, Cost: 1},
					{From: 2, To: 3, Capacity: 100, CurrentFlow: 96, Cost: 1},
					{From: 3, To: 4, Capacity: 100, CurrentFlow: 97, Cost: 1},
				},
			},
			threshold:     0.9,
			topN:          2,
			expectedCount: 2,
		},
		{
			name: "sorts by utilization descending",
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 100, CurrentFlow: 90, Cost: 1},
					{From: 2, To: 3, Capacity: 100, CurrentFlow: 100, Cost: 1},
					{From: 3, To: 4, Capacity: 100, CurrentFlow: 95, Cost: 1},
				},
			},
			threshold:     0.9,
			topN:          10,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindBottlenecks(tt.graph, tt.threshold, tt.topN)

			if len(result.Bottlenecks) != tt.expectedCount {
				t.Errorf("FindBottlenecks() returned %d bottlenecks, want %d",
					len(result.Bottlenecks), tt.expectedCount)
			}

			if tt.expectedCount > 0 && tt.expectedSeverity != 0 {
				if result.Bottlenecks[0].Severity != tt.expectedSeverity {
					t.Errorf("First bottleneck severity = %v, want %v",
						result.Bottlenecks[0].Severity, tt.expectedSeverity)
				}
			}

			// Verify sorted order
			for i := 1; i < len(result.Bottlenecks); i++ {
				if result.Bottlenecks[i].Utilization > result.Bottlenecks[i-1].Utilization {
					t.Errorf("Bottlenecks not sorted by utilization descending")
				}
			}
		})
	}
}

func TestCalculateSeverity(t *testing.T) {
	tests := []struct {
		name        string
		utilization float64
		expected    analyticsv1.BottleneckSeverity
	}{
		{
			name:        "critical at 100%",
			utilization: 1.0,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL,
		},
		{
			name:        "critical at 99.99%",
			utilization: 0.9999,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL,
		},
		{
			name:        "high at 95%",
			utilization: 0.95,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
		},
		{
			name:        "medium at 90%",
			utilization: 0.90,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_MEDIUM,
		},
		{
			name:        "low at 85%",
			utilization: 0.85,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_LOW,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSeverity(tt.utilization)
			if result != tt.expected {
				t.Errorf("calculateSeverity(%v) = %v, want %v",
					tt.utilization, result, tt.expected)
			}
		})
	}
}

func TestCalculateImpactScore(t *testing.T) {
	tests := []struct {
		name          string
		edge          *commonv1.Edge
		graph         *commonv1.Graph
		expectedScore float64
	}{
		{
			name: "half of total flow",
			edge: &commonv1.Edge{From: 1, To: 2, CurrentFlow: 50},
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 50},
					{From: 2, To: 3, CurrentFlow: 50},
				},
			},
			expectedScore: 0.5,
		},
		{
			name: "all flow through one edge",
			edge: &commonv1.Edge{From: 1, To: 2, CurrentFlow: 100},
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 100},
				},
			},
			expectedScore: 1.0,
		},
		{
			name: "zero total flow",
			edge: &commonv1.Edge{From: 1, To: 2, CurrentFlow: 0},
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, CurrentFlow: 0},
				},
			},
			expectedScore: 0.0,
		},
		{
			name: "ignores virtual nodes in total",
			edge: &commonv1.Edge{From: 1, To: 2, CurrentFlow: 50},
			graph: &commonv1.Graph{
				Edges: []*commonv1.Edge{
					{From: -1, To: 1, CurrentFlow: 100},
					{From: 1, To: 2, CurrentFlow: 50},
					{From: 2, To: 3, CurrentFlow: 50},
				},
			},
			expectedScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateImpactScore(tt.edge, tt.graph)
			if !floatEquals(result, tt.expectedScore, 0.0001) {
				t.Errorf("calculateImpactScore() = %v, want %v",
					result, tt.expectedScore)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name          string
		bottlenecks   []*analyticsv1.Bottleneck
		expectedCount int
	}{
		{
			name: "generates recommendations for high severity",
			bottlenecks: []*analyticsv1.Bottleneck{
				{
					Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
					Utilization: 0.95,
					Edge:        &commonv1.Edge{From: 1, To: 2, Capacity: 100},
				},
			},
			expectedCount: 1,
		},
		{
			name: "generates recommendations for critical severity",
			bottlenecks: []*analyticsv1.Bottleneck{
				{
					Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL,
					Utilization: 1.0,
					Edge:        &commonv1.Edge{From: 1, To: 2, Capacity: 100},
				},
			},
			expectedCount: 1,
		},
		{
			name: "no recommendations for low severity",
			bottlenecks: []*analyticsv1.Bottleneck{
				{
					Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_LOW,
					Utilization: 0.85,
					Edge:        &commonv1.Edge{From: 1, To: 2, Capacity: 100},
				},
			},
			expectedCount: 0,
		},
		{
			name:          "empty bottlenecks",
			bottlenecks:   []*analyticsv1.Bottleneck{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRecommendations(tt.bottlenecks)
			if len(result) != tt.expectedCount {
				t.Errorf("generateRecommendations() returned %d recommendations, want %d",
					len(result), tt.expectedCount)
			}
		})
	}
}

func TestCalculateSeverity_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		utilization float64
		expected    analyticsv1.BottleneckSeverity
	}{
		{
			name:        "exactly 99%",
			utilization: 0.99,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL,
		},
		{
			name:        "just below 99%",
			utilization: 0.9899,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
		},
		{
			name:        "exactly 95%",
			utilization: 0.95,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
		},
		{
			name:        "just below 95%",
			utilization: 0.9499,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_MEDIUM,
		},
		{
			name:        "exactly 90%",
			utilization: 0.90,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_MEDIUM,
		},
		{
			name:        "just below 90%",
			utilization: 0.8999,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_LOW,
		},
		{
			name:        "zero utilization",
			utilization: 0.0,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_LOW,
		},
		{
			name:        "over 100%",
			utilization: 1.5,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL,
		},
		{
			name:        "negative utilization",
			utilization: -0.1,
			expected:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_LOW,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSeverity(tt.utilization)
			if result != tt.expected {
				t.Errorf("calculateSeverity(%v) = %v, want %v",
					tt.utilization, result, tt.expected)
			}
		})
	}
}

func TestFindBottlenecks_EmptyGraph(t *testing.T) {
	graph := &commonv1.Graph{
		Edges: []*commonv1.Edge{},
	}

	result := FindBottlenecks(graph, 0.9, 10)

	if len(result.Bottlenecks) != 0 {
		t.Errorf("Expected no bottlenecks for empty graph, got %d", len(result.Bottlenecks))
	}
	if len(result.Recommendations) != 0 {
		t.Errorf("Expected no recommendations for empty graph, got %d", len(result.Recommendations))
	}
}

func TestFindBottlenecks_AllSaturated(t *testing.T) {
	graph := &commonv1.Graph{
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 100, CurrentFlow: 100, Cost: 1},
			{From: 2, To: 3, Capacity: 100, CurrentFlow: 100, Cost: 1},
			{From: 3, To: 4, Capacity: 100, CurrentFlow: 100, Cost: 1},
		},
	}

	result := FindBottlenecks(graph, 0.9, 0) // topN=0 — без ограничений

	if len(result.Bottlenecks) != 3 {
		t.Errorf("Expected 3 bottlenecks, got %d", len(result.Bottlenecks))
	}

	// Все должны быть CRITICAL
	for i, b := range result.Bottlenecks {
		if b.Severity != analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL {
			t.Errorf("Bottleneck %d: expected CRITICAL severity, got %v", i, b.Severity)
		}
	}
}

func TestGenerateRecommendations_MultipleSeverities(t *testing.T) {
	bottlenecks := []*analyticsv1.Bottleneck{
		{
			Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_CRITICAL,
			Utilization: 1.0,
			Edge:        &commonv1.Edge{From: 1, To: 2, Capacity: 100},
		},
		{
			Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_HIGH,
			Utilization: 0.96,
			Edge:        &commonv1.Edge{From: 2, To: 3, Capacity: 100},
		},
		{
			Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_MEDIUM,
			Utilization: 0.91,
			Edge:        &commonv1.Edge{From: 3, To: 4, Capacity: 100},
		},
		{
			Severity:    analyticsv1.BottleneckSeverity_BOTTLENECK_SEVERITY_LOW,
			Utilization: 0.85,
			Edge:        &commonv1.Edge{From: 4, To: 5, Capacity: 100},
		},
	}

	result := generateRecommendations(bottlenecks)

	// Рекомендации только для HIGH и CRITICAL
	if len(result) != 2 {
		t.Errorf("Expected 2 recommendations (CRITICAL + HIGH), got %d", len(result))
	}
}
