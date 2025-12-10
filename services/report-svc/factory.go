// services/report-svc/factory.go
package reportsvc

import (
	"time"

	reportv1 "logistics/gen/go/logistics/report/v1"
	"logistics/services/report-svc/internal/repository"
	"logistics/services/report-svc/internal/service"
)

// NewBenchmarkServer создаёт экземпляр сервиса для бенчмарков.
func NewBenchmarkServer() reportv1.ReportServiceServer {
	cfg := service.ServiceConfig{
		Version:       "benchmark",
		DefaultTTL:    24 * time.Hour,
		SaveToStorage: false,
	}
	return service.NewReportService(cfg, nil)
}

// NewBenchmarkServerWithRepo создаёт сервис с репозиторием.
func NewBenchmarkServerWithRepo(repo repository.Repository) reportv1.ReportServiceServer {
	cfg := service.ServiceConfig{
		Version:       "benchmark",
		DefaultTTL:    24 * time.Hour,
		SaveToStorage: true,
	}
	return service.NewReportService(cfg, repo)
}
