package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/pkg/cache"
	"logistics/pkg/logger"
)

func TestMain(m *testing.M) {
	// Инициализируем логгер для тестов
	logger.Init("error")

	os.Exit(m.Run())
}

func TestNewSolverService(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	if svc == nil {
		t.Fatal("NewSolverService returned nil")
	}
	if svc.version != "1.0.0" {
		t.Errorf("Version = %s, want 1.0.0", svc.version)
	}
}

func TestSolverService_Solve_NilGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	_, err := svc.Solve(ctx, req)
	if err == nil {
		t.Error("Expected error for nil graph")
	}
}

func TestSolverService_Solve_EmptyGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{},
			Edges: []*commonv1.Edge{},
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	_, err := svc.Solve(ctx, req)
	if err == nil {
		t.Error("Expected error for empty graph")
	}
}

func TestSolverService_Solve_InvalidSource(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1},
				{Id: 2},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0},
			},
			SourceId: 99, // Invalid
			SinkId:   2,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	_, err := svc.Solve(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid source")
	}
}

func TestSolverService_Solve_InvalidSink(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1},
				{Id: 2},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0},
			},
			SourceId: 1,
			SinkId:   99, // Invalid
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	_, err := svc.Solve(ctx, req)
	if err == nil {
		t.Error("Expected error for invalid sink")
	}
}

func TestSolverService_Solve_SourceEqualsSink(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1},
				{Id: 2},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0},
			},
			SourceId: 1,
			SinkId:   1, // Same as source
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	_, err := svc.Solve(ctx, req)
	if err == nil {
		t.Error("Expected error when source equals sink")
	}
}

func TestSolverService_Solve_SimpleGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// Simple: 1 -> 2 -> 3 with capacity 10
	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
				{From: 2, To: 3, Capacity: 10.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success")
	}
	if resp.Result.MaxFlow != 10.0 {
		t.Errorf("MaxFlow = %f, want 10.0", resp.Result.MaxFlow)
	}
	if resp.Result.Status != commonv1.FlowStatus_FLOW_STATUS_OPTIMAL {
		t.Errorf("Status = %v, want OPTIMAL", resp.Result.Status)
	}
}

func TestSolverService_Solve_DiamondGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// Diamond: 1 -> 2 -> 4
	//          1 -> 3 -> 4
	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
				{From: 1, To: 3, Capacity: 10.0, Cost: 1.0},
				{From: 2, To: 4, Capacity: 10.0, Cost: 1.0},
				{From: 3, To: 4, Capacity: 10.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   4,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if resp.Result.MaxFlow != 20.0 {
		t.Errorf("MaxFlow = %f, want 20.0", resp.Result.MaxFlow)
	}
}

func TestSolverService_Solve_BottleneckGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// Bottleneck: 1 -100-> 2 -5-> 3 -100-> 4
	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 100.0, Cost: 1.0},
				{From: 2, To: 3, Capacity: 5.0, Cost: 1.0}, // Bottleneck
				{From: 3, To: 4, Capacity: 100.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   4,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if resp.Result.MaxFlow != 5.0 {
		t.Errorf("MaxFlow = %f, want 5.0 (bottleneck)", resp.Result.MaxFlow)
	}
}

func TestSolverService_Solve_NoPathGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// Disconnected: 1 -> 2, 3 -> 4 (no path 1->4)
	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
				{From: 3, To: 4, Capacity: 10.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   4,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if resp.Result.MaxFlow != 0.0 {
		t.Errorf("MaxFlow = %f, want 0.0 (no path)", resp.Result.MaxFlow)
	}
}

