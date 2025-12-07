package clients

import (
	"context"
	"io"

	"google.golang.org/grpc"

	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/pkg/config"
)

// SimulationClient клиент для simulation-svc
type SimulationClient struct {
	conn   *grpc.ClientConn
	client simulationv1.SimulationServiceClient
}

// NewSimulationClient создаёт клиент
func NewSimulationClient(ctx context.Context, endpoint config.ServiceEndpoint) (*SimulationClient, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &SimulationClient{
		conn:   conn,
		client: simulationv1.NewSimulationServiceClient(conn),
	}, nil
}

// RunWhatIf запускает what-if анализ
func (c *SimulationClient) RunWhatIf(ctx context.Context, req *simulationv1.RunWhatIfRequest) (*simulationv1.RunWhatIfResponse, error) {
	return c.client.RunWhatIf(ctx, req)
}

// RunMonteCarlo запускает Monte Carlo симуляцию
func (c *SimulationClient) RunMonteCarlo(ctx context.Context, req *simulationv1.RunMonteCarloRequest) (*simulationv1.RunMonteCarloResponse, error) {
	return c.client.RunMonteCarlo(ctx, req)
}

// RunMonteCarloStream запускает Monte Carlo с потоковым прогрессом
func (c *SimulationClient) RunMonteCarloStream(ctx context.Context, req *simulationv1.RunMonteCarloRequest) (<-chan *simulationv1.MonteCarloProgress, <-chan error) {
	progressCh := make(chan *simulationv1.MonteCarloProgress, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(progressCh)
		defer close(errCh)

		stream, err := c.client.RunMonteCarloStream(ctx, req)
		if err != nil {
			errCh <- err
			return
		}

		for {
			progress, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				errCh <- err
				return
			}

			select {
			case progressCh <- progress:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
	}()

	return progressCh, errCh
}

// AnalyzeSensitivity анализ чувствительности
func (c *SimulationClient) AnalyzeSensitivity(ctx context.Context, req *simulationv1.AnalyzeSensitivityRequest) (*simulationv1.AnalyzeSensitivityResponse, error) {
	return c.client.AnalyzeSensitivity(ctx, req)
}

// AnalyzeResilience анализ устойчивости
func (c *SimulationClient) AnalyzeResilience(ctx context.Context, req *simulationv1.AnalyzeResilienceRequest) (*simulationv1.AnalyzeResilienceResponse, error) {
	return c.client.AnalyzeResilience(ctx, req)
}

// SimulateFailures симуляция отказов
func (c *SimulationClient) SimulateFailures(ctx context.Context, req *simulationv1.SimulateFailuresRequest) (*simulationv1.SimulateFailuresResponse, error) {
	return c.client.SimulateFailures(ctx, req)
}

// FindCriticalElements поиск критических элементов
func (c *SimulationClient) FindCriticalElements(ctx context.Context, req *simulationv1.FindCriticalElementsRequest) (*simulationv1.FindCriticalElementsResponse, error) {
	return c.client.FindCriticalElements(ctx, req)
}

// CompareScenarios сравнение сценариев
func (c *SimulationClient) CompareScenarios(ctx context.Context, req *simulationv1.CompareScenariosRequest) (*simulationv1.CompareScenariosResponse, error) {
	return c.client.CompareScenarios(ctx, req)
}

// GetSimulation получение симуляции
func (c *SimulationClient) GetSimulation(ctx context.Context, id string) (*simulationv1.GetSimulationResponse, error) {
	return c.client.GetSimulation(ctx, &simulationv1.GetSimulationRequest{
		SimulationId: id,
	})
}

// ListSimulations список симуляций
func (c *SimulationClient) ListSimulations(ctx context.Context, req *simulationv1.ListSimulationsRequest) (*simulationv1.ListSimulationsResponse, error) {
	return c.client.ListSimulations(ctx, req)
}

// Raw возвращает сырой gRPC клиент
func (c *SimulationClient) Raw() simulationv1.SimulationServiceClient {
	return c.client
}
