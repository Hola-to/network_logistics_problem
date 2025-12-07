package clients

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"logistics/pkg/config"
	"logistics/pkg/logger"
)

// Manager управляет всеми gRPC клиентами
type Manager struct {
	mu sync.RWMutex

	auth       *AuthClient
	solver     *SolverClient
	analytics  *AnalyticsClient
	validation *ValidationClient
	simulation *SimulationClient
	history    *HistoryClient
	report     *ReportClient
	audit      *AuditClient

	connections []*grpc.ClientConn
	config      *Config
}

// Config конфигурация менеджера клиентов
type Config struct {
	Auth       config.ServiceEndpoint
	Solver     config.ServiceEndpoint
	Analytics  config.ServiceEndpoint
	Validation config.ServiceEndpoint
	Simulation config.ServiceEndpoint
	History    config.ServiceEndpoint
	Report     config.ServiceEndpoint
	Audit      config.ServiceEndpoint
}

// NewManager создаёт менеджер клиентов
func NewManager(ctx context.Context, cfg *Config) (*Manager, error) {
	m := &Manager{
		config:      cfg,
		connections: make([]*grpc.ClientConn, 0, 8),
	}

	var err error

	// Auth
	m.auth, err = NewAuthClient(ctx, cfg.Auth)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to connect to auth-svc: %w", err)
	}
	m.connections = append(m.connections, m.auth.conn)
	logger.Log.Info("Connected to auth-svc", "address", cfg.Auth.Address())

	// Solver
	m.solver, err = NewSolverClient(ctx, cfg.Solver)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to connect to solver-svc: %w", err)
	}
	m.connections = append(m.connections, m.solver.conn)
	logger.Log.Info("Connected to solver-svc", "address", cfg.Solver.Address())

	// Analytics
	m.analytics, err = NewAnalyticsClient(ctx, cfg.Analytics)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to connect to analytics-svc: %w", err)
	}
	m.connections = append(m.connections, m.analytics.conn)
	logger.Log.Info("Connected to analytics-svc", "address", cfg.Analytics.Address())

	// Validation
	m.validation, err = NewValidationClient(ctx, cfg.Validation)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to connect to validation-svc: %w", err)
	}
	m.connections = append(m.connections, m.validation.conn)
	logger.Log.Info("Connected to validation-svc", "address", cfg.Validation.Address())

	// Simulation
	m.simulation, err = NewSimulationClient(ctx, cfg.Simulation)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to connect to simulation-svc: %w", err)
	}
	m.connections = append(m.connections, m.simulation.conn)
	logger.Log.Info("Connected to simulation-svc", "address", cfg.Simulation.Address())

	// History
	m.history, err = NewHistoryClient(ctx, cfg.History)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to connect to history-svc: %w", err)
	}
	m.connections = append(m.connections, m.history.conn)
	logger.Log.Info("Connected to history-svc", "address", cfg.History.Address())

	// Report
	m.report, err = NewReportClient(ctx, cfg.Report)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to connect to report-svc: %w", err)
	}
	m.connections = append(m.connections, m.report.conn)
	logger.Log.Info("Connected to report-svc", "address", cfg.Report.Address())

	// Audit
	m.audit, err = NewAuditClient(ctx, cfg.Audit)
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("failed to connect to audit-svc: %w", err)
	}
	m.connections = append(m.connections, m.audit.conn)
	logger.Log.Info("Connected to audit-svc", "address", cfg.Audit.Address())

	return m, nil
}

// Getters
func (m *Manager) Auth() *AuthClient             { return m.auth }
func (m *Manager) Solver() *SolverClient         { return m.solver }
func (m *Manager) Analytics() *AnalyticsClient   { return m.analytics }
func (m *Manager) Validation() *ValidationClient { return m.validation }
func (m *Manager) Simulation() *SimulationClient { return m.simulation }
func (m *Manager) History() *HistoryClient       { return m.history }
func (m *Manager) Report() *ReportClient         { return m.report }
func (m *Manager) Audit() *AuditClient           { return m.audit }

// ServiceHealth информация о здоровье сервиса
type ServiceHealth struct {
	Name      string
	Address   string
	Status    string
	LatencyMs int64
	Error     string
	Version   string
}

// CheckHealth проверяет здоровье всех сервисов
func (m *Manager) CheckHealth(ctx context.Context) map[string]*ServiceHealth {
	results := make(map[string]*ServiceHealth)

	services := []struct {
		name    string
		conn    *grpc.ClientConn
		address string
	}{
		{"auth", m.auth.conn, m.config.Auth.Address()},
		{"solver", m.solver.conn, m.config.Solver.Address()},
		{"analytics", m.analytics.conn, m.config.Analytics.Address()},
		{"validation", m.validation.conn, m.config.Validation.Address()},
		{"simulation", m.simulation.conn, m.config.Simulation.Address()},
		{"history", m.history.conn, m.config.History.Address()},
		{"report", m.report.conn, m.config.Report.Address()},
		{"audit", m.audit.conn, m.config.Audit.Address()},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, svc := range services {
		wg.Add(1)
		go func(name string, conn *grpc.ClientConn, address string) {
			defer wg.Done()

			health := &ServiceHealth{
				Name:    name,
				Address: address,
			}

			start := time.Now()
			client := grpc_health_v1.NewHealthClient(conn)

			healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			resp, err := client.Check(healthCtx, &grpc_health_v1.HealthCheckRequest{})
			health.LatencyMs = time.Since(start).Milliseconds()

			if err != nil {
				health.Status = "UNHEALTHY"
				health.Error = err.Error()
			} else if resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
				health.Status = "HEALTHY"
			} else {
				health.Status = resp.Status.String()
			}

			mu.Lock()
			results[name] = health
			mu.Unlock()
		}(svc.name, svc.conn, svc.address)
	}

	wg.Wait()
	return results
}

// Close закрывает все соединения
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, conn := range m.connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

// dialOptions возвращает общие опции для gRPC соединений
func dialOptions(endpoint config.ServiceEndpoint) []grpc.DialOption {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(50*1024*1024),
			grpc.MaxCallSendMsgSize(50*1024*1024),
		),
	}
	return opts
}

// dial создаёт соединение с сервисом
func dial(_ context.Context, endpoint config.ServiceEndpoint) (*grpc.ClientConn, error) {
	return grpc.NewClient(endpoint.Address(), dialOptions(endpoint)...)
}