func TestSolverService_Solve_AllAlgorithms(t *testing.T) {
	algorithms := []commonv1.Algorithm{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
		commonv1.Algorithm_ALGORITHM_MIN_COST,
		commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
	}

	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
			{From: 1, To: 3, Capacity: 10.0, Cost: 2.0},
			{From: 2, To: 4, Capacity: 10.0, Cost: 1.0},
			{From: 3, To: 4, Capacity: 10.0, Cost: 2.0},
		},
		SourceId: 1,
		SinkId:   4,
	}

	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	for _, algo := range algorithms {
		t.Run(algo.String(), func(t *testing.T) {
			req := &optimizationv1.SolveRequest{
				Graph:     graph,
				Algorithm: algo,
			}

			resp, err := svc.Solve(ctx, req)
			if err != nil {
				t.Fatalf("Algorithm %s failed: %v", algo, err)
			}

			if !resp.Success {
				t.Errorf("Algorithm %s: expected success", algo)
			}
			if resp.Result.MaxFlow != 20.0 {
				t.Errorf("Algorithm %s: MaxFlow = %f, want 20.0", algo, resp.Result.MaxFlow)
			}
		})
	}
}

func TestSolverService_Solve_WithOptions(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
				{From: 2, To: 3, Capacity: 10.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		Options: &optimizationv1.SolveOptions{
			TimeoutSeconds: 10,
			ReturnPaths:    true,
			MaxIterations:  100,
			Epsilon:        1e-6,
		},
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve with options failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success")
	}
}

func TestSolverService_Solve_ReturnsSolvedGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1, Name: "Source"},
				{Id: 2, Name: "Middle"},
				{Id: 3, Name: "Sink"},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
				{From: 2, To: 3, Capacity: 8.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   3,
			Name:     "Test Graph",
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if resp.SolvedGraph == nil {
		t.Fatal("SolvedGraph is nil")
	}

	// Check flow on edges
	hasFlow := false
	for _, edge := range resp.SolvedGraph.Edges {
		if edge.CurrentFlow > 0 {
			hasFlow = true
			break
		}
	}
	if !hasFlow {
		t.Error("Solved graph should have flow on edges")
	}
}

func TestSolverService_Solve_ReturnsMetrics(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
				{From: 2, To: 3, Capacity: 10.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if resp.Metrics == nil {
		t.Fatal("Metrics is nil")
	}
	if resp.Metrics.ComputationTimeMs < 0 {
		t.Error("ComputationTimeMs should be non-negative")
	}
	if resp.Metrics.Iterations < 0 {
		t.Error("Iterations should be non-negative")
	}
}

func TestSolverService_Solve_LargeGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// Grid graph 10x10
	nodes := make([]*commonv1.Node, 100)
	var edges []*commonv1.Edge

	for i := 0; i < 100; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}

		// Right edge
		if i%10 != 9 {
			edges = append(edges, &commonv1.Edge{
				From:     int64(i),
				To:       int64(i + 1),
				Capacity: 10.0,
				Cost:     1.0,
			})
		}
		// Down edge
		if i < 90 {
			edges = append(edges, &commonv1.Edge{
				From:     int64(i),
				To:       int64(i + 10),
				Capacity: 10.0,
				Cost:     1.0,
			})
		}
	}

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes:    nodes,
			Edges:    edges,
			SourceId: 0,
			SinkId:   99,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Large graph solve failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success for large graph")
	}
	if resp.Result.MaxFlow <= 0 {
		t.Error("Expected positive max flow")
	}
}

func TestSolverService_Solve_MinCostAlgorithm(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// Two paths: cheap (1->2->4, cost=2) vs expensive (1->3->4, cost=20)
	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},  // Cheap
				{From: 2, To: 4, Capacity: 10.0, Cost: 1.0},  // Cheap
				{From: 1, To: 3, Capacity: 10.0, Cost: 10.0}, // Expensive
				{From: 3, To: 4, Capacity: 10.0, Cost: 10.0}, // Expensive
			},
			SourceId: 1,
			SinkId:   4,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_MIN_COST,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Min cost solve failed: %v", err)
	}

	if resp.Result.MaxFlow != 20.0 {
		t.Errorf("MaxFlow = %f, want 20.0", resp.Result.MaxFlow)
	}

	// Total cost should use cheaper path first
	// First 10 units through cheap path: 10 * 2 = 20
	// Next 10 units through expensive path: 10 * 20 = 200
	// Total = 220
	if resp.Result.TotalCost != 220.0 {
		t.Errorf("TotalCost = %f, want 220.0", resp.Result.TotalCost)
	}
}

