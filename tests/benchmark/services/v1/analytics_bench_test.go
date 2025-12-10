package services_benchmark

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	analyticssvc "logistics/services/analytics-svc"
)

var (
	analyticsListener *bufconn.Listener
	analyticsClient   analyticsv1.AnalyticsServiceClient
)

func init() {
	analyticsListener = bufconn.Listen(bufSize)

	server := grpc.NewServer()
	svc := analyticssvc.NewBenchmarkServer()
	analyticsv1.RegisterAnalyticsServiceServer(server, svc)

	go func() {
		if err := server.Serve(analyticsListener); err != nil {
			log.Fatalf("Analytics server exited with error: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return analyticsListener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to dial bufnet: %v", err)
	}

	analyticsClient = analyticsv1.NewAnalyticsServiceClient(conn)
}

// =============================================================================
// GRAPH GENERATORS FOR ANALYTICS
// =============================================================================

func generateAnalyticsGraph(nodes, edges int, withFlow bool) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	nodeList := make([]*commonv1.Node, nodes)
	for i := 0; i < nodes; i++ {
		nodeList[i] = &commonv1.Node{
			Id:   int64(i),
			Type: commonv1.NodeType(r.Intn(5)),
		}
	}

	edgeList := make([]*commonv1.Edge, 0, edges)
	for i := 0; i < edges; i++ {
		from := int64(r.Intn(nodes - 1))
		to := from + 1 + int64(r.Intn(nodes-int(from)-1))
		if to >= int64(nodes) {
			to = int64(nodes - 1)
		}

		capacity := float64(r.Intn(100) + 10)
		flow := 0.0
		if withFlow {
			flow = capacity * (0.3 + r.Float64()*0.7) // 30-100% utilization
		}

		edgeList = append(edgeList, &commonv1.Edge{
			From:        from,
			To:          to,
			Capacity:    capacity,
			Cost:        float64(r.Intn(50) + 1),
			CurrentFlow: flow,
			RoadType:    commonv1.RoadType(r.Intn(5)),
		})
	}

	return &commonv1.Graph{
		Nodes:    nodeList,
		Edges:    edgeList,
		SourceId: 0,
		SinkId:   int64(nodes - 1),
	}
}

func generateBottleneckGraph(nodes int, bottleneckCount int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	nodeList := make([]*commonv1.Node, nodes)
	for i := 0; i < nodes; i++ {
		nodeList[i] = &commonv1.Node{Id: int64(i)}
	}

	var edgeList []*commonv1.Edge

	// Create edges with some at high utilization (bottlenecks)
	for i := 0; i < nodes-1; i++ {
		capacity := float64(r.Intn(100) + 10)
		utilization := 0.5 + r.Float64()*0.3 // Normal: 50-80%

		if i < bottleneckCount {
			utilization = 0.9 + r.Float64()*0.1 // Bottleneck: 90-100%
		}

		edgeList = append(edgeList, &commonv1.Edge{
			From:        int64(i),
			To:          int64(i + 1),
			Capacity:    capacity,
			Cost:        float64(r.Intn(10) + 1),
			CurrentFlow: capacity * utilization,
		})
	}

	return &commonv1.Graph{
		Nodes:    nodeList,
		Edges:    edgeList,
		SourceId: 0,
		SinkId:   int64(nodes - 1),
	}
}

// =============================================================================
// CALCULATE COST BENCHMARKS
// =============================================================================

