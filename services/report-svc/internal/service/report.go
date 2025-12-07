// services/report-svc/internal/service/report.go
package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	reportv1 "logistics/gen/go/logistics/report/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/telemetry"
	"logistics/services/report-svc/internal/generator"
	"logistics/services/report-svc/internal/repository"
)

var startTime = time.Now()

// ReportService реализация gRPC сервиса отчётов
type ReportService struct {
	reportv1.UnimplementedReportServiceServer

	version          string
	reportsGenerated atomic.Int64
	generators       map[reportv1.ReportFormat]generator.Generator
	repository       repository.Repository

	// Настройки
	defaultTTL    time.Duration
	saveToStorage bool
}

// ServiceConfig конфигурация сервиса
type ServiceConfig struct {
	Version       string
	DefaultTTL    time.Duration
	SaveToStorage bool
}

// NewReportService создаёт новый сервис
func NewReportService(cfg ServiceConfig, repo repository.Repository) *ReportService {
	return &ReportService{
		version: cfg.Version,
		generators: map[reportv1.ReportFormat]generator.Generator{
			reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN: generator.NewMarkdownGenerator(),
			reportv1.ReportFormat_REPORT_FORMAT_CSV:      generator.NewCSVGenerator(),
			reportv1.ReportFormat_REPORT_FORMAT_EXCEL:    generator.NewExcelGenerator(),
			reportv1.ReportFormat_REPORT_FORMAT_PDF:      generator.NewPDFGenerator(),
			reportv1.ReportFormat_REPORT_FORMAT_HTML:     generator.NewHTMLGenerator(),
			reportv1.ReportFormat_REPORT_FORMAT_JSON:     generator.NewJSONGenerator(),
		},
		repository:    repo,
		defaultTTL:    cfg.DefaultTTL,
		saveToStorage: cfg.SaveToStorage,
	}
}

// GenerateFlowReport генерирует отчёт по потоку
func (s *ReportService) GenerateFlowReport(
	ctx context.Context,
	req *reportv1.GenerateFlowReportRequest,
) (*reportv1.GenerateFlowReportResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GenerateFlowReport",
		trace.WithAttributes(
			attribute.String("format", req.Format.String()),
		),
	)
	defer span.End()

	start := time.Now()

	gen, err := s.getGenerator(req.Format)
	if err != nil {
		return &reportv1.GenerateFlowReportResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Конвертируем данные для генератора
	data := &generator.ReportData{
		Type:       reportv1.ReportType_REPORT_TYPE_FLOW,
		Options:    req.Options,
		Graph:      req.Graph,
		FlowResult: req.Result,
	}

	// Добавляем дополнительные данные
	if req.Metrics != nil {
		data.FlowData = &generator.FlowReportData{
			Metrics: req.Metrics,
		}
	}

	// Конвертируем edges для генератора
	if req.Result != nil {
		data.FlowEdges = generator.ConvertFlowEdges(req.Result.Edges)
	}

	content, err := gen.Generate(ctx, data)
	if err != nil {
		telemetry.SetError(ctx, err)
		return &reportv1.GenerateFlowReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to generate report: %v", err),
		}, nil
	}

	duration := time.Since(start)
	s.reportsGenerated.Add(1)

	metadata := s.buildMetadata(
		reportv1.ReportType_REPORT_TYPE_FLOW,
		req.Format,
		content,
		duration,
		req.Options,
		req.CalculationId,
		req.GraphId,
	)

	// Сохраняем в хранилище
	if s.shouldSave(req.Options) && s.repository != nil {
		if err := s.saveReport(ctx, req.Options, metadata, content, ""); err != nil {
			telemetry.SetError(ctx, err)
			// Логируем, но не фейлим запрос
		}
	}

	return &reportv1.GenerateFlowReportResponse{
		Success:  true,
		Metadata: metadata,
		Content: &reportv1.ReportContent{
			Data:        content,
			ContentType: getContentType(req.Format),
			Filename:    metadata.Filename,
			SizeBytes:   int64(len(content)),
		},
	}, nil
}