func TestSolverService_GetAlgorithms(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	resp, err := svc.GetAlgorithms(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetAlgorithms failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if len(resp.Algorithms) == 0 {
		t.Error("Expected at least one algorithm")
	}

	// Check required algorithms present
	expectedAlgorithms := map[commonv1.Algorithm]bool{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP:   false,
		commonv1.Algorithm_ALGORITHM_DINIC:          false,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL:   false,
		commonv1.Algorithm_ALGORITHM_MIN_COST:       false,
		commonv1.Algorithm_ALGORITHM_FORD_FULKERSON: false,
	}

	for _, algo := range resp.Algorithms {
		expectedAlgorithms[algo.Algorithm] = true

		// Check required fields
		if algo.Name == "" {
			t.Errorf("Algorithm %v has empty name", algo.Algorithm)
		}
		if algo.Description == "" {
			t.Errorf("Algorithm %v has empty description", algo.Algorithm)
		}
		if algo.TimeComplexity == "" {
			t.Errorf("Algorithm %v has empty time complexity", algo.Algorithm)
		}
	}

	for algo, found := range expectedAlgorithms {
		if !found {
			t.Errorf("Algorithm %v not found in response", algo)
		}
	}
}

func TestSolverService_GetAlgorithms_MinCostSupport(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	resp, err := svc.GetAlgorithms(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetAlgorithms failed: %v", err)
	}

	for _, algo := range resp.Algorithms {
		if algo.Algorithm == commonv1.Algorithm_ALGORITHM_MIN_COST {
			if !algo.SupportsMinCost {
				t.Error("MIN_COST algorithm should support min cost")
			}
			if !algo.SupportsNegativeCosts {
				t.Error("MIN_COST algorithm should support negative costs")
			}
		}
	}
}

func TestSolverService_Solve_ContextCancellation(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0},
			},
			SourceId: 1,
			SinkId:   2,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	// Depending on implementation, may succeed (fast enough) or fail
	_, err := svc.Solve(ctx, req)
	// Just checking it doesn't panic
	_ = err
}

func TestSolverService_Solve_Timeout(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Small graph should complete quickly
	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0},
			},
			SourceId: 1,
			SinkId:   2,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		Options: &optimizationv1.SolveOptions{
			TimeoutSeconds: 0.1,
		},
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		// May fail due to context timeout, but shouldn't panic
		return
	}

	if resp != nil && resp.Success {
		// OK - completed before timeout
	}
}

func TestSolverService_Solve_ZeroCapacityEdge(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 0.0, Cost: 1.0}, // Zero capacity
				{From: 2, To: 3, Capacity: 10.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if resp.Result.MaxFlow != 0.0 {
		t.Errorf("MaxFlow = %f, want 0.0 (zero capacity edge)", resp.Result.MaxFlow)
	}
}

// Mock stream for testing SolveStream
type mockSolveStream struct {
	optimizationv1.SolverService_SolveStreamServer
	messages []*optimizationv1.SolveProgress
	ctx      context.Context
}

func (m *mockSolveStream) Send(progress *optimizationv1.SolveProgress) error {
	m.messages = append(m.messages, progress)
	return nil
}

func (m *mockSolveStream) Context() context.Context {
	return m.ctx
}

