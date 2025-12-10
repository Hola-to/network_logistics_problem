package services_benchmark

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"

	solversvc "logistics/services/solver-svc"
)

const bufSize = 1024 * 1024

var (
	listener *bufconn.Listener
	client   optimizationv1.SolverServiceClient
)

// init инициализирует in-memory gRPC сервер для бенчмарков
func init() {
	listener = bufconn.Listen(bufSize)

	server := grpc.NewServer()
	svc := solversvc.NewBenchmarkServer()
	optimizationv1.RegisterSolverServiceServer(server, svc)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// Создаем клиент
	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to dial bufnet: %v", err)
	}

	client = optimizationv1.NewSolverServiceClient(conn)
}

// =============================================================================
// GRAPH GENERATORS
// =============================================================================

// generateGridProtoGraph создает граф-решетку NxN в формате Proto
func generateGridProtoGraph(n int) *commonv1.Graph {
	nodes := make([]*commonv1.Node, n*n)
	var edges []*commonv1.Edge

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			id := int64(i*n + j)
			nodes[id] = &commonv1.Node{Id: id}

			// Вправо
			if j < n-1 {
				edges = append(edges, &commonv1.Edge{
					From:     id,
					To:       id + 1,
					Capacity: 10.0,
					Cost:     1.0,
				})
			}
			// Вниз
			if i < n-1 {
				edges = append(edges, &commonv1.Edge{
					From:     id,
					To:       id + int64(n),
					Capacity: 10.0,
					Cost:     1.0,
				})
			}
		}
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: 0,
		SinkId:   int64(n*n - 1),
	}
}

// generateLineProtoGraph создает линейный граф
func generateLineProtoGraph(n int) *commonv1.Graph {
	nodes := make([]*commonv1.Node, n)
	edges := make([]*commonv1.Edge, n-1)

	for i := 0; i < n; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
		if i > 0 {
			edges[i-1] = &commonv1.Edge{
				From:     int64(i - 1),
				To:       int64(i),
				Capacity: 100.0,
				Cost:     1.0,
			}
		}
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: 0,
		SinkId:   int64(n - 1),
	}
}

// generateLayeredProtoGraph создает слоистый граф
func generateLayeredProtoGraph(layers, width, connectionsPerNode int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	totalNodes := layers*width + 2
	nodes := make([]*commonv1.Node, totalNodes)
	var edges []*commonv1.Edge

	source := int64(0)
	sink := int64(totalNodes - 1)

	for i := 0; i < totalNodes; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	// Source -> первый слой
	for i := 0; i < width; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     source,
			To:       int64(1 + i),
			Capacity: 100.0,
			Cost:     1.0,
		})
	}

	// Слои между собой
	for l := 0; l < layers-1; l++ {
		for i := 0; i < width; i++ {
			from := int64(1 + l*width + i)
			for c := 0; c < connectionsPerNode; c++ {
				to := int64(1 + (l+1)*width + r.Intn(width))
				edges = append(edges, &commonv1.Edge{
					From:     from,
					To:       to,
					Capacity: float64(r.Intn(50) + 10),
					Cost:     float64(r.Intn(10) + 1),
				})
			}
		}
	}

	// Последний слой -> Sink
	for i := 0; i < width; i++ {
		from := int64(1 + (layers-1)*width + i)
		edges = append(edges, &commonv1.Edge{
			From:     from,
			To:       sink,
			Capacity: 100.0,
			Cost:     1.0,
		})
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: source,
		SinkId:   sink,
	}
}

// generateDenseProtoGraph создает плотный граф
func generateDenseProtoGraph(n int, densityPercent int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	nodes := make([]*commonv1.Node, n)
	var edges []*commonv1.Edge

	for i := 0; i < n; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if r.Intn(100) < densityPercent {
				edges = append(edges, &commonv1.Edge{
					From:     int64(i),
					To:       int64(j),
					Capacity: float64(r.Intn(100) + 1),
					Cost:     float64(r.Intn(10) + 1),
				})
			}
		}
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: 0,
		SinkId:   int64(n - 1),
	}
}

// generateDiamondProtoGraph создает diamond-граф для быстрых тестов
func generateDiamondProtoGraph() *commonv1.Graph {
	return &commonv1.Graph{
		Nodes: []*commonv1.Node{
			{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4},
		},
		Edges: []*commonv1.Edge{
			{From: 1, To: 2, Capacity: 10, Cost: 1},
			{From: 1, To: 3, Capacity: 10, Cost: 1},
			{From: 2, To: 4, Capacity: 10, Cost: 1},
			{From: 3, To: 4, Capacity: 10, Cost: 1},
		},
		SourceId: 1,
		SinkId:   4,
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func solveGraph(b *testing.B, graph *commonv1.Graph, algorithm commonv1.Algorithm) {
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: algorithm,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Solve(ctx, req)
		if err != nil {
			b.Fatalf("Solve failed: %v", err)
		}
	}
}

func solveGraphWithOptions(b *testing.B, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts *optimizationv1.SolveOptions) {
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: algorithm,
		Options:   opts,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Solve(ctx, req)
		if err != nil {
			b.Fatalf("Solve failed: %v", err)
		}
	}
}

