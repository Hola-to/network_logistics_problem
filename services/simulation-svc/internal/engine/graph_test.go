package engine

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneGraph(t *testing.T) {
	tests := []struct {
		name  string
		graph *commonv1.Graph
	}{
		{
			name:  "nil graph",
			graph: nil,
		},
		{
			name: "empty graph",
			graph: &commonv1.Graph{
				SourceId: 1,
				SinkId:   2,
				Name:     "empty",
				Metadata: map[string]string{},
				Nodes:    []*commonv1.Node{},
				Edges:    []*commonv1.Edge{},
			},
		},
		{
			name:  "simple graph",
			graph: createTestGraph(),
		},
		{
			name: "graph with metadata",
			graph: &commonv1.Graph{
				SourceId: 1,
				SinkId:   4,
				Name:     "test",
				Metadata: map[string]string{"key1": "value1", "key2": "value2"},
				Nodes: []*commonv1.Node{
					{Id: 1, X: 0, Y: 0, Type: commonv1.NodeType_NODE_TYPE_SOURCE, Metadata: map[string]string{"node_key": "node_value"}},
					{Id: 2, X: 1, Y: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
				},
				Edges: []*commonv1.Edge{
					{From: 1, To: 2, Capacity: 10, Cost: 5},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := CloneGraph(tt.graph)

			if tt.graph == nil {
				assert.Nil(t, clone)
				return
			}

			require.NotNil(t, clone)

			// Проверяем, что это копия, а не тот же объект
			assert.NotSame(t, tt.graph, clone)

			// Проверяем базовые поля
			assert.Equal(t, tt.graph.SourceId, clone.SourceId)
			assert.Equal(t, tt.graph.SinkId, clone.SinkId)
			assert.Equal(t, tt.graph.Name, clone.Name)

			// Проверяем metadata
			assert.Equal(t, len(tt.graph.Metadata), len(clone.Metadata))
			for k, v := range tt.graph.Metadata {
				assert.Equal(t, v, clone.Metadata[k])
			}

			// Проверяем nodes
			assert.Equal(t, len(tt.graph.Nodes), len(clone.Nodes))
			for i, node := range tt.graph.Nodes {
				assert.NotSame(t, node, clone.Nodes[i])
				assert.Equal(t, node.Id, clone.Nodes[i].Id)
				assert.Equal(t, node.X, clone.Nodes[i].X)
				assert.Equal(t, node.Y, clone.Nodes[i].Y)
				assert.Equal(t, node.Type, clone.Nodes[i].Type)
			}

			// Проверяем edges
			assert.Equal(t, len(tt.graph.Edges), len(clone.Edges))
			for i, edge := range tt.graph.Edges {
				assert.NotSame(t, edge, clone.Edges[i])
				assert.Equal(t, edge.From, clone.Edges[i].From)
				assert.Equal(t, edge.To, clone.Edges[i].To)
				assert.Equal(t, edge.Capacity, clone.Edges[i].Capacity)
				assert.Equal(t, edge.Cost, clone.Edges[i].Cost)
			}
		})
	}
}

func TestCloneGraph_DeepCopy(t *testing.T) {
	original := createTestGraph()
	clone := CloneGraph(original)

	// Модифицируем клон
	clone.Edges[0].Capacity = 999
	clone.Nodes[0].X = 999
	clone.Metadata["new_key"] = "new_value"

	// Проверяем, что оригинал не изменился
	assert.NotEqual(t, float64(999), original.Edges[0].Capacity)
	assert.NotEqual(t, float64(999), original.Nodes[0].X)
	_, exists := original.Metadata["new_key"]
	assert.False(t, exists)
}

func TestApplyModifications(t *testing.T) {
	tests := []struct {
		name          string
		graph         *commonv1.Graph
		modifications []*simulationv1.Modification
		validate      func(t *testing.T, result *commonv1.Graph)
	}{
		{
			name:          "empty modifications",
			graph:         createTestGraph(),
			modifications: []*simulationv1.Modification{},
			validate: func(t *testing.T, result *commonv1.Graph) {
				assert.Equal(t, 4, len(result.Nodes))
				assert.Equal(t, 4, len(result.Edges))
			},
		},
		{
			name:  "update edge capacity absolute",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
					EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
					Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
					Change:  &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 50},
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				for _, e := range result.Edges {
					if e.From == 1 && e.To == 2 {
						assert.Equal(t, 50.0, e.Capacity)
						return
					}
				}
				t.Error("edge 1->2 not found")
			},
		},
		{
			name:  "update edge capacity relative",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
					EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
					Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
					Change:  &simulationv1.Modification_RelativeChange{RelativeChange: 1.5}, // +50%
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				for _, e := range result.Edges {
					if e.From == 1 && e.To == 2 {
						assert.Equal(t, 15.0, e.Capacity) // 10 * 1.5
						return
					}
				}
				t.Error("edge 1->2 not found")
			},
		},
		{
			name:  "update edge capacity delta",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
					EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
					Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
					Change:  &simulationv1.Modification_Delta{Delta: 5},
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				for _, e := range result.Edges {
					if e.From == 1 && e.To == 2 {
						assert.Equal(t, 15.0, e.Capacity) // 10 + 5
						return
					}
				}
				t.Error("edge 1->2 not found")
			},
		},
		{
			name:  "update edge cost",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
					EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
					Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_COST,
					Change:  &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 100},
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				for _, e := range result.Edges {
					if e.From == 1 && e.To == 2 {
						assert.Equal(t, 100.0, e.Cost)
						return
					}
				}
				t.Error("edge 1->2 not found")
			},
		},
		{
			name:  "remove edge",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_REMOVE_EDGE,
					EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				for _, e := range result.Edges {
					if e.From == 1 && e.To == 2 {
						t.Error("edge 1->2 should be removed")
						return
					}
				}
				assert.Equal(t, 3, len(result.Edges))
			},
		},
		{
			name:  "add edge",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_ADD_EDGE,
					EdgeKey: &commonv1.EdgeKey{From: 1, To: 4},
					Change:  &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 25},
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				assert.Equal(t, 5, len(result.Edges))
				found := false
				for _, e := range result.Edges {
					if e.From == 1 && e.To == 4 {
						found = true
						assert.Equal(t, 25.0, e.Capacity)
					}
				}
				assert.True(t, found, "new edge 1->4 should exist")
			},
		},
		{
			name:  "remove node",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:   simulationv1.ModificationType_MODIFICATION_TYPE_REMOVE_NODE,
					NodeId: 2,
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				assert.Equal(t, 3, len(result.Nodes))
				// Проверяем, что связанные рёбра удалены
				for _, e := range result.Edges {
					assert.NotEqual(t, int64(2), e.From)
					assert.NotEqual(t, int64(2), e.To)
				}
			},
		},
		{
			name:  "update node supply",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:   simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_NODE,
					NodeId: 1,
					Target: simulationv1.ModificationTarget_MODIFICATION_TARGET_SUPPLY,
					Change: &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 100},
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				for _, n := range result.Nodes {
					if n.Id == 1 {
						assert.Equal(t, 100.0, n.Supply)
						return
					}
				}
				t.Error("node 1 not found")
			},
		},
		{
			name:  "disable node",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:   simulationv1.ModificationType_MODIFICATION_TYPE_DISABLE_NODE,
					NodeId: 2,
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				for _, e := range result.Edges {
					if e.From == 2 || e.To == 2 {
						assert.Equal(t, 0.0, e.Capacity)
					}
				}
			},
		},
		{
			name:  "multiple modifications",
			graph: createTestGraph(),
			modifications: []*simulationv1.Modification{
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
					EdgeKey: &commonv1.EdgeKey{From: 1, To: 2},
					Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
					Change:  &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 50},
				},
				{
					Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
					EdgeKey: &commonv1.EdgeKey{From: 2, To: 4},
					Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
					Change:  &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 60},
				},
			},
			validate: func(t *testing.T, result *commonv1.Graph) {
				for _, e := range result.Edges {
					if e.From == 1 && e.To == 2 {
						assert.Equal(t, 50.0, e.Capacity)
					}
					if e.From == 2 && e.To == 4 {
						assert.Equal(t, 60.0, e.Capacity)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyModifications(tt.graph, tt.modifications)
			require.NotNil(t, result)
			// Проверяем, что оригинал не изменён
			assert.NotSame(t, tt.graph, result)
			tt.validate(t, result)
		})
	}
}