func TestSolverService_SolveStream_SimpleGraph(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
				{From: 1, To: 3, Capacity: 10.0, Cost: 1.0},
				{From: 2, To: 4, Capacity: 10.0, Cost: 1.0},
				{From: 3, To: 4, Capacity: 10.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   4,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	stream := &mockSolveStream{
		ctx:      context.Background(),
		messages: make([]*optimizationv1.SolveProgress, 0),
	}

	err := svc.SolveStream(req, stream)
	if err != nil {
		t.Fatalf("SolveStream failed: %v", err)
	}

	if len(stream.messages) == 0 {
		t.Error("Expected at least one progress message")
	}

	// Check last message
	lastMsg := stream.messages[len(stream.messages)-1]
	if lastMsg.Status != "completed" {
		t.Errorf("Last status = %s, want 'completed'", lastMsg.Status)
	}
	if lastMsg.ProgressPercent != 100.0 {
		t.Errorf("Last progress = %f, want 100.0", lastMsg.ProgressPercent)
	}
	if lastMsg.CurrentFlow != 20.0 {
		t.Errorf("Final flow = %f, want 20.0", lastMsg.CurrentFlow)
	}
}

func TestSolverService_Solve_Validation(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	tests := []struct {
		name string
		req  *optimizationv1.SolveRequest
	}{
		{"NilGraph", &optimizationv1.SolveRequest{Graph: nil}},
		{"EmptyGraph", &optimizationv1.SolveRequest{Graph: &commonv1.Graph{Nodes: []*commonv1.Node{}}}},
		{"InvalidSource", &optimizationv1.SolveRequest{Graph: &commonv1.Graph{Nodes: []*commonv1.Node{{Id: 1}}, SourceId: 99, SinkId: 1}}},
		{"InvalidSink", &optimizationv1.SolveRequest{Graph: &commonv1.Graph{Nodes: []*commonv1.Node{{Id: 1}}, SourceId: 1, SinkId: 99}}},
		{"SourceEqualsSink", &optimizationv1.SolveRequest{Graph: &commonv1.Graph{Nodes: []*commonv1.Node{{Id: 1}}, SourceId: 1, SinkId: 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Solve(ctx, tt.req)
			if err == nil {
				t.Errorf("%s: Expected error, got nil", tt.name)
			}
		})
	}
}

func TestSolverService_Solve_Algorithms(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// Diamond Graph: 1->2->4, 1->3->4. Cap 10 everywhere. MaxFlow=20
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1},
			{From: 1, To: 3, Capacity: 10, Cost: 1},
			{From: 2, To: 4, Capacity: 10, Cost: 1},
			{From: 3, To: 4, Capacity: 10, Cost: 1},
		},
		SourceId: 1, SinkId: 4,
	}

	algos := []commonv1.Algorithm{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
	}

	for _, algo := range algos {
		t.Run(algo.String(), func(t *testing.T) {
			req := &optimizationv1.SolveRequest{Graph: graph, Algorithm: algo}
			resp, err := svc.Solve(ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !resp.Success {
				t.Error("Expected Success=true")
			}
			if resp.Result.MaxFlow != 20.0 {
				t.Errorf("MaxFlow: got %f, want 20.0", resp.Result.MaxFlow)
			}
		})
	}
}

func TestSolverService_Solve_MinCost(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// 1->2 (Cost 1, Cap 10), 1->3 (Cost 10, Cap 10). Both go to 4.
	// Total Cap 20. Max Flow should fill cheap path first.
	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1},
			{From: 2, To: 4, Capacity: 10, Cost: 1},
			{From: 1, To: 3, Capacity: 10, Cost: 10},
			{From: 3, To: 4, Capacity: 10, Cost: 10},
		},
		SourceId: 1, SinkId: 4,
	}

	req := &optimizationv1.SolveRequest{Graph: graph, Algorithm: commonv1.Algorithm_ALGORITHM_MIN_COST}
	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 10 units * 2 cost = 20
	// 10 units * 20 cost = 200
	// Total cost = 220
	if resp.Result.TotalCost != 220.0 {
		t.Errorf("TotalCost: got %f, want 220.0", resp.Result.TotalCost)
	}
}

func TestSolverService_SolveStream_Cancellation(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	// Создаем большой граф, чтобы алгоритм не завершился мгновенно
	nodes := make([]*commonv1.Node, 1000)
	var edges []*commonv1.Edge
	for i := 0; i < 1000; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
		if i > 0 {
			edges = append(edges, &commonv1.Edge{
				From:     int64(i - 1),
				To:       int64(i),
				Capacity: 100.0,
			})
		}
	}

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes:    nodes,
			Edges:    edges,
			SourceId: 0,
			SinkId:   999,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	// Отменяем контекст СРАЗУ
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stream := &mockSolveStream{
		ctx:      ctx,
		messages: make([]*optimizationv1.SolveProgress, 0),
	}

	// Запускаем метод
	err := svc.SolveStream(req, stream)

	// ПРОВЕРКИ
	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	} else if err != context.Canceled {
		// Некоторые реализации gRPC могут оборачивать ошибку, но внутри сервиса мы возвращаем ctx.Err()
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}

	// Проверяем, что стрим не был "успешно завершен"
	if len(stream.messages) > 0 {
		lastMsg := stream.messages[len(stream.messages)-1]
		if lastMsg.Status == "completed" {
			t.Error("Stream sent 'completed' status despite cancellation")
		}
	}
}