// GenerateAnalyticsReport генерирует аналитический отчёт
func (s *ReportService) GenerateAnalyticsReport(
	ctx context.Context,
	req *reportv1.GenerateAnalyticsReportRequest,
) (*reportv1.GenerateAnalyticsReportResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GenerateAnalyticsReport")
	defer span.End()

	start := time.Now()

	gen, err := s.getGenerator(req.Format)
	if err != nil {
		return &reportv1.GenerateAnalyticsReportResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Конвертируем данные для генератора
	analyticsData := &generator.AnalyticsReportData{
		FlowStats:  req.FlowStats,
		GraphStats: req.GraphStats,
	}

	// Обрабатываем cost response
	if req.Cost != nil {
		analyticsData.TotalCost = req.Cost.TotalCost
		analyticsData.Currency = req.Cost.Currency
		analyticsData.CostBreakdown = generator.ConvertCostBreakdown(req.Cost.Breakdown)
	}

	// Обрабатываем bottlenecks
	if req.Bottlenecks != nil {
		analyticsData.Bottlenecks = generator.ConvertBottlenecks(req.Bottlenecks.Bottlenecks)
		analyticsData.Recommendations = generator.ConvertRecommendations(req.Bottlenecks.Recommendations)
	}

	// Обрабатываем efficiency
	analyticsData.Efficiency = generator.ConvertEfficiency(req.Efficiency)

	data := &generator.ReportData{
		Type:          reportv1.ReportType_REPORT_TYPE_ANALYTICS,
		Options:       req.Options,
		Graph:         req.Graph,
		AnalyticsData: analyticsData,
	}

	content, err := gen.Generate(ctx, data)
	if err != nil {
		telemetry.SetError(ctx, err)
		return &reportv1.GenerateAnalyticsReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to generate report: %v", err),
		}, nil
	}

	duration := time.Since(start)
	s.reportsGenerated.Add(1)

	metadata := s.buildMetadata(
		reportv1.ReportType_REPORT_TYPE_ANALYTICS,
		req.Format,
		content,
		duration,
		req.Options,
		req.CalculationId,
		req.GraphId,
	)

	if s.shouldSave(req.Options) && s.repository != nil {
		if err := s.saveReport(ctx, req.Options, metadata, content, ""); err != nil {
			telemetry.SetError(ctx, err)
		}
	}

	return &reportv1.GenerateAnalyticsReportResponse{
		Success:  true,
		Metadata: metadata,
		Content: &reportv1.ReportContent{
			Data:        content,
			ContentType: getContentType(req.Format),
			Filename:    metadata.Filename,
			SizeBytes:   int64(len(content)),
		},
	}, nil
}

// GenerateSimulationReport генерирует отчёт по симуляции
func (s *ReportService) GenerateSimulationReport(
	ctx context.Context,
	req *reportv1.GenerateSimulationReportRequest,
) (*reportv1.GenerateSimulationReportResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GenerateSimulationReport")
	defer span.End()

	start := time.Now()

	gen, err := s.getGenerator(req.Format)
	if err != nil {
		return &reportv1.GenerateSimulationReportResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Конвертируем данные симуляции
	simData := s.convertSimulationResult(req)

	data := &generator.ReportData{
		Type:           reportv1.ReportType_REPORT_TYPE_SIMULATION,
		Options:        req.Options,
		Graph:          req.BaselineGraph,
		SimulationData: simData,
	}

	content, err := gen.Generate(ctx, data)
	if err != nil {
		telemetry.SetError(ctx, err)
		return &reportv1.GenerateSimulationReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to generate report: %v", err),
		}, nil
	}

	duration := time.Since(start)
	s.reportsGenerated.Add(1)

	metadata := s.buildMetadata(
		reportv1.ReportType_REPORT_TYPE_SIMULATION,
		req.Format,
		content,
		duration,
		req.Options,
		req.CalculationId,
		req.GraphId,
	)

	if s.shouldSave(req.Options) && s.repository != nil {
		if err := s.saveReport(ctx, req.Options, metadata, content, ""); err != nil {
			telemetry.SetError(ctx, err)
		}
	}

	return &reportv1.GenerateSimulationReportResponse{
		Success:  true,
		Metadata: metadata,
		Content: &reportv1.ReportContent{
			Data:        content,
			ContentType: getContentType(req.Format),
			Filename:    metadata.Filename,
			SizeBytes:   int64(len(content)),
		},
	}, nil
}