func BenchmarkAnalytics_CalculateCost_Small(b *testing.B) {
	graph := generateAnalyticsGraph(50, 100, true)
	ctx := context.Background()
	req := &analyticsv1.CalculateCostRequest{
		Graph: graph,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.CalculateCost(ctx, req)
		if err != nil {
			b.Fatalf("CalculateCost failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_CalculateCost_Medium(b *testing.B) {
	graph := generateAnalyticsGraph(200, 500, true)
	ctx := context.Background()
	req := &analyticsv1.CalculateCostRequest{
		Graph: graph,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.CalculateCost(ctx, req)
		if err != nil {
			b.Fatalf("CalculateCost failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_CalculateCost_Large(b *testing.B) {
	graph := generateAnalyticsGraph(1000, 3000, true)
	ctx := context.Background()
	req := &analyticsv1.CalculateCostRequest{
		Graph: graph,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.CalculateCost(ctx, req)
		if err != nil {
			b.Fatalf("CalculateCost failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_CalculateCost_WithOptions(b *testing.B) {
	graph := generateAnalyticsGraph(200, 500, true)
	ctx := context.Background()
	req := &analyticsv1.CalculateCostRequest{
		Graph: graph,
		Options: &analyticsv1.CostOptions{
			Currency:          "USD",
			IncludeFixedCosts: true,
			CostMultipliers: map[string]float64{
				"HIGHWAY":   1.5,
				"PRIMARY":   1.2,
				"SECONDARY": 1.0,
			},
			DiscountPercent: 10,
			MarkupPercent:   5,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.CalculateCost(ctx, req)
		if err != nil {
			b.Fatalf("CalculateCost failed: %v", err)
		}
	}
}

// =============================================================================
// FIND BOTTLENECKS BENCHMARKS
// =============================================================================

func BenchmarkAnalytics_FindBottlenecks_Small(b *testing.B) {
	graph := generateBottleneckGraph(50, 5)
	ctx := context.Background()
	req := &analyticsv1.FindBottlenecksRequest{
		Graph:                graph,
		UtilizationThreshold: 0.9,
		TopN:                 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.FindBottlenecks(ctx, req)
		if err != nil {
			b.Fatalf("FindBottlenecks failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_FindBottlenecks_Medium(b *testing.B) {
	graph := generateBottleneckGraph(200, 20)
	ctx := context.Background()
	req := &analyticsv1.FindBottlenecksRequest{
		Graph:                graph,
		UtilizationThreshold: 0.9,
		TopN:                 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.FindBottlenecks(ctx, req)
		if err != nil {
			b.Fatalf("FindBottlenecks failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_FindBottlenecks_Large(b *testing.B) {
	graph := generateBottleneckGraph(1000, 100)
	ctx := context.Background()
	req := &analyticsv1.FindBottlenecksRequest{
		Graph:                graph,
		UtilizationThreshold: 0.9,
		TopN:                 20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.FindBottlenecks(ctx, req)
		if err != nil {
			b.Fatalf("FindBottlenecks failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_FindBottlenecks_LowThreshold(b *testing.B) {
	graph := generateBottleneckGraph(200, 50)
	ctx := context.Background()
	req := &analyticsv1.FindBottlenecksRequest{
		Graph:                graph,
		UtilizationThreshold: 0.5, // Lower threshold = more bottlenecks
		TopN:                 0,   // All
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.FindBottlenecks(ctx, req)
		if err != nil {
			b.Fatalf("FindBottlenecks failed: %v", err)
		}
	}
}

// =============================================================================
// ANALYZE FLOW BENCHMARKS
// =============================================================================

func BenchmarkAnalytics_AnalyzeFlow_Small(b *testing.B) {
	graph := generateAnalyticsGraph(50, 100, true)
	ctx := context.Background()
	req := &analyticsv1.AnalyzeFlowRequest{
		Graph: graph,
		Options: &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        true,
			FindBottlenecks:     true,
			CalculateStatistics: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.AnalyzeFlow(ctx, req)
		if err != nil {
			b.Fatalf("AnalyzeFlow failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_AnalyzeFlow_Medium(b *testing.B) {
	graph := generateAnalyticsGraph(200, 500, true)
	ctx := context.Background()
	req := &analyticsv1.AnalyzeFlowRequest{
		Graph: graph,
		Options: &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        true,
			FindBottlenecks:     true,
			CalculateStatistics: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.AnalyzeFlow(ctx, req)
		if err != nil {
			b.Fatalf("AnalyzeFlow failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_AnalyzeFlow_Large(b *testing.B) {
	graph := generateAnalyticsGraph(1000, 3000, true)
	ctx := context.Background()
	req := &analyticsv1.AnalyzeFlowRequest{
		Graph: graph,
		Options: &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        true,
			FindBottlenecks:     true,
			CalculateStatistics: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.AnalyzeFlow(ctx, req)
		if err != nil {
			b.Fatalf("AnalyzeFlow failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_AnalyzeFlow_OnlyStats(b *testing.B) {
	graph := generateAnalyticsGraph(500, 1500, true)
	ctx := context.Background()
	req := &analyticsv1.AnalyzeFlowRequest{
		Graph: graph,
		Options: &analyticsv1.AnalysisOptions{
			CalculateStatistics: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.AnalyzeFlow(ctx, req)
		if err != nil {
			b.Fatalf("AnalyzeFlow failed: %v", err)
		}
	}
}

// =============================================================================
// COMPARE SCENARIOS BENCHMARKS
// =============================================================================

func BenchmarkAnalytics_CompareScenarios_2Scenarios(b *testing.B) {
	baseline := generateAnalyticsGraph(100, 200, true)
	scenario1 := generateAnalyticsGraph(100, 200, true)
	scenario2 := generateAnalyticsGraph(100, 200, true)

	ctx := context.Background()
	req := &analyticsv1.CompareScenariosRequest{
		Baseline:      baseline,
		Scenarios:     []*commonv1.Graph{scenario1, scenario2},
		ScenarioNames: []string{"Scenario A", "Scenario B"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.CompareScenarios(ctx, req)
		if err != nil {
			b.Fatalf("CompareScenarios failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_CompareScenarios_5Scenarios(b *testing.B) {
	baseline := generateAnalyticsGraph(100, 200, true)
	scenarios := make([]*commonv1.Graph, 5)
	names := make([]string, 5)
	for i := 0; i < 5; i++ {
		scenarios[i] = generateAnalyticsGraph(100, 200, true)
		names[i] = fmt.Sprintf("Scenario %c", 'A'+i)
	}

	ctx := context.Background()
	req := &analyticsv1.CompareScenariosRequest{
		Baseline:      baseline,
		Scenarios:     scenarios,
		ScenarioNames: names,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.CompareScenarios(ctx, req)
		if err != nil {
			b.Fatalf("CompareScenarios failed: %v", err)
		}
	}
}

func BenchmarkAnalytics_CompareScenarios_10Scenarios(b *testing.B) {
	baseline := generateAnalyticsGraph(100, 200, true)
	scenarios := make([]*commonv1.Graph, 10)
	names := make([]string, 10)
	for i := 0; i < 10; i++ {
		scenarios[i] = generateAnalyticsGraph(100, 200, true)
		names[i] = fmt.Sprintf("Scenario %d", i+1)
	}

	ctx := context.Background()
	req := &analyticsv1.CompareScenariosRequest{
		Baseline:      baseline,
		Scenarios:     scenarios,
		ScenarioNames: names,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyticsClient.CompareScenarios(ctx, req)
		if err != nil {
			b.Fatalf("CompareScenarios failed: %v", err)
		}
	}
}

// =============================================================================
// SCALABILITY BENCHMARKS
// =============================================================================

func BenchmarkAnalytics_Scalability_CalculateCost(b *testing.B) {
	sizes := []struct {
		nodes int
		edges int
	}{
		{50, 100},
		{100, 300},
		{200, 600},
		{500, 1500},
		{1000, 3000},
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("N%d_E%d", size.nodes, size.edges), func(b *testing.B) {
			graph := generateAnalyticsGraph(size.nodes, size.edges, true)
			ctx := context.Background()
			req := &analyticsv1.CalculateCostRequest{Graph: graph}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = analyticsClient.CalculateCost(ctx, req)
			}
		})
	}
}

func BenchmarkAnalytics_Scalability_AnalyzeFlow(b *testing.B) {
	sizes := []struct {
		nodes int
		edges int
	}{
		{50, 100},
		{100, 300},
		{200, 600},
		{500, 1500},
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("N%d_E%d", size.nodes, size.edges), func(b *testing.B) {
			graph := generateAnalyticsGraph(size.nodes, size.edges, true)
			ctx := context.Background()
			req := &analyticsv1.AnalyzeFlowRequest{
				Graph: graph,
				Options: &analyticsv1.AnalysisOptions{
					AnalyzeCosts:        true,
					FindBottlenecks:     true,
					CalculateStatistics: true,
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = analyticsClient.AnalyzeFlow(ctx, req)
			}
		})
	}
}

// =============================================================================
// PARALLEL BENCHMARKS
// =============================================================================

func BenchmarkAnalytics_Parallel_CalculateCost(b *testing.B) {
	graph := generateAnalyticsGraph(200, 500, true)
	ctx := context.Background()
	req := &analyticsv1.CalculateCostRequest{Graph: graph}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := analyticsClient.CalculateCost(ctx, req)
			if err != nil {
				b.Errorf("CalculateCost failed: %v", err)
			}
		}
	})
}

func BenchmarkAnalytics_Parallel_AnalyzeFlow(b *testing.B) {
	graph := generateAnalyticsGraph(200, 500, true)
	ctx := context.Background()
	req := &analyticsv1.AnalyzeFlowRequest{
		Graph: graph,
		Options: &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        true,
			FindBottlenecks:     true,
			CalculateStatistics: true,
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := analyticsClient.AnalyzeFlow(ctx, req)
			if err != nil {
				b.Errorf("AnalyzeFlow failed: %v", err)
			}
		}
	})
}

// =============================================================================
// MEMORY BENCHMARKS
// =============================================================================

func BenchmarkAnalytics_Memory_CalculateCost(b *testing.B) {
	graph := generateAnalyticsGraph(500, 1500, true)
	ctx := context.Background()
	req := &analyticsv1.CalculateCostRequest{Graph: graph}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyticsClient.CalculateCost(ctx, req)
	}
}

func BenchmarkAnalytics_Memory_AnalyzeFlow(b *testing.B) {
	graph := generateAnalyticsGraph(500, 1500, true)
	ctx := context.Background()
	req := &analyticsv1.AnalyzeFlowRequest{
		Graph: graph,
		Options: &analyticsv1.AnalysisOptions{
			AnalyzeCosts:        true,
			FindBottlenecks:     true,
			CalculateStatistics: true,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyticsClient.AnalyzeFlow(ctx, req)
	}
}
