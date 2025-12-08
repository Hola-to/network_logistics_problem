package v1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "logistics/gen/go/logistics/common/v1"
	historyv1 "logistics/gen/go/logistics/history/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/tests/integration/testutil"
)

func TestHistoryService_SaveCalculation(t *testing.T) {
	client := SetupHistoryClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	userID := "testuser_" + testutil.RandomString(8)

	resp, err := client.SaveCalculation(ctx, &historyv1.SaveCalculationRequest{
		UserId: userID,
		Name:   "Test Calculation",
		Request: &optimizationv1.SolveRequest{
			Graph:     CreateSimpleGraph(),
			Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		},
		Response: &optimizationv1.SolveResponse{
			Success: true,
			Result:  CreateFlowResult(),
		},
		Tags: map[string]string{
			"environment": "test",
			"type":        "integration",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.CalculationId)
	assert.NotNil(t, resp.CreatedAt)
}

func TestHistoryService_GetCalculation(t *testing.T) {
	client := SetupHistoryClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	userID := "testuser_" + testutil.RandomString(8)

	// Save first
	saveResp, err := client.SaveCalculation(ctx, &historyv1.SaveCalculationRequest{
		UserId: userID,
		Name:   "Get Test Calculation",
		Request: &optimizationv1.SolveRequest{
			Graph:     CreateSimpleGraph(),
			Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		},
		Response: &optimizationv1.SolveResponse{
			Success: true,
			Result:  CreateFlowResult(),
		},
	})
	require.NoError(t, err)

	// Then get
	getResp, err := client.GetCalculation(ctx, &historyv1.GetCalculationRequest{
		UserId:        userID,
		CalculationId: saveResp.CalculationId,
	})

	require.NoError(t, err)
	require.NotNil(t, getResp)
	require.NotNil(t, getResp.Record)
	assert.Equal(t, saveResp.CalculationId, getResp.Record.CalculationId)
	assert.Equal(t, "Get Test Calculation", getResp.Record.Name)
}

func TestHistoryService_ListCalculations(t *testing.T) {
	client := SetupHistoryClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	userID := "listuser_" + testutil.RandomString(8)

	// Create several calculations
	for i := 0; i < 5; i++ {
		_, err := client.SaveCalculation(ctx, &historyv1.SaveCalculationRequest{
			UserId: userID,
			Name:   "List Test " + testutil.RandomString(4),
			Request: &optimizationv1.SolveRequest{
				Graph:     CreateSimpleGraph(),
				Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
			},
			Response: &optimizationv1.SolveResponse{
				Success: true,
				Result:  CreateFlowResult(),
			},
		})
		require.NoError(t, err)
	}

	// List all
	listResp, err := client.ListCalculations(ctx, &historyv1.ListCalculationsRequest{
		UserId: userID,
		Pagination: &commonv1.PaginationRequest{
			Page:     1,
			PageSize: 10,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, listResp)
	assert.GreaterOrEqual(t, len(listResp.Calculations), 5)
	assert.NotNil(t, listResp.Pagination)
}

func TestHistoryService_ListCalculationsWithFilter(t *testing.T) {
	client := SetupHistoryClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	userID := "filteruser_" + testutil.RandomString(8)

	// Create calculations with different algorithms
	algorithms := []commonv1.Algorithm{
		commonv1.Algorithm_ALGORITHM_DINIC,
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
		commonv1.Algorithm_ALGORITHM_DINIC,
	}

	for _, alg := range algorithms {
		_, err := client.SaveCalculation(ctx, &historyv1.SaveCalculationRequest{
			UserId: userID,
			Name:   "Filter Test",
			Request: &optimizationv1.SolveRequest{
				Graph:     CreateSimpleGraph(),
				Algorithm: alg,
			},
			Response: &optimizationv1.SolveResponse{
				Success: true,
				Result:  CreateFlowResult(),
			},
		})
		require.NoError(t, err)
	}

	// Filter by algorithm
	listResp, err := client.ListCalculations(ctx, &historyv1.ListCalculationsRequest{
		UserId: userID,
		Filter: &historyv1.HistoryFilter{
			Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, listResp)
	assert.GreaterOrEqual(t, len(listResp.Calculations), 2)
}

func TestHistoryService_DeleteCalculation(t *testing.T) {
	client := SetupHistoryClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	userID := "deleteuser_" + testutil.RandomString(8)

	// Save
	saveResp, err := client.SaveCalculation(ctx, &historyv1.SaveCalculationRequest{
		UserId: userID,
		Name:   "To Delete",
		Request: &optimizationv1.SolveRequest{
			Graph:     CreateSimpleGraph(),
			Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		},
		Response: &optimizationv1.SolveResponse{
			Success: true,
			Result:  CreateFlowResult(),
		},
	})
	require.NoError(t, err)

	// Delete
	deleteResp, err := client.DeleteCalculation(ctx, &historyv1.DeleteCalculationRequest{
		UserId:        userID,
		CalculationId: saveResp.CalculationId,
	})

	require.NoError(t, err)
	require.NotNil(t, deleteResp)
	assert.True(t, deleteResp.Success)

	// Verify deletion
	_, err = client.GetCalculation(ctx, &historyv1.GetCalculationRequest{
		UserId:        userID,
		CalculationId: saveResp.CalculationId,
	})
	require.Error(t, err)
}

func TestHistoryService_GetStatistics(t *testing.T) {
	client := SetupHistoryClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	userID := "statsuser_" + testutil.RandomString(8)

	// Create some calculations
	for i := 0; i < 3; i++ {
		_, err := client.SaveCalculation(ctx, &historyv1.SaveCalculationRequest{
			UserId: userID,
			Name:   "Stats Test",
			Request: &optimizationv1.SolveRequest{
				Graph:     CreateSimpleGraph(),
				Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
			},
			Response: &optimizationv1.SolveResponse{
				Success: true,
				Result:  CreateFlowResult(),
			},
		})
		require.NoError(t, err)
	}

	// Get statistics
	statsResp, err := client.GetStatistics(ctx, &historyv1.GetStatisticsRequest{
		UserId: userID,
	})

	require.NoError(t, err)
	require.NotNil(t, statsResp)
	assert.GreaterOrEqual(t, statsResp.TotalCalculations, int32(3))
}
