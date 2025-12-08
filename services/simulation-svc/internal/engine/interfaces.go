// services/simulation-svc/internal/engine/interfaces.go
package engine

import (
	"context"

	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/pkg/client"
)

// SolverClientInterface интерфейс для solver клиента
// Позволяет использовать как реальный клиент, так и моки в тестах
type SolverClientInterface interface {
	Solve(ctx context.Context, graph *commonv1.Graph, algorithm commonv1.Algorithm, opts interface{}) (*client.SolveResult, error)
}
