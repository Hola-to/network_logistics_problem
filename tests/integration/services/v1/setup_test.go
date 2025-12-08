package v1_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	auditv1 "logistics/gen/go/logistics/audit/v1"
	authv1 "logistics/gen/go/logistics/auth/v1"
	historyv1 "logistics/gen/go/logistics/history/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/tests/integration/testutil"
)

// Service addresses (environment variables)
const (
	EnvAnalyticsAddr  = "ANALYTICS_SVC_ADDR"
	EnvAuditAddr      = "AUDIT_SVC_ADDR"
	EnvAuthAddr       = "AUTH_SVC_ADDR"
	EnvHistoryAddr    = "HISTORY_SVC_ADDR"
	EnvReportAddr     = "REPORT_SVC_ADDR"
	EnvSimulationAddr = "SIMULATION_SVC_ADDR"
	EnvSolverAddr     = "SOLVER_SVC_ADDR"
	EnvValidationAddr = "VALIDATION_SVC_ADDR"

	DefaultAnalyticsAddr  = "localhost:50053"
	DefaultAuditAddr      = "localhost:50057"
	DefaultAuthAddr       = "localhost:50055"
	DefaultHistoryAddr    = "localhost:50056"
	DefaultReportAddr     = "localhost:50059"
	DefaultSimulationAddr = "localhost:50058"
	DefaultSolverAddr     = "localhost:50054"
	DefaultValidationAddr = "localhost:50052"
)

// TestClients holds all gRPC clients for testing
type TestClients struct {
	Analytics  analyticsv1.AnalyticsServiceClient
	Audit      auditv1.AuditServiceClient
	Auth       authv1.AuthServiceClient
	History    historyv1.HistoryServiceClient
	Report     reportv1.ReportServiceClient
	Simulation simulationv1.SimulationServiceClient
	Solver     optimizationv1.SolverServiceClient
	Validation validationv1.ValidationServiceClient

	conns []*grpc.ClientConn
}

// Close closes all connections
func (tc *TestClients) Close() {
	for _, conn := range tc.conns {
		if conn != nil {
			conn.Close()
		}
	}
}

// dialService creates a gRPC connection to a service
func dialService(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("failed to dial %s: %v", addr, err)
	}

	return conn
}

// SetupAnalyticsClient creates analytics client
func SetupAnalyticsClient(t *testing.T) analyticsv1.AnalyticsServiceClient {
	t.Helper()
	addr := testutil.RequireService(t, EnvAnalyticsAddr, DefaultAnalyticsAddr)
	conn := dialService(t, addr)
	t.Cleanup(func() { conn.Close() })
	return analyticsv1.NewAnalyticsServiceClient(conn)
}

// SetupAuditClient creates audit client
func SetupAuditClient(t *testing.T) auditv1.AuditServiceClient {
	t.Helper()
	addr := testutil.RequireService(t, EnvAuditAddr, DefaultAuditAddr)
	conn := dialService(t, addr)
	t.Cleanup(func() { conn.Close() })
	return auditv1.NewAuditServiceClient(conn)
}

// SetupAuthClient creates auth client
func SetupAuthClient(t *testing.T) authv1.AuthServiceClient {
	t.Helper()
	addr := testutil.RequireService(t, EnvAuthAddr, DefaultAuthAddr)
	conn := dialService(t, addr)
	t.Cleanup(func() { conn.Close() })
	return authv1.NewAuthServiceClient(conn)
}

// SetupHistoryClient creates history client
func SetupHistoryClient(t *testing.T) historyv1.HistoryServiceClient {
	t.Helper()
	addr := testutil.RequireService(t, EnvHistoryAddr, DefaultHistoryAddr)
	conn := dialService(t, addr)
	t.Cleanup(func() { conn.Close() })
	return historyv1.NewHistoryServiceClient(conn)
}

// SetupReportClient creates report client
func SetupReportClient(t *testing.T) reportv1.ReportServiceClient {
	t.Helper()
	addr := testutil.RequireService(t, EnvReportAddr, DefaultReportAddr)
	conn := dialService(t, addr)
	t.Cleanup(func() { conn.Close() })
	return reportv1.NewReportServiceClient(conn)
}

// SetupSimulationClient creates simulation client
func SetupSimulationClient(t *testing.T) simulationv1.SimulationServiceClient {
	t.Helper()
	addr := testutil.RequireService(t, EnvSimulationAddr, DefaultSimulationAddr)
	conn := dialService(t, addr)
	t.Cleanup(func() { conn.Close() })
	return simulationv1.NewSimulationServiceClient(conn)
}

// SetupSolverClient creates solver client
func SetupSolverClient(t *testing.T) optimizationv1.SolverServiceClient {
	t.Helper()
	addr := testutil.RequireService(t, EnvSolverAddr, DefaultSolverAddr)
	conn := dialService(t, addr)
	t.Cleanup(func() { conn.Close() })
	return optimizationv1.NewSolverServiceClient(conn)
}

// SetupValidationClient creates validation client
func SetupValidationClient(t *testing.T) validationv1.ValidationServiceClient {
	t.Helper()
	addr := testutil.RequireService(t, EnvValidationAddr, DefaultValidationAddr)
	conn := dialService(t, addr)
	t.Cleanup(func() { conn.Close() })
	return validationv1.NewValidationServiceClient(conn)
}
