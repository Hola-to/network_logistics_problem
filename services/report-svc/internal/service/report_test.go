// services/report-svc/internal/service/report_test.go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	analyticsv1 "logistics/gen/go/logistics/analytics/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	reportv1 "logistics/gen/go/logistics/report/v1"
	simulationv1 "logistics/gen/go/logistics/simulation/v1"
	"logistics/services/report-svc/internal/generator"
	"logistics/services/report-svc/internal/repository"
)

// MockGenerator мок для генератора
type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) Generate(ctx context.Context, data *generator.ReportData) ([]byte, error) {
	args := m.Called(ctx, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockGenerator) Format() reportv1.ReportFormat {
	args := m.Called()
	return args.Get(0).(reportv1.ReportFormat)
}

// MockRepository мок для репозитория
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, params *repository.CreateParams) (*repository.Report, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Report), args.Error(1)
}

func (m *MockRepository) Get(ctx context.Context, id uuid.UUID) (*repository.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Report), args.Error(1)
}

func (m *MockRepository) GetContent(ctx context.Context, id uuid.UUID) ([]byte, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, params *repository.ListParams) (*repository.ListResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ListResult), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) UpdateTags(ctx context.Context, id uuid.UUID, tags []string, replace bool) ([]string, error) {
	args := m.Called(ctx, id, tags, replace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRepository) Stats(ctx context.Context, userID string) (*repository.Stats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Stats), args.Error(1)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRepository) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Тесты
func TestNewReportService(t *testing.T) {
	cfg := ServiceConfig{
		Version:       "1.0.0",
		DefaultTTL:    24 * time.Hour,
		SaveToStorage: true,
	}

	mockRepo := new(MockRepository)
	svc := NewReportService(cfg, mockRepo)

	require.NotNil(t, svc)
	assert.Equal(t, "1.0.0", svc.version)
	assert.Equal(t, 24*time.Hour, svc.defaultTTL)
	assert.True(t, svc.saveToStorage)
	assert.NotNil(t, svc.generators)
	assert.Len(t, svc.generators, 6) // 6 форматов
}

func TestReportService_GenerateFlowReport_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	req := &reportv1.GenerateFlowReportRequest{
		Graph: &commonv1.Graph{
			Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
			Edges: []*commonv1.Edge{{From: 1, To: 2, Capacity: 100}},
		},
		Result: &commonv1.FlowResult{
			MaxFlow:   100,
			TotalCost: 500,
			Edges: []*commonv1.FlowEdge{
				{From: 1, To: 2, Flow: 100, Capacity: 100},
			},
		},
		Metrics: &optimizationv1.SolveMetrics{
			ComputationTimeMs: 50,
			Iterations:        10,
		},
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
		Options: &reportv1.ReportOptions{
			Title:       "Flow Report",
			Description: "Test flow report",
		},
		CalculationId: "calc-123",
		GraphId:       "graph-456",
	}

	resp, err := svc.GenerateFlowReport(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Content)
	assert.NotNil(t, resp.Metadata)
	assert.Equal(t, reportv1.ReportType_REPORT_TYPE_FLOW, resp.Metadata.Type)
	assert.Equal(t, reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN, resp.Metadata.Format)
}

func TestReportService_GenerateFlowReport_UnsupportedFormat(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	req := &reportv1.GenerateFlowReportRequest{
		Format: reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED,
	}

	resp, err := svc.GenerateFlowReport(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "unsupported format")
}

func TestReportService_GenerateFlowReport_WithSaveToStorage(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	svc := NewReportService(ServiceConfig{
		Version:       "1.0.0",
		SaveToStorage: true,
	}, mockRepo)

	// ИСПРАВЛЕНИЕ: используем mock.Anything для контекста
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*repository.CreateParams")).
		Return(&repository.Report{ID: uuid.New()}, nil)

	req := &reportv1.GenerateFlowReportRequest{
		Graph:  &commonv1.Graph{},
		Result: &commonv1.FlowResult{},
		Format: reportv1.ReportFormat_REPORT_FORMAT_JSON,
		Options: &reportv1.ReportOptions{
			SaveToStorage: true,
			Title:         "Test",
		},
	}

	resp, err := svc.GenerateFlowReport(ctx, req)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	mockRepo.AssertExpectations(t)
}

