package v1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "logistics/gen/go/logistics/common/v1"
	validationv1 "logistics/gen/go/logistics/validation/v1"
	"logistics/tests/integration/testutil"
)

func TestValidationService_ValidateGraph(t *testing.T) {
	client := SetupValidationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	tests := []struct {
		name      string
		graph     *commonv1.Graph
		level     validationv1.ValidationLevel
		wantValid bool
	}{
		{
			name:      "valid simple graph",
			graph:     CreateSimpleGraph(),
			level:     validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
			wantValid: true,
		},
		{
			name:      "valid graph with full validation",
			graph:     CreateSimpleGraph(),
			level:     validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
			wantValid: true,
		},
		{
			name:      "invalid graph - bad sink",
			graph:     CreateInvalidGraph(),
			level:     validationv1.ValidationLevel_VALIDATION_LEVEL_BASIC,
			wantValid: false,
		},
		{
			name:      "disconnected graph",
			graph:     CreateDisconnectedGraph(),
			level:     validationv1.ValidationLevel_VALIDATION_LEVEL_STANDARD,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.ValidateGraph(ctx, &validationv1.ValidateGraphRequest{
				Graph:              tt.graph,
				Level:              tt.level,
				CheckConnectivity:  true,
				CheckBusinessRules: true,
				CheckTopology:      true,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.Result)
			assert.Equal(t, tt.wantValid, resp.Result.IsValid)

			if !tt.wantValid {
				assert.NotEmpty(t, resp.Result.Errors)
			}

			assert.NotNil(t, resp.Metrics)
			assert.Greater(t, resp.Metrics.TotalChecks, int32(0))
		})
	}
}

func TestValidationService_ValidateFlow(t *testing.T) {
	client := SetupValidationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	tests := []struct {
		name            string
		graph           *commonv1.Graph
		expectedMaxFlow float64
		wantValid       bool
	}{
		{
			name:            "valid flow",
			graph:           CreateSolvedGraph(),
			expectedMaxFlow: 15,
			wantValid:       true,
		},
		{
			name:            "flow mismatch",
			graph:           CreateSolvedGraph(),
			expectedMaxFlow: 100, // Wrong expected
			wantValid:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.ValidateFlow(ctx, &validationv1.ValidateFlowRequest{
				Graph:           tt.graph,
				ExpectedMaxFlow: tt.expectedMaxFlow,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantValid, resp.IsValid)

			if !tt.wantValid {
				assert.NotEmpty(t, resp.Violations)
			}

			assert.NotNil(t, resp.Summary)
		})
	}
}

func TestValidationService_ValidateForAlgorithm(t *testing.T) {
	client := SetupValidationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	tests := []struct {
		name       string
		graph      *commonv1.Graph
		algorithm  commonv1.Algorithm
		wantCompat bool
	}{
		{
			name:       "compatible with Dinic",
			graph:      CreateSimpleGraph(),
			algorithm:  commonv1.Algorithm_ALGORITHM_DINIC,
			wantCompat: true,
		},
		{
			name:       "compatible with Edmonds-Karp",
			graph:      CreateSimpleGraph(),
			algorithm:  commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			wantCompat: true,
		},
		{
			name:       "compatible with Min-Cost-Flow",
			graph:      CreateSimpleGraph(),
			algorithm:  commonv1.Algorithm_ALGORITHM_MIN_COST,
			wantCompat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.ValidateForAlgorithm(ctx, &validationv1.ValidateForAlgorithmRequest{
				Graph:     tt.graph,
				Algorithm: tt.algorithm,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantCompat, resp.IsCompatible)

			if resp.Complexity != nil {
				assert.NotEmpty(t, resp.Complexity.TimeComplexity)
			}
		})
	}
}

func TestValidationService_ValidateAll(t *testing.T) {
	client := SetupValidationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.ValidateAll(ctx, &validationv1.ValidateAllRequest{
		Graph:     CreateSimpleGraph(),
		Level:     validationv1.ValidationLevel_VALIDATION_LEVEL_FULL,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.IsValid)
	assert.NotNil(t, resp.GraphValidation)
	assert.NotNil(t, resp.FlowValidation)
	assert.NotNil(t, resp.AlgorithmValidation)
	assert.NotNil(t, resp.Metrics)
}

func TestValidationService_Health(t *testing.T) {
	client := SetupValidationClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.Health(ctx, &validationv1.HealthRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "SERVING", resp.Status)
	assert.NotEmpty(t, resp.Version)
	assert.GreaterOrEqual(t, resp.UptimeSeconds, int64(0))
}
