// services/analytics-svc/factory.go
package analyticssvc

import (
	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	"logistics/services/analytics-svc/internal/service"
)

// NewBenchmarkServer создаёт экземпляр сервиса для бенчмарков.
func NewBenchmarkServer() analyticsv1.AnalyticsServiceServer {
	return service.NewAnalyticsService()
}