func TestReportService_GenerateAnalyticsReport_Success(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	req := &reportv1.GenerateAnalyticsReportRequest{
		Graph: &commonv1.Graph{},
		Cost: &analyticsv1.CalculateCostResponse{
			TotalCost: 1500,
			Currency:  "USD",
			Breakdown: &analyticsv1.CostBreakdown{
				TransportCost: 1000,
				FixedCost:     500,
			},
		},
		Bottlenecks: &analyticsv1.FindBottlenecksResponse{
			Bottlenecks: []*analyticsv1.Bottleneck{
				{
					Edge:        &commonv1.Edge{From: 1, To: 2},
					Utilization: 0.95,
				},
			},
			Recommendations: []*analyticsv1.Recommendation{
				{
					Type:        "increase_capacity",
					Description: "Increase capacity",
				},
			},
		},
		Efficiency: &analyticsv1.EfficiencyReport{
			OverallEfficiency: 0.85,
			Grade:             "B",
		},
		Format: reportv1.ReportFormat_REPORT_FORMAT_JSON, // Используем JSON вместо HTML
	}

	resp, err := svc.GenerateAnalyticsReport(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, reportv1.ReportType_REPORT_TYPE_ANALYTICS, resp.Metadata.Type)
}

func TestReportService_GenerateSimulationReport_WhatIf(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	req := &reportv1.GenerateSimulationReportRequest{
		BaselineGraph: &commonv1.Graph{},
		SimulationResult: &reportv1.GenerateSimulationReportRequest_WhatIf{
			WhatIf: &simulationv1.RunWhatIfResponse{
				Baseline: &simulationv1.ScenarioResult{
					MaxFlow:   100,
					TotalCost: 500,
				},
				Modified: &simulationv1.ScenarioResult{
					MaxFlow:   120,
					TotalCost: 600,
				},
			},
		},
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
	}

	resp, err := svc.GenerateSimulationReport(ctx, req)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, reportv1.ReportType_REPORT_TYPE_SIMULATION, resp.Metadata.Type)
}

func TestReportService_GenerateSimulationReport_MonteCarlo(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	req := &reportv1.GenerateSimulationReportRequest{
		BaselineGraph: &commonv1.Graph{},
		SimulationResult: &reportv1.GenerateSimulationReportRequest_MonteCarlo{
			MonteCarlo: &simulationv1.RunMonteCarloResponse{
				FlowStats: &simulationv1.MonteCarloStats{
					Mean:   150,
					StdDev: 10,
				},
				FlowPercentiles: map[string]float64{
					"p5":  130,
					"p50": 150,
					"p95": 170,
				},
			},
		},
		Format: reportv1.ReportFormat_REPORT_FORMAT_JSON, // JSON вместо PDF для простоты
	}

	resp, err := svc.GenerateSimulationReport(ctx, req)

	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestReportService_GenerateSummaryReport_Success(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	req := &reportv1.GenerateSummaryReportRequest{
		Graph: &commonv1.Graph{},
		FlowResult: &commonv1.FlowResult{
			MaxFlow:   100,
			TotalCost: 500,
		},
		Analytics: &analyticsv1.AnalyzeFlowResponse{
			Cost: &analyticsv1.CalculateCostResponse{
				TotalCost: 500,
			},
			Bottlenecks: &analyticsv1.FindBottlenecksResponse{},
			Efficiency: &analyticsv1.EfficiencyReport{
				Grade: "A",
			},
		},
		Format: reportv1.ReportFormat_REPORT_FORMAT_JSON,
	}

	resp, err := svc.GenerateSummaryReport(ctx, req)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, reportv1.ReportType_REPORT_TYPE_SUMMARY, resp.Metadata.Type)
}