func TestSolverService_SolveStream_Success(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	graph := &commonv1.Graph{
		Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10},
			{From: 2, To: 3, Capacity: 10},
		},
		SourceId: 1, SinkId: 3,
	}

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if len(stream.messages) == 0 {
		t.Fatal("Expected messages in stream")
	}

	lastMsg := stream.messages[len(stream.messages)-1]
	if lastMsg.Status != "completed" {
		t.Errorf("Last status expected 'completed', got %s", lastMsg.Status)
	}
	if lastMsg.CurrentFlow != 10.0 {
		t.Errorf("Expected flow 10.0, got %f", lastMsg.CurrentFlow)
	}
}

func TestSolverService_Solve_ParallelEdges(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	// Граф с ребрами в обоих направлениях
	// Важно: ребро 2->1 НЕ увеличивает max flow из 1 в 3
	// Поток идет ТОЛЬКО 1->2->3
	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10.0, Cost: 1.0},
				{From: 2, To: 3, Capacity: 10.0, Cost: 1.0},
				// Обратное ребро 2->1 не помогает потоку 1->3
				{From: 2, To: 1, Capacity: 5.0, Cost: 1.0},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Max flow ограничен ребром 2->3 = 10
	if resp.Result.MaxFlow != 10.0 {
		t.Errorf("MaxFlow = %f, want 10.0", resp.Result.MaxFlow)
	}
}

// Вспомогательные моки для тестов кэша

type mockSolverCache struct {
	getResult *cache.CachedSolveResult
	getFound  bool
	getError  error
	setError  error
	setCalled bool
}

func (m *mockSolverCache) Get(ctx context.Context, g *commonv1.Graph, algo commonv1.Algorithm) (*cache.CachedSolveResult, bool, error) {
	return m.getResult, m.getFound, m.getError
}

func (m *mockSolverCache) SetFromResponse(ctx context.Context, g *commonv1.Graph, algo commonv1.Algorithm, resp *optimizationv1.SolveResponse, ttl time.Duration) error {
	m.setCalled = true
	return m.setError
}

// Мок для интерфейса cache.Cache
type mockCache struct {
	data      map[string][]byte
	getError  error
	setError  error
	shouldHit bool
	hitData   []byte
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string][]byte),
	}
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	if m.shouldHit && m.hitData != nil {
		return m.hitData, nil
	}
	if data, ok := m.data[key]; ok {
		return data, nil
	}
	return nil, cache.ErrKeyNotFound
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if m.setError != nil {
		return m.setError
	}
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockCache) GetWithTTL(ctx context.Context, key string) ([]byte, time.Duration, error) {
	if m.getError != nil {
		return nil, 0, m.getError
	}
	if m.shouldHit && m.hitData != nil {
		return m.hitData, 5 * time.Minute, nil
	}
	if data, ok := m.data[key]; ok {
		return data, 5 * time.Minute, nil
	}
	return nil, 0, cache.ErrKeyNotFound
}

func (m *mockCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, key := range keys {
		if data, ok := m.data[key]; ok {
			result[key] = data
		}
	}
	return result, nil
}

func (m *mockCache) MSet(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	if m.setError != nil {
		return m.setError
	}
	for key, value := range entries {
		m.data[key] = value
	}
	return nil
}

func (m *mockCache) MDelete(ctx context.Context, keys []string) (int64, error) {
	var count int64
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			count++
		}
	}
	return count, nil
}

func (m *mockCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys, nil
}

func (m *mockCache) DeleteByPattern(ctx context.Context, pattern string) (int64, error) {
	// Простая реализация - удаляем всё
	count := int64(len(m.data))
	m.data = make(map[string][]byte)
	return count, nil
}

