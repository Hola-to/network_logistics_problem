package v1_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "logistics/gen/go/logistics/common/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/tests/integration/testutil"
)

func TestSolverService_Solve(t *testing.T) {
	client := SetupSolverClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	tests := []struct {
		name      string
		graph     *commonv1.Graph
		algorithm commonv1.Algorithm
		wantFlow  float64
		wantErr   bool
	}{
		{
			name:      "simple graph with Dinic",
			graph:     CreateSimpleGraph(),
			algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
			wantFlow:  15,
			wantErr:   false,
		},
		{
			name:      "simple graph with Edmonds-Karp",
			graph:     CreateSimpleGraph(),
			algorithm: commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			wantFlow:  15,
			wantErr:   false,
		},
		{
			name:      "simple graph with Push-Relabel",
			graph:     CreateSimpleGraph(),
			algorithm: commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
			wantFlow:  15,
			wantErr:   false,
		},
		{
			name:      "larger graph",
			graph:     CreateLargeGraph(10),
			algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
			wantFlow:  10, // Expected based on graph structure
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Solve(ctx, &optimizationv1.SolveRequest{
				Graph:     tt.graph,
				Algorithm: tt.algorithm,
			})

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.True(t, resp.Success)
			assert.NotNil(t, resp.Result)
			assert.Equal(t, tt.wantFlow, resp.Result.MaxFlow)
			assert.NotNil(t, resp.SolvedGraph)
			assert.NotNil(t, resp.Metrics)
			assert.Greater(t, resp.Metrics.ComputationTimeMs, 0.0)
		})
	}
}

func TestSolverService_SolveWithOptions(t *testing.T) {
	client := SetupSolverClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.Solve(ctx, &optimizationv1.SolveRequest{
		Graph:     CreateSimpleGraph(),
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
		Options: &optimizationv1.SolveOptions{
			TimeoutSeconds: 30,
			ReturnPaths:    true,
			MaxIterations:  1000,
			Epsilon:        1e-9,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestSolverService_SolveStream(t *testing.T) {
	client := SetupSolverClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	stream, err := client.SolveStream(ctx, &optimizationv1.SolveRequestForBigGraphs{
		Graph:     CreateLargeGraph(20),
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})
	require.NoError(t, err)

	var lastProgress *optimizationv1.SolveProgress
	progressCount := 0

	for {
		progress, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		progressCount++
		lastProgress = progress

		// Verify progress structure
		assert.GreaterOrEqual(t, progress.CurrentFlow, 0.0)
		assert.GreaterOrEqual(t, progress.ProgressPercent, 0.0)
		assert.LessOrEqual(t, progress.ProgressPercent, 100.0)
	}

	assert.Greater(t, progressCount, 0)
	require.NotNil(t, lastProgress)
	assert.Equal(t, "completed", lastProgress.Status)
}

func TestSolverService_GetAlgorithms(t *testing.T) {
	client := SetupSolverClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.GetAlgorithms(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Algorithms)

	// Verify all expected algorithms are present
	expectedAlgorithms := map[commonv1.Algorithm]bool{
		commonv1.Algorithm_ALGORITHM_DINIC:        false,
		commonv1.Algorithm_ALGORITHM_EDMONDS_KARP: false,
		commonv1.Algorithm_ALGORITHM_PUSH_RELABEL: false,
		commonv1.Algorithm_ALGORITHM_MIN_COST:     false,
	}

	for _, alg := range resp.Algorithms {
		if _, ok := expectedAlgorithms[alg.Algorithm]; ok {
			expectedAlgorithms[alg.Algorithm] = true
		}
		// Verify algorithm info
		assert.NotEmpty(t, alg.Name)
		assert.NotEmpty(t, alg.Description)
		assert.NotEmpty(t, alg.TimeComplexity)
	}

	for alg, found := range expectedAlgorithms {
		assert.True(t, found, "algorithm %s not found", alg)
	}
}

func TestSolverService_SolveInvalidGraph(t *testing.T) {
	client := SetupSolverClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	// Test with nil graph
	_, err := client.Solve(ctx, &optimizationv1.SolveRequest{
		Graph:     nil,
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})
	require.Error(t, err)

	// Test with invalid source/sink
	_, err = client.Solve(ctx, &optimizationv1.SolveRequest{
		Graph:     CreateInvalidGraph(),
		Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
	})
	require.Error(t, err)
}

func TestSolverService_MinCostFlow(t *testing.T) {
	client := SetupSolverClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.Solve(ctx, &optimizationv1.SolveRequest{
		Graph:     CreateSimpleGraph(),
		Algorithm: commonv1.Algorithm_ALGORITHM_MIN_COST,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Result)
	assert.Greater(t, resp.Result.MaxFlow, 0.0)
	assert.Greater(t, resp.Result.TotalCost, 0.0)
}