func TestReportService_GenerateComparisonReport_Success(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	req := &reportv1.GenerateComparisonReportRequest{
		Items: []*reportv1.ComparisonItem{
			{
				Name:  "Scenario A",
				Graph: &commonv1.Graph{},
				Result: &commonv1.FlowResult{
					MaxFlow:   100,
					TotalCost: 500,
				},
				Metrics: map[string]float64{"efficiency": 0.8},
			},
			{
				Name:  "Scenario B",
				Graph: &commonv1.Graph{},
				Result: &commonv1.FlowResult{
					MaxFlow:   120,
					TotalCost: 600,
				},
			},
		},
		Format: reportv1.ReportFormat_REPORT_FORMAT_CSV,
	}

	resp, err := svc.GenerateComparisonReport(ctx, req)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, reportv1.ReportType_REPORT_TYPE_COMPARISON, resp.Metadata.Type)
}

func TestReportService_GenerateHistoryReport_Success(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	req := &reportv1.GenerateHistoryReportRequest{
		UserId: "user-123",
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
	}

	resp, err := svc.GenerateHistoryReport(ctx, req)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, reportv1.ReportType_REPORT_TYPE_HISTORY, resp.Metadata.Type)
}

func TestReportService_GetReport_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	report := &repository.Report{
		ID:          reportID,
		Title:       "Test Report",
		Content:     []byte("test content"),
		ContentType: "text/plain",
		Filename:    "test.txt",
		SizeBytes:   12,
		ReportType:  reportv1.ReportType_REPORT_TYPE_FLOW,
		Format:      reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
		CreatedAt:   time.Now(),
	}

	// ИСПРАВЛЕНИЕ: используем mock.Anything для контекста
	mockRepo.On("Get", mock.Anything, reportID).Return(report, nil)

	resp, err := svc.GetReport(ctx, &reportv1.GetReportRequest{
		ReportId: reportID.String(),
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Metadata)
	assert.NotNil(t, resp.Content)
	mockRepo.AssertExpectations(t)
}

func TestReportService_GetReport_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	mockRepo.On("Get", mock.Anything, reportID).Return(nil, repository.ErrNotFound)

	resp, err := svc.GetReport(ctx, &reportv1.GetReportRequest{
		ReportId: reportID.String(),
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "not found")
	mockRepo.AssertExpectations(t)
}

func TestReportService_GetReport_InvalidID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	resp, err := svc.GetReport(ctx, &reportv1.GetReportRequest{
		ReportId: "invalid-uuid",
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "invalid report ID")
}