// GenerateSummaryReport генерирует сводный отчёт
func (s *ReportService) GenerateSummaryReport(
	ctx context.Context,
	req *reportv1.GenerateSummaryReportRequest,
) (*reportv1.GenerateSummaryReportResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GenerateSummaryReport")
	defer span.End()

	start := time.Now()

	gen, err := s.getGenerator(req.Format)
	if err != nil {
		return &reportv1.GenerateSummaryReportResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	data := &generator.ReportData{
		Type:       reportv1.ReportType_REPORT_TYPE_SUMMARY,
		Options:    req.Options,
		Graph:      req.Graph,
		FlowResult: req.FlowResult,
	}

	// Добавляем аналитику если есть
	if req.Analytics != nil {
		data.AnalyticsData = &generator.AnalyticsReportData{}
		if req.Analytics.Cost != nil {
			data.AnalyticsData.TotalCost = req.Analytics.Cost.TotalCost
			data.AnalyticsData.Currency = req.Analytics.Cost.Currency
			data.AnalyticsData.CostBreakdown = generator.ConvertCostBreakdown(req.Analytics.Cost.Breakdown)
		}
		if req.Analytics.Bottlenecks != nil {
			data.AnalyticsData.Bottlenecks = generator.ConvertBottlenecks(req.Analytics.Bottlenecks.Bottlenecks)
			data.AnalyticsData.Recommendations = generator.ConvertRecommendations(req.Analytics.Bottlenecks.Recommendations)
		}
		data.AnalyticsData.Efficiency = generator.ConvertEfficiency(req.Analytics.Efficiency)
		data.AnalyticsData.FlowStats = req.Analytics.FlowStats
		data.AnalyticsData.GraphStats = req.Analytics.GraphStats
	}

	// Добавляем симуляции если есть
	if len(req.Simulations) > 0 {
		data.SimulationData = &generator.SimulationReportData{
			SimulationType: "Multiple",
		}
		// Можно добавить обработку simulations
	}

	content, err := gen.Generate(ctx, data)
	if err != nil {
		telemetry.SetError(ctx, err)
		return &reportv1.GenerateSummaryReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to generate report: %v", err),
		}, nil
	}

	duration := time.Since(start)
	s.reportsGenerated.Add(1)

	metadata := s.buildMetadata(
		reportv1.ReportType_REPORT_TYPE_SUMMARY,
		req.Format,
		content,
		duration,
		req.Options,
		req.CalculationId,
		req.GraphId,
	)

	if s.shouldSave(req.Options) && s.repository != nil {
		if err := s.saveReport(ctx, req.Options, metadata, content, ""); err != nil {
			telemetry.SetError(ctx, err)
		}
	}

	return &reportv1.GenerateSummaryReportResponse{
		Success:  true,
		Metadata: metadata,
		Content: &reportv1.ReportContent{
			Data:        content,
			ContentType: getContentType(req.Format),
			Filename:    metadata.Filename,
			SizeBytes:   int64(len(content)),
		},
	}, nil
}

// GenerateComparisonReport генерирует отчёт сравнения
func (s *ReportService) GenerateComparisonReport(
	ctx context.Context,
	req *reportv1.GenerateComparisonReportRequest,
) (*reportv1.GenerateComparisonReportResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GenerateComparisonReport")
	defer span.End()

	start := time.Now()

	gen, err := s.getGenerator(req.Format)
	if err != nil {
		return &reportv1.GenerateComparisonReportResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Конвертируем items
	compData := make([]*generator.ComparisonItemData, 0, len(req.Items))
	for _, item := range req.Items {
		if item == nil {
			continue
		}
		cd := &generator.ComparisonItemData{
			Name:    item.Name,
			Metrics: item.Metrics,
		}
		if item.Result != nil {
			cd.MaxFlow = item.Result.MaxFlow
			cd.TotalCost = item.Result.TotalCost
		}
		compData = append(compData, cd)
	}

	data := &generator.ReportData{
		Type:           reportv1.ReportType_REPORT_TYPE_COMPARISON,
		Options:        req.Options,
		ComparisonData: compData,
	}

	content, err := gen.Generate(ctx, data)
	if err != nil {
		telemetry.SetError(ctx, err)
		return &reportv1.GenerateComparisonReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to generate report: %v", err),
		}, nil
	}

	duration := time.Since(start)
	s.reportsGenerated.Add(1)

	metadata := s.buildMetadata(
		reportv1.ReportType_REPORT_TYPE_COMPARISON,
		req.Format,
		content,
		duration,
		req.Options,
		req.CalculationId,
		"",
	)

	if s.shouldSave(req.Options) && s.repository != nil {
		if err := s.saveReport(ctx, req.Options, metadata, content, ""); err != nil {
			telemetry.SetError(ctx, err)
		}
	}

	return &reportv1.GenerateComparisonReportResponse{
		Success:  true,
		Metadata: metadata,
		Content: &reportv1.ReportContent{
			Data:        content,
			ContentType: getContentType(req.Format),
			Filename:    metadata.Filename,
			SizeBytes:   int64(len(content)),
		},
	}, nil
}

