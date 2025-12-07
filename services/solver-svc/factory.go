// services/solver-svc/factory.go
package solversvc

import (
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/services/solver-svc/internal/service"
)

// NewBenchmarkServer создаёт экземпляр сервиса для внешних бенчмарков.
// Он возвращает интерфейс, скрывая внутреннюю структуру реализации.
func NewBenchmarkServer() optimizationv1.SolverServiceServer {
	// Здесь мы вызываем внутренний конструктор
	return service.NewSolverService("benchmark", nil)
}
