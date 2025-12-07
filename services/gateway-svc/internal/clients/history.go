package clients

import (
	"context"

	"google.golang.org/grpc"

	historyv1 "logistics/gen/go/logistics/history/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/pkg/config"
)

// HistoryClient клиент для history-svc
type HistoryClient struct {
	conn   *grpc.ClientConn
	client historyv1.HistoryServiceClient
}

// NewHistoryClient создаёт клиент
func NewHistoryClient(ctx context.Context, endpoint config.ServiceEndpoint) (*HistoryClient, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &HistoryClient{
		conn:   conn,
		client: historyv1.NewHistoryServiceClient(conn),
	}, nil
}

// SaveCalculation сохраняет расчёт
func (c *HistoryClient) SaveCalculation(ctx context.Context, userID, name string, req *optimizationv1.SolveRequest, resp *optimizationv1.SolveResponse, tags map[string]string) (*historyv1.SaveCalculationResponse, error) {
	return c.client.SaveCalculation(ctx, &historyv1.SaveCalculationRequest{
		UserId:   userID,
		Request:  req,
		Response: resp,
		Name:     name,
		Tags:     tags,
	})
}

// GetCalculation получает расчёт
func (c *HistoryClient) GetCalculation(ctx context.Context, userID, calcID string) (*historyv1.GetCalculationResponse, error) {
	return c.client.GetCalculation(ctx, &historyv1.GetCalculationRequest{
		CalculationId: calcID,
		UserId:        userID,
	})
}

// ListCalculations список расчётов
func (c *HistoryClient) ListCalculations(ctx context.Context, req *historyv1.ListCalculationsRequest) (*historyv1.ListCalculationsResponse, error) {
	return c.client.ListCalculations(ctx, req)
}

// DeleteCalculation удаляет расчёт
func (c *HistoryClient) DeleteCalculation(ctx context.Context, userID, calcID string) (*historyv1.DeleteCalculationResponse, error) {
	return c.client.DeleteCalculation(ctx, &historyv1.DeleteCalculationRequest{
		CalculationId: calcID,
		UserId:        userID,
	})
}

// GetStatistics получает статистику
func (c *HistoryClient) GetStatistics(ctx context.Context, req *historyv1.GetStatisticsRequest) (*historyv1.GetStatisticsResponse, error) {
	return c.client.GetStatistics(ctx, req)
}

// Raw возвращает сырой gRPC клиент
func (c *HistoryClient) Raw() historyv1.HistoryServiceClient {
	return c.client
}
