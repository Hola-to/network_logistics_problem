package handlers

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	"logistics/gen/go/logistics/gateway/v1/gatewayv1connect"
	"logistics/pkg/config"
	"logistics/services/gateway-svc/internal/clients"
)

// Константы
const (
	statusHealthy = "HEALTHY"
)

// GatewayHandler реализует gatewayv1connect.GatewayServiceHandler
type GatewayHandler struct {
	gatewayv1connect.UnimplementedGatewayServiceHandler

	clients   *clients.Manager
	config    *config.Config
	startedAt time.Time

	// Sub-handlers
	auth       *AuthHandler
	solver     *SolverHandler
	analytics  *AnalyticsHandler
	validation *ValidationHandler
	simulation *SimulationHandler
	history    *HistoryHandler
	report     *ReportHandler
	audit      *AuditHandler
}

// NewGatewayHandler создаёт handler
func NewGatewayHandler(clients *clients.Manager, cfg *config.Config) *GatewayHandler {
	h := &GatewayHandler{
		clients:   clients,
		config:    cfg,
		startedAt: time.Now(),
	}

	// Инициализируем sub-handlers
	h.auth = NewAuthHandler(clients)
	h.solver = NewSolverHandler(clients, cfg)
	h.analytics = NewAnalyticsHandler(clients)
	h.validation = NewValidationHandler(clients)
	h.simulation = NewSimulationHandler(clients)
	h.history = NewHistoryHandler(clients)
	h.report = NewReportHandler(clients)
	h.audit = NewAuditHandler(clients)

	return h
}

// ==================== Health & Info ====================

func (h *GatewayHandler) Health(
	ctx context.Context,
	_ *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.HealthResponse], error) {
	healthResults := h.clients.CheckHealth(ctx)

	services := make(map[string]*gatewayv1.ServiceHealth)
	allHealthy := true

	for name, health := range healthResults {
		services[name] = &gatewayv1.ServiceHealth{
			Name:      health.Name,
			Status:    health.Status,
			Address:   health.Address,
			LatencyMs: health.LatencyMs,
			Error:     health.Error,
			Version:   health.Version,
		}
		if health.Status != statusHealthy {
			allHealthy = false
		}
	}

	status := statusHealthy
	if !allHealthy {
		status = "DEGRADED"
	}

	return connect.NewResponse(&gatewayv1.HealthResponse{
		Status:    status,
		Timestamp: timestamppb.Now(),
		Services:  services,
	}), nil
}

func (h *GatewayHandler) ReadinessCheck(
	ctx context.Context,
	_ *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.ReadinessResponse], error) {
	healthResults := h.clients.CheckHealth(ctx)

	dependencies := make(map[string]bool)
	allReady := true

	for name, health := range healthResults {
		isHealthy := health.Status == statusHealthy
		dependencies[name] = isHealthy
		if !isHealthy {
			allReady = false
		}
	}

	return connect.NewResponse(&gatewayv1.ReadinessResponse{
		Ready:        allReady,
		Dependencies: dependencies,
	}), nil
}

func (h *GatewayHandler) Info(
	ctx context.Context,
	_ *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.InfoResponse], error) {
	return connect.NewResponse(&gatewayv1.InfoResponse{
		Name:          h.config.App.Name,
		Version:       h.config.App.Version,
		Environment:   h.config.App.Environment,
		StartedAt:     timestamppb.New(h.startedAt),
		UptimeSeconds: int64(time.Since(h.startedAt).Seconds()),
		Features: []string{
			"optimization", "validation", "analytics",
			"simulation", "history", "reports", "auth",
		},
		RateLimit: &gatewayv1.RateLimitInfo{
			Enabled:           h.config.RateLimit.Enabled,
			RequestsPerMinute: int32(h.config.RateLimit.Requests),
			BurstSize:         int32(h.config.RateLimit.BurstSize),
		},
		BuildInfo: map[string]string{
			"go_version": "1.23",
			"build_time": h.startedAt.Format(time.RFC3339),
		},
	}), nil
}

func (h *GatewayHandler) GetAlgorithms(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.AlgorithmsResponse], error) {
	return h.solver.GetAlgorithms(ctx, req)
}

// ==================== Auth ====================

func (h *GatewayHandler) Register(
	ctx context.Context,
	req *connect.Request[gatewayv1.RegisterRequest],
) (*connect.Response[gatewayv1.AuthResponse], error) {
	return h.auth.Register(ctx, req)
}

func (h *GatewayHandler) Login(
	ctx context.Context,
	req *connect.Request[gatewayv1.LoginRequest],
) (*connect.Response[gatewayv1.AuthResponse], error) {
	return h.auth.Login(ctx, req)
}

func (h *GatewayHandler) RefreshToken(
	ctx context.Context,
	req *connect.Request[gatewayv1.RefreshTokenRequest],
) (*connect.Response[gatewayv1.AuthResponse], error) {
	return h.auth.RefreshToken(ctx, req)
}

