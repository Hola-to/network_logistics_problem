// services/validation-svc/factory.go
package validationsvc

import (
	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/services/validation-svc/internal/service"
)

// NewBenchmarkServer создаёт экземпляр сервиса для бенчмарков.
func NewBenchmarkServer() validationv1.ValidationServiceServer {
	return service.NewValidationService("benchmark")
}