// GenerateHistoryReport генерирует отчёт истории
func (s *ReportService) GenerateHistoryReport(
	ctx context.Context,
	req *reportv1.GenerateHistoryReportRequest,
) (*reportv1.GenerateHistoryReportResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GenerateHistoryReport")
	defer span.End()

	start := time.Now()

	gen, err := s.getGenerator(req.Format)
	if err != nil {
		return &reportv1.GenerateHistoryReportResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	data := &generator.ReportData{
		Type:    reportv1.ReportType_REPORT_TYPE_HISTORY,
		Options: req.Options,
		// Для history нужно добавить специальную обработку
	}

	content, err := gen.Generate(ctx, data)
	if err != nil {
		telemetry.SetError(ctx, err)
		return &reportv1.GenerateHistoryReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to generate report: %v", err),
		}, nil
	}

	duration := time.Since(start)
	s.reportsGenerated.Add(1)

	metadata := s.buildMetadata(
		reportv1.ReportType_REPORT_TYPE_HISTORY,
		req.Format,
		content,
		duration,
		req.Options,
		"",
		"",
	)

	return &reportv1.GenerateHistoryReportResponse{
		Success:  true,
		Metadata: metadata,
		Content: &reportv1.ReportContent{
			Data:        content,
			ContentType: getContentType(req.Format),
			Filename:    metadata.Filename,
			SizeBytes:   int64(len(content)),
		},
	}, nil
}

// GetReport возвращает сохранённый отчёт
func (s *ReportService) GetReport(
	ctx context.Context,
	req *reportv1.GetReportRequest,
) (*reportv1.GetReportResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GetReport",
		trace.WithAttributes(attribute.String("report_id", req.ReportId)),
	)
	defer span.End()

	if s.repository == nil {
		return &reportv1.GetReportResponse{
			Success:      false,
			ErrorMessage: "storage not configured",
		}, nil
	}

	id, err := uuid.Parse(req.ReportId)
	if err != nil {
		return &reportv1.GetReportResponse{
			Success:      false,
			ErrorMessage: "invalid report ID",
		}, nil
	}

	report, err := s.repository.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &reportv1.GetReportResponse{
				Success:      false,
				ErrorMessage: "report not found",
			}, nil
		}
		telemetry.SetError(ctx, err)
		return &reportv1.GetReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to get report: %v", err),
		}, nil
	}

	return &reportv1.GetReportResponse{
		Success:  true,
		Metadata: report.ToMetadata(),
		Content:  report.ToContent(),
	}, nil
}

// GetReportInfo возвращает информацию об отчёте без контента
func (s *ReportService) GetReportInfo(
	ctx context.Context,
	req *reportv1.GetReportInfoRequest,
) (*reportv1.GetReportInfoResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GetReportInfo",
		trace.WithAttributes(attribute.String("report_id", req.ReportId)),
	)
	defer span.End()

	if s.repository == nil {
		return &reportv1.GetReportInfoResponse{
			Success:      false,
			ErrorMessage: "storage not configured",
		}, nil
	}

	id, err := uuid.Parse(req.ReportId)
	if err != nil {
		return &reportv1.GetReportInfoResponse{
			Success:      false,
			ErrorMessage: "invalid report ID",
		}, nil
	}

	report, err := s.repository.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &reportv1.GetReportInfoResponse{
				Success:      false,
				ErrorMessage: "report not found",
			}, nil
		}
		return &reportv1.GetReportInfoResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to get report: %v", err),
		}, nil
	}

	return &reportv1.GetReportInfoResponse{
		Success:  true,
		Metadata: report.ToMetadata(),
	}, nil
}