func (m *mockCache) Stats(ctx context.Context) (*cache.Stats, error) {
	return &cache.Stats{
		TotalKeys: int64(len(m.data)),
		Backend:   "mock",
	}, nil
}

func (m *mockCache) Clear(ctx context.Context) error {
	m.data = make(map[string][]byte)
	return nil
}

func (m *mockCache) Close() error {
	return nil
}

func TestSolverService_Solve_WithCache_Hit(t *testing.T) {
	// Создаём кэшированный результат
	cachedResult := &cache.CachedSolveResult{
		MaxFlow:   42.0,
		TotalCost: 100.0,
		Status:    "FLOW_STATUS_OPTIMAL",
	}
	cachedData, _ := json.Marshal(cachedResult)

	mockC := newMockCache()
	mockC.shouldHit = true
	mockC.hitData = cachedData

	solverCache := cache.NewSolverCache(mockC, 10*time.Minute)
	svc := NewSolverService("1.0.0", solverCache)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
			SourceId: 1,
			SinkId:   2,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, 42.0, resp.Result.MaxFlow)
	assert.Equal(t, float64(0), resp.Metrics.ComputationTimeMs, "Cache hit should have 0 computation time")
}

func TestSolverService_Solve_WithCache_Miss(t *testing.T) {
	mockC := newMockCache()
	solverCache := cache.NewSolverCache(mockC, 10*time.Minute)
	svc := NewSolverService("1.0.0", solverCache)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
			SourceId: 1,
			SinkId:   2,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, 10.0, resp.Result.MaxFlow)

	assert.Eventually(t, func() bool {
		return len(mockC.data) > 0
	}, time.Second, 10*time.Millisecond, "Cache Set should be called on miss")
}

func TestSolverService_Solve_CacheSetError(t *testing.T) {
	mockC := newMockCache()
	mockC.setError = errors.New("cache write error")

	solverCache := cache.NewSolverCache(mockC, 10*time.Minute)
	svc := NewSolverService("1.0.0", solverCache)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
			SourceId: 1,
			SinkId:   2,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	// Ошибка записи в кэш не должна влиять на результат
	resp, err := svc.Solve(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 10.0, resp.Result.MaxFlow)
}

func TestSolverService_SolveStream_IterationLimit(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	// Большой граф
	nodes := make([]*commonv1.Node, 100)
	var edges []*commonv1.Edge
	for i := 0; i < 100; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
		if i > 0 {
			edges = append(edges, &commonv1.Edge{From: int64(i - 1), To: int64(i), Capacity: 10})
		}
	}

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes:    nodes,
			Edges:    edges,
			SourceId: 0,
			SinkId:   99,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		Options: &optimizationv1.SolveOptions{
			MaxIterations: 1, // Очень маленький лимит
		},
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)
	require.NoError(t, err)

	// Должен завершиться, но с неполным потоком
	require.NotEmpty(t, stream.messages)
	lastMsg := stream.messages[len(stream.messages)-1]
	assert.Equal(t, "completed", lastMsg.Status)
}

func TestSolverService_SolveStream_Timeout(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	nodes := make([]*commonv1.Node, 50)
	var edges []*commonv1.Edge
	for i := 0; i < 50; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
		if i > 0 {
			edges = append(edges, &commonv1.Edge{From: int64(i - 1), To: int64(i), Capacity: 10})
		}
	}

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes:    nodes,
			Edges:    edges,
			SourceId: 0,
			SinkId:   49,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		Options: &optimizationv1.SolveOptions{
			TimeoutSeconds: 0.0000001, // Очень маленький таймаут
		},
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)
	require.NoError(t, err)

	// Должен завершиться (возможно, до таймаута на малом графе)
	require.NotEmpty(t, stream.messages)
	lastMsg := stream.messages[len(stream.messages)-1]
	assert.Equal(t, "completed", lastMsg.Status)
}

func TestSolverService_SolveStream_SendError(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10},
				{From: 2, To: 3, Capacity: 10},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	stream := &mockSolveStreamWithError{
		ctx:       context.Background(),
		sendErr:   errors.New("stream send error"),
		failAfter: 0, // Ошибка сразу на первой отправке
	}

	err := svc.SolveStream(req, stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stream send error")
}

