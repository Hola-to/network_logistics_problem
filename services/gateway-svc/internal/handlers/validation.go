package handlers

import (
	"context"

	"connectrpc.com/connect"

	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/services/gateway-svc/internal/clients"
)

// ValidationHandler обработчики валидации
type ValidationHandler struct {
	clients *clients.Manager
}

// NewValidationHandler создаёт handler
func NewValidationHandler(clients *clients.Manager) *ValidationHandler {
	return &ValidationHandler{clients: clients}
}

func (h *ValidationHandler) ValidateGraph(
	ctx context.Context,
	req *connect.Request[gatewayv1.ValidateGraphRequest],
) (*connect.Response[gatewayv1.ValidateGraphResponse], error) {
	msg := req.Msg

	level := validationv1.ValidationLevel(msg.Level)
	resp, err := h.clients.Validation().Raw().ValidateGraph(ctx, &validationv1.ValidateGraphRequest{
		Graph:              msg.Graph,
		Level:              level,
		CheckConnectivity:  msg.CheckConnectivity,
		CheckBusinessRules: msg.CheckBusinessRules,
		CheckTopology:      msg.CheckTopology,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.ValidateGraphResponse{
		IsValid:    resp.Result.IsValid,
		Errors:     resp.Result.Errors,
		Warnings:   resp.Warnings,
		Statistics: resp.Statistics,
		Metrics: &gatewayv1.ValidationMetrics{
			TotalChecks:  resp.Metrics.TotalChecks,
			PassedChecks: resp.Metrics.PassedChecks,
			FailedChecks: resp.Metrics.FailedChecks,
			DurationMs:   resp.Metrics.DurationMs,
		},
	}), nil
}

func (h *ValidationHandler) ValidateForAlgorithm(
	ctx context.Context,
	req *connect.Request[gatewayv1.ValidateForAlgorithmRequest],
) (*connect.Response[gatewayv1.ValidateForAlgorithmResponse], error) {
	msg := req.Msg

	resp, err := h.clients.Validation().ValidateForAlgorithm(ctx, msg.Graph, msg.Algorithm)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var complexity *gatewayv1.AlgorithmComplexityEstimate
	if resp.Complexity != nil {
		complexity = &gatewayv1.AlgorithmComplexityEstimate{
			TimeComplexity:      resp.Complexity.TimeComplexity,
			SpaceComplexity:     resp.Complexity.SpaceComplexity,
			EstimatedIterations: resp.Complexity.EstimatedIterations,
			Recommendation:      resp.Complexity.Recommendation,
		}
	}

	return connect.NewResponse(&gatewayv1.ValidateForAlgorithmResponse{
		IsCompatible:    resp.IsCompatible,
		Issues:          resp.Issues,
		Recommendations: resp.Recommendations,
		Complexity:      complexity,
	}), nil
}
