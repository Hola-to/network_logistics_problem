package services_benchmark

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"

	solversvc "logistics/services/solver-svc"
)

const bufSize = 1024 * 1024

var (
	listener *bufconn.Listener
	client   optimizationv1.SolverServiceClient
)

// init initializes an in-memory gRPC server for benchmarks
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

	// Create client connection
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

// generateGridProtoGraph creates an NxN grid graph in Proto format
// Grid graphs are good for testing algorithms on regular structures
func generateGridProtoGraph(n int) *commonv1.Graph {
	nodes := make([]*commonv1.Node, n*n)
	var edges []*commonv1.Edge

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			id := int64(i*n + j)
			nodes[id] = &commonv1.Node{Id: id}

			// Edge to the right
			if j < n-1 {
				edges = append(edges, &commonv1.Edge{
					From:     id,
					To:       id + 1,
					Capacity: 10.0,
					Cost:     1.0,
				})
			}
			// Edge downward
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

// generateLineProtoGraph creates a linear graph (chain)
// Linear graphs represent the simplest case with single path
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

// generateLayeredProtoGraph creates a layered graph
// Layered graphs are typical for network flow problems
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

	// Source -> first layer
	for i := 0; i < width; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     source,
			To:       int64(1 + i),
			Capacity: 100.0,
			Cost:     1.0,
		})
	}

	// Inter-layer connections
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

	// Last layer -> Sink
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

// generateDenseProtoGraph creates a dense graph with specified density percentage
// Dense graphs test algorithm performance on highly connected networks
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

// generateDiamondProtoGraph creates a diamond-shaped graph for quick tests
// Simple 4-node graph ideal for basic functionality verification
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

// generateHighCapacityProtoGraph creates a graph with high capacity values (> 10^6)
// Specifically designed for testing Capacity Scaling algorithm
func generateHighCapacityProtoGraph(n int) *commonv1.Graph {
	nodes := make([]*commonv1.Node, n)
	var edges []*commonv1.Edge

	for i := 0; i < n; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	r := rand.New(rand.NewSource(42))
	for i := 0; i < n-1; i++ {
		// Main path with high capacity
		edges = append(edges, &commonv1.Edge{
			From:     int64(i),
			To:       int64(i + 1),
			Capacity: float64(1e6 + r.Intn(1e6)),
			Cost:     float64(r.Intn(10) + 1),
		})

		// Additional edges for some nodes
		if i < n-2 && r.Intn(100) < 50 {
			edges = append(edges, &commonv1.Edge{
				From:     int64(i),
				To:       int64(i + 2),
				Capacity: float64(5e5 + r.Intn(5e5)),
				Cost:     float64(r.Intn(5) + 1),
			})
		}
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: 0,
		SinkId:   int64(n - 1),
	}
}

// generateBipartiteProtoGraph creates a bipartite graph
// Common structure for matching problems
func generateBipartiteProtoGraph(leftSize, rightSize, edgesPerLeft int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	totalNodes := leftSize + rightSize + 2
	nodes := make([]*commonv1.Node, totalNodes)
	var edges []*commonv1.Edge

	source := int64(0)
	sink := int64(totalNodes - 1)

	for i := 0; i < totalNodes; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	// Source -> left partition
	for i := 0; i < leftSize; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     source,
			To:       int64(1 + i),
			Capacity: 1.0,
			Cost:     0,
		})
	}

	// Left partition -> right partition
	for i := 0; i < leftSize; i++ {
		from := int64(1 + i)
		for e := 0; e < edgesPerLeft; e++ {
			to := int64(1 + leftSize + r.Intn(rightSize))
			edges = append(edges, &commonv1.Edge{
				From:     from,
				To:       to,
				Capacity: 1.0,
				Cost:     float64(r.Intn(10) + 1),
			})
		}
	}

	// Right partition -> Sink
	for i := 0; i < rightSize; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     int64(1 + leftSize + i),
			To:       sink,
			Capacity: 1.0,
			Cost:     0,
		})
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: source,
		SinkId:   sink,
	}
}

// generateCompleteProtoGraph creates a complete graph (every node connected to every other)
// Ideal for testing Push-Relabel on highly connected graphs
func generateCompleteProtoGraph(n int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	nodes := make([]*commonv1.Node, n)
	var edges []*commonv1.Edge

	for i := 0; i < n; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	// All edges from lower index to higher index (directed complete graph)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			edges = append(edges, &commonv1.Edge{
				From:     int64(i),
				To:       int64(j),
				Capacity: float64(r.Intn(100) + 10),
				Cost:     float64(r.Intn(10) + 1),
			})
		}
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: 0,
		SinkId:   int64(n - 1),
	}
}

