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
	reportv1 "logistics/gen/go/logistics/report/v1"
	reportsvc "logistics/services/report-svc"
)

var (
	reportListener *bufconn.Listener
	reportClient   reportv1.ReportServiceClient
)

func init() {
	reportListener = bufconn.Listen(bufSize)

	server := grpc.NewServer()
	svc := reportsvc.NewBenchmarkServer()
	reportv1.RegisterReportServiceServer(server, svc)

	go func() {
		if err := server.Serve(reportListener); err != nil {
			log.Fatalf("Report server exited with error: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return reportListener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to dial bufnet: %v", err)
	}

	reportClient = reportv1.NewReportServiceClient(conn)
}

// =============================================================================
// GRAPH GENERATORS
// =============================================================================

func generateReportGraph(nodes, edges int) *commonv1.Graph {
	r := rand.New(rand.NewSource(42))

	nodeList := make([]*commonv1.Node, nodes)
	for i := 0; i < nodes; i++ {
		nodeList[i] = &commonv1.Node{
			Id:   int64(i),
			Name: fmt.Sprintf("Node %d", i),
			Type: commonv1.NodeType(r.Intn(3) + 1),
		}
	}

	edgeList := make([]*commonv1.Edge, edges)
	for i := 0; i < edges; i++ {
		from := int64(r.Intn(nodes - 1))
		to := from + 1 + int64(r.Intn(nodes-int(from)-1))
		if to >= int64(nodes) {
			to = int64(nodes - 1)
		}

		capacity := float64(r.Intn(100) + 10)
		flow := float64(r.Intn(int(capacity)))

		edgeList[i] = &commonv1.Edge{
			From:        from,
			To:          to,
			Capacity:    capacity,
			CurrentFlow: flow,
			Cost:        float64(r.Intn(10) + 1),
		}
	}

	return &commonv1.Graph{
		Nodes:    nodeList,
		Edges:    edgeList,
		SourceId: 0,
		SinkId:   int64(nodes - 1),
		Name:     "Test Graph",
	}
}

func generateFlowResult(edges int) *commonv1.FlowResult {
	r := rand.New(rand.NewSource(42))

	flowEdges := make([]*commonv1.FlowEdge, edges)
	for i := 0; i < edges; i++ {
		capacity := float64(r.Intn(100) + 10)
		flow := float64(r.Intn(int(capacity)))
		flowEdges[i] = &commonv1.FlowEdge{
			From:        int64(i),
			To:          int64(i + 1),
			Flow:        flow,
			Capacity:    capacity,
			Utilization: flow / capacity,
		}
	}

	return &commonv1.FlowResult{
		MaxFlow:           float64(r.Intn(1000) + 100),
		TotalCost:         float64(r.Intn(5000) + 500),
		Edges:             flowEdges,
		Status:            commonv1.FlowStatus_FLOW_STATUS_OPTIMAL,
		Iterations:        int32(r.Intn(100) + 10),
		ComputationTimeMs: float64(r.Intn(100)) + 0.5,
	}
}

// =============================================================================
// GENERATE FLOW REPORT BENCHMARKS
// =============================================================================

func BenchmarkReport_GenerateFlowReport_Markdown_Small(b *testing.B) {
	graph := generateReportGraph(20, 50)
	result := generateFlowResult(50)
	ctx := context.Background()

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  graph,
		Result: result,
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reportClient.GenerateFlowReport(ctx, req)
		if err != nil {
			b.Fatalf("GenerateFlowReport failed: %v", err)
		}
	}
}

func BenchmarkReport_GenerateFlowReport_Markdown_Large(b *testing.B) {
	graph := generateReportGraph(200, 600)
	result := generateFlowResult(600)
	ctx := context.Background()

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  graph,
		Result: result,
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reportClient.GenerateFlowReport(ctx, req)
		if err != nil {
			b.Fatalf("GenerateFlowReport failed: %v", err)
		}
	}
}

func BenchmarkReport_GenerateFlowReport_CSV(b *testing.B) {
	graph := generateReportGraph(100, 300)
	result := generateFlowResult(300)
	ctx := context.Background()

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  graph,
		Result: result,
		Format: reportv1.ReportFormat_REPORT_FORMAT_CSV,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reportClient.GenerateFlowReport(ctx, req)
		if err != nil {
			b.Fatalf("GenerateFlowReport failed: %v", err)
		}
	}
}

func BenchmarkReport_GenerateFlowReport_JSON(b *testing.B) {
	graph := generateReportGraph(100, 300)
	result := generateFlowResult(300)
	ctx := context.Background()

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  graph,
		Result: result,
		Format: reportv1.ReportFormat_REPORT_FORMAT_JSON,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reportClient.GenerateFlowReport(ctx, req)
		if err != nil {
			b.Fatalf("GenerateFlowReport failed: %v", err)
		}
	}
}