// ListReports возвращает список отчётов
func (s *ReportService) ListReports(
	ctx context.Context,
	req *reportv1.ListReportsRequest,
) (*reportv1.ListReportsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.ListReports")
	defer span.End()

	if s.repository == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeUnimplemented, "storage not configured"),
		)
	}

	params := &repository.ListParams{
		Limit:     req.Limit,
		Offset:    req.Offset,
		UserID:    req.UserId,
		Tags:      req.Tags,
		OrderBy:   req.OrderBy,
		OrderDesc: req.OrderDesc,
	}

	if req.ReportType != reportv1.ReportType_REPORT_TYPE_UNSPECIFIED {
		params.ReportType = &req.ReportType
	}
	if req.Format != reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED {
		params.Format = &req.Format
	}
	if req.CalculationId != "" {
		params.CalculationID = req.CalculationId
	}
	if req.GraphId != "" {
		params.GraphID = req.GraphId
	}
	if req.CreatedAfter != nil {
		t := req.CreatedAfter.AsTime()
		params.CreatedAfter = &t
	}
	if req.CreatedBefore != nil {
		t := req.CreatedBefore.AsTime()
		params.CreatedBefore = &t
	}

	result, err := s.repository.List(ctx, params)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to list reports"),
		)
	}

	reports := make([]*reportv1.ReportMetadata, len(result.Reports))
	for i, r := range result.Reports {
		reports[i] = r.ToMetadata()
	}

	return &reportv1.ListReportsResponse{
		Reports:    reports,
		TotalCount: result.TotalCount,
		HasMore:    result.HasMore,
	}, nil
}

// DeleteReport удаляет отчёт
func (s *ReportService) DeleteReport(
	ctx context.Context,
	req *reportv1.DeleteReportRequest,
) (*reportv1.DeleteReportResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.DeleteReport",
		trace.WithAttributes(attribute.String("report_id", req.ReportId)),
	)
	defer span.End()

	if s.repository == nil {
		return &reportv1.DeleteReportResponse{
			Success:      false,
			ErrorMessage: "storage not configured",
		}, nil
	}

	id, err := uuid.Parse(req.ReportId)
	if err != nil {
		return &reportv1.DeleteReportResponse{
			Success:      false,
			ErrorMessage: "invalid report ID",
		}, nil
	}

	if req.HardDelete {
		err = s.repository.HardDelete(ctx, id)
	} else {
		err = s.repository.Delete(ctx, id)
	}

	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &reportv1.DeleteReportResponse{
				Success:      false,
				ErrorMessage: "report not found",
			}, nil
		}
		telemetry.SetError(ctx, err)
		return &reportv1.DeleteReportResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to delete report: %v", err),
		}, nil
	}

	return &reportv1.DeleteReportResponse{
		Success: true,
	}, nil
}

// UpdateReportTags обновляет теги отчёта
func (s *ReportService) UpdateReportTags(
	ctx context.Context,
	req *reportv1.UpdateReportTagsRequest,
) (*reportv1.UpdateReportTagsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.UpdateReportTags")
	defer span.End()

	if s.repository == nil {
		return &reportv1.UpdateReportTagsResponse{
			Success:      false,
			ErrorMessage: "storage not configured",
		}, nil
	}

	id, err := uuid.Parse(req.ReportId)
	if err != nil {
		return &reportv1.UpdateReportTagsResponse{
			Success:      false,
			ErrorMessage: "invalid report ID",
		}, nil
	}

	tags, err := s.repository.UpdateTags(ctx, id, req.Tags, req.Replace)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &reportv1.UpdateReportTagsResponse{
				Success:      false,
				ErrorMessage: "report not found",
			}, nil
		}
		return &reportv1.UpdateReportTagsResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to update tags: %v", err),
		}, nil
	}

	return &reportv1.UpdateReportTagsResponse{
		Success: true,
		Tags:    tags,
	}, nil
}

// GetRepositoryStats возвращает статистику хранилища
func (s *ReportService) GetRepositoryStats(
	ctx context.Context,
	req *reportv1.GetRepositoryStatsRequest,
) (*reportv1.GetRepositoryStatsResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ReportService.GetRepositoryStats")
	defer span.End()

	if s.repository == nil {
		return nil, pkgerrors.ToGRPC(
			pkgerrors.New(pkgerrors.CodeUnimplemented, "storage not configured"),
		)
	}

	stats, err := s.repository.Stats(ctx, req.UserId)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(
			pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get stats"),
		)
	}

	return stats.ToProto(), nil
}