// generateTournamentProtoGraph creates a tournament graph
// (exactly one edge between each pair of nodes in random direction)
func generateTournamentProtoGraph(n int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	nodes := make([]*commonv1.Node, n)
	var edges []*commonv1.Edge

	for i := 0; i < n; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			from, to := int64(i), int64(j)
			if r.Intn(2) == 0 {
				from, to = to, from
			}
			edges = append(edges, &commonv1.Edge{
				From:     from,
				To:       to,
				Capacity: float64(r.Intn(100) + 10),
				Cost:     float64(r.Intn(10) + 1),
			})
		}
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: 0,
		SinkId:   int64(n - 1),
	}
}

// generateUnitCapacityProtoGraph creates a graph with unit capacities
// Ideal for Dinic (O(E√V)) and bipartite matching problems
func generateUnitCapacityProtoGraph(layers, width int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	totalNodes := layers*width + 2
	nodes := make([]*commonv1.Node, totalNodes)
	var edges []*commonv1.Edge

	source := int64(0)
	sink := int64(totalNodes - 1)

	for i := 0; i < totalNodes; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	// Source -> first layer
	for i := 0; i < width; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     source,
			To:       int64(1 + i),
			Capacity: 1.0, // Unit capacity
			Cost:     1.0,
		})
	}

	// Inter-layer connections
	for l := 0; l < layers-1; l++ {
		for i := 0; i < width; i++ {
			from := int64(1 + l*width + i)
			// Each node connects to 2-3 nodes in next layer
			connections := 2 + r.Intn(2)
			for c := 0; c < connections; c++ {
				to := int64(1 + (l+1)*width + r.Intn(width))
				edges = append(edges, &commonv1.Edge{
					From:     from,
					To:       to,
					Capacity: 1.0, // Unit capacity
					Cost:     float64(r.Intn(10) + 1),
				})
			}
		}
	}

	// Last layer -> Sink
	for i := 0; i < width; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     int64(1 + (layers-1)*width + i),
			To:       sink,
			Capacity: 1.0,
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

// generateVeryDenseProtoGraph creates a very dense graph (70-90% edges)
// Tests algorithm behavior on almost complete graphs
func generateVeryDenseProtoGraph(n int, densityPercent int) *commonv1.Graph {
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
				// Add reverse edge for strong connectivity
				if r.Intn(100) < densityPercent/2 {
					edges = append(edges, &commonv1.Edge{
						From:     int64(j),
						To:       int64(i),
						Capacity: float64(r.Intn(100) + 1),
						Cost:     float64(r.Intn(10) + 1),
					})
				}
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

// generateMultiSourceSinkProtoGraph creates a graph with virtual source/sink
// connected to multiple real sources/sinks
func generateMultiSourceSinkProtoGraph(sources, sinks, middleNodes int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	// 0 = super source, 1..sources = sources, sources+1..sources+middleNodes = middle
	// sources+middleNodes+1..sources+middleNodes+sinks = sinks, last = super sink
	totalNodes := 2 + sources + middleNodes + sinks
	nodes := make([]*commonv1.Node, totalNodes)
	var edges []*commonv1.Edge

	superSource := int64(0)
	superSink := int64(totalNodes - 1)

	for i := 0; i < totalNodes; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	// Super source -> sources
	for i := 0; i < sources; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     superSource,
			To:       int64(1 + i),
			Capacity: float64(100 + r.Intn(100)),
			Cost:     0,
		})
	}

	// Sources -> middle nodes
	for i := 0; i < sources; i++ {
		from := int64(1 + i)
		for j := 0; j < middleNodes; j++ {
			if r.Intn(100) < 50 {
				to := int64(1 + sources + j)
				edges = append(edges, &commonv1.Edge{
					From:     from,
					To:       to,
					Capacity: float64(r.Intn(50) + 10),
					Cost:     float64(r.Intn(10) + 1),
				})
			}
		}
	}

	// Middle nodes interconnections
	for i := 0; i < middleNodes; i++ {
		for j := i + 1; j < middleNodes; j++ {
			if r.Intn(100) < 30 {
				from := int64(1 + sources + i)
				to := int64(1 + sources + j)
				edges = append(edges, &commonv1.Edge{
					From:     from,
					To:       to,
					Capacity: float64(r.Intn(50) + 10),
					Cost:     float64(r.Intn(10) + 1),
				})
			}
		}
	}

	// Middle nodes -> sinks
	for i := 0; i < middleNodes; i++ {
		from := int64(1 + sources + i)
		for j := 0; j < sinks; j++ {
			if r.Intn(100) < 50 {
				to := int64(1 + sources + middleNodes + j)
				edges = append(edges, &commonv1.Edge{
					From:     from,
					To:       to,
					Capacity: float64(r.Intn(50) + 10),
					Cost:     float64(r.Intn(10) + 1),
				})
			}
		}
	}

	// Sinks -> super sink
	for i := 0; i < sinks; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     int64(1 + sources + middleNodes + i),
			To:       superSink,
			Capacity: float64(100 + r.Intn(100)),
			Cost:     0,
		})
	}

	return &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: superSource,
		SinkId:   superSink,
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// solveGraph executes benchmark for solving graph with specified algorithm
func solveGraph(b *testing.B, graph *commonv1.Graph, algorithm commonv1.Algorithm) {
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: algorithm,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Solve(ctx, req)
		if err != nil {
			b.Fatalf("Solve failed: %v", err)
		}
		if !resp.Success {
			b.Fatalf("Solve returned unsuccessful: %s", resp.ErrorMessage)
		}
	}
}