func TestApplyModifications_NonexistentEdge(t *testing.T) {
	graph := createTestGraph()
	mods := []*simulationv1.Modification{
		{
			Type:    simulationv1.ModificationType_MODIFICATION_TYPE_UPDATE_EDGE,
			EdgeKey: &commonv1.EdgeKey{From: 99, To: 100},
			Target:  simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY,
			Change:  &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 50},
		},
	}

	result := ApplyModifications(graph, mods)
	// Не должно быть паники, граф не изменён
	assert.Equal(t, len(graph.Edges), len(result.Edges))
}

func TestResetFlow(t *testing.T) {
	graph := createTestGraph()
	// Устанавливаем flow
	for _, e := range graph.Edges {
		e.CurrentFlow = 5
	}

	ResetFlow(graph)

	for _, e := range graph.Edges {
		assert.Equal(t, 0.0, e.CurrentFlow)
	}
}

func TestResetFlow_NilGraph(t *testing.T) {
	// Не должно паниковать
	ResetFlow(nil)
}

func TestEdgeKey(t *testing.T) {
	tests := []struct {
		from, to int64
		expected string
	}{
		{1, 2, "1->2"},
		{0, 0, "0->0"},
		{100, 200, "100->200"},
	}

	for _, tt := range tests {
		result := edgeKey(tt.from, tt.to)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetTargetValue(t *testing.T) {
	edge := &commonv1.Edge{
		From:     1,
		To:       2,
		Capacity: 100,
		Cost:     50,
		Length:   25,
	}

	tests := []struct {
		target   simulationv1.ModificationTarget
		expected float64
	}{
		{simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY, 100},
		{simulationv1.ModificationTarget_MODIFICATION_TARGET_COST, 50},
		{simulationv1.ModificationTarget_MODIFICATION_TARGET_LENGTH, 25},
		{simulationv1.ModificationTarget_MODIFICATION_TARGET_UNSPECIFIED, 100}, // default
	}

	for _, tt := range tests {
		result := getTargetValue(edge, tt.target)
		assert.Equal(t, tt.expected, result)
	}
}

func TestSetTargetValue(t *testing.T) {
	edge := &commonv1.Edge{
		From:     1,
		To:       2,
		Capacity: 100,
		Cost:     50,
		Length:   25,
	}

	setTargetValue(edge, simulationv1.ModificationTarget_MODIFICATION_TARGET_CAPACITY, 200)
	assert.Equal(t, 200.0, edge.Capacity)

	setTargetValue(edge, simulationv1.ModificationTarget_MODIFICATION_TARGET_COST, 75)
	assert.Equal(t, 75.0, edge.Cost)

	setTargetValue(edge, simulationv1.ModificationTarget_MODIFICATION_TARGET_LENGTH, 30)
	assert.Equal(t, 30.0, edge.Length)
}

func TestCalculateNewValue(t *testing.T) {
	tests := []struct {
		name     string
		current  float64
		mod      *simulationv1.Modification
		expected float64
	}{
		{
			name:    "absolute value",
			current: 100,
			mod: &simulationv1.Modification{
				Change: &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 50},
			},
			expected: 50,
		},
		{
			name:    "relative change",
			current: 100,
			mod: &simulationv1.Modification{
				Change: &simulationv1.Modification_RelativeChange{RelativeChange: 1.5},
			},
			expected: 150,
		},
		{
			name:    "delta positive",
			current: 100,
			mod: &simulationv1.Modification{
				Change: &simulationv1.Modification_Delta{Delta: 25},
			},
			expected: 125,
		},
		{
			name:    "delta negative",
			current: 100,
			mod: &simulationv1.Modification{
				Change: &simulationv1.Modification_Delta{Delta: -25},
			},
			expected: 75,
		},
		{
			name:     "no change",
			current:  100,
			mod:      &simulationv1.Modification{},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateNewValue(tt.current, tt.mod)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetModValue(t *testing.T) {
	tests := []struct {
		name         string
		mod          *simulationv1.Modification
		defaultValue float64
		expected     float64
	}{
		{
			name: "absolute value",
			mod: &simulationv1.Modification{
				Change: &simulationv1.Modification_AbsoluteValue{AbsoluteValue: 50},
			},
			defaultValue: 0,
			expected:     50,
		},
		{
			name:         "no change returns default",
			mod:          &simulationv1.Modification{},
			defaultValue: 100,
			expected:     100,
		},
		{
			name: "relative change returns default",
			mod: &simulationv1.Modification{
				Change: &simulationv1.Modification_RelativeChange{RelativeChange: 1.5},
			},
			defaultValue: 100,
			expected:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModValue(tt.mod, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

func createTestGraph() *commonv1.Graph {
	return &commonv1.Graph{
		SourceId: 1,
		SinkId:   4,
		Name:     "test-graph",
		Metadata: map[string]string{"env": "test"},
		Nodes: []*commonv1.Node{
			{Id: 1, X: 0, Y: 0, Type: commonv1.NodeType_NODE_TYPE_SOURCE, Metadata: map[string]string{}},
			{Id: 2, X: 1, Y: 0, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Metadata: map[string]string{}},
			{Id: 3, X: 1, Y: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE, Metadata: map[string]string{}},
			{Id: 4, X: 2, Y: 0, Type: commonv1.NodeType_NODE_TYPE_SINK, Metadata: map[string]string{}},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1},
			{From: 1, To: 3, Capacity: 15, Cost: 2},
			{From: 2, To: 4, Capacity: 10, Cost: 1},
			{From: 3, To: 4, Capacity: 15, Cost: 2},
		},
	}
}
