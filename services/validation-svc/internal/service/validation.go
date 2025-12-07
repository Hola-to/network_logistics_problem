package service

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	commonv1 "logistics/gen/go/logistics/common/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/pkg/logger"
	"logistics/pkg/telemetry"
	"logistics/services/validation-svc/internal/validators"
)

var startTime = time.Now()

type ValidationService struct {
	validationv1.UnimplementedValidationServiceServer
	version string
}

func NewValidationService(version string) *ValidationService {
	return &ValidationService{version: version}
}

// ValidateGraph валидирует структуру графа
func (s *ValidationService) ValidateGraph(
	ctx context.Context,
	req *validationv1.ValidateGraphRequest,
) (*validationv1.ValidateGraphResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ValidationService.ValidateGraph",
		trace.WithAttributes(
			attribute.String("level", req.Level.String()),
		),
	)
	defer span.End()

	start := time.Now()

	response := &validationv1.ValidateGraphResponse{
		Warnings: []string{},
		Metrics:  &validationv1.ValidationMetrics{},
	}

	// Добавляем атрибуты графа
	if req.Graph != nil {
		telemetry.SetAttributes(ctx, telemetry.GraphAttributes(
			len(req.Graph.Nodes),
			len(req.Graph.Edges),
			req.Graph.SourceId,
			req.Graph.SinkId,
		)...)
	}

	var allErrors []*commonv1.ValidationError
	var totalChecks, passedChecks, failedChecks, warningChecks int32

	// 1. Базовая структура (всегда)
	ctx, structSpan := telemetry.StartSpan(ctx, "ValidateStructure")
	structureErrors := validators.ValidateStructure(req.Graph)
	structSpan.End()

	allErrors = append(allErrors, structureErrors...)
	if len(structureErrors) > 0 {
		failedChecks += int32(len(structureErrors))

		telemetry.AddEvent(ctx, "structure_validation_failed",
			attribute.Int("errors", len(structureErrors)),
		)

		response.Result = &commonv1.ValidationResult{
			IsValid: false,
			Errors:  allErrors,
		}
		response.Metrics = buildMetrics(totalChecks+failedChecks, passedChecks, failedChecks, warningChecks, start)
		return response, nil
	}
	passedChecks++
	totalChecks++

	// Определяем уровень валидации
	level := req.Level
	if level == validationv1.ValidationLevel_VALIDATION_LEVEL_UNSPECIFIED {
		level = validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD
	}

	// 2. Связность
	if req.CheckConnectivity || level >= validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD {
		_, connSpan := telemetry.StartSpan(ctx, "ValidateConnectivity")
		connErrors := validators.ValidateConnectivity(req.Graph)
		connSpan.End()

		allErrors = append(allErrors, connErrors...)
		if len(connErrors) > 0 {
			failedChecks += int32(len(connErrors))
		} else {
			passedChecks++
		}
		totalChecks++
	}

	// 3. Бизнес-правила
	if req.CheckBusinessRules || level >= validationv1.ValidationLevel_VALIDATION_LEVEL_STRICT {
		_, bizSpan := telemetry.StartSpan(ctx, "ValidateBusinessRules")
		businessErrors := validators.ValidateBusinessRules(req.Graph)
		bizSpan.End()

		allErrors = append(allErrors, businessErrors...)
		if len(businessErrors) > 0 {
			failedChecks += int32(len(businessErrors))
		} else {
			passedChecks++
		}
		totalChecks++
	}

	// 4. Топология
	if req.CheckTopology || level >= validationv1.ValidationLevel_VALIDATION_LEVEL_FULL {
		_, topoSpan := telemetry.StartSpan(ctx, "ValidateTopology")
		topoResult := validators.ValidateTopology(req.Graph)
		topoSpan.End()

		allErrors = append(allErrors, topoResult.Errors...)
		response.Warnings = append(response.Warnings, topoResult.Warnings...)
		failedChecks += int32(len(topoResult.Errors))
		warningChecks += int32(len(topoResult.Warnings))
		if len(topoResult.Errors) == 0 {
			passedChecks++
		}
		totalChecks++
	}

	// Статистика графа
	response.Statistics = validators.CalculateGraphStatistics(req.Graph)

	isValid := len(allErrors) == 0
	response.Result = &commonv1.ValidationResult{
		IsValid: isValid,
		Errors:  allErrors,
	}
	response.Metrics = buildMetrics(totalChecks, passedChecks, failedChecks, warningChecks, start)

	// Добавляем результаты в span
	telemetry.SetAttributes(ctx, telemetry.ValidationAttributes(
		level.String(),
		len(allErrors),
		isValid,
	)...)

	telemetry.AddEvent(ctx, "validation_completed",
		attribute.Bool("valid", isValid),
		attribute.Int("total_checks", int(totalChecks)),
		attribute.Int("passed_checks", int(passedChecks)),
		attribute.Int("failed_checks", int(failedChecks)),
	)

	return response, nil
}