// =============================================================================
// EDMONDS-KARP BENCHMARKS
// =============================================================================

func BenchmarkClient_EdmondsKarp_Diamond(b *testing.B) {
	graph := generateDiamondProtoGraph()
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Grid_30x30(b *testing.B) {
	graph := generateGridProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Line_100(b *testing.B) {
	graph := generateLineProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Line_500(b *testing.B) {
	graph := generateLineProtoGraph(500)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Line_1000(b *testing.B) {
	graph := generateLineProtoGraph(1000)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

// =============================================================================
// DINIC BENCHMARKS
// =============================================================================

func BenchmarkClient_Dinic_Diamond(b *testing.B) {
	graph := generateDiamondProtoGraph()
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Grid_30x30(b *testing.B) {
	graph := generateGridProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Grid_50x50(b *testing.B) {
	graph := generateGridProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Grid_70x70(b *testing.B) {
	graph := generateGridProtoGraph(70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Line_1000(b *testing.B) {
	graph := generateLineProtoGraph(1000)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Layered_5x20(b *testing.B) {
	graph := generateLayeredProtoGraph(5, 20, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Layered_15x100(b *testing.B) {
	graph := generateLayeredProtoGraph(15, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Dense_50_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Dense_100_20pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

// =============================================================================
// PUSH-RELABEL BENCHMARKS
// =============================================================================

func BenchmarkClient_PushRelabel_Diamond(b *testing.B) {
	graph := generateDiamondProtoGraph()
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Grid_30x30(b *testing.B) {
	graph := generateGridProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Dense_50_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

// =============================================================================
// MIN-COST FLOW BENCHMARKS
// =============================================================================

func BenchmarkClient_MinCost_Diamond(b *testing.B) {
	graph := generateDiamondProtoGraph()
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Grid_15x15(b *testing.B) {
	graph := generateGridProtoGraph(15)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Layered_5x20(b *testing.B) {
	graph := generateLayeredProtoGraph(5, 20, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

// =============================================================================
// FORD-FULKERSON BENCHMARKS
// =============================================================================

func BenchmarkClient_FordFulkerson_Diamond(b *testing.B) {
	graph := generateDiamondProtoGraph()
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

// =============================================================================
// ALGORITHM COMPARISON BENCHMARKS
// =============================================================================

func BenchmarkClient_Compare_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"EdmondsKarp", commonv1.Algorithm_ALGORITHM_EDMONDS_KARP},
		{"Dinic", commonv1.Algorithm_ALGORITHM_DINIC},
		{"PushRelabel", commonv1.Algorithm_ALGORITHM_PUSH_RELABEL},
		{"MinCost", commonv1.Algorithm_ALGORITHM_MIN_COST},
	}

	for _, alg := range algorithms {
		b.Run(alg.name, func(b *testing.B) {
			solveGraph(b, graph, alg.algo)
		})
	}
}

func BenchmarkClient_Compare_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"EdmondsKarp", commonv1.Algorithm_ALGORITHM_EDMONDS_KARP},
		{"Dinic", commonv1.Algorithm_ALGORITHM_DINIC},
		{"PushRelabel", commonv1.Algorithm_ALGORITHM_PUSH_RELABEL},
	}

	for _, alg := range algorithms {
		b.Run(alg.name, func(b *testing.B) {
			solveGraph(b, graph, alg.algo)
		})
	}
}

func BenchmarkClient_Compare_Dense_50(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"EdmondsKarp", commonv1.Algorithm_ALGORITHM_EDMONDS_KARP},
		{"Dinic", commonv1.Algorithm_ALGORITHM_DINIC},
		{"PushRelabel", commonv1.Algorithm_ALGORITHM_PUSH_RELABEL},
	}

	for _, alg := range algorithms {
		b.Run(alg.name, func(b *testing.B) {
			solveGraph(b, graph, alg.algo)
		})
	}
}

// =============================================================================
// OPTIONS BENCHMARKS
// =============================================================================

func BenchmarkClient_WithOptions_ReturnPaths(b *testing.B) {
	graph := generateGridProtoGraph(20)
	opts := &optimizationv1.SolveOptions{
		ReturnPaths: true,
	}
	solveGraphWithOptions(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, opts)
}

func BenchmarkClient_WithOptions_MaxIterations(b *testing.B) {
	graph := generateGridProtoGraph(20)
	opts := &optimizationv1.SolveOptions{
		MaxIterations: 10,
	}
	solveGraphWithOptions(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, opts)
}

func BenchmarkClient_WithOptions_CustomEpsilon(b *testing.B) {
	graph := generateGridProtoGraph(20)
	opts := &optimizationv1.SolveOptions{
		Epsilon: 1e-6,
	}
	solveGraphWithOptions(b, graph, commonv1.Algorithm_ALGORITHM_DINIC, opts)
}

func BenchmarkClient_WithOptions_Timeout(b *testing.B) {
	graph := generateGridProtoGraph(20)
	opts := &optimizationv1.SolveOptions{
		TimeoutSeconds: 5.0,
	}
	solveGraphWithOptions(b, graph, commonv1.Algorithm_ALGORITHM_DINIC, opts)
}

// =============================================================================
// SCALABILITY BENCHMARKS
// =============================================================================

func BenchmarkClient_Scalability_Dinic_Grid(b *testing.B) {
	sizes := []int{5, 10, 15, 20, 25, 30}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			graph := generateGridProtoGraph(size)
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
		})
	}
}

func BenchmarkClient_Scalability_Dinic_Line(b *testing.B) {
	sizes := []int{50, 100, 200, 500, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("N%d", size), func(b *testing.B) {
			graph := generateLineProtoGraph(size)
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
		})
	}
}

func BenchmarkClient_Scalability_Layered(b *testing.B) {
	configs := []struct {
		layers int
		width  int
	}{
		{3, 10},
		{5, 20},
		{10, 30},
		{15, 50},
	}

	for _, cfg := range configs {
		b.Run(fmt.Sprintf("L%d_W%d", cfg.layers, cfg.width), func(b *testing.B) {
			graph := generateLayeredProtoGraph(cfg.layers, cfg.width, 3)
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
		})
	}
}

// =============================================================================
// MEMORY BENCHMARKS
// =============================================================================

func BenchmarkClient_Memory_Small(b *testing.B) {
	graph := generateGridProtoGraph(10)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Solve(ctx, req)
	}
}

func BenchmarkClient_Memory_Medium(b *testing.B) {
	graph := generateGridProtoGraph(30)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Solve(ctx, req)
	}
}

func BenchmarkClient_Memory_Large(b *testing.B) {
	graph := generateGridProtoGraph(50)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Solve(ctx, req)
	}
}

// =============================================================================
// PARALLEL BENCHMARKS
// =============================================================================

func BenchmarkClient_Parallel_Dinic_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
		}
	})
}

func BenchmarkClient_Parallel_EdmondsKarp_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
		}
	})
}

