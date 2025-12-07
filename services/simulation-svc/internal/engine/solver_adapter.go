// services/simulation-svc/internal/engine/solver_adapter.go
package engine

import (
	"context"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/pkg/client"
)

// SolverClientAdapter адаптирует *client.SolverClient к интерфейсу
type SolverClientAdapter struct {
	client *client.SolverClient
}

// NewSolverClientAdapter создаёт адаптер
func NewSolverClientAdapter(c *client.SolverClient) *SolverClientAdapter {
	return &SolverClientAdapter{client: c}
}

// Solve реализует SolverClientInterface
func (a *SolverClientAdapter) Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error) {
	var solveOpts *optimizationv1.SolveOptions
	if opts != nil {
		if o, ok := opts.(*optimizationv1.SolveOptions); ok {
			solveOpts = o
		}
	}
	return a.client.Solve(ctx, graph, algorithm, solveOpts)
}