// ValidateFlow валидирует поток в графе
func (s *ValidationService) ValidateFlow(
	ctx context.Context,
	req *validationv1.ValidateFlowRequest,
) (*validationv1.ValidateFlowResponse, error) {
	_, span := telemetry.StartSpan(ctx, "ValidationService.ValidateFlow")
	defer span.End()

	violations := validators.ValidateFlowLogic(req.Graph)

	flowViolations := make([]*validationv1.FlowViolation, 0, len(violations))
	for _, err := range violations {
		flowViolations = append(flowViolations, &validationv1.FlowViolation{
			Code:    err.Code,
			Message: err.Message,
			Field:   err.Field,
		})
	}

	summary := validators.CalculateFlowSummary(req.Graph)

	if req.ExpectedMaxFlow > 0 {
		if abs(summary.TotalFlow-req.ExpectedMaxFlow) > validators.Epsilon {
			flowViolations = append(flowViolations, &validationv1.FlowViolation{
				Code:     "UNEXPECTED_MAX_FLOW",
				Message:  "Фактический поток не соответствует ожидаемому",
				Expected: req.ExpectedMaxFlow,
				Actual:   summary.TotalFlow,
			})
		}
	}

	isValid := len(flowViolations) == 0
	span.SetAttributes(
		attribute.Bool("valid", isValid),
		attribute.Int("violations", len(flowViolations)),
	)

	return &validationv1.ValidateFlowResponse{
		IsValid:    isValid,
		Violations: flowViolations,
		Summary:    summary,
	}, nil
}

// ValidateForAlgorithm проверяет совместимость с алгоритмом
func (s *ValidationService) ValidateForAlgorithm(
	ctx context.Context,
	req *validationv1.ValidateForAlgorithmRequest,
) (*validationv1.ValidateForAlgorithmResponse, error) {
	_, span := telemetry.StartSpan(ctx, "ValidationService.ValidateForAlgorithm",
		trace.WithAttributes(
			attribute.String("algorithm", req.Algorithm.String()),
		),
	)
	defer span.End()

	result := validators.ValidateForAlgorithm(req.Graph, req.Algorithm)

	span.SetAttributes(
		attribute.Bool("compatible", result.IsCompatible),
		attribute.Int("issues", len(result.Issues)),
	)

	return result, nil
}

// ValidateAll выполняет полную валидацию
func (s *ValidationService) ValidateAll(
	ctx context.Context,
	req *validationv1.ValidateAllRequest,
) (*validationv1.ValidateAllResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "ValidationService.ValidateAll",
		trace.WithAttributes(
			attribute.String("level", req.Level.String()),
		),
	)
	defer span.End()

	start := time.Now()

	// Валидация графа
	graphResp, err := s.ValidateGraph(ctx, &validationv1.ValidateGraphRequest{
		Graph:              req.Graph,
		Level:              req.Level,
		CheckConnectivity:  true,
		CheckBusinessRules: true,
		CheckTopology:      req.Level >= validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
	})
	if err != nil {
		logger.Log.Warn("Graph validation failed in ValidateAll", "error", err)
	}

	// Валидация потока
	flowResp, err := s.ValidateFlow(ctx, &validationv1.ValidateFlowRequest{
		Graph: req.Graph,
	})
	if err != nil {
		logger.Log.Warn("Flow validation failed in ValidateAll", "error", err)
	}

	// Валидация алгоритма (если указан)
	var algoResp *validationv1.ValidateForAlgorithmResponse
	if req.Algorithm != commonv1.Algorithm_ALGORITHM_UNSPECIFIED {
		algoResp, err = s.ValidateForAlgorithm(ctx, &validationv1.ValidateForAlgorithmRequest{
			Graph:     req.Graph,
			Algorithm: req.Algorithm,
		})
		if err != nil {
			logger.Log.Warn("Algorithm validation failed in ValidateAll", "error", err)
		}
	}

	isValid := graphResp != nil && graphResp.Result.IsValid &&
		flowResp != nil && flowResp.IsValid
	if algoResp != nil {
		isValid = isValid && algoResp.IsCompatible
	}

	span.SetAttributes(
		attribute.Bool("valid", isValid),
	)

	telemetry.AddEvent(ctx, "full_validation_completed",
		attribute.Bool("graph_valid", graphResp != nil && graphResp.Result.IsValid),
		attribute.Bool("flow_valid", flowResp != nil && flowResp.IsValid),
	)

	return &validationv1.ValidateAllResponse{
		IsValid:             isValid,
		GraphValidation:     graphResp,
		FlowValidation:      flowResp,
		AlgorithmValidation: algoResp,
		Metrics: &validationv1.ValidationMetrics{
			DurationMs: float64(time.Since(start).Milliseconds()),
		},
	}, nil
}

// Health возвращает статус сервиса
func (s *ValidationService) Health(
	ctx context.Context,
	_ *validationv1.HealthRequest,
) (*validationv1.HealthResponse, error) {
	_, span := telemetry.StartSpan(ctx, "ValidationService.Health")
	defer span.End()

	return &validationv1.HealthResponse{
		Status:        "SERVING",
		Version:       s.version,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
	}, nil
}

func buildMetrics(total, passed, failed, warnings int32, start time.Time) *validationv1.ValidationMetrics {
	return &validationv1.ValidationMetrics{
		TotalChecks:   total,
		PassedChecks:  passed,
		FailedChecks:  failed,
		WarningChecks: warnings,
		DurationMs:    float64(time.Since(start).Microseconds()) / 1000.0,
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
