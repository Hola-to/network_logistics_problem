package v1_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	auditv1 "logistics/gen/go/logistics/audit/v1"
	commonv1 "logistics/gen/go/logistics/common/v1"
	"logistics/tests/integration/testutil"
)

func TestAuditService_LogEvent(t *testing.T) {
	client := SetupAuditClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.LogEvent(ctx, &auditv1.LogEventRequest{
		Entry: &auditv1.AuditEntry{
			Timestamp:    timestamppb.Now(),
			Service:      "test-service",
			Method:       "TestMethod",
			Action:       auditv1.AuditAction_AUDIT_ACTION_CREATE,
			Outcome:      auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS,
			UserId:       "test-user-" + testutil.RandomString(8),
			Username:     "testuser",
			ResourceType: "test",
			ResourceId:   testutil.RandomString(8),
			DurationMs:   100,
			Metadata: map[string]string{
				"test": "value",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.EventId)
}

func TestAuditService_LogEventBatch(t *testing.T) {
	client := SetupAuditClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	entries := make([]*auditv1.AuditEntry, 5)
	for i := 0; i < 5; i++ {
		entries[i] = &auditv1.AuditEntry{
			Timestamp:    timestamppb.Now(),
			Service:      "test-service",
			Method:       "BatchTestMethod",
			Action:       auditv1.AuditAction_AUDIT_ACTION_READ,
			Outcome:      auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS,
			UserId:       "batch-user-" + testutil.RandomString(8),
			ResourceType: "batch-test",
			ResourceId:   testutil.RandomString(8),
		}
	}

	resp, err := client.LogEventBatch(ctx, &auditv1.LogEventBatchRequest{
		Entries: entries,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int32(5), resp.LoggedCount)
	assert.Equal(t, int32(0), resp.FailedCount)
}

func TestAuditService_GetAuditLogs(t *testing.T) {
	client := SetupAuditClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	service := "getlogs-service-" + testutil.RandomString(8)

	// Log some events first
	for i := 0; i < 3; i++ {
		_, err := client.LogEvent(ctx, &auditv1.LogEventRequest{
			Entry: &auditv1.AuditEntry{
				Timestamp:    timestamppb.Now(),
				Service:      service,
				Method:       "GetLogsTest",
				Action:       auditv1.AuditAction_AUDIT_ACTION_READ,
				Outcome:      auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS,
				UserId:       "logs-user",
				ResourceType: "test",
			},
		})
		require.NoError(t, err)
	}

	// Get logs
	resp, err := client.GetAuditLogs(ctx, &auditv1.GetAuditLogsRequest{
		Filter: &auditv1.AuditFilter{
			Services: []string{service},
		},
		Pagination: &commonv1.PaginationRequest{
			Page:     1,
			PageSize: 10,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Entries), 3)
}

func TestAuditService_GetResourceHistory(t *testing.T) {
	client := SetupAuditClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resourceType := "test-resource"
	resourceID := "res-" + testutil.RandomString(8)

	// Create history for resource
	actions := []auditv1.AuditAction{
		auditv1.AuditAction_AUDIT_ACTION_CREATE,
		auditv1.AuditAction_AUDIT_ACTION_UPDATE,
		auditv1.AuditAction_AUDIT_ACTION_READ,
	}

	for _, action := range actions {
		_, err := client.LogEvent(ctx, &auditv1.LogEventRequest{
			Entry: &auditv1.AuditEntry{
				Timestamp:    timestamppb.Now(),
				Service:      "history-test",
				Method:       "ResourceHistory",
				Action:       action,
				Outcome:      auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS,
				ResourceType: resourceType,
				ResourceId:   resourceID,
				UserId:       "history-user",
			},
		})
		require.NoError(t, err)
	}

	// Get resource history
	resp, err := client.GetResourceHistory(ctx, &auditv1.GetResourceHistoryRequest{
		ResourceType: resourceType,
		ResourceId:   resourceID,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Entries), 3)
	assert.NotNil(t, resp.Summary)
}

func TestAuditService_GetUserActivity(t *testing.T) {
	client := SetupAuditClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	userID := "activity-user-" + testutil.RandomString(8)

	// Create activity for user
	for i := 0; i < 5; i++ {
		_, err := client.LogEvent(ctx, &auditv1.LogEventRequest{
			Entry: &auditv1.AuditEntry{
				Timestamp:    timestamppb.Now(),
				Service:      "activity-test",
				Method:       "UserActivity",
				Action:       auditv1.AuditAction_AUDIT_ACTION_READ,
				Outcome:      auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS,
				UserId:       userID,
				ResourceType: "activity",
			},
		})
		require.NoError(t, err)
	}

	// Get user activity
	now := time.Now()
	resp, err := client.GetUserActivity(ctx, &auditv1.GetUserActivityRequest{
		UserId: userID,
		TimeRange: &commonv1.TimeRange{
			StartTimestamp: now.Add(-1 * time.Hour).Unix(),
			EndTimestamp:   now.Add(1 * time.Hour).Unix(),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Entries), 5)
	assert.NotNil(t, resp.Summary)
	assert.GreaterOrEqual(t, resp.Summary.TotalActions, int32(5))
}

func TestAuditService_GetAuditStats(t *testing.T) {
	client := SetupAuditClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	// Log some events
	for i := 0; i < 3; i++ {
		_, err := client.LogEvent(ctx, &auditv1.LogEventRequest{
			Entry: &auditv1.AuditEntry{
				Timestamp: timestamppb.Now(),
				Service:   "stats-test",
				Method:    "GetStats",
				Action:    auditv1.AuditAction_AUDIT_ACTION_ANALYZE,
				Outcome:   auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS,
				UserId:    "stats-user-" + testutil.RandomString(4),
			},
		})
		require.NoError(t, err)
	}

	now := time.Now()
	resp, err := client.GetAuditStats(ctx, &auditv1.GetAuditStatsRequest{
		TimeRange: &commonv1.TimeRange{
			StartTimestamp: now.Add(-24 * time.Hour).Unix(),
			EndTimestamp:   now.Add(1 * time.Hour).Unix(),
		},
		GroupBy: "service",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Summary)
	assert.Greater(t, resp.Summary.TotalEvents, int64(0))
}

func TestAuditService_Health(t *testing.T) {
	client := SetupAuditClient(t)
	ctx, cancel := testutil.Context(t)
	defer cancel()

	resp, err := client.Health(ctx, &auditv1.HealthRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "SERVING", resp.Status)
	assert.NotEmpty(t, resp.Version)
}
