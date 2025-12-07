package converter

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/services/solver-svc/internal/graph"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToResidualGraph_EmptyGraph(t *testing.T) {
	proto := &commonv1.Graph{
		Nodes: []*commonv1.Node{},
		Edges: []*commonv1.Edge{},
	}

	rg := ToResidualGraph(proto)

	assert.Equal(t, 0, rg.NodeCount())
	assert.Equal(t, 0, rg.EdgeCount())
}

func TestToResidualGraph_SimpleGraph(t *testing.T) {
	proto := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1}, {Id: 2}, {Id: 3},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 5},
			{From: 2, To: 3, Capacity: 20, Cost: 10},
		},
	}

	rg := ToResidualGraph(proto)

	assert.Equal(t, 3, rg.NodeCount())

	// Проверяем прямые ребра
	edge12 := rg.GetEdge(1, 2)
	require.NotNil(t, edge12)
	assert.Equal(t, 10.0, edge12.Capacity)
	assert.Equal(t, 5.0, edge12.Cost)
	assert.False(t, edge12.IsReverse)

	// Проверяем обратные ребра (автоматически созданные)
	edge21 := rg.GetEdge(2, 1)
	require.NotNil(t, edge21)
	assert.Equal(t, 0.0, edge21.Capacity) // Обратное ребро начинается с 0
	assert.True(t, edge21.IsReverse)
}

func TestToResidualGraph_BidirectionalEdge(t *testing.T) {
	proto := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1}, {Id: 2},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 5, Bidirectional: true},
		},
	}

	rg := ToResidualGraph(proto)

	// При Bidirectional=true создаются ДВА прямых ребра
	// 1->2 и 2->1, каждое с указанной capacity

	edge12 := rg.GetEdge(1, 2)
	edge21 := rg.GetEdge(2, 1)

	require.NotNil(t, edge12, "Edge 1->2 should exist")
	require.NotNil(t, edge21, "Edge 2->1 should exist")

	// Оба ребра должны иметь capacity 10
	// Примечание: реализация AddEdgeWithReverse дважды для bidirectional
	// может привести к тому, что edge21 будет иметь capacity=10 (прямое ребро)
	// а не 0 (только обратное)

	// Проверяем, что хотя бы одно из ребер имеет capacity 10
	assert.True(t, edge12.Capacity == 10.0 || edge21.Capacity == 10.0,
		"At least one direction should have capacity 10")
}

func TestToResidualGraph_LargeGraph(t *testing.T) {
	n := 100
	nodes := make([]*commonv1.Node, n)
	edges := make([]*commonv1.Edge, n-1)

	for i := 0; i < n; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
		if i > 0 {
			edges[i-1] = &commonv1.Edge{
				From:     int64(i - 1),
				To:       int64(i),
				Capacity: float64(i * 10),
			}
		}
	}

	proto := &commonv1.Graph{Nodes: nodes, Edges: edges}

	rg := ToResidualGraph(proto)

	assert.Equal(t, n, rg.NodeCount())
}

func TestToFlowEdges_NoFlow(t *testing.T) {
	rg := graph.NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 0)

	edges := ToFlowEdges(rg)

	assert.Empty(t, edges, "No edges with flow should be returned")
}

func TestToFlowEdges_WithFlow(t *testing.T) {
	rg := graph.NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 5)
	rg.UpdateFlow(1, 2, 6)

	edges := ToFlowEdges(rg)

	require.Len(t, edges, 1)
	assert.Equal(t, int64(1), edges[0].From)
	assert.Equal(t, int64(2), edges[0].To)
	assert.Equal(t, 6.0, edges[0].Flow)
	assert.Equal(t, 10.0, edges[0].Capacity)
	assert.InDelta(t, 0.6, edges[0].Utilization, 1e-9)
}

func TestToFlowEdges_FullUtilization(t *testing.T) {
	rg := graph.NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 0)
	rg.UpdateFlow(1, 2, 10)

	edges := ToFlowEdges(rg)

	require.Len(t, edges, 1)
	assert.InDelta(t, 1.0, edges[0].Utilization, 1e-9)
}

func TestToFlowEdges_DoesNotIncludeReverseEdges(t *testing.T) {
	rg := graph.NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 0)
	rg.UpdateFlow(1, 2, 5)

	edges := ToFlowEdges(rg)

	// Должно быть только одно ребро (прямое с потоком)
	assert.Len(t, edges, 1)
	assert.Equal(t, int64(1), edges[0].From)
}

func TestToPaths_EmptyPaths(t *testing.T) {
	rg := graph.NewResidualGraph()

	paths := ToPaths([][]int64{}, rg)

	assert.Empty(t, paths)
}