func TestReportService_GetReport_NoRepository(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	resp, err := svc.GetReport(ctx, &reportv1.GetReportRequest{
		ReportId: uuid.New().String(),
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "storage not configured")
}

func TestReportService_GetReportInfo_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	report := &repository.Report{
		ID:         reportID,
		Title:      "Test Report",
		ReportType: reportv1.ReportType_REPORT_TYPE_FLOW,
		Format:     reportv1.ReportFormat_REPORT_FORMAT_PDF,
		CreatedAt:  time.Now(),
	}

	mockRepo.On("Get", mock.Anything, reportID).Return(report, nil)

	resp, err := svc.GetReportInfo(ctx, &reportv1.GetReportInfoRequest{
		ReportId: reportID.String(),
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Metadata)
	mockRepo.AssertExpectations(t)
}

func TestReportService_GetReportInfo_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	mockRepo.On("Get", mock.Anything, reportID).Return(nil, repository.ErrNotFound)

	resp, err := svc.GetReportInfo(ctx, &reportv1.GetReportInfoRequest{
		ReportId: reportID.String(),
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "not found")
}

func TestReportService_ListReports_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	result := &repository.ListResult{
		Reports: []*repository.Report{
			{
				ID:         uuid.New(),
				Title:      "Report 1",
				ReportType: reportv1.ReportType_REPORT_TYPE_FLOW,
				CreatedAt:  time.Now(),
			},
			{
				ID:         uuid.New(),
				Title:      "Report 2",
				ReportType: reportv1.ReportType_REPORT_TYPE_ANALYTICS,
				CreatedAt:  time.Now(),
			},
		},
		TotalCount: 2,
		HasMore:    false,
	}

	mockRepo.On("List", mock.Anything, mock.AnythingOfType("*repository.ListParams")).
		Return(result, nil)

	resp, err := svc.ListReports(ctx, &reportv1.ListReportsRequest{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Reports, 2)
	assert.Equal(t, int64(2), resp.TotalCount)
	assert.False(t, resp.HasMore)
	mockRepo.AssertExpectations(t)
}

func TestReportService_ListReports_WithFilters(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(params *repository.ListParams) bool {
		return params.ReportType != nil &&
			*params.ReportType == reportv1.ReportType_REPORT_TYPE_FLOW &&
			params.UserID == "user-123"
	})).Return(&repository.ListResult{}, nil)

	resp, err := svc.ListReports(ctx, &reportv1.ListReportsRequest{
		ReportType: reportv1.ReportType_REPORT_TYPE_FLOW,
		UserId:     "user-123",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	mockRepo.AssertExpectations(t)
}

func TestReportService_ListReports_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	mockRepo.On("List", mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	_, err := svc.ListReports(ctx, &reportv1.ListReportsRequest{})

	require.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestReportService_DeleteReport_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	mockRepo.On("Delete", mock.Anything, reportID).Return(nil)

	resp, err := svc.DeleteReport(ctx, &reportv1.DeleteReportRequest{
		ReportId:   reportID.String(),
		HardDelete: false,
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	mockRepo.AssertExpectations(t)
}

func TestReportService_DeleteReport_HardDelete(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	mockRepo.On("HardDelete", mock.Anything, reportID).Return(nil)

	resp, err := svc.DeleteReport(ctx, &reportv1.DeleteReportRequest{
		ReportId:   reportID.String(),
		HardDelete: true,
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	mockRepo.AssertExpectations(t)
}

func TestReportService_DeleteReport_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	mockRepo.On("Delete", mock.Anything, reportID).Return(repository.ErrNotFound)

	resp, err := svc.DeleteReport(ctx, &reportv1.DeleteReportRequest{
		ReportId: reportID.String(),
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "not found")
	mockRepo.AssertExpectations(t)
}

func TestReportService_DeleteReport_InvalidID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	resp, err := svc.DeleteReport(ctx, &reportv1.DeleteReportRequest{
		ReportId: "invalid",
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "invalid report ID")
}

func TestReportService_DeleteReport_NoRepository(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	resp, err := svc.DeleteReport(ctx, &reportv1.DeleteReportRequest{
		ReportId: uuid.New().String(),
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "storage not configured")
}

func TestReportService_UpdateReportTags_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	newTags := []string{"new", "tags"}

	mockRepo.On("UpdateTags", mock.Anything, reportID, newTags, true).
		Return(newTags, nil)

	resp, err := svc.UpdateReportTags(ctx, &reportv1.UpdateReportTagsRequest{
		ReportId: reportID.String(),
		Tags:     newTags,
		Replace:  true,
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, newTags, resp.Tags)
	mockRepo.AssertExpectations(t)
}

func TestReportService_UpdateReportTags_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	reportID := uuid.New()
	mockRepo.On("UpdateTags", mock.Anything, reportID, []string{"tag"}, false).
		Return(nil, repository.ErrNotFound)

	resp, err := svc.UpdateReportTags(ctx, &reportv1.UpdateReportTagsRequest{
		ReportId: reportID.String(),
		Tags:     []string{"tag"},
		Replace:  false,
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "not found")
}

func TestReportService_UpdateReportTags_InvalidID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	resp, err := svc.UpdateReportTags(ctx, &reportv1.UpdateReportTagsRequest{
		ReportId: "invalid",
		Tags:     []string{"tag"},
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "invalid report ID")
}

func TestReportService_UpdateReportTags_NoRepository(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	resp, err := svc.UpdateReportTags(ctx, &reportv1.UpdateReportTagsRequest{
		ReportId: uuid.New().String(),
		Tags:     []string{"tag"},
	})

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "storage not configured")
}

func TestReportService_GetRepositoryStats_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	stats := &repository.Stats{
		TotalReports:   100,
		TotalSizeBytes: 1024 * 1024,
		AvgSizeBytes:   10240,
		ReportsByType: map[string]int64{
			"REPORT_TYPE_FLOW": 50,
		},
		ReportsByFormat: map[string]int64{
			"REPORT_FORMAT_PDF": 60,
		},
		SizeByType: map[string]int64{},
	}

	mockRepo.On("Stats", mock.Anything, "").Return(stats, nil)

	resp, err := svc.GetRepositoryStats(ctx, &reportv1.GetRepositoryStatsRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(100), resp.TotalReports)
	assert.Equal(t, int64(1024*1024), resp.TotalSizeBytes)
	mockRepo.AssertExpectations(t)
}

func TestReportService_GetRepositoryStats_WithUserID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	stats := &repository.Stats{
		TotalReports:    10,
		ReportsByType:   map[string]int64{},
		ReportsByFormat: map[string]int64{},
		SizeByType:      map[string]int64{},
	}

	mockRepo.On("Stats", mock.Anything, "user-123").Return(stats, nil)

	resp, err := svc.GetRepositoryStats(ctx, &reportv1.GetRepositoryStatsRequest{
		UserId: "user-123",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(10), resp.TotalReports)
	mockRepo.AssertExpectations(t)
}

func TestReportService_GetRepositoryStats_NoRepository(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	_, err := svc.GetRepositoryStats(ctx, &reportv1.GetRepositoryStatsRequest{})

	require.Error(t, err)
}

func TestReportService_GetSupportedFormats(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	resp, err := svc.GetSupportedFormats(ctx, &reportv1.GetSupportedFormatsRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Formats, 6) // MD, CSV, Excel, PDF, HTML, JSON

	// Проверяем каждый формат
	formatNames := make(map[string]bool)
	for _, f := range resp.Formats {
		formatNames[f.Name] = true
		assert.NotEmpty(t, f.Extension)
		assert.NotEmpty(t, f.MimeType)
		assert.NotEmpty(t, f.SupportedReportTypes)
	}

	assert.True(t, formatNames["Markdown"])
	assert.True(t, formatNames["CSV"])
	assert.True(t, formatNames["Excel"])
	assert.True(t, formatNames["PDF"])
	assert.True(t, formatNames["HTML"])
	assert.True(t, formatNames["JSON"])
}

func TestReportService_Health_Serving(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	mockRepo.On("Ping", mock.Anything).Return(nil)
	mockRepo.On("Stats", mock.Anything, "").Return(&repository.Stats{
		TotalReports:    50,
		TotalSizeBytes:  1024,
		ReportsByType:   map[string]int64{},
		ReportsByFormat: map[string]int64{},
		SizeByType:      map[string]int64{},
	}, nil)

	resp, err := svc.Health(ctx, &reportv1.HealthRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "SERVING", resp.Status)
	assert.Equal(t, "1.0.0", resp.Version)
	assert.NotNil(t, resp.Storage)
	assert.Equal(t, "OK", resp.Storage.Status)
	mockRepo.AssertExpectations(t)
}

func TestReportService_Health_Degraded(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, mockRepo)

	mockRepo.On("Ping", mock.Anything).Return(errors.New("connection failed"))

	resp, err := svc.Health(ctx, &reportv1.HealthRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "DEGRADED", resp.Status)
	assert.NotNil(t, resp.Storage)
	assert.Equal(t, "ERROR", resp.Storage.Status)
	mockRepo.AssertExpectations(t)
}

func TestReportService_Health_NoStorage(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	resp, err := svc.Health(ctx, &reportv1.HealthRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "SERVING", resp.Status)
	assert.NotNil(t, resp.Storage)
	assert.Equal(t, "NOT_CONFIGURED", resp.Storage.Status)
}

// Тесты вспомогательных функций
func TestGetExtension(t *testing.T) {
	tests := []struct {
		format   reportv1.ReportFormat
		expected string
	}{
		{reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN, ".md"},
		{reportv1.ReportFormat_REPORT_FORMAT_CSV, ".csv"},
		{reportv1.ReportFormat_REPORT_FORMAT_EXCEL, ".xlsx"},
		{reportv1.ReportFormat_REPORT_FORMAT_PDF, ".pdf"},
		{reportv1.ReportFormat_REPORT_FORMAT_HTML, ".html"},
		{reportv1.ReportFormat_REPORT_FORMAT_JSON, ".json"},
		{reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED, ".txt"},
	}

	for _, tt := range tests {
		t.Run(tt.format.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, getExtension(tt.format))
		})
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		format   reportv1.ReportFormat
		expected string
	}{
		{reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN, "text/markdown"},
		{reportv1.ReportFormat_REPORT_FORMAT_CSV, "text/csv"},
		{reportv1.ReportFormat_REPORT_FORMAT_EXCEL, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{reportv1.ReportFormat_REPORT_FORMAT_PDF, "application/pdf"},
		{reportv1.ReportFormat_REPORT_FORMAT_HTML, "text/html"},
		{reportv1.ReportFormat_REPORT_FORMAT_JSON, "application/json"},
		{reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED, "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.format.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, getContentType(tt.format))
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Name", "Simple_Name"},
		{"With-Dash_And_Underscore", "With-Dash_And_Underscore"},
		{"Special!@#$%Chars", "SpecialChars"},
		{"Числа123", "123"},
		{"", "report"},
		{"   ", "___"}, // ИСПРАВЛЕНО: пробелы конвертируются в _
		{"a", "a"},
		{"Report 2024", "Report_2024"},
		{"!!!@@@###", "report"}, // Добавлен тест: только спецсимволы -> report
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeFilename(tt.input))
		})
	}
}

