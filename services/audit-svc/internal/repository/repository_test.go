package repository

import (
	"testing"
	"time"
)

func TestAuditEntry_Fields(t *testing.T) {
	entry := &AuditEntry{
		ID:           "audit-123",
		Timestamp:    time.Now(),
		Service:      "test-service",
		Method:       "TestMethod",
		RequestID:    "req-456",
		Action:       "CREATE",
		Outcome:      "SUCCESS",
		UserID:       "user-789",
		Username:     "testuser",
		UserRole:     "admin",
		ClientIP:     "192.168.1.1",
		UserAgent:    "TestAgent/1.0",
		ResourceType: "Document",
		ResourceID:   "doc-123",
		ResourceName: "Test Document",
		DurationMs:   150,
		ErrorCode:    "",
		ErrorMessage: "",
		Metadata:     map[string]string{"key": "value"},
	}

	if entry.ID != "audit-123" {
		t.Errorf("ID = %v, want audit-123", entry.ID)
	}
	if entry.Action != "CREATE" {
		t.Errorf("Action = %v, want CREATE", entry.Action)
	}
	if entry.Metadata["key"] != "value" {
		t.Error("Metadata not set correctly")
	}
}

func TestAuditFilter_Fields(t *testing.T) {
	filter := &AuditFilter{
		TimeRange: &TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
		Services: []string{"service1", "service2"},
		Methods:  []string{"Method1"},
		Actions:  []string{"CREATE", "UPDATE"},
		Outcomes: []string{"SUCCESS"},
		UserID:   "user-123",
	}

	if len(filter.Services) != 2 {
		t.Errorf("Services count = %d, want 2", len(filter.Services))
	}
	if filter.TimeRange == nil {
		t.Error("TimeRange should not be nil")
	}
}

func TestListOptions_Defaults(t *testing.T) {
	opts := &ListOptions{}

	// Check that zero values don't cause issues
	if opts.Limit != 0 {
		t.Errorf("Default Limit = %d, want 0", opts.Limit)
	}
	if opts.Offset != 0 {
		t.Errorf("Default Offset = %d, want 0", opts.Offset)
	}
}

func TestTimeRange_Validity(t *testing.T) {
	tests := []struct {
		name    string
		tr      *TimeRange
		isValid bool
	}{
		{
			name: "valid range",
			tr: &TimeRange{
				Start: time.Now().Add(-24 * time.Hour),
				End:   time.Now(),
			},
			isValid: true,
		},
		{
			name: "start after end",
			tr: &TimeRange{
				Start: time.Now(),
				End:   time.Now().Add(-24 * time.Hour),
			},
			isValid: false,
		},
		{
			name: "same start and end",
			tr: &TimeRange{
				Start: time.Now(),
				End:   time.Now(),
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := !tt.tr.Start.After(tt.tr.End)
			if valid != tt.isValid {
				t.Errorf("TimeRange validity = %v, want %v", valid, tt.isValid)
			}
		})
	}
}

func TestUserActivitySummary_Initialization(t *testing.T) {
	summary := &UserActivitySummary{
		TotalActions:      100,
		SuccessfulActions: 90,
		FailedActions:     8,
		DeniedActions:     2,
		ActionsByType:     map[string]int{"CREATE": 50, "READ": 50},
		ActionsByService:  map[string]int{"service1": 60, "service2": 40},
		FirstActivity:     time.Now().Add(-30 * 24 * time.Hour),
		LastActivity:      time.Now(),
	}

	if summary.TotalActions != 100 {
		t.Errorf("TotalActions = %d, want 100", summary.TotalActions)
	}

	// Verify counts add up
	total := summary.SuccessfulActions + summary.FailedActions + summary.DeniedActions
	if total != summary.TotalActions {
		t.Errorf("Action counts don't add up: %d != %d", total, summary.TotalActions)
	}
}

func TestAuditStats_Fields(t *testing.T) {
	stats := &AuditStats{
		TotalEvents:      1000,
		SuccessfulEvents: 900,
		FailedEvents:     80,
		DeniedEvents:     20,
		UniqueUsers:      50,
		UniqueResources:  200,
		AvgDurationMs:    125.5,
		ByService:        map[string]int64{"svc1": 500, "svc2": 500},
		ByAction:         map[string]int64{"CREATE": 300, "READ": 700},
		ByOutcome:        map[string]int64{"SUCCESS": 900, "FAILURE": 100},
		Timeline:         []TimelinePoint{},
		TopUsers:         []TopUser{{UserID: "u1", ActionCount: 100}},
		TopResources:     []TopResource{{ResourceType: "Doc", ActionCount: 50}},
	}

	if stats.TotalEvents != 1000 {
		t.Errorf("TotalEvents = %d, want 1000", stats.TotalEvents)
	}

	// Verify success + failure + denied equals total
	total := stats.SuccessfulEvents + stats.FailedEvents + stats.DeniedEvents
	if total != stats.TotalEvents {
		t.Errorf("Event counts don't add up: %d != %d", total, stats.TotalEvents)
	}
}

func TestResourceSummary_Fields(t *testing.T) {
	now := time.Now()
	summary := &ResourceSummary{
		CreatedAt:      now.Add(-24 * time.Hour),
		CreatedBy:      "user1",
		LastModifiedAt: now,
		LastModifiedBy: "user2",
		TotalChanges:   10,
	}

	if summary.TotalChanges != 10 {
		t.Errorf("TotalChanges = %d, want 10", summary.TotalChanges)
	}
	if summary.CreatedBy != "user1" {
		t.Errorf("CreatedBy = %s, want user1", summary.CreatedBy)
	}
	if summary.LastModifiedAt.Before(summary.CreatedAt) {
		t.Error("LastModifiedAt should be after CreatedAt")
	}
}

func TestTopUser_Fields(t *testing.T) {
	topUser := TopUser{
		UserID:      "user-123",
		Username:    "testuser",
		ActionCount: 500,
	}

	if topUser.ActionCount != 500 {
		t.Errorf("ActionCount = %d, want 500", topUser.ActionCount)
	}
}

func TestTopResource_Fields(t *testing.T) {
	topResource := TopResource{
		ResourceType: "Document",
		ResourceID:   "doc-456",
		ActionCount:  250,
	}

	if topResource.ResourceType != "Document" {
		t.Errorf("ResourceType = %s, want Document", topResource.ResourceType)
	}
}

func TestTimelinePoint_Fields(t *testing.T) {
	point := TimelinePoint{
		Timestamp:    time.Now(),
		Count:        100,
		SuccessCount: 90,
		FailureCount: 10,
	}

	if point.SuccessCount+point.FailureCount != point.Count {
		t.Error("Success + Failure should equal Count")
	}
}

func TestErrAuditNotFound(t *testing.T) {
	err := ErrAuditNotFound
	if err.Error() != "audit entry not found" {
		t.Errorf("Error message = %s, want 'audit entry not found'", err.Error())
	}
}