// GetSupportedFormats возвращает список поддерживаемых форматов
func (s *ReportService) GetSupportedFormats(
	ctx context.Context,
	req *reportv1.GetSupportedFormatsRequest,
) (*reportv1.GetSupportedFormatsResponse, error) {
	formats := []*reportv1.FormatInfo{
		{
			Format:          reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
			Name:            "Markdown",
			Extension:       ".md",
			MimeType:        "text/markdown",
			SupportsCharts:  false,
			SupportsStyling: true,
			SupportedReportTypes: []reportv1.ReportType{
				reportv1.ReportType_REPORT_TYPE_FLOW,
				reportv1.ReportType_REPORT_TYPE_ANALYTICS,
				reportv1.ReportType_REPORT_TYPE_SIMULATION,
				reportv1.ReportType_REPORT_TYPE_SUMMARY,
				reportv1.ReportType_REPORT_TYPE_COMPARISON,
				reportv1.ReportType_REPORT_TYPE_HISTORY,
			},
		},
		{
			Format:          reportv1.ReportFormat_REPORT_FORMAT_CSV,
			Name:            "CSV",
			Extension:       ".csv",
			MimeType:        "text/csv",
			SupportsCharts:  false,
			SupportsStyling: false,
			SupportedReportTypes: []reportv1.ReportType{
				reportv1.ReportType_REPORT_TYPE_FLOW,
				reportv1.ReportType_REPORT_TYPE_ANALYTICS,
				reportv1.ReportType_REPORT_TYPE_COMPARISON,
			},
		},
		{
			Format:          reportv1.ReportFormat_REPORT_FORMAT_EXCEL,
			Name:            "Excel",
			Extension:       ".xlsx",
			MimeType:        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			SupportsCharts:  true,
			SupportsStyling: true,
			SupportedReportTypes: []reportv1.ReportType{
				reportv1.ReportType_REPORT_TYPE_FLOW,
				reportv1.ReportType_REPORT_TYPE_ANALYTICS,
				reportv1.ReportType_REPORT_TYPE_SIMULATION,
				reportv1.ReportType_REPORT_TYPE_SUMMARY,
				reportv1.ReportType_REPORT_TYPE_COMPARISON,
			},
		},
		{
			Format:          reportv1.ReportFormat_REPORT_FORMAT_PDF,
			Name:            "PDF",
			Extension:       ".pdf",
			MimeType:        "application/pdf",
			SupportsCharts:  true,
			SupportsStyling: true,
			SupportedReportTypes: []reportv1.ReportType{
				reportv1.ReportType_REPORT_TYPE_FLOW,
				reportv1.ReportType_REPORT_TYPE_ANALYTICS,
				reportv1.ReportType_REPORT_TYPE_SIMULATION,
				reportv1.ReportType_REPORT_TYPE_SUMMARY,
				reportv1.ReportType_REPORT_TYPE_COMPARISON,
			},
		},
		{
			Format:          reportv1.ReportFormat_REPORT_FORMAT_HTML,
			Name:            "HTML",
			Extension:       ".html",
			MimeType:        "text/html",
			SupportsCharts:  true,
			SupportsStyling: true,
			SupportedReportTypes: []reportv1.ReportType{
				reportv1.ReportType_REPORT_TYPE_FLOW,
				reportv1.ReportType_REPORT_TYPE_ANALYTICS,
				reportv1.ReportType_REPORT_TYPE_SIMULATION,
				reportv1.ReportType_REPORT_TYPE_SUMMARY,
				reportv1.ReportType_REPORT_TYPE_COMPARISON,
			},
		},
		{
			Format:          reportv1.ReportFormat_REPORT_FORMAT_JSON,
			Name:            "JSON",
			Extension:       ".json",
			MimeType:        "application/json",
			SupportsCharts:  false,
			SupportsStyling: false,
			SupportedReportTypes: []reportv1.ReportType{
				reportv1.ReportType_REPORT_TYPE_FLOW,
				reportv1.ReportType_REPORT_TYPE_ANALYTICS,
				reportv1.ReportType_REPORT_TYPE_SIMULATION,
				reportv1.ReportType_REPORT_TYPE_SUMMARY,
				reportv1.ReportType_REPORT_TYPE_COMPARISON,
			},
		},
	}

	return &reportv1.GetSupportedFormatsResponse{
		Formats: formats,
	}, nil
}