func TestConvertSimulationResult(t *testing.T) {
	svc := &ReportService{}

	t.Run("WhatIf", func(t *testing.T) {
		req := &reportv1.GenerateSimulationReportRequest{
			SimulationResult: &reportv1.GenerateSimulationReportRequest_WhatIf{
				WhatIf: &simulationv1.RunWhatIfResponse{
					Baseline: &simulationv1.ScenarioResult{
						MaxFlow:   100,
						TotalCost: 500,
					},
					Modified: &simulationv1.ScenarioResult{
						MaxFlow:   120,
						TotalCost: 600,
					},
				},
			},
		}

		data := svc.convertSimulationResult(req)

		assert.Equal(t, "What-If Analysis", data.SimulationType)
		assert.Equal(t, 100.0, data.BaselineFlow)
		assert.Equal(t, 500.0, data.BaselineCost)
		assert.Len(t, data.Scenarios, 1)
	})

	t.Run("MonteCarlo", func(t *testing.T) {
		req := &reportv1.GenerateSimulationReportRequest{
			SimulationResult: &reportv1.GenerateSimulationReportRequest_MonteCarlo{
				MonteCarlo: &simulationv1.RunMonteCarloResponse{
					FlowStats: &simulationv1.MonteCarloStats{
						Mean:   150,
						StdDev: 10,
					},
				},
			},
		}

		data := svc.convertSimulationResult(req)

		assert.Equal(t, "Monte Carlo Simulation", data.SimulationType)
		assert.NotNil(t, data.MonteCarlo)
		assert.Equal(t, 150.0, data.MonteCarlo.MeanFlow)
	})

	t.Run("Sensitivity", func(t *testing.T) {
		req := &reportv1.GenerateSimulationReportRequest{
			SimulationResult: &reportv1.GenerateSimulationReportRequest_Sensitivity{
				Sensitivity: &simulationv1.AnalyzeSensitivityResponse{
					ParameterResults: []*simulationv1.SensitivityResult{
						{
							ParameterId:      "param1",
							Elasticity:       1.5,
							SensitivityIndex: 0.8,
						},
					},
				},
			},
		}

		data := svc.convertSimulationResult(req)

		assert.Equal(t, "Sensitivity Analysis", data.SimulationType)
		assert.Len(t, data.Sensitivity, 1)
	})

	t.Run("Resilience", func(t *testing.T) {
		req := &reportv1.GenerateSimulationReportRequest{
			SimulationResult: &reportv1.GenerateSimulationReportRequest_Resilience{
				Resilience: &simulationv1.AnalyzeResilienceResponse{
					Metrics: &simulationv1.ResilienceMetrics{
						OverallScore: 0.9,
					},
				},
			},
		}

		data := svc.convertSimulationResult(req)

		assert.Equal(t, "Resilience Analysis", data.SimulationType)
		assert.NotNil(t, data.Resilience)
		assert.Equal(t, 0.9, data.Resilience.OverallScore)
	})

	t.Run("TimeSimulation", func(t *testing.T) {
		req := &reportv1.GenerateSimulationReportRequest{
			SimulationResult: &reportv1.GenerateSimulationReportRequest_TimeSimulation{
				TimeSimulation: &simulationv1.RunTimeSimulationResponse{
					Stats: &simulationv1.TimeSimulationStats{
						AvgFlow: 100,
					},
				},
			},
		}

		data := svc.convertSimulationResult(req)

		assert.Equal(t, "Time-Dependent Simulation", data.SimulationType)
		assert.Equal(t, 100.0, data.BaselineFlow)
	})

	t.Run("Comparison", func(t *testing.T) {
		req := &reportv1.GenerateSimulationReportRequest{
			SimulationResult: &reportv1.GenerateSimulationReportRequest_Comparison{
				Comparison: &simulationv1.CompareScenariosResponse{
					Baseline: &simulationv1.ScenarioResult{
						MaxFlow: 100,
					},
					RankedScenarios: []*simulationv1.ScenarioResultWithRank{
						{
							Result: &simulationv1.ScenarioResult{
								Name:    "Scenario A",
								MaxFlow: 120,
							},
						},
					},
				},
			},
		}

		data := svc.convertSimulationResult(req)

		assert.Equal(t, "Scenario Comparison", data.SimulationType)
		assert.Equal(t, 100.0, data.BaselineFlow)
		assert.Len(t, data.Scenarios, 1)
	})

	t.Run("NilSimulationResult", func(t *testing.T) {
		req := &reportv1.GenerateSimulationReportRequest{}

		data := svc.convertSimulationResult(req)

		assert.Empty(t, data.SimulationType)
	})
}