// mockSolveStreamWithError - мок стрима с ошибкой отправки
type mockSolveStreamWithError struct {
	optimizationv1.SolverService_SolveStreamServer
	ctx       context.Context
	sendErr   error
	failAfter int
	sendCount int
}

func (m *mockSolveStreamWithError) Send(progress *optimizationv1.SolveProgress) error {
	if m.sendCount >= m.failAfter {
		return m.sendErr
	}
	m.sendCount++
	return nil
}

func (m *mockSolveStreamWithError) Context() context.Context {
	return m.ctx
}

func TestSolverService_SolveStream_PathEmpty(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	// Граф без пути от source к sink
	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10},
				// Нет пути от 2 к 4
				{From: 3, To: 4, Capacity: 10},
			},
			SourceId: 1,
			SinkId:   4,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)
	require.NoError(t, err)

	// Должен завершиться с потоком 0
	require.NotEmpty(t, stream.messages)
	lastMsg := stream.messages[len(stream.messages)-1]
	assert.Equal(t, "completed", lastMsg.Status)
	assert.Equal(t, 0.0, lastMsg.CurrentFlow)
}

func TestSolverService_SolveStream_PathFlowBelowEpsilon(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	// Граф с очень маленькой capacity
	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10},
				{From: 2, To: 3, Capacity: 1e-15}, // Очень маленькая capacity
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		Options: &optimizationv1.SolveOptions{
			Epsilon: 1e-9,
		},
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)
	require.NoError(t, err)

	// Должен завершиться с потоком ~0
	require.NotEmpty(t, stream.messages)
	lastMsg := stream.messages[len(stream.messages)-1]
	assert.Equal(t, "completed", lastMsg.Status)
	assert.InDelta(t, 0.0, lastMsg.CurrentFlow, 1e-9)
}

func TestSolverService_SolveStream_NoPathFromStart(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	// Граф где source изолирован
	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				// Нет рёбер из source (1)
				{From: 2, To: 3, Capacity: 10},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)
	require.NoError(t, err)

	require.NotEmpty(t, stream.messages)
	lastMsg := stream.messages[len(stream.messages)-1]
	assert.Equal(t, "completed", lastMsg.Status)
	assert.Equal(t, 0.0, lastMsg.CurrentFlow)
}

func TestSolverService_Solve_ReturnPaths(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10, Cost: 1},
				{From: 2, To: 3, Capacity: 10, Cost: 1},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		Options: &optimizationv1.SolveOptions{
			ReturnPaths: true,
		},
	}

	resp, err := svc.Solve(ctx, req)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Result.Paths)
}

func TestSolverService_Solve_FlowEdges(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{
				{Id: 1}, {Id: 2}, {Id: 3},
			},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10, Cost: 1},
				{From: 2, To: 3, Capacity: 10, Cost: 1},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	resp, err := svc.Solve(ctx, req)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.Result.Edges)
	// Check that edges have flow
	hasFlow := false
	for _, edge := range resp.Result.Edges {
		if edge.Flow > 0 {
			hasFlow = true
			break
		}
	}
	assert.True(t, hasFlow)
}

func TestSolverService_Stats_Increment(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)
	ctx := context.Background()

	statsBefore := svc.GetStats()

	req := &optimizationv1.SolveRequest{
		Graph: &commonv1.Graph{
			Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
			SourceId: 1,
			SinkId:   2,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	_, _ = svc.Solve(ctx, req)
	_, _ = svc.Solve(ctx, req)
	_, _ = svc.Solve(ctx, req)

	statsAfter := svc.GetStats()

	assert.Equal(t, statsBefore.RequestsTotal+3, statsAfter.RequestsTotal)
	assert.Equal(t, statsBefore.RequestsSuccess+3, statsAfter.RequestsSuccess)
}

func TestSolverService_SolveStream_Dinic(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4}},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10},
				{From: 1, To: 3, Capacity: 10},
				{From: 2, To: 4, Capacity: 10},
				{From: 3, To: 4, Capacity: 10},
			},
			SourceId: 1,
			SinkId:   4,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)

	require.NoError(t, err)
	require.NotEmpty(t, stream.messages)

	lastMsg := stream.messages[len(stream.messages)-1]
	assert.Equal(t, "completed", lastMsg.Status)
	assert.Equal(t, 20.0, lastMsg.CurrentFlow)
}