// Health возвращает статус сервиса
func (s *ReportService) Health(
	ctx context.Context,
	req *reportv1.HealthRequest,
) (*reportv1.HealthResponse, error) {
	resp := &reportv1.HealthResponse{
		Status:           "SERVING",
		Version:          s.version,
		UptimeSeconds:    int64(time.Since(startTime).Seconds()),
		ReportsGenerated: s.reportsGenerated.Load(),
	}

	// Проверяем storage
	if s.repository != nil {
		if err := s.repository.Ping(ctx); err != nil {
			resp.Status = "DEGRADED"
			resp.Storage = &reportv1.StorageHealth{
				Status:       "ERROR",
				ErrorMessage: err.Error(),
			}
		} else {
			stats, err := s.repository.Stats(ctx, "")
			if err != nil {
				resp.Storage = &reportv1.StorageHealth{
					Status:       "ERROR",
					ErrorMessage: err.Error(),
				}
			} else {
				resp.Storage = &reportv1.StorageHealth{
					Status:         "OK",
					StoredReports:  stats.TotalReports,
					TotalSizeBytes: stats.TotalSizeBytes,
				}
			}
		}
	} else {
		resp.Storage = &reportv1.StorageHealth{
			Status: "NOT_CONFIGURED",
		}
	}

	return resp, nil
}

// === Вспомогательные методы ===

func (s *ReportService) getGenerator(format reportv1.ReportFormat) (generator.Generator, error) {
	gen, ok := s.generators[format]
	if !ok {
		return nil, fmt.Errorf("unsupported format: %s", format.String())
	}
	return gen, nil
}

func (s *ReportService) shouldSave(opts *reportv1.ReportOptions) bool {
	if opts != nil && opts.SaveToStorage {
		return true
	}
	return s.saveToStorage
}

func (s *ReportService) buildMetadata(
	reportType reportv1.ReportType,
	format reportv1.ReportFormat,
	content []byte,
	duration time.Duration,
	opts *reportv1.ReportOptions,
	calculationID, graphID string,
) *reportv1.ReportMetadata {
	ext := getExtension(format)
	title := "report"
	if opts != nil && opts.Title != "" {
		title = sanitizeFilename(opts.Title)
	}

	meta := &reportv1.ReportMetadata{
		ReportId:         uuid.New().String(),
		Type:             reportType,
		Format:           format,
		GeneratedAt:      timestamppb.Now(),
		GenerationTimeMs: float64(duration.Milliseconds()),
		SizeBytes:        int64(len(content)),
		Filename:         fmt.Sprintf("%s_%s%s", title, time.Now().Format("20060102_150405"), ext),
		CalculationId:    calculationID,
		GraphId:          graphID,
	}

	if opts != nil {
		meta.Title = opts.Title
		meta.Description = opts.Description
		meta.GeneratedBy = opts.Author
		meta.Tags = opts.Tags
		meta.CustomFields = opts.CustomFields

		if opts.TtlSeconds > 0 {
			meta.ExpiresAt = timestamppb.New(time.Now().Add(time.Duration(opts.TtlSeconds) * time.Second))
		}
	}

	return meta
}

func (s *ReportService) saveReport(
	ctx context.Context,
	opts *reportv1.ReportOptions,
	metadata *reportv1.ReportMetadata,
	content []byte,
	userID string,
) error {
	if s.repository == nil {
		return nil
	}

	title := "Untitled Report"
	description := ""
	author := "System"
	var tags []string
	var customFields map[string]string

	if opts != nil {
		if opts.Title != "" {
			title = opts.Title
		}
		if opts.Description != "" {
			description = opts.Description
		}
		if opts.Author != "" {
			author = opts.Author
		}
		tags = opts.Tags
		customFields = opts.CustomFields
	}

	ttl := s.defaultTTL
	if opts != nil && opts.TtlSeconds > 0 {
		ttl = time.Duration(opts.TtlSeconds) * time.Second
	}

	params := &repository.CreateParams{
		Title:            title,
		Description:      description,
		Author:           author,
		ReportType:       metadata.Type,
		Format:           metadata.Format,
		Content:          content,
		ContentType:      getContentType(metadata.Format),
		Filename:         metadata.Filename,
		CalculationID:    metadata.CalculationId,
		GraphID:          metadata.GraphId,
		UserID:           userID,
		GenerationTimeMs: metadata.GenerationTimeMs,
		Version:          s.version,
		Tags:             tags,
		CustomFields:     customFields,
		TTL:              ttl,
	}

	_, err := s.repository.Create(ctx, params)
	return err
}