func BenchmarkClient_Parallel_Mixed_Algorithms(b *testing.B) {
	graphs := []*commonv1.Graph{
		generateGridProtoGraph(10),
		generateGridProtoGraph(15),
		generateLayeredProtoGraph(5, 20, 3),
	}

	algorithms := []commonv1.Algorithm{
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
	}

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := &optimizationv1.SolveRequest{
				Graph:     graphs[i%len(graphs)],
				Algorithm: algorithms[i%len(algorithms)],
			}
			_, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
			i++
		}
	})
}

// =============================================================================
// LATENCY BENCHMARKS
// =============================================================================

func BenchmarkClient_Latency_Minimal(b *testing.B) {
	// Минимальный граф для измерения накладных расходов
	graph := &commonv1.Graph{
		Nodes:    []*commonv1.Node{{Id: 1}, {Id: 2}},
		Edges:    []*commonv1.Edge{{From: 1, To: 2, Capacity: 10}},
		SourceId: 1,
		SinkId:   2,
	}

	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Solve(ctx, req)
	}
}

func BenchmarkClient_Latency_WithContext(b *testing.B) {
	graph := generateDiamondProtoGraph()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req := &optimizationv1.SolveRequest{
			Graph:     graph,
			Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		}
		_, _ = client.Solve(ctx, req)
		cancel()
	}
}

// =============================================================================
// GET ALGORITHMS BENCHMARK
// =============================================================================

func BenchmarkClient_GetAlgorithms(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetAlgorithms(ctx, nil)
		if err != nil {
			b.Fatalf("GetAlgorithms failed: %v", err)
		}
	}
}

// =============================================================================
// STREAMING BENCHMARKS
// =============================================================================

func BenchmarkClient_SolveStream_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		// Consume all messages
		for {
			msg, err := stream.Recv()
			if err != nil {
				break
			}
			if msg.Status == "completed" {
				break
			}
		}
	}
}

func BenchmarkClient_SolveStream_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		for {
			msg, err := stream.Recv()
			if err != nil {
				break
			}
			if msg.Status == "completed" {
				break
			}
		}
	}
}