func (h *GatewayHandler) Logout(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[emptypb.Empty], error) {
	return h.auth.Logout(ctx, req)
}

func (h *GatewayHandler) GetProfile(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.UserProfile], error) {
	return h.auth.GetProfile(ctx, req)
}

func (h *GatewayHandler) ValidateToken(
	ctx context.Context,
	req *connect.Request[gatewayv1.ValidateTokenRequest],
) (*connect.Response[gatewayv1.ValidateTokenResponse], error) {
	return h.auth.ValidateToken(ctx, req)
}

// ==================== Optimization ====================

func (h *GatewayHandler) CalculateLogistics(
	ctx context.Context,
	req *connect.Request[gatewayv1.CalculateLogisticsRequest],
) (*connect.Response[gatewayv1.CalculateLogisticsResponse], error) {
	return h.solver.CalculateLogistics(ctx, req)
}

func (h *GatewayHandler) SolveGraph(
	ctx context.Context,
	req *connect.Request[gatewayv1.SolveGraphRequest],
) (*connect.Response[gatewayv1.SolveGraphResponse], error) {
	return h.solver.SolveGraph(ctx, req)
}

func (h *GatewayHandler) SolveGraphStream(
	ctx context.Context,
	req *connect.Request[gatewayv1.SolveGraphRequest],
	stream *connect.ServerStream[gatewayv1.SolveProgressEvent],
) error {
	return h.solver.SolveGraphStream(ctx, req, stream)
}

func (h *GatewayHandler) BatchSolve(
	ctx context.Context,
	req *connect.Request[gatewayv1.BatchSolveRequest],
) (*connect.Response[gatewayv1.BatchSolveResponse], error) {
	return h.solver.BatchSolve(ctx, req)
}

// ==================== Validation ====================

func (h *GatewayHandler) ValidateGraph(
	ctx context.Context,
	req *connect.Request[gatewayv1.ValidateGraphRequest],
) (*connect.Response[gatewayv1.ValidateGraphResponse], error) {
	return h.validation.ValidateGraph(ctx, req)
}

func (h *GatewayHandler) ValidateForAlgorithm(
	ctx context.Context,
	req *connect.Request[gatewayv1.ValidateForAlgorithmRequest],
) (*connect.Response[gatewayv1.ValidateForAlgorithmResponse], error) {
	return h.validation.ValidateForAlgorithm(ctx, req)
}

// ==================== Analytics ====================

func (h *GatewayHandler) AnalyzeGraph(
	ctx context.Context,
	req *connect.Request[gatewayv1.AnalyzeGraphRequest],
) (*connect.Response[gatewayv1.AnalyzeGraphResponse], error) {
	return h.analytics.AnalyzeGraph(ctx, req)
}

func (h *GatewayHandler) CalculateCost(
	ctx context.Context,
	req *connect.Request[gatewayv1.CalculateCostRequest],
) (*connect.Response[gatewayv1.CalculateCostResponse], error) {
	return h.analytics.CalculateCost(ctx, req)
}

func (h *GatewayHandler) GetBottlenecks(
	ctx context.Context,
	req *connect.Request[gatewayv1.BottlenecksRequest],
) (*connect.Response[gatewayv1.BottlenecksResponse], error) {
	return h.analytics.GetBottlenecks(ctx, req)
}

func (h *GatewayHandler) CompareScenarios(
	ctx context.Context,
	req *connect.Request[gatewayv1.CompareScenariosRequest],
) (*connect.Response[gatewayv1.CompareScenariosResponse], error) {
	return h.analytics.CompareScenarios(ctx, req)
}

// ==================== Simulation ====================

func (h *GatewayHandler) RunWhatIf(
	ctx context.Context,
	req *connect.Request[gatewayv1.WhatIfRequest],
) (*connect.Response[gatewayv1.WhatIfResponse], error) {
	return h.simulation.RunWhatIf(ctx, req)
}

func (h *GatewayHandler) RunMonteCarlo(
	ctx context.Context,
	req *connect.Request[gatewayv1.MonteCarloRequest],
) (*connect.Response[gatewayv1.MonteCarloResponse], error) {
	return h.simulation.RunMonteCarlo(ctx, req)
}

func (h *GatewayHandler) RunMonteCarloStream(
	ctx context.Context,
	req *connect.Request[gatewayv1.MonteCarloRequest],
	stream *connect.ServerStream[gatewayv1.MonteCarloProgressEvent],
) error {
	return h.simulation.RunMonteCarloStream(ctx, req, stream)
}

func (h *GatewayHandler) AnalyzeSensitivity(
	ctx context.Context,
	req *connect.Request[gatewayv1.SensitivityRequest],
) (*connect.Response[gatewayv1.SensitivityResponse], error) {
	return h.simulation.AnalyzeSensitivity(ctx, req)
}