func (s *ReportService) convertSimulationResult(req *reportv1.GenerateSimulationReportRequest) *generator.SimulationReportData {
	data := &generator.SimulationReportData{}

	switch result := req.SimulationResult.(type) {
	case *reportv1.GenerateSimulationReportRequest_WhatIf:
		data.SimulationType = "What-If Analysis"
		if result.WhatIf != nil {
			if result.WhatIf.Baseline != nil {
				data.BaselineFlow = result.WhatIf.Baseline.MaxFlow
				data.BaselineCost = result.WhatIf.Baseline.TotalCost
			}
			if result.WhatIf.Modified != nil {
				data.Scenarios = append(data.Scenarios, &generator.ScenarioData{
					Name:      "Modified",
					MaxFlow:   result.WhatIf.Modified.MaxFlow,
					TotalCost: result.WhatIf.Modified.TotalCost,
				})
			}
		}

	case *reportv1.GenerateSimulationReportRequest_Comparison:
		data.SimulationType = "Scenario Comparison"
		if result.Comparison != nil {
			if result.Comparison.Baseline != nil {
				data.BaselineFlow = result.Comparison.Baseline.MaxFlow
				data.BaselineCost = result.Comparison.Baseline.TotalCost
			}
			data.Scenarios = generator.ConvertScenarioResults(result.Comparison.RankedScenarios)
		}

	case *reportv1.GenerateSimulationReportRequest_MonteCarlo:
		data.SimulationType = "Monte Carlo Simulation"
		if result.MonteCarlo != nil {
			data.MonteCarlo = generator.ConvertMonteCarloStats(result.MonteCarlo)
		}

	case *reportv1.GenerateSimulationReportRequest_Sensitivity:
		data.SimulationType = "Sensitivity Analysis"
		if result.Sensitivity != nil {
			data.Sensitivity = generator.ConvertSensitivityResults(result.Sensitivity.ParameterResults)
		}

	case *reportv1.GenerateSimulationReportRequest_Resilience:
		data.SimulationType = "Resilience Analysis"
		if result.Resilience != nil {
			data.Resilience = generator.ConvertResilienceMetrics(result.Resilience)
		}

	case *reportv1.GenerateSimulationReportRequest_TimeSimulation:
		data.SimulationType = "Time-Dependent Simulation"
		if result.TimeSimulation != nil && result.TimeSimulation.Stats != nil {
			stats := result.TimeSimulation.Stats
			data.BaselineFlow = stats.AvgFlow
		}
	}

	return data
}

func getExtension(format reportv1.ReportFormat) string {
	switch format {
	case reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN:
		return ".md"
	case reportv1.ReportFormat_REPORT_FORMAT_CSV:
		return ".csv"
	case reportv1.ReportFormat_REPORT_FORMAT_EXCEL:
		return ".xlsx"
	case reportv1.ReportFormat_REPORT_FORMAT_PDF:
		return ".pdf"
	case reportv1.ReportFormat_REPORT_FORMAT_HTML:
		return ".html"
	case reportv1.ReportFormat_REPORT_FORMAT_JSON:
		return ".json"
	default:
		return ".txt"
	}
}

func getContentType(format reportv1.ReportFormat) string {
	switch format {
	case reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN:
		return "text/markdown"
	case reportv1.ReportFormat_REPORT_FORMAT_CSV:
		return "text/csv"
	case reportv1.ReportFormat_REPORT_FORMAT_EXCEL:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case reportv1.ReportFormat_REPORT_FORMAT_PDF:
		return "application/pdf"
	case reportv1.ReportFormat_REPORT_FORMAT_HTML:
		return "text/html"
	case reportv1.ReportFormat_REPORT_FORMAT_JSON:
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func sanitizeFilename(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			result = append(result, r)
		} else if r == ' ' {
			result = append(result, '_')
		}
	}
	if len(result) == 0 {
		return "report"
	}
	return string(result)
}
