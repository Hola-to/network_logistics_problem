package clients

import (
	"context"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/pkg/config"
)

// SolverClient клиент для solver-svc
type SolverClient struct {
	conn   *grpc.ClientConn
	client optimizationv1.SolverServiceClient
}

// NewSolverClient создаёт клиент
func NewSolverClient(ctx context.Context, endpoint config.ServiceEndpoint) (*SolverClient, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &SolverClient{
		conn:   conn,
		client: optimizationv1.NewSolverServiceClient(conn),
	}, nil
}

// SolveResult результат решения
type SolveResult struct {
	Success      bool
	Result       *commonv1.FlowResult
	SolvedGraph  *commonv1.Graph
	Metrics      *optimizationv1.SolveMetrics
	ErrorMessage string
}

// Solve решает задачу потока
func (c *SolverClient) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts *optimizationv1.SolveOptions) (*SolveResult, error) {
	resp, err := c.client.Solve(ctx, &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: algorithm,
		Options:   opts,
	})
	if err != nil {
		return nil, err
	}

	return &SolveResult{
		Success:      resp.Success,
		Result:       resp.Result,
		SolvedGraph:  resp.SolvedGraph,
		Metrics:      resp.Metrics,
		ErrorMessage: resp.ErrorMessage,
	}, nil
}

// SolveStream решает с потоковой передачей прогресса
func (c *SolverClient) SolveStream(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts *optimizationv1.SolveOptions) (<-chan *optimizationv1.SolveProgress, <-chan error) {
	progressCh := make(chan *optimizationv1.SolveProgress, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(progressCh)
		defer close(errCh)

		stream, err := c.client.SolveStream(ctx, &optimizationv1.SolveRequestForBigGraphs{
			Graph:     graph,
			Algorithm: algorithm,
			Options:   opts,
		})
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

// GetAlgorithms возвращает список алгоритмов
func (c *SolverClient) GetAlgorithms(ctx context.Context) ([]*optimizationv1.AlgorithmInfo, error) {
	resp, err := c.client.GetAlgorithms(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return resp.Algorithms, nil
}

// Raw возвращает сырой gRPC клиент
func (c *SolverClient) Raw() optimizationv1.SolverServiceClient {
	return c.client
}