func TestSolverService_SolveStream_PushRelabel(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10},
				{From: 2, To: 3, Capacity: 10},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)

	require.NoError(t, err)
	require.NotEmpty(t, stream.messages)

	lastMsg := stream.messages[len(stream.messages)-1]
	assert.Equal(t, "completed", lastMsg.Status)
	assert.Equal(t, 10.0, lastMsg.CurrentFlow)
}

func TestSolverService_SolveStream_MinCost(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}, {Id: 3}},
			Edges: []*commonv1.Edge{
				{From: 1, To: 2, Capacity: 10, Cost: 1},
				{From: 2, To: 3, Capacity: 10, Cost: 1},
			},
			SourceId: 1,
			SinkId:   3,
		},
		Algorithm: commonv1.Algorithm_ALGORITHM_MIN_COST,
	}

	stream := &mockSolveStream{ctx: context.Background()}
	err := svc.SolveStream(req, stream)

	require.NoError(t, err)
	require.NotEmpty(t, stream.messages)

	lastMsg := stream.messages[len(stream.messages)-1]
	assert.Equal(t, "completed", lastMsg.Status)
	assert.Equal(t, 10.0, lastMsg.CurrentFlow)
}

func TestSolverService_OptionsValidation(t *testing.T) {
	svc := NewSolverService("1.0.0", nil)

	tests := []struct {
		name string
		opts *optimizationv1.SolveOptions
	}{
		{
			name: "very_small_epsilon",
			opts: &optimizationv1.SolveOptions{Epsilon: 1e-20},
		},
		{
			name: "very_large_epsilon",
			opts: &optimizationv1.SolveOptions{Epsilon: 1},
		},
		{
			name: "very_small_timeout",
			opts: &optimizationv1.SolveOptions{TimeoutSeconds: 0.001},
		},
		{
			name: "very_large_timeout",
			opts: &optimizationv1.SolveOptions{TimeoutSeconds: 100000},
		},
		{
			name: "very_small_iterations",
			opts: &optimizationv1.SolveOptions{MaxIterations: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.buildSolverOptions(tt.opts)

			// Should not panic and should have valid values
			assert.GreaterOrEqual(t, result.Epsilon, MinEpsilon)
			assert.LessOrEqual(t, result.Epsilon, MaxEpsilon)
			if tt.opts.TimeoutSeconds > 0 {
				assert.GreaterOrEqual(t, result.Timeout.Seconds(), MinTimeoutSeconds)
				assert.LessOrEqual(t, result.Timeout.Seconds(), MaxTimeoutSeconds)
			}
			if tt.opts.MaxIterations > 0 {
				assert.GreaterOrEqual(t, result.MaxIterations, MinIterations)
			}
		})
	}
}

func TestMemStatsCache_Refresh(t *testing.T) {
	cache := newMemStatsCache(10 * time.Millisecond)

	// First read forces refresh
	alloc1 := cache.get()
	assert.Greater(t, alloc1, uint64(0))

	// Immediate second read should be cached
	alloc2 := cache.get()
	assert.Equal(t, alloc1, alloc2)

	// Wait for cache to expire
	time.Sleep(15 * time.Millisecond)

	// Should refresh now
	alloc3 := cache.get()
	// May or may not be equal depending on GC, but should not be 0
	assert.Greater(t, alloc3, uint64(0))
}

func TestProgressTracker(t *testing.T) {
	// Simple test for progress tracker fields
	start := time.Now()
	cache := newMemStatsCache(time.Second)
	tracker := &progressTracker{
		stream:        nil,
		start:         start,
		lastSendTime:  start,
		memStatsCache: cache,
	}

	assert.Equal(t, start, tracker.start)
	assert.Equal(t, start, tracker.lastSendTime)
	assert.NotNil(t, tracker.memStatsCache)
}
