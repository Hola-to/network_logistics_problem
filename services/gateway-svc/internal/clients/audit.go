package clients

import (
	"context"

	"google.golang.org/grpc"

	auditv1 "logistics/gen/go/logistics/audit/v1"
	"logistics/pkg/config"
)

// AuditClient клиент для audit-svc
type AuditClient struct {
	conn   *grpc.ClientConn
	client auditv1.AuditServiceClient
}

// NewAuditClient создаёт клиент
func NewAuditClient(ctx context.Context, endpoint config.ServiceEndpoint) (*AuditClient, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &AuditClient{
		conn:   conn,
		client: auditv1.NewAuditServiceClient(conn),
	}, nil
}

// LogEvent логирует событие
func (c *AuditClient) LogEvent(ctx context.Context, entry *auditv1.AuditEntry) (*auditv1.LogEventResponse, error) {
	return c.client.LogEvent(ctx, &auditv1.LogEventRequest{Entry: entry})
}

// LogEventBatch логирует пачку событий
func (c *AuditClient) LogEventBatch(ctx context.Context, entries []*auditv1.AuditEntry) (*auditv1.LogEventBatchResponse, error) {
	return c.client.LogEventBatch(ctx, &auditv1.LogEventBatchRequest{Entries: entries})
}

// GetAuditLogs получает логи
func (c *AuditClient) GetAuditLogs(ctx context.Context, req *auditv1.GetAuditLogsRequest) (*auditv1.GetAuditLogsResponse, error) {
	return c.client.GetAuditLogs(ctx, req)
}

// GetResourceHistory история ресурса
func (c *AuditClient) GetResourceHistory(ctx context.Context, resourceType, resourceID string) (*auditv1.GetResourceHistoryResponse, error) {
	return c.client.GetResourceHistory(ctx, &auditv1.GetResourceHistoryRequest{
		ResourceType: resourceType,
		ResourceId:   resourceID,
	})
}

// GetUserActivity активность пользователя
func (c *AuditClient) GetUserActivity(ctx context.Context, req *auditv1.GetUserActivityRequest) (*auditv1.GetUserActivityResponse, error) {
	return c.client.GetUserActivity(ctx, req)
}

// GetAuditStats статистика аудита
func (c *AuditClient) GetAuditStats(ctx context.Context, req *auditv1.GetAuditStatsRequest) (*auditv1.GetAuditStatsResponse, error) {
	return c.client.GetAuditStats(ctx, req)
}

// Raw возвращает сырой gRPC клиент
func (c *AuditClient) Raw() auditv1.AuditServiceClient {
	return c.client
}
