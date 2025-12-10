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

	commonv1 "logistics/gen/go/logistics/common/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
	validationsvc "logistics/services/validation-svc"
)

var (
	validationListener *bufconn.Listener
	validationClient   validationv1.ValidationServiceClient
)

func init() {
	validationListener = bufconn.Listen(bufSize)

	server := grpc.NewServer()
	svc := validationsvc.NewBenchmarkServer()
	validationv1.RegisterValidationServiceServer(server, svc)

	go func() {
		if err := server.Serve(validationListener); err != nil {
			log.Fatalf("Validation server exited with error: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return validationListener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to dial bufnet: %v", err)
	}

	validationClient = validationv1.NewValidationServiceClient(conn)
}

// =============================================================================
// GRAPH GENERATORS FOR VALIDATION
// =============================================================================

func generateValidGraph(nodes, edges int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	nodeList := make([]*commonv1.Node, nodes)
	for i := 0; i < nodes; i++ {
		nodeList[i] = &commonv1.Node{
			Id:   int64(i),
			Type: commonv1.NodeType(r.Intn(5)),
		}
	}

	edgeList := make([]*commonv1.Edge, 0, edges)
	for i := 0; i < edges && i < nodes-1; i++ {
		edgeList = append(edgeList, &commonv1.Edge{
			From:     int64(i),
			To:       int64(i + 1),
			Capacity: float64(r.Intn(100) + 10),
			Cost:     float64(r.Intn(50) + 1),
		})
	}

	// Add extra random edges
	for i := len(edgeList); i < edges; i++ {
		from := int64(r.Intn(nodes - 1))
		to := from + 1 + int64(r.Intn(nodes-int(from)-1))
		if to >= int64(nodes) {
			to = int64(nodes - 1)
		}
		edgeList = append(edgeList, &commonv1.Edge{
			From:     from,
			To:       to,
			Capacity: float64(r.Intn(100) + 10),
			Cost:     float64(r.Intn(50) + 1),
		})
	}

	return &commonv1.Graph{
		Nodes:    nodeList,
		Edges:    edgeList,
		SourceId: 0,
		SinkId:   int64(nodes - 1),
	}
}

func generateInvalidGraph(nodes int, invalidationType string) *commonv1.Graph {
	graph := generateValidGraph(nodes, nodes*2)

	switch invalidationType {
	case "disconnected":
		// Remove edges to make graph disconnected
		if len(graph.Edges) > 2 {
			graph.Edges = graph.Edges[:len(graph.Edges)/2]
		}
	case "negative_capacity":
		if len(graph.Edges) > 0 {
			graph.Edges[0].Capacity = -10
		}
	case "self_loop":
		graph.Edges = append(graph.Edges, &commonv1.Edge{
			From:     0,
			To:       0,
			Capacity: 10,
		})
	case "invalid_source":
		graph.SourceId = 9999
	case "no_edges":
		graph.Edges = nil
	}

	return graph
}

func generateGraphWithFlow(nodes, edges int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))
	graph := generateValidGraph(nodes, edges)

	for _, edge := range graph.Edges {
		edge.CurrentFlow = edge.Capacity * r.Float64()
	}

	return graph
}

// =============================================================================
// VALIDATE GRAPH BENCHMARKS
// =============================================================================

