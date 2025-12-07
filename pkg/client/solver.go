// pkg/client/solver.go
package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
)

// SolverClient клиент для solver-svc
type SolverClient struct {
	conn   *grpc.ClientConn
	client optimizationv1.SolverServiceClient
}

// SolverClientConfig конфигурация клиента
type SolverClientConfig struct {
	Address    string
	Timeout    time.Duration
	MaxRetries int
	EnableTLS  bool
	CertFile   string
}

// DefaultSolverClientConfig возвращает конфигурацию по умолчанию
func DefaultSolverClientConfig() *SolverClientConfig {
	return &SolverClientConfig{
		Address:    "localhost:50054",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		EnableTLS:  false,
	}
}

// NewSolverClient создаёт нового клиента
func NewSolverClient(cfg *SolverClientConfig) (*SolverClient, error) {
	if cfg == nil {
		cfg = DefaultSolverClientConfig()
	}

	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(50*1024*1024), // 50MB
			grpc.MaxCallSendMsgSize(50*1024*1024),
		),
	}

	if cfg.EnableTLS {
		// TODO: добавить TLS
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to solver service: %w", err)
	}

	return &SolverClient{
		conn:   conn,
		client: optimizationv1.NewSolverServiceClient(conn),
	}, nil
}

// Close закрывает соединение
func (c *SolverClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SolveResult результат решения
type SolveResult struct {
	MaxFlow            float64
	TotalCost          float64
	AverageUtilization float64
	SaturatedEdges     int32
	ActivePaths        int32
	Status             commonv1.FlowStatus
	ComputationTimeMs  float64
	Graph              *commonv1.Graph
	Iterations         int32
	Error              error
}

// Solve решает задачу потока
func (c *SolverClient) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts *optimizationv1.SolveOptions) (*SolveResult, error) {
	req := &optimizationv1.SolveRequest{
		Graph:     graph,
		Algorithm: algorithm,
		Options:   opts,
	}

	resp, err := c.client.Solve(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("solver request failed: %w", err)
	}

	if !resp.Success {
		return &SolveResult{
			Status: commonv1.FlowStatus_FLOW_STATUS_ERROR,
			Error:  fmt.Errorf("solver returned error: %s", resp.ErrorMessage),
		}, nil
	}

	// Вычисляем статистику
	avgUtil, saturated, activePaths := calculateFlowStats(resp.SolvedGraph)

	return &SolveResult{
		MaxFlow:            resp.Result.MaxFlow,
		TotalCost:          resp.Result.TotalCost,
		AverageUtilization: avgUtil,
		SaturatedEdges:     saturated,
		ActivePaths:        activePaths,
		Status:             resp.Result.Status,
		ComputationTimeMs:  resp.Metrics.ComputationTimeMs,
		Graph:              resp.SolvedGraph,
		Iterations:         resp.Metrics.Iterations,
	}, nil
}

// SolveWithTimeout решает с таймаутом
func (c *SolverClient) SolveWithTimeout(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, timeout time.Duration) (*SolveResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	opts := &optimizationv1.SolveOptions{
		TimeoutSeconds: timeout.Seconds(),
	}

	return c.Solve(ctx, graph, algorithm, opts)
}

// GetAlgorithms возвращает список алгоритмов
func (c *SolverClient) GetAlgorithms(ctx context.Context) ([]*optimizationv1.AlgorithmInfo, error) {
	resp, err := c.client.GetAlgorithms(ctx, nil)
	if err != nil {
		return nil, err
	}
	return resp.Algorithms, nil
}

func calculateFlowStats(g *commonv1.Graph) (avgUtilization float64, saturated int32, activePaths int32) {
	if g == nil {
		return 0, 0, 0
	}

	var totalUtil float64
	var activeEdges int

	for _, edge := range g.Edges {
		if edge.CurrentFlow > 1e-9 {
			activeEdges++
			if edge.Capacity > 0 {
				util := edge.CurrentFlow / edge.Capacity
				totalUtil += util
				if util >= 0.99 {
					saturated++
				}
			}
		}
	}

	if activeEdges > 0 {
		avgUtilization = totalUtil / float64(activeEdges)
	}

	return avgUtilization, saturated, int32(activeEdges)
}