func (h *GatewayHandler) AnalyzeResilience(
	ctx context.Context,
	req *connect.Request[gatewayv1.ResilienceRequest],
) (*connect.Response[gatewayv1.ResilienceResponse], error) {
	return h.simulation.AnalyzeResilience(ctx, req)
}

func (h *GatewayHandler) SimulateFailures(
	ctx context.Context,
	req *connect.Request[gatewayv1.FailureSimulationRequest],
) (*connect.Response[gatewayv1.FailureSimulationResponse], error) {
	return h.simulation.SimulateFailures(ctx, req)
}

func (h *GatewayHandler) FindCriticalElements(
	ctx context.Context,
	req *connect.Request[gatewayv1.CriticalElementsRequest],
) (*connect.Response[gatewayv1.CriticalElementsResponse], error) {
	return h.simulation.FindCriticalElements(ctx, req)
}

func (h *GatewayHandler) GetSimulation(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetSimulationRequest],
) (*connect.Response[gatewayv1.SimulationRecord], error) {
	return h.simulation.GetSimulation(ctx, req)
}

func (h *GatewayHandler) ListSimulations(
	ctx context.Context,
	req *connect.Request[gatewayv1.ListSimulationsRequest],
) (*connect.Response[gatewayv1.ListSimulationsResponse], error) {
	return h.simulation.ListSimulations(ctx, req)
}

func (h *GatewayHandler) DeleteSimulation(
	ctx context.Context,
	req *connect.Request[gatewayv1.DeleteSimulationRequest],
) (*connect.Response[emptypb.Empty], error) {
	return h.simulation.DeleteSimulation(ctx, req)
}

// ==================== History ====================

func (h *GatewayHandler) SaveCalculation(
	ctx context.Context,
	req *connect.Request[gatewayv1.SaveCalculationRequest],
) (*connect.Response[gatewayv1.SaveCalculationResponse], error) {
	return h.history.SaveCalculation(ctx, req)
}

func (h *GatewayHandler) GetCalculation(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetCalculationRequest],
) (*connect.Response[gatewayv1.CalculationRecord], error) {
	return h.history.GetCalculation(ctx, req)
}

func (h *GatewayHandler) ListCalculations(
	ctx context.Context,
	req *connect.Request[gatewayv1.ListCalculationsRequest],
) (*connect.Response[gatewayv1.ListCalculationsResponse], error) {
	return h.history.ListCalculations(ctx, req)
}

func (h *GatewayHandler) DeleteCalculation(
	ctx context.Context,
	req *connect.Request[gatewayv1.DeleteCalculationRequest],
) (*connect.Response[emptypb.Empty], error) {
	return h.history.DeleteCalculation(ctx, req)
}

func (h *GatewayHandler) GetStatistics(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetStatisticsRequest],
) (*connect.Response[gatewayv1.StatisticsResponse], error) {
	return h.history.GetStatistics(ctx, req)
}

// ==================== Reports ====================

func (h *GatewayHandler) GenerateReport(
	ctx context.Context,
	req *connect.Request[gatewayv1.GenerateReportRequest],
) (*connect.Response[gatewayv1.GenerateReportResponse], error) {
	return h.report.GenerateReport(ctx, req)
}

func (h *GatewayHandler) GetReport(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetReportRequest],
) (*connect.Response[gatewayv1.ReportRecord], error) {
	return h.report.GetReport(ctx, req)
}

func (h *GatewayHandler) DownloadReport(
	ctx context.Context,
	req *connect.Request[gatewayv1.DownloadReportRequest],
	stream *connect.ServerStream[gatewayv1.ReportChunk],
) error {
	return h.report.DownloadReport(ctx, req, stream)
}

func (h *GatewayHandler) ListReports(
	ctx context.Context,
	req *connect.Request[gatewayv1.ListReportsRequest],
) (*connect.Response[gatewayv1.ListReportsResponse], error) {
	return h.report.ListReports(ctx, req)
}

func (h *GatewayHandler) DeleteReport(
	ctx context.Context,
	req *connect.Request[gatewayv1.DeleteReportRequest],
) (*connect.Response[emptypb.Empty], error) {
	return h.report.DeleteReport(ctx, req)
}

func (h *GatewayHandler) GetReportFormats(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.ReportFormatsResponse], error) {
	return h.report.GetReportFormats(ctx, req)
}

// ==================== Audit ====================

func (h *GatewayHandler) GetAuditLogs(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetAuditLogsRequest],
) (*connect.Response[gatewayv1.AuditLogsResponse], error) {
	return h.audit.GetAuditLogs(ctx, req)
}

func (h *GatewayHandler) GetUserActivity(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetUserActivityRequest],
) (*connect.Response[gatewayv1.UserActivityResponse], error) {
	return h.audit.GetUserActivity(ctx, req)
}

func (h *GatewayHandler) GetAuditStats(
	ctx context.Context,
	req *connect.Request[gatewayv1.GetAuditStatsRequest],
) (*connect.Response[gatewayv1.AuditStatsResponse], error) {
	return h.audit.GetAuditStats(ctx, req)
}