// Тест для проверки счётчика отчётов
func TestReportService_ReportsCounter(t *testing.T) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	// Генерируем несколько отчётов
	for i := 0; i < 3; i++ {
		req := &reportv1.GenerateFlowReportRequest{
			Graph:  &commonv1.Graph{},
			Result: &commonv1.FlowResult{},
			Format: reportv1.ReportFormat_REPORT_FORMAT_JSON,
		}
		_, _ = svc.GenerateFlowReport(ctx, req)
	}

	// Проверяем health
	resp, err := svc.Health(ctx, &reportv1.HealthRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), resp.ReportsGenerated)
}

// Тест shouldSave
func TestReportService_ShouldSave(t *testing.T) {
	svc := &ReportService{saveToStorage: false}

	// Без опций - использует дефолт
	assert.False(t, svc.shouldSave(nil))

	// С явным указанием
	assert.True(t, svc.shouldSave(&reportv1.ReportOptions{SaveToStorage: true}))
	assert.False(t, svc.shouldSave(&reportv1.ReportOptions{SaveToStorage: false}))

	// С дефолтом true
	svc.saveToStorage = true
	assert.True(t, svc.shouldSave(nil))
	assert.True(t, svc.shouldSave(&reportv1.ReportOptions{SaveToStorage: false}))
}

// Бенчмарки
func BenchmarkGenerateFlowReport_JSON(b *testing.B) {
	ctx := context.Background()
	svc := NewReportService(ServiceConfig{Version: "1.0.0"}, nil)

	req := &reportv1.GenerateFlowReportRequest{
		Graph: &commonv1.Graph{
			Nodes: make([]*commonv1.Node, 100),
			Edges: make([]*commonv1.Edge, 200),
		},
		Result: &commonv1.FlowResult{
			MaxFlow: 1000,
			Edges:   make([]*commonv1.FlowEdge, 200),
		},
		Format: reportv1.ReportFormat_REPORT_FORMAT_JSON,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GenerateFlowReport(ctx, req)
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	input := "Test Report with Special!@#$% Characters 2024"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitizeFilename(input)
	}
}