func BenchmarkReport_GenerateFlowReport_HTML(b *testing.B) {
	graph := generateReportGraph(100, 300)
	result := generateFlowResult(300)
	ctx := context.Background()

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  graph,
		Result: result,
		Format: reportv1.ReportFormat_REPORT_FORMAT_HTML,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reportClient.GenerateFlowReport(ctx, req)
		if err != nil {
			b.Fatalf("GenerateFlowReport failed: %v", err)
		}
	}
}

func BenchmarkReport_GenerateFlowReport_WithOptions(b *testing.B) {
	graph := generateReportGraph(100, 300)
	result := generateFlowResult(300)
	ctx := context.Background()

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  graph,
		Result: result,
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
		Options: &reportv1.ReportOptions{
			Title:               "Benchmark Report",
			Description:         "Performance test report",
			Author:              "Benchmark",
			IncludeGraphDetails: true,
			IncludeEdgeList:     true,
			IncludePathDetails:  true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reportClient.GenerateFlowReport(ctx, req)
		if err != nil {
			b.Fatalf("GenerateFlowReport failed: %v", err)
		}
	}
}

// =============================================================================
// FORMAT COMPARISON BENCHMARKS
// =============================================================================

func BenchmarkReport_FormatComparison(b *testing.B) {
	graph := generateReportGraph(100, 300)
	result := generateFlowResult(300)
	ctx := context.Background()

	formats := []struct {
		name   string
		format reportv1.ReportFormat
	}{
		{"Markdown", reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN},
		{"CSV", reportv1.ReportFormat_REPORT_FORMAT_CSV},
		{"JSON", reportv1.ReportFormat_REPORT_FORMAT_JSON},
		{"HTML", reportv1.ReportFormat_REPORT_FORMAT_HTML},
	}

	for _, f := range formats {
		b.Run(f.name, func(b *testing.B) {
			req := &reportv1.GenerateFlowReportRequest{
				Graph:  graph,
				Result: result,
				Format: f.format,
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = reportClient.GenerateFlowReport(ctx, req)
			}
		})
	}
}

// =============================================================================
// SUPPORTED FORMATS BENCHMARK
// =============================================================================

func BenchmarkReport_GetSupportedFormats(b *testing.B) {
	ctx := context.Background()
	req := &reportv1.GetSupportedFormatsRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reportClient.GetSupportedFormats(ctx, req)
		if err != nil {
			b.Fatalf("GetSupportedFormats failed: %v", err)
		}
	}
}

// =============================================================================
// HEALTH BENCHMARK
// =============================================================================

func BenchmarkReport_Health(b *testing.B) {
	ctx := context.Background()
	req := &reportv1.HealthRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reportClient.Health(ctx, req)
		if err != nil {
			b.Fatalf("Health failed: %v", err)
		}
	}
}

// =============================================================================
// PARALLEL BENCHMARKS
// =============================================================================

func BenchmarkReport_Parallel_GenerateFlowReport(b *testing.B) {
	graph := generateReportGraph(100, 300)
	result := generateFlowResult(300)
	ctx := context.Background()

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  graph,
		Result: result,
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := reportClient.GenerateFlowReport(ctx, req)
			if err != nil {
				b.Errorf("GenerateFlowReport failed: %v", err)
			}
		}
	})
}

func BenchmarkReport_Parallel_MixedFormats(b *testing.B) {
	graph := generateReportGraph(50, 150)
	result := generateFlowResult(150)
	ctx := context.Background()

	formats := []reportv1.ReportFormat{
		reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
		reportv1.ReportFormat_REPORT_FORMAT_CSV,
		reportv1.ReportFormat_REPORT_FORMAT_JSON,
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := &reportv1.GenerateFlowReportRequest{
				Graph:  graph,
				Result: result,
				Format: formats[i%len(formats)],
			}
			_, _ = reportClient.GenerateFlowReport(ctx, req)
			i++
		}
	})
}

// =============================================================================
// MEMORY BENCHMARKS
// =============================================================================

func BenchmarkReport_Memory_LargeReport(b *testing.B) {
	graph := generateReportGraph(500, 2000)
	result := generateFlowResult(2000)
	ctx := context.Background()

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  graph,
		Result: result,
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
		Options: &reportv1.ReportOptions{
			IncludeGraphDetails: true,
			IncludeEdgeList:     true,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reportClient.GenerateFlowReport(ctx, req)
	}
}

// =============================================================================
// SIZE SCALING BENCHMARKS
// =============================================================================

func BenchmarkReport_Scaling_Markdown(b *testing.B) {
	sizes := []struct {
		name  string
		nodes int
		edges int
	}{
		{"Small_20x50", 20, 50},
		{"Medium_100x300", 100, 300},
		{"Large_500x1500", 500, 1500},
	}

	ctx := context.Background()

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			graph := generateReportGraph(size.nodes, size.edges)
			result := generateFlowResult(size.edges)

			req := &reportv1.GenerateFlowReportRequest{
				Graph:  graph,
				Result: result,
				Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = reportClient.GenerateFlowReport(ctx, req)
			}
		})
	}
}
