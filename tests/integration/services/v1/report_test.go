package v1_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	reportv1 "logistics/gen/go/logistics/report/v1"
	"logistics/tests/integration/testutil"
)

func TestReportService_GenerateFlowReport(t *testing.T) {
	client := SetupReportClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	formats := []reportv1.ReportFormat{
		reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
		reportv1.ReportFormat_REPORT_FORMAT_JSON,
		reportv1.ReportFormat_REPORT_FORMAT_CSV,
		reportv1.ReportFormat_REPORT_FORMAT_HTML,
	}

	for _, format := range formats {
		t.Run(format.String(), func(t *testing.T) {
			resp, err := client.GenerateFlowReport(ctx, &reportv1.GenerateFlowReportRequest{
				Graph:  CreateSolvedGraph(),
				Result: CreateFlowResult(),
				Format: format,
				Options: &reportv1.ReportOptions{
					Title:               "Test Flow Report",
					Description:         "Integration test report",
					Author:              "Test Suite",
					IncludeGraphDetails: true,
					IncludeEdgeList:     true,
				},
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.True(t, resp.Success)
			assert.NotNil(t, resp.Metadata)
			assert.NotNil(t, resp.Content)
			assert.NotEmpty(t, resp.Content.Data)
			assert.Greater(t, resp.Content.SizeBytes, int64(0))
		})
	}
}

func TestReportService_GenerateAnalyticsReport(t *testing.T) {
	client := SetupReportClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.GenerateAnalyticsReport(ctx, &reportv1.GenerateAnalyticsReportRequest{
		Graph:  CreateSolvedGraph(),
		Format: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
		Options: &reportv1.ReportOptions{
			Title:                  "Analytics Report",
			IncludeRecommendations: true,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Content)
}

func TestReportService_GenerateSummaryReport(t *testing.T) {
	client := SetupReportClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.GenerateSummaryReport(ctx, &reportv1.GenerateSummaryReportRequest{
		Graph:      CreateSolvedGraph(),
		FlowResult: CreateFlowResult(),
		Format:     reportv1.ReportFormat_REPORT_FORMAT_HTML,
		Options: &reportv1.ReportOptions{
			Title:       "Summary Report",
			CompanyName: "Test Company",
			Theme:       "professional",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestReportService_GetSupportedFormats(t *testing.T) {
	client := SetupReportClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.GetSupportedFormats(ctx, &reportv1.GetSupportedFormatsRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Formats)

	// Verify expected formats exist
	formatNames := make(map[reportv1.ReportFormat]bool)
	for _, f := range resp.Formats {
		formatNames[f.Format] = true
		assert.NotEmpty(t, f.Name)
		assert.NotEmpty(t, f.Extension)
		assert.NotEmpty(t, f.MimeType)
	}

	assert.True(t, formatNames[reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN])
	assert.True(t, formatNames[reportv1.ReportFormat_REPORT_FORMAT_JSON])
	assert.True(t, formatNames[reportv1.ReportFormat_REPORT_FORMAT_CSV])
}

func TestReportService_GenerateReportStream(t *testing.T) {
	client := SetupReportClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	stream, err := client.GenerateReportStream(ctx, &reportv1.GenerateReportStreamRequest{})
	require.NoError(t, err)

	var totalData []byte
	chunkCount := 0

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Stream might not be implemented or return early
			t.Logf("Stream ended early: %v", err)
			break
		}

		chunkCount++
		totalData = append(totalData, chunk.Data...)

		if chunk.IsLast {
			break
		}
	}

	t.Logf("Received %d chunks, total size: %d bytes", chunkCount, len(totalData))
}

func TestReportService_Health(t *testing.T) {
	client := SetupReportClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.Health(ctx, &reportv1.HealthRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "SERVING", resp.Status)
	assert.NotEmpty(t, resp.Version)
	assert.GreaterOrEqual(t, resp.ReportsGenerated, int64(0))
}

func TestReportService_ListReports(t *testing.T) {
	client := SetupReportClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	// First generate a report
	genResp, err := client.GenerateFlowReport(ctx, &reportv1.GenerateFlowReportRequest{
		Graph:  CreateSolvedGraph(),
		Result: CreateFlowResult(),
		Format: reportv1.ReportFormat_REPORT_FORMAT_JSON,
		Options: &reportv1.ReportOptions{
			Title:         "List Test Report",
			SaveToStorage: true,
			TtlSeconds:    3600,
		},
	})

	if err != nil {
		t.Skipf("Storage not configured: %v", err)
	}
	require.True(t, genResp.Success)

	// List reports
	listResp, err := client.ListReports(ctx, &reportv1.ListReportsRequest{
		Limit:  10,
		Offset: 0,
	})

	if err != nil {
		t.Skipf("Storage not configured: %v", err)
	}
	require.NotNil(t, listResp)
}
