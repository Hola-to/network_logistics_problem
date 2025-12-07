package clients

import (
	"context"
	"io"

	"google.golang.org/grpc"

	reportv1 "logistics/gen/go/logistics/report/v1"
	"logistics/pkg/config"
)

// ReportClient клиент для report-svc
type ReportClient struct {
	conn   *grpc.ClientConn
	client reportv1.ReportServiceClient
}

// NewReportClient создаёт клиент
func NewReportClient(ctx context.Context, endpoint config.ServiceEndpoint) (*ReportClient, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &ReportClient{
		conn:   conn,
		client: reportv1.NewReportServiceClient(conn),
	}, nil
}

// GenerateFlowReport генерирует отчёт по потоку
func (c *ReportClient) GenerateFlowReport(ctx context.Context, req *reportv1.GenerateFlowReportRequest) (*reportv1.GenerateFlowReportResponse, error) {
	return c.client.GenerateFlowReport(ctx, req)
}

// GenerateAnalyticsReport генерирует аналитический отчёт
func (c *ReportClient) GenerateAnalyticsReport(ctx context.Context, req *reportv1.GenerateAnalyticsReportRequest) (*reportv1.GenerateAnalyticsReportResponse, error) {
	return c.client.GenerateAnalyticsReport(ctx, req)
}

// GenerateSimulationReport генерирует отчёт по симуляции
func (c *ReportClient) GenerateSimulationReport(ctx context.Context, req *reportv1.GenerateSimulationReportRequest) (*reportv1.GenerateSimulationReportResponse, error) {
	return c.client.GenerateSimulationReport(ctx, req)
}

// GenerateSummaryReport генерирует сводный отчёт
func (c *ReportClient) GenerateSummaryReport(ctx context.Context, req *reportv1.GenerateSummaryReportRequest) (*reportv1.GenerateSummaryReportResponse, error) {
	return c.client.GenerateSummaryReport(ctx, req)
}

// GetReport получает отчёт
func (c *ReportClient) GetReport(ctx context.Context, reportID string) (*reportv1.GetReportResponse, error) {
	return c.client.GetReport(ctx, &reportv1.GetReportRequest{ReportId: reportID})
}

// DownloadReportStream скачивает отчёт потоково
func (c *ReportClient) DownloadReportStream(ctx context.Context, reportID string) (<-chan []byte, <-chan error) {
	dataCh := make(chan []byte, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(dataCh)
		defer close(errCh)

		stream, err := c.client.GenerateReportStream(ctx, &reportv1.GenerateReportStreamRequest{})
		if err != nil {
			errCh <- err
			return
		}

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				errCh <- err
				return
			}

			select {
			case dataCh <- chunk.Data:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
	}()

	return dataCh, errCh
}

// ListReports список отчётов
func (c *ReportClient) ListReports(ctx context.Context, req *reportv1.ListReportsRequest) (*reportv1.ListReportsResponse, error) {
	return c.client.ListReports(ctx, req)
}

// DeleteReport удаляет отчёт
func (c *ReportClient) DeleteReport(ctx context.Context, reportID string) (*reportv1.DeleteReportResponse, error) {
	return c.client.DeleteReport(ctx, &reportv1.DeleteReportRequest{ReportId: reportID})
}

// GetSupportedFormats возвращает поддерживаемые форматы
func (c *ReportClient) GetSupportedFormats(ctx context.Context) (*reportv1.GetSupportedFormatsResponse, error) {
	return c.client.GetSupportedFormats(ctx, &reportv1.GetSupportedFormatsRequest{})
}

// Health проверяет здоровье - ИСПРАВЛЕНО
func (c *ReportClient) Health(ctx context.Context) (*reportv1.HealthResponse, error) {
	return c.client.Health(ctx, &reportv1.HealthRequest{})
}

// Raw возвращает сырой gRPC клиент
func (c *ReportClient) Raw() reportv1.ReportServiceClient {
	return c.client
}
