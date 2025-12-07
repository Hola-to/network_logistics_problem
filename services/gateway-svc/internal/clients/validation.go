package clients

import (
	"context"

	"google.golang.org/grpc"

	commonv1 "logistics/gen/go/logistics/common/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/pkg/config"
)

// ValidationClient клиент для validation-svc
type ValidationClient struct {
	conn   *grpc.ClientConn
	client validationv1.ValidationServiceClient
}

// NewValidationClient создаёт клиент
func NewValidationClient(ctx context.Context, endpoint config.ServiceEndpoint) (*ValidationClient, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &ValidationClient{
		conn:   conn,
		client: validationv1.NewValidationServiceClient(conn),
	}, nil
}

// ValidateGraph валидирует граф
func (c *ValidationClient) ValidateGraph(ctx context.Context, graph *commonv1.Graph, level validationv1.ValidationLevel) (*validationv1.ValidateGraphResponse, error) {
	return c.client.ValidateGraph(ctx, &validationv1.ValidateGraphRequest{
		Graph:              graph,
		Level:              level,
		CheckConnectivity:  true,
		CheckBusinessRules: true,
		CheckTopology:      level == validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
	})
}

// ValidateForAlgorithm проверяет совместимость с алгоритмом
func (c *ValidationClient) ValidateForAlgorithm(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm) (*validationv1.ValidateForAlgorithmResponse, error) {
	return c.client.ValidateForAlgorithm(ctx, &validationv1.ValidateForAlgorithmRequest{
		Graph:     graph,
		Algorithm: algorithm,
	})
}

// ValidateAll выполняет полную валидацию
func (c *ValidationClient) ValidateAll(ctx context.Context, graph *commonv1.Graph, level validationv1.ValidationLevel, algorithm commonv1.Algorithm) (*validationv1.ValidateAllResponse, error) {
	return c.client.ValidateAll(ctx, &validationv1.ValidateAllRequest{
		Graph:     graph,
		Level:     level,
		Algorithm: algorithm,
	})
}

// Raw возвращает сырой gRPC клиент
func (c *ValidationClient) Raw() validationv1.ValidationServiceClient {
	return c.client
}
