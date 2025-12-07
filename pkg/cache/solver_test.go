package cache

import (
	"context"
	"testing"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
)

func TestSolverCache_SetGet(t *testing.T) {
	memCache := NewMemoryCache(nil)
	defer memCache.Close()

	solverCache := NewSolverCache(memCache, 5*time.Minute)

	ctx := context.Background()
	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   3,
		Nodes: []*commonv1.Node{
			{Id: 1, Type: commonv1.NodeType_NODE_TYPE_SOURCE},
			{Id: 2, Type: commonv1.NodeType_NODE_TYPE_WAREHOUSE},
			{Id: 3, Type: commonv1.NodeType_NODE_TYPE_SINK},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1},
			{From: 2, To: 3, Capacity: 10, Cost: 1},
		},
	}

	result := &CachedSolveResult{
		MaxFlow:           10,
		TotalCost:         20,
		Status:            "FLOW_STATUS_OPTIMAL",
		Iterations:        5,
		ComputationTimeMs: 1.5,
		FlowEdges: []*FlowEdgeCache{
			{From: 1, To: 2, Flow: 10, Capacity: 10, Utilization: 1.0},
			{From: 2, To: 3, Flow: 10, Capacity: 10, Utilization: 1.0},
		},
	}

	// Set
	err := solverCache.Set(ctx, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, result, 0)
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	// Get
	got, found, err := solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if !found {
		t.Fatal("expected to find cached result")
	}

	if got.MaxFlow != result.MaxFlow {
		t.Errorf("expected max flow %f, got %f", result.MaxFlow, got.MaxFlow)
	}
	if got.TotalCost != result.TotalCost {
		t.Errorf("expected total cost %f, got %f", result.TotalCost, got.TotalCost)
	}
	if len(got.FlowEdges) != 2 {
		t.Errorf("expected 2 flow edges, got %d", len(got.FlowEdges))
	}
}

func TestSolverCache_GetNotFound(t *testing.T) {
	memCache := NewMemoryCache(nil)
	defer memCache.Close()

	solverCache := NewSolverCache(memCache, 5*time.Minute)

	ctx := context.Background()
	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   2,
	}

	result, found, err := solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected not found")
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestSolverCache_DifferentAlgorithm(t *testing.T) {
	memCache := NewMemoryCache(nil)
	defer memCache.Close()

	solverCache := NewSolverCache(memCache, 5*time.Minute)

	ctx := context.Background()
	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   2,
	}

	result := &CachedSolveResult{MaxFlow: 10}

	// Set for one algorithm
	solverCache.Set(ctx, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, result, 0)

	// Try to get for different algorithm
	_, found, _ := solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC)
	if found {
		t.Error("should not find result for different algorithm")
	}
}

func TestSolverCache_SetFromResponse(t *testing.T) {
	memCache := NewMemoryCache(nil)
	defer memCache.Close()

	solverCache := NewSolverCache(memCache, 5*time.Minute)

	ctx := context.Background()
	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   2,
	}

	resp := &optimizationv1.SolveResponse{
		Result: &commonv1.FlowResult{
			MaxFlow:           15,
			TotalCost:         30,
			Status:            commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
			Iterations:        10,
			ComputationTimeMs: 2.5,
			Edges: []*commonv1.FlowEdge{
				{From: 1, To: 2, Flow: 15, Capacity: 20, Utilization: 0.75},
			},
		},
	}

	err := solverCache.SetFromResponse(ctx, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL, resp, 0)
	if err != nil {
		t.Fatalf("failed to set from response: %v", err)
	}

	got, found, _ := solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
	if !found {
		t.Fatal("expected to find cached result")
	}

	if got.MaxFlow != 15 {
		t.Errorf("expected max flow 15, got %f", got.MaxFlow)
	}
}

func TestSolverCache_SetFromResponse_NilResponse(t *testing.T) {
	memCache := NewMemoryCache(nil)
	defer memCache.Close()

	solverCache := NewSolverCache(memCache, 5*time.Minute)

	ctx := context.Background()
	graph := &commonv1.Graph{}

	// Should not error on nil response
	err := solverCache.SetFromResponse(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC, nil, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = solverCache.SetFromResponse(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC, &optimizationv1.SolveResponse{}, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSolverCache_Invalidate(t *testing.T) {
	memCache := NewMemoryCache(nil)
	defer memCache.Close()

	solverCache := NewSolverCache(memCache, 5*time.Minute)

	ctx := context.Background()
	graph := &commonv1.Graph{
		SourceId: 1,
		SinkId:   2,
	}

	result := &CachedSolveResult{MaxFlow: 10}

	// Set
	solverCache.Set(ctx, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, result, 0)
	solverCache.Set(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC, result, 0)

	// Invalidate
	err := solverCache.Invalidate(ctx, graph)
	if err != nil {
		t.Fatalf("failed to invalidate: %v", err)
	}

	// Both should be gone
	_, found1, _ := solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
	_, found2, _ := solverCache.Get(ctx, graph, commonv1.Algorithm_ALGORITHM_DINIC)

	if found1 || found2 {
		t.Error("expected cache to be invalidated")
	}
}

func TestSolverCache_InvalidateAll(t *testing.T) {
	memCache := NewMemoryCache(nil)
	defer memCache.Close()

	solverCache := NewSolverCache(memCache, 5*time.Minute)

	ctx := context.Background()

	graph1 := &commonv1.Graph{SourceId: 1, SinkId: 2}
	graph2 := &commonv1.Graph{SourceId: 3, SinkId: 4}

	result := &CachedSolveResult{MaxFlow: 10}

	solverCache.Set(ctx, graph1, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, result, 0)
	solverCache.Set(ctx, graph2, commonv1.Algorithm_ALGORITHM_DINIC, result, 0)

	count, err := solverCache.InvalidateAll(ctx)
	if err != nil {
		t.Fatalf("failed to invalidate all: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 invalidated, got %d", count)
	}
}

func TestCachedSolveResult_ToFlowResult(t *testing.T) {
	cached := &CachedSolveResult{
		MaxFlow:           20,
		TotalCost:         40,
		Status:            "FLOW_STATUS_OPTIMAL",
		Iterations:        15,
		ComputationTimeMs: 3.5,
		FlowEdges: []*FlowEdgeCache{
			{From: 1, To: 2, Flow: 10, Capacity: 15, Utilization: 0.67},
			{From: 2, To: 3, Flow: 10, Capacity: 10, Utilization: 1.0},
		},
	}

	result := cached.ToFlowResult()

	if result.MaxFlow != 20 {
		t.Errorf("expected max flow 20, got %f", result.MaxFlow)
	}
	if result.TotalCost != 40 {
		t.Errorf("expected total cost 40, got %f", result.TotalCost)
	}
	if result.Iterations != 15 {
		t.Errorf("expected 15 iterations, got %d", result.Iterations)
	}
	if len(result.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(result.Edges))
	}
}