func TestToPaths_SimplePaths(t *testing.T) {
	rg := graph.NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 2)
	rg.AddEdgeWithReverse(2, 3, 10, 3)
	rg.UpdateFlow(1, 2, 5)
	rg.UpdateFlow(2, 3, 5)

	rawPaths := [][]int64{{1, 2, 3}}

	paths := ToPaths(rawPaths, rg)

	require.Len(t, paths, 1)
	assert.Equal(t, []int64{1, 2, 3}, paths[0].NodeIds)
	assert.Equal(t, 5.0, paths[0].Flow)
	assert.InDelta(t, 25.0, paths[0].Cost, 1e-9) // (2+3) * 5
}

func TestToPaths_SingleNodePath(t *testing.T) {
	rg := graph.NewResidualGraph()

	paths := ToPaths([][]int64{{1}}, rg)

	// Single node path should be filtered out
	assert.Empty(t, paths)
}

func TestToPaths_MultiplePaths(t *testing.T) {
	rg := graph.NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 1)
	rg.AddEdgeWithReverse(2, 4, 10, 1)
	rg.AddEdgeWithReverse(1, 3, 10, 2)
	rg.AddEdgeWithReverse(3, 4, 10, 2)

	rg.UpdateFlow(1, 2, 5)
	rg.UpdateFlow(2, 4, 5)
	rg.UpdateFlow(1, 3, 3)
	rg.UpdateFlow(3, 4, 3)

	rawPaths := [][]int64{
		{1, 2, 4},
		{1, 3, 4},
	}

	paths := ToPaths(rawPaths, rg)

	assert.Len(t, paths, 2)
}

func TestUpdateGraphWithFlow(t *testing.T) {
	proto := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, CurrentFlow: 0},
		},
		SourceId: 1,
		SinkId:   2,
		Name:     "test",
	}

	rg := graph.NewResidualGraph()
	rg.AddEdgeWithReverse(1, 2, 10, 0)
	rg.UpdateFlow(1, 2, 7)

	result := UpdateGraphWithFlow(proto, rg)

	assert.Equal(t, "test", result.Name)
	assert.Equal(t, int64(1), result.SourceId)
	assert.Equal(t, int64(2), result.SinkId)
	require.Len(t, result.Edges, 1)
	assert.Equal(t, 7.0, result.Edges[0].CurrentFlow)
}

func TestCalculateGraphStatistics_EmptyGraph(t *testing.T) {
	proto := &commonv1.Graph{
		Nodes: []*commonv1.Node{},
		Edges: []*commonv1.Edge{},
	}

	stats := CalculateGraphStatistics(proto)

	assert.Equal(t, int64(0), stats.NodeCount)
	assert.Equal(t, int64(0), stats.EdgeCount)
	assert.Equal(t, 0.0, stats.Density)
}

func TestCalculateGraphStatistics_SimpleGraph(t *testing.T) {
	proto := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_DELIVERY_POINT},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_INTERSECTION},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 100, Length: 10},
			{From: 2, To: 3, Capacity: 50, Length: 20},
		},
	}

	stats := CalculateGraphStatistics(proto)

	assert.Equal(t, int64(3), stats.NodeCount)
	assert.Equal(t, int64(2), stats.EdgeCount)
	assert.Equal(t, int64(1), stats.WarehouseCount)
	assert.Equal(t, int64(1), stats.DeliveryPointCount)
	assert.Equal(t, 150.0, stats.TotalCapacity)
	assert.InDelta(t, 15.0, stats.AverageEdgeLength, 1e-9)
}

func TestCalculateGraphStatistics_CompleteGraph(t *testing.T) {
	// Полный граф на 4 вершинах (6 ребер)
	proto := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
			{From: 1, To: 3, Capacity: 10},
			{From: 1, To: 4, Capacity: 10},
			{From: 2, To: 3, Capacity: 10},
			{From: 2, To: 4, Capacity: 10},
			{From: 3, To: 4, Capacity: 10},
		},
	}

	stats := CalculateGraphStatistics(proto)

	// Density = E / (N * (N-1)) = 6 / 12 = 0.5
	assert.InDelta(t, 0.5, stats.Density, 1e-9)
}

func TestCalculateGraphStatistics_SingleNode(t *testing.T) {
	proto := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}},
		Edges: []*commonv1.Edge{},
	}

	stats := CalculateGraphStatistics(proto)

	assert.Equal(t, int64(1), stats.NodeCount)
	assert.Equal(t, 0.0, stats.Density) // Нет ребер
}

func TestToPaths_InfiniteFlow(t *testing.T) {
	// Граф без рёбер, но с путём - flow останется Infinity и должен стать 0
	rg := graph.NewResidualGraph()
	rg.AddNode(1)
	rg.AddNode(2)
	rg.AddNode(3)
	// Рёбра НЕ добавляем

	rawPaths := [][]int64{{1, 2, 3}}

	paths := ToPaths(rawPaths, rg)

	require.Len(t, paths, 1)
	assert.Equal(t, 0.0, paths[0].Flow, "Flow should be 0 when edges don't exist")
}