func BenchmarkValidation_ValidateGraph_Basic_Small(b *testing.B) {
	graph := generateValidGraph(50, 100)
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateGraph(ctx, req)
		if err != nil {
			b.Fatalf("ValidateGraph failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateGraph_Basic_Medium(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateGraph(ctx, req)
		if err != nil {
			b.Fatalf("ValidateGraph failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateGraph_Basic_Large(b *testing.B) {
	graph := generateValidGraph(1000, 3000)
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateGraph(ctx, req)
		if err != nil {
			b.Fatalf("ValidateGraph failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateGraph_Standard(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph:             graph,
		Level:             validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
		CheckConnectivity: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateGraph(ctx, req)
		if err != nil {
			b.Fatalf("ValidateGraph failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateGraph_Strict(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph:              graph,
		Level:              validationv1.ValidationLevel_VALIDATION_LEVEL_STRICT,
		CheckConnectivity:  true,
		CheckBusinessRules: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateGraph(ctx, req)
		if err != nil {
			b.Fatalf("ValidateGraph failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateGraph_Full(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph:              graph,
		Level:              validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
		CheckConnectivity:  true,
		CheckBusinessRules: true,
		CheckTopology:      true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateGraph(ctx, req)
		if err != nil {
			b.Fatalf("ValidateGraph failed: %v", err)
		}
	}
}

// =============================================================================
// VALIDATE FLOW BENCHMARKS
// =============================================================================

func BenchmarkValidation_ValidateFlow_Small(b *testing.B) {
	graph := generateGraphWithFlow(50, 100)
	ctx := context.Background()
	req := &validationv1.ValidateFlowRequest{
		Graph: graph,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateFlow(ctx, req)
		if err != nil {
			b.Fatalf("ValidateFlow failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateFlow_Medium(b *testing.B) {
	graph := generateGraphWithFlow(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateFlowRequest{
		Graph: graph,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateFlow(ctx, req)
		if err != nil {
			b.Fatalf("ValidateFlow failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateFlow_Large(b *testing.B) {
	graph := generateGraphWithFlow(1000, 3000)
	ctx := context.Background()
	req := &validationv1.ValidateFlowRequest{
		Graph: graph,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateFlow(ctx, req)
		if err != nil {
			b.Fatalf("ValidateFlow failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateFlow_WithExpected(b *testing.B) {
	graph := generateGraphWithFlow(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateFlowRequest{
		Graph:           graph,
		ExpectedMaxFlow: 100.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateFlow(ctx, req)
		if err != nil {
			b.Fatalf("ValidateFlow failed: %v", err)
		}
	}
}

// =============================================================================
// VALIDATE FOR ALGORITHM BENCHMARKS
// =============================================================================

func BenchmarkValidation_ValidateForAlgorithm_EdmondsKarp(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateForAlgorithmRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateForAlgorithm(ctx, req)
		if err != nil {
			b.Fatalf("ValidateForAlgorithm failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateForAlgorithm_Dinic(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateForAlgorithmRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateForAlgorithm(ctx, req)
		if err != nil {
			b.Fatalf("ValidateForAlgorithm failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateForAlgorithm_MinCost(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateForAlgorithmRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_MIN_COST,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateForAlgorithm(ctx, req)
		if err != nil {
			b.Fatalf("ValidateForAlgorithm failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateForAlgorithm_AllAlgorithms(b *testing.B) {
	graph := generateValidGraph(200, 500)
	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"EdmondsKarp", commonv1.Algorithm_ALGORITHM_EDMONDS_KARP},
		{"Dinic", commonv1.Algorithm_ALGORITHM_DINIC},
		{"PushRelabel", commonv1.Algorithm_ALGORITHM_PUSH_RELABEL},
		{"MinCost", commonv1.Algorithm_ALGORITHM_MIN_COST},
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
	}

	ctx := context.Background()

	for _, alg := range algorithms {
		b.Run(alg.name, func(b *testing.B) {
			req := &validationv1.ValidateForAlgorithmRequest{
				Graph:     graph,
				Algorithm: alg.algo,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = validationClient.ValidateForAlgorithm(ctx, req)
			}
		})
	}
}

// =============================================================================
// VALIDATE ALL BENCHMARKS
// =============================================================================

func BenchmarkValidation_ValidateAll_Basic(b *testing.B) {
	graph := generateValidGraph(100, 200)
	ctx := context.Background()
	req := &validationv1.ValidateAllRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateAll(ctx, req)
		if err != nil {
			b.Fatalf("ValidateAll failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateAll_Full(b *testing.B) {
	graph := generateValidGraph(100, 200)
	ctx := context.Background()
	req := &validationv1.ValidateAllRequest{
		Graph:     graph,
		Level:     validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateAll(ctx, req)
		if err != nil {
			b.Fatalf("ValidateAll failed: %v", err)
		}
	}
}

func BenchmarkValidation_ValidateAll_WithAlgorithm(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateAllRequest{
		Graph:     graph,
		Level:     validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.ValidateAll(ctx, req)
		if err != nil {
			b.Fatalf("ValidateAll failed: %v", err)
		}
	}
}

// =============================================================================
// INVALID GRAPH BENCHMARKS
// =============================================================================

func BenchmarkValidation_InvalidGraph_Disconnected(b *testing.B) {
	graph := generateInvalidGraph(100, "disconnected")
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph:             graph,
		Level:             validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
		CheckConnectivity: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validationClient.ValidateGraph(ctx, req)
	}
}

func BenchmarkValidation_InvalidGraph_NegativeCapacity(b *testing.B) {
	graph := generateInvalidGraph(100, "negative_capacity")
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validationClient.ValidateGraph(ctx, req)
	}
}

func BenchmarkValidation_InvalidGraph_SelfLoop(b *testing.B) {
	graph := generateInvalidGraph(100, "self_loop")
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_STRICT,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validationClient.ValidateGraph(ctx, req)
	}
}

// =============================================================================
// SCALABILITY BENCHMARKS
// =============================================================================

func BenchmarkValidation_Scalability_ValidateGraph(b *testing.B) {
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
			graph := generateValidGraph(size.nodes, size.edges)
			ctx := context.Background()
			req := &validationv1.ValidateGraphRequest{
				Graph: graph,
				Level: validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = validationClient.ValidateGraph(ctx, req)
			}
		})
	}
}

func BenchmarkValidation_Scalability_ValidateAll(b *testing.B) {
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
			graph := generateValidGraph(size.nodes, size.edges)
			ctx := context.Background()
			req := &validationv1.ValidateAllRequest{
				Graph: graph,
				Level: validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = validationClient.ValidateAll(ctx, req)
			}
		})
	}
}

// =============================================================================
// PARALLEL BENCHMARKS
// =============================================================================

func BenchmarkValidation_Parallel_ValidateGraph(b *testing.B) {
	graph := generateValidGraph(200, 500)
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := validationClient.ValidateGraph(ctx, req)
			if err != nil {
				b.Errorf("ValidateGraph failed: %v", err)
			}
		}
	})
}

func BenchmarkValidation_Parallel_ValidateAll(b *testing.B) {
	graph := generateValidGraph(100, 200)
	ctx := context.Background()
	req := &validationv1.ValidateAllRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := validationClient.ValidateAll(ctx, req)
			if err != nil {
				b.Errorf("ValidateAll failed: %v", err)
			}
		}
	})
}

// =============================================================================
// MEMORY BENCHMARKS
// =============================================================================

func BenchmarkValidation_Memory_ValidateGraph(b *testing.B) {
	graph := generateValidGraph(500, 1500)
	ctx := context.Background()
	req := &validationv1.ValidateGraphRequest{
		Graph: graph,
		Level: validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validationClient.ValidateGraph(ctx, req)
	}
}

func BenchmarkValidation_Memory_ValidateAll(b *testing.B) {
	graph := generateValidGraph(500, 1500)
	ctx := context.Background()
	req := &validationv1.ValidateAllRequest{
		Graph:     graph,
		Level:     validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validationClient.ValidateAll(ctx, req)
	}
}

// =============================================================================
// HEALTH BENCHMARK
// =============================================================================

func BenchmarkValidation_Health(b *testing.B) {
	ctx := context.Background()
	req := &validationv1.HealthRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validationClient.Health(ctx, req)
		if err != nil {
			b.Fatalf("Health failed: %v", err)
		}
	}
}
