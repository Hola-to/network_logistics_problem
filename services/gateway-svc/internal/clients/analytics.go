package clients

import (
	"context"

	"google.golang.org/grpc"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/pkg/config"
)

// AnalyticsClient клиент для analytics-svc
type AnalyticsClient struct {
	conn   *grpc.ClientConn
	client analyticsv1.AnalyticsServiceClient
}

// NewAnalyticsClient создаёт клиент
func NewAnalyticsClient(ctx context.Context, endpoint config.ServiceEndpoint) (*AnalyticsClient, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &AnalyticsClient{
		conn:   conn,
		client: analyticsv1.NewAnalyticsServiceClient(conn),
	}, nil
}

// CalculateCost рассчитывает стоимость
func (c *AnalyticsClient) CalculateCost(ctx context.Context, graph *commonv1.Graph, opts *analyticsv1.CostOptions) (*analyticsv1.CalculateCostResponse, error) {
	return c.client.CalculateCost(ctx, &analyticsv1.CalculateCostRequest{
		Graph:   graph,
		Options: opts,
	})
}

// FindBottlenecks находит узкие места
func (c *AnalyticsClient) FindBottlenecks(ctx context.Context, graph *commonv1.Graph, threshold float64, topN int32) (*analyticsv1.FindBottlenecksResponse, error) {
	return c.client.FindBottlenecks(ctx, &analyticsv1.FindBottlenecksRequest{
		Graph:                graph,
		UtilizationThreshold: threshold,
		TopN:                 topN,
	})
}

// AnalyzeFlow выполняет полный анализ потока
func (c *AnalyticsClient) AnalyzeFlow(ctx context.Context, graph *commonv1.Graph, opts *analyticsv1.AnalysisOptions) (*analyticsv1.AnalyzeFlowResponse, error) {
	return c.client.AnalyzeFlow(ctx, &analyticsv1.AnalyzeFlowRequest{
		Graph:   graph,
		Options: opts,
	})
}

// CompareScenarios сравнивает сценарии
func (c *AnalyticsClient) CompareScenarios(ctx context.Context, baseline *commonv1.Graph, scenarios []*commonv1.Graph, names []string) (*analyticsv1.CompareScenariosResponse, error) {
	return c.client.CompareScenarios(ctx, &analyticsv1.CompareScenariosRequest{
		Baseline:      baseline,
		Scenarios:     scenarios,
		ScenarioNames: names,
	})
}

// Raw возвращает сырой gRPC клиент
func (c *AnalyticsClient) Raw() analyticsv1.AnalyticsServiceClient {
	return c.client
}