// solveGraphWithOptions executes benchmark with custom solve options
func solveGraphWithOptions(b *testing.B, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts *optimizationv1.SolveOptions) {
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: algorithm,
		Options:   opts,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Solve(ctx, req)
		if err != nil {
			b.Fatalf("Solve failed: %v", err)
		}
		if !resp.Success {
			b.Fatalf("Solve returned unsuccessful: %s", resp.ErrorMessage)
		}
	}
}

// consumeStream reads all messages from streaming response
func consumeStream(stream optimizationv1.SolverService_SolveStreamClient) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if msg.Status == "completed" {
			return nil
		}
	}
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

func BenchmarkClient_FordFulkerson_Grid_30x30(b *testing.B) {
	graph := generateGridProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Grid_50x50(b *testing.B) {
	graph := generateGridProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Grid_70x70(b *testing.B) {
	graph := generateGridProtoGraph(70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Grid_100x100(b *testing.B) {
	graph := generateGridProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Line_100(b *testing.B) {
	graph := generateLineProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Line_500(b *testing.B) {
	graph := generateLineProtoGraph(500)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Line_1000(b *testing.B) {
	graph := generateLineProtoGraph(1000)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Layered_5x20(b *testing.B) {
	graph := generateLayeredProtoGraph(5, 20, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Layered_15x100(b *testing.B) {
	graph := generateLayeredProtoGraph(15, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Dense_30_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(30, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Dense_50_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Dense_50_50pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Dense_100_20pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Dense_100_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Complete_30(b *testing.B) {
	graph := generateCompleteProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Complete_50(b *testing.B) {
	graph := generateCompleteProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Complete_75(b *testing.B) {
	graph := generateCompleteProtoGraph(75)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Bipartite_50x50(b *testing.B) {
	graph := generateBipartiteProtoGraph(50, 50, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Bipartite_100x100(b *testing.B) {
	graph := generateBipartiteProtoGraph(100, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Tournament_50(b *testing.B) {
	graph := generateTournamentProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_Tournament_100(b *testing.B) {
	graph := generateTournamentProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_UnitCapacity_10x50(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_UnitCapacity_15x100(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(15, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_VeryDense_50_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_VeryDense_75_60pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 60)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_MultiSourceSink_10_10_50(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(10, 10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_MultiSourceSink_20_20_100(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(20, 20, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_HighCapacity_50(b *testing.B) {
	graph := generateHighCapacityProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
}

func BenchmarkClient_FordFulkerson_HighCapacity_100(b *testing.B) {
	graph := generateHighCapacityProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
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

func BenchmarkClient_EdmondsKarp_Grid_50x50(b *testing.B) {
	graph := generateGridProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Grid_70x70(b *testing.B) {
	graph := generateGridProtoGraph(70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Grid_100x100(b *testing.B) {
	graph := generateGridProtoGraph(100)
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

func BenchmarkClient_EdmondsKarp_Layered_5x20(b *testing.B) {
	graph := generateLayeredProtoGraph(5, 20, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Layered_15x100(b *testing.B) {
	graph := generateLayeredProtoGraph(15, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Dense_50_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Dense_50_50pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Dense_100_20pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Dense_100_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Complete_30(b *testing.B) {
	graph := generateCompleteProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Complete_50(b *testing.B) {
	graph := generateCompleteProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Complete_75(b *testing.B) {
	graph := generateCompleteProtoGraph(75)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Bipartite_50x50(b *testing.B) {
	graph := generateBipartiteProtoGraph(50, 50, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Bipartite_100x100(b *testing.B) {
	graph := generateBipartiteProtoGraph(100, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Tournament_50(b *testing.B) {
	graph := generateTournamentProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_Tournament_100(b *testing.B) {
	graph := generateTournamentProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_UnitCapacity_10x50(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_UnitCapacity_15x100(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(15, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_UnitCapacity_20x150(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(20, 150)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_VeryDense_50_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_VeryDense_75_60pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 60)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_VeryDense_50_90pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 90)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_VeryDense_75_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_VeryDense_100_50pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(100, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_MultiSourceSink_10_10_50(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(10, 10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_MultiSourceSink_20_20_100(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(20, 20, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_HighCapacity_50(b *testing.B) {
	graph := generateHighCapacityProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_HighCapacity_100(b *testing.B) {
	graph := generateHighCapacityProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
}

func BenchmarkClient_EdmondsKarp_HighCapacity_200(b *testing.B) {
	graph := generateHighCapacityProtoGraph(200)
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

func BenchmarkClient_Dinic_Grid_100x100(b *testing.B) {
	graph := generateGridProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Line_100(b *testing.B) {
	graph := generateLineProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Line_500(b *testing.B) {
	graph := generateLineProtoGraph(500)
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

func BenchmarkClient_Dinic_Dense_50_50pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Dense_100_20pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Dense_100_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Complete_30(b *testing.B) {
	graph := generateCompleteProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Complete_50(b *testing.B) {
	graph := generateCompleteProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Complete_75(b *testing.B) {
	graph := generateCompleteProtoGraph(75)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Complete_100(b *testing.B) {
	graph := generateCompleteProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Bipartite_50x50(b *testing.B) {
	graph := generateBipartiteProtoGraph(50, 50, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Bipartite_100x100(b *testing.B) {
	graph := generateBipartiteProtoGraph(100, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Tournament_50(b *testing.B) {
	graph := generateTournamentProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Tournament_100(b *testing.B) {
	graph := generateTournamentProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_Tournament_150(b *testing.B) {
	graph := generateTournamentProtoGraph(150)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_UnitCapacity_10x50(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_UnitCapacity_15x100(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(15, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_UnitCapacity_20x150(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(20, 150)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_VeryDense_50_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_VeryDense_75_60pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 60)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_VeryDense_50_90pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 90)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_VeryDense_75_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_VeryDense_100_50pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(100, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_VeryDense_100_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(100, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_MultiSourceSink_10_10_50(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(10, 10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_MultiSourceSink_20_20_100(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(20, 20, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_HighCapacity_50(b *testing.B) {
	graph := generateHighCapacityProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_HighCapacity_100(b *testing.B) {
	graph := generateHighCapacityProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_Dinic_HighCapacity_200(b *testing.B) {
	graph := generateHighCapacityProtoGraph(200)
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

func BenchmarkClient_PushRelabel_Grid_50x50(b *testing.B) {
	graph := generateGridProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Grid_70x70(b *testing.B) {
	graph := generateGridProtoGraph(70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Grid_100x100(b *testing.B) {
	graph := generateGridProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Line_100(b *testing.B) {
	graph := generateLineProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Line_500(b *testing.B) {
	graph := generateLineProtoGraph(500)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Line_1000(b *testing.B) {
	graph := generateLineProtoGraph(1000)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Layered_5x20(b *testing.B) {
	graph := generateLayeredProtoGraph(5, 20, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Layered_15x100(b *testing.B) {
	graph := generateLayeredProtoGraph(15, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Dense_50_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Dense_50_50pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Dense_100_20pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Dense_100_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Dense_100_50pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Complete_30(b *testing.B) {
	graph := generateCompleteProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Complete_50(b *testing.B) {
	graph := generateCompleteProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Complete_75(b *testing.B) {
	graph := generateCompleteProtoGraph(75)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Complete_100(b *testing.B) {
	graph := generateCompleteProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Bipartite_50x50(b *testing.B) {
	graph := generateBipartiteProtoGraph(50, 50, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Bipartite_100x100(b *testing.B) {
	graph := generateBipartiteProtoGraph(100, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Tournament_50(b *testing.B) {
	graph := generateTournamentProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Tournament_100(b *testing.B) {
	graph := generateTournamentProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_Tournament_150(b *testing.B) {
	graph := generateTournamentProtoGraph(150)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_UnitCapacity_10x50(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_UnitCapacity_15x100(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(15, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_UnitCapacity_20x150(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(20, 150)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_VeryDense_50_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_VeryDense_75_60pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 60)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_VeryDense_50_90pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 90)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_VeryDense_75_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_VeryDense_100_50pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(100, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_VeryDense_100_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(100, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_MultiSourceSink_10_10_50(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(10, 10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_MultiSourceSink_20_20_100(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(20, 20, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_HighCapacity_50(b *testing.B) {
	graph := generateHighCapacityProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_HighCapacity_100(b *testing.B) {
	graph := generateHighCapacityProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
}

func BenchmarkClient_PushRelabel_HighCapacity_200(b *testing.B) {
	graph := generateHighCapacityProtoGraph(200)
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

func BenchmarkClient_MinCost_Grid_30x30(b *testing.B) {
	graph := generateGridProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Grid_50x50(b *testing.B) {
	graph := generateGridProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Grid_70x70(b *testing.B) {
	graph := generateGridProtoGraph(70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Line_100(b *testing.B) {
	graph := generateLineProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Line_500(b *testing.B) {
	graph := generateLineProtoGraph(500)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Line_1000(b *testing.B) {
	graph := generateLineProtoGraph(1000)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Layered_5x20(b *testing.B) {
	graph := generateLayeredProtoGraph(5, 20, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Layered_10x30(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 30, 4)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Layered_15x100(b *testing.B) {
	graph := generateLayeredProtoGraph(15, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Dense_50_30pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Dense_50_50pct(b *testing.B) {
	graph := generateDenseProtoGraph(50, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Dense_100_20pct(b *testing.B) {
	graph := generateDenseProtoGraph(100, 20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Complete_20(b *testing.B) {
	graph := generateCompleteProtoGraph(20)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Complete_30(b *testing.B) {
	graph := generateCompleteProtoGraph(30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Complete_50(b *testing.B) {
	graph := generateCompleteProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Bipartite_30x30(b *testing.B) {
	graph := generateBipartiteProtoGraph(30, 30, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Bipartite_50x50(b *testing.B) {
	graph := generateBipartiteProtoGraph(50, 50, 3)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Bipartite_100x100(b *testing.B) {
	graph := generateBipartiteProtoGraph(100, 100, 5)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Tournament_50(b *testing.B) {
	graph := generateTournamentProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_Tournament_100(b *testing.B) {
	graph := generateTournamentProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_UnitCapacity_10x30(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(10, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_UnitCapacity_10x50(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_UnitCapacity_15x100(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(15, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_VeryDense_50_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 70)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_VeryDense_75_60pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 60)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_MultiSourceSink_5_5_30(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(5, 5, 30)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_MultiSourceSink_10_10_50(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(10, 10, 50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_MultiSourceSink_20_20_100(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(20, 20, 100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_HighCapacity_50(b *testing.B) {
	graph := generateHighCapacityProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_HighCapacity_100(b *testing.B) {
	graph := generateHighCapacityProtoGraph(100)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

func BenchmarkClient_MinCost_HighCapacity_200(b *testing.B) {
	graph := generateHighCapacityProtoGraph(200)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
}

// =============================================================================
// ALGORITHM COMPARISON BENCHMARKS
// Compares all algorithms on the same graph structure
// =============================================================================

func BenchmarkClient_Compare_Grid_20x20(b *testing.B) {
	graph := generateGridProtoGraph(20)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_Grid_50x50(b *testing.B) {
	graph := generateGridProtoGraph(50)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_Grid_70x70(b *testing.B) {
	graph := generateGridProtoGraph(70)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_Dense_50(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_Complete_50(b *testing.B) {
	graph := generateCompleteProtoGraph(50)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_Bipartite_50x50(b *testing.B) {
	graph := generateBipartiteProtoGraph(50, 50, 3)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_Bipartite_100x100(b *testing.B) {
	graph := generateBipartiteProtoGraph(100, 100, 5)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_VeryDense_50_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(50, 70)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_VeryDense_75_70pct(b *testing.B) {
	graph := generateVeryDenseProtoGraph(75, 70)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_Tournament_75(b *testing.B) {
	graph := generateTournamentProtoGraph(75)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_UnitCapacity_15x100(b *testing.B) {
	graph := generateUnitCapacityProtoGraph(15, 100)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_MultiSourceSink(b *testing.B) {
	graph := generateMultiSourceSinkProtoGraph(15, 15, 75)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

func BenchmarkClient_Compare_HighCapacity_100(b *testing.B) {
	graph := generateHighCapacityProtoGraph(100)

	algorithms := []struct {
		name string
		algo commonv1.Algorithm
	}{
		{"FordFulkerson", commonv1.Algorithm_ALGORITHM_FORD_FULKERSON},
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

// =============================================================================
// SCALABILITY COMPARISON BY ALGORITHM
// Tests how algorithms scale with increasing graph sizes
// =============================================================================

func BenchmarkClient_Scalability_Complete_All(b *testing.B) {
	sizes := []int{20, 30, 40, 50, 60}

	for _, size := range sizes {
		graph := generateCompleteProtoGraph(size)

		b.Run(fmt.Sprintf("N%d/FordFulkerson", size), func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
		})
		b.Run(fmt.Sprintf("N%d/EdmondsKarp", size), func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
		})
		b.Run(fmt.Sprintf("N%d/Dinic", size), func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
		})
		b.Run(fmt.Sprintf("N%d/PushRelabel", size), func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
		})
		b.Run(fmt.Sprintf("N%d/MinCost", size), func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
		})
	}
}

func BenchmarkClient_Scalability_VeryDense_All(b *testing.B) {
	configs := []struct {
		nodes   int
		density int
	}{
		{30, 70},
		{50, 70},
		{75, 70},
		{100, 50},
	}

	for _, cfg := range configs {
		graph := generateVeryDenseProtoGraph(cfg.nodes, cfg.density)
		name := fmt.Sprintf("N%d_D%dpct", cfg.nodes, cfg.density)

		b.Run(name+"/FordFulkerson", func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_FORD_FULKERSON)
		})
		b.Run(name+"/EdmondsKarp", func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP)
		})
		b.Run(name+"/Dinic", func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
		})
		b.Run(name+"/PushRelabel", func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_PUSH_RELABEL)
		})
		b.Run(name+"/MinCost", func(b *testing.B) {
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_MIN_COST)
		})
	}
}

func BenchmarkClient_Scalability_Dinic_Grid(b *testing.B) {
	sizes := []int{5, 10, 15, 20, 25, 30, 40}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dx%d", size, size), func(b *testing.B) {
			graph := generateGridProtoGraph(size)
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
		})
	}
}

func BenchmarkClient_Scalability_Dinic_Line(b *testing.B) {
	sizes := []int{50, 100, 200, 500, 1000, 2000}

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
		{20, 75},
	}

	for _, cfg := range configs {
		b.Run(fmt.Sprintf("L%d_W%d", cfg.layers, cfg.width), func(b *testing.B) {
			graph := generateLayeredProtoGraph(cfg.layers, cfg.width, 3)
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
		})
	}
}

func BenchmarkClient_Scalability_Dense(b *testing.B) {
	configs := []struct {
		nodes   int
		density int
	}{
		{30, 30},
		{50, 30},
		{75, 20},
		{100, 15},
	}

	for _, cfg := range configs {
		b.Run(fmt.Sprintf("N%d_D%dpct", cfg.nodes, cfg.density), func(b *testing.B) {
			graph := generateDenseProtoGraph(cfg.nodes, cfg.density)
			solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
		})
	}
}

// =============================================================================
// OPTIONS BENCHMARKS
// Tests impact of different solve options on performance
// =============================================================================

func BenchmarkClient_WithOptions_ReturnPaths(b *testing.B) {
	graph := generateGridProtoGraph(20)
	opts := &optimizationv1.SolveOptions{
		ReturnPaths: true,
	}
	solveGraphWithOptions(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, opts)
}

func BenchmarkClient_WithOptions_NoReturnPaths(b *testing.B) {
	graph := generateGridProtoGraph(20)
	opts := &optimizationv1.SolveOptions{
		ReturnPaths: false,
	}
	solveGraphWithOptions(b, graph, commonv1.Algorithm_ALGORITHM_EDMONDS_KARP, opts)
}

func BenchmarkClient_WithOptions_MaxIterations(b *testing.B) {
	graph := generateGridProtoGraph(20)
	opts := &optimizationv1.SolveOptions{
		MaxIterations: 100,
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

func BenchmarkClient_WithOptions_Combined(b *testing.B) {
	graph := generateGridProtoGraph(25)
	opts := &optimizationv1.SolveOptions{
		ReturnPaths:    true,
		Epsilon:        1e-9,
		TimeoutSeconds: 10.0,
		MaxIterations:  1000,
	}
	solveGraphWithOptions(b, graph, commonv1.Algorithm_ALGORITHM_DINIC, opts)
}

// =============================================================================
// MEMORY BENCHMARKS
// Measures memory allocations during solve operations
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

func BenchmarkClient_Memory_WithPaths(b *testing.B) {
	graph := generateGridProtoGraph(20)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		Options: &optimizationv1.SolveOptions{
			ReturnPaths: true,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Solve(ctx, req)
	}
}

// =============================================================================
// PARALLEL BENCHMARKS
// Tests concurrent request handling performance
// =============================================================================

func BenchmarkClient_Parallel_Dinic_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
			if !resp.Success {
				b.Errorf("Solve returned unsuccessful: %s", resp.ErrorMessage)
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
			resp, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
			if !resp.Success {
				b.Errorf("Solve returned unsuccessful: %s", resp.ErrorMessage)
			}
		}
	})
}

func BenchmarkClient_Parallel_PushRelabel_Dense(b *testing.B) {
	graph := generateDenseProtoGraph(50, 30)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
			if !resp.Success {
				b.Errorf("Solve returned unsuccessful: %s", resp.ErrorMessage)
			}
		}
	})
}

func BenchmarkClient_Parallel_FordFulkerson_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
			if !resp.Success {
				b.Errorf("Solve returned unsuccessful: %s", resp.ErrorMessage)
			}
		}
	})
}

func BenchmarkClient_Parallel_MinCost_Grid_10x10(b *testing.B) {
	graph := generateGridProtoGraph(10)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_MIN_COST,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
			if !resp.Success {
				b.Errorf("Solve returned unsuccessful: %s", resp.ErrorMessage)
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
		commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
		commonv1.Algorithm_ALGORITHM_MIN_COST,
	}

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := &optimizationv1.SolveRequest{
				Graph:     graphs[i%len(graphs)],
				Algorithm: algorithms[i%len(algorithms)],
			}
			resp, err := client.Solve(ctx, req)
			if err != nil {
				b.Errorf("Solve failed: %v", err)
			}
			if !resp.Success {
				b.Errorf("Solve returned unsuccessful: %s", resp.ErrorMessage)
			}
			i++
		}
	})
}

func BenchmarkClient_Parallel_HighContention(b *testing.B) {
	// Same graph for all goroutines - maximum contention
	graph := generateGridProtoGraph(15)
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.SetParallelism(16) // High parallelism level
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.Solve(ctx, req)
		}
	})
}

// =============================================================================
// LATENCY BENCHMARKS
// Measures request/response overhead and latency
// =============================================================================

func BenchmarkClient_Latency_Minimal(b *testing.B) {
	// Minimal graph to measure overhead
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

func BenchmarkClient_Latency_WithDeadline(b *testing.B) {
	graph := generateDiamondProtoGraph()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
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
// Tests the GetAlgorithms endpoint performance
// =============================================================================

func BenchmarkClient_GetAlgorithms(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetAlgorithms(ctx, &emptypb.Empty{})
		if err != nil {
			b.Fatalf("GetAlgorithms failed: %v", err)
		}
	}
}

func BenchmarkClient_GetAlgorithms_Parallel(b *testing.B) {
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.GetAlgorithms(ctx, &emptypb.Empty{})
			if err != nil {
				b.Errorf("GetAlgorithms failed: %v", err)
			}
		}
	})
}

// =============================================================================
// STREAMING BENCHMARKS
// Tests streaming solve functionality for large graphs
// =============================================================================

func BenchmarkClient_SolveStream_Grid_20x20_FordFulkerson(b *testing.B) {
	graph := generateGridProtoGraph(20)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		if err := consumeStream(stream); err != nil {
			b.Fatalf("Stream consumption failed: %v", err)
		}
	}
}

func BenchmarkClient_SolveStream_Grid_20x20_EdmondsKarp(b *testing.B) {
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

		if err := consumeStream(stream); err != nil {
			b.Fatalf("Stream consumption failed: %v", err)
		}
	}
}

func BenchmarkClient_SolveStream_Grid_20x20_Dinic(b *testing.B) {
	graph := generateGridProtoGraph(20)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		if err := consumeStream(stream); err != nil {
			b.Fatalf("Stream consumption failed: %v", err)
		}
	}
}

func BenchmarkClient_SolveStream_Grid_20x20_PushRelabel(b *testing.B) {
	graph := generateGridProtoGraph(20)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		if err := consumeStream(stream); err != nil {
			b.Fatalf("Stream consumption failed: %v", err)
		}
	}
}

func BenchmarkClient_SolveStream_Grid_20x20_MinCost(b *testing.B) {
	graph := generateGridProtoGraph(20)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_MIN_COST,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		if err := consumeStream(stream); err != nil {
			b.Fatalf("Stream consumption failed: %v", err)
		}
	}
}

func BenchmarkClient_SolveStream_Layered_10x50(b *testing.B) {
	graph := generateLayeredProtoGraph(10, 50, 5)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		if err := consumeStream(stream); err != nil {
			b.Fatalf("Stream consumption failed: %v", err)
		}
	}
}

func BenchmarkClient_SolveStream_Large_Grid_50x50(b *testing.B) {
	graph := generateGridProtoGraph(50)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		if err := consumeStream(stream); err != nil {
			b.Fatalf("Stream consumption failed: %v", err)
		}
	}
}

func BenchmarkClient_SolveStream_WithOptions(b *testing.B) {
	graph := generateGridProtoGraph(25)
	ctx := context.Background()

	req := &optimizationv1.SolveRequestForBigGraphs{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		Options: &optimizationv1.SolveOptions{
			ReturnPaths:    true,
			TimeoutSeconds: 30.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := client.SolveStream(ctx, req)
		if err != nil {
			b.Fatalf("SolveStream failed: %v", err)
		}

		if err := consumeStream(stream); err != nil {
			b.Fatalf("Stream consumption failed: %v", err)
		}
	}
}

// =============================================================================
// RESPONSE VALIDATION BENCHMARKS
// =============================================================================

func BenchmarkClient_ValidateResponse_Small(b *testing.B) {
	graph := generateDiamondProtoGraph()
	ctx := context.Background()
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		Options: &optimizationv1.SolveOptions{
			ReturnPaths: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Solve(ctx, req)
		if err != nil {
			b.Fatalf("Solve failed: %v", err)
		}
		if !resp.Success {
			b.Fatalf("Solve returned unsuccessful")
		}
		if resp.Result == nil {
			b.Fatalf("Result is nil")
		}
		if resp.Result.MaxFlow <= 0 {
			b.Fatalf("MaxFlow should be positive")
		}
		if resp.Metrics == nil {
			b.Fatalf("Metrics is nil")
		}
	}
}

// =============================================================================
// EDGE CASE BENCHMARKS
// =============================================================================

func BenchmarkClient_EdgeCase_SinglePath(b *testing.B) {
	graph := generateLineProtoGraph(50)
	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_EdgeCase_ManyParallelPaths(b *testing.B) {
	// Граф с множеством параллельных путей от source к sink
	nodes := []*commonv1.Node{{Id: 0}, {Id: 1}}
	var edges []*commonv1.Edge

	for i := 0; i < 100; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     0,
			To:       1,
			Capacity: 10.0,
			Cost:     float64(i + 1),
		})
	}

	graph := &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: 0,
		SinkId:   1,
	}

	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}

func BenchmarkClient_EdgeCase_HighDegreeNode(b *testing.B) {
	// Звездообразный граф: центр соединен со всеми остальными
	n := 100
	nodes := make([]*commonv1.Node, n+2) // центр + периферия + source + sink
	var edges []*commonv1.Edge

	for i := 0; i < n+2; i++ {
		nodes[i] = &commonv1.Node{Id: int64(i)}
	}

	source := int64(0)
	center := int64(1)
	sink := int64(n + 1)

	// source -> center
	edges = append(edges, &commonv1.Edge{
		From:     source,
		To:       center,
		Capacity: float64(n * 10),
		Cost:     1,
	})

	// center -> all peripheral nodes -> sink
	for i := 2; i <= n; i++ {
		edges = append(edges, &commonv1.Edge{
			From:     center,
			To:       int64(i),
			Capacity: 10.0,
			Cost:     1,
		})
		edges = append(edges, &commonv1.Edge{
			From:     int64(i),
			To:       sink,
			Capacity: 10.0,
			Cost:     1,
		})
	}

	graph := &commonv1.Graph{
		Nodes:    nodes,
		Edges:    edges,
		SourceId: source,
		SinkId:   sink,
	}

	solveGraph(b, graph, commonv1.Algorithm_ALGORITHM_DINIC)
}
