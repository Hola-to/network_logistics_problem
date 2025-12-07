package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	auditv1 "logistics/gen/go/logistics/audit/v1"
	"logistics/services/audit-svc/internal/repository"
)

// Mock repository
type mockAuditRepository struct {
	entries map[string]*repository.AuditEntry
	nextID  int
}

func newMockAuditRepository() *mockAuditRepository {
	return &mockAuditRepository{
		entries: make(map[string]*repository.AuditEntry),
		nextID:  1,
	}
}

func (m *mockAuditRepository) Create(ctx context.Context, entry *repository.AuditEntry) error {
	entry.ID = fmt.Sprintf("audit-%d", m.nextID)
	m.nextID++
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockAuditRepository) CreateBatch(ctx context.Context, entries []*repository.AuditEntry) (int, error) {
	for _, entry := range entries {
		if err := m.Create(ctx, entry); err != nil {
			return 0, err
		}
	}
	return len(entries), nil
}

func (m *mockAuditRepository) GetByID(ctx context.Context, id string) (*repository.AuditEntry, error) {
	if entry, ok := m.entries[id]; ok {
		return entry, nil
	}
	return nil, repository.ErrAuditNotFound
}

func (m *mockAuditRepository) List(ctx context.Context, filter *repository.AuditFilter, opts *repository.ListOptions) ([]*repository.AuditEntry, int64, error) {
	result := make([]*repository.AuditEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		result = append(result, entry)
	}
	return result, int64(len(result)), nil
}

func (m *mockAuditRepository) GetResourceHistory(ctx context.Context, resourceType, resourceID string, opts *repository.ListOptions) ([]*repository.AuditEntry, *repository.ResourceSummary, int64, error) {
	return nil, &repository.ResourceSummary{}, 0, nil
}

func (m *mockAuditRepository) GetUserActivity(ctx context.Context, userID string, timeRange *repository.TimeRange, opts *repository.ListOptions) ([]*repository.AuditEntry, *repository.UserActivitySummary, int64, error) {
	return nil, &repository.UserActivitySummary{
		ActionsByType:    make(map[string]int),
		ActionsByService: make(map[string]int),
	}, 0, nil
}

func (m *mockAuditRepository) GetStats(ctx context.Context, timeRange *repository.TimeRange, groupBy string) (*repository.AuditStats, error) {
	return &repository.AuditStats{
		ByService: make(map[string]int64),
		ByAction:  make(map[string]int64),
		ByOutcome: make(map[string]int64),
	}, nil
}

func (m *mockAuditRepository) Count(ctx context.Context) (int64, error) {
	return int64(len(m.entries)), nil
}

func (m *mockAuditRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func TestAuditService_LogEvent(t *testing.T) {
	repo := newMockAuditRepository()
	svc := NewAuditService(repo, "1.0.0")
	ctx := context.Background()

	tests := []struct {
		name        string
		entry       *auditv1.AuditEntry
		wantSuccess bool
	}{
		{
			name: "successful log",
			entry: &auditv1.AuditEntry{
				Service:   "test-service",
				Method:    "TestMethod",
				Action:    auditv1.AuditAction_AUDIT_ACTION_CREATE,
				Outcome:   auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS,
				Timestamp: timestamppb.Now(),
			},
			wantSuccess: true,
		},
		{
			name:        "nil entry",
			entry:       nil,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.LogEvent(ctx, &auditv1.LogEventRequest{Entry: tt.entry})
			if err != nil {
				t.Fatalf("LogEvent() error = %v", err)
			}

			if resp.Success != tt.wantSuccess {
				t.Errorf("Success = %v, want %v", resp.Success, tt.wantSuccess)
			}
		})
	}
}

func TestAuditService_LogEventBatch(t *testing.T) {
	repo := newMockAuditRepository()
	svc := NewAuditService(repo, "1.0.0")
	ctx := context.Background()

	entries := []*auditv1.AuditEntry{
		{Service: "svc1", Method: "Method1", Action: auditv1.AuditAction_AUDIT_ACTION_READ},
		{Service: "svc2", Method: "Method2", Action: auditv1.AuditAction_AUDIT_ACTION_UPDATE},
	}

	resp, err := svc.LogEventBatch(ctx, &auditv1.LogEventBatchRequest{Entries: entries})
	if err != nil {
		t.Fatalf("LogEventBatch() error = %v", err)
	}

	if resp.LoggedCount != 2 {
		t.Errorf("LoggedCount = %d, want 2", resp.LoggedCount)
	}
	if resp.FailedCount != 0 {
		t.Errorf("FailedCount = %d, want 0", resp.FailedCount)
	}
}

func TestAuditService_Health(t *testing.T) {
	repo := newMockAuditRepository()
	svc := NewAuditService(repo, "2.0.0")
	ctx := context.Background()

	resp, err := svc.Health(ctx, &auditv1.HealthRequest{})
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if resp.Status != "SERVING" {
		t.Errorf("Status = %v, want SERVING", resp.Status)
	}
	if resp.Version != "2.0.0" {
		t.Errorf("Version = %v, want 2.0.0", resp.Version)
	}
}

func TestAuditService_ParseAction(t *testing.T) {
	svc := &AuditService{}

	tests := []struct {
		input    string
		expected auditv1.AuditAction
	}{
		{"CREATE", auditv1.AuditAction_AUDIT_ACTION_CREATE},
		{"AUDIT_ACTION_CREATE", auditv1.AuditAction_AUDIT_ACTION_CREATE},
		{"READ", auditv1.AuditAction_AUDIT_ACTION_READ},
		{"UPDATE", auditv1.AuditAction_AUDIT_ACTION_UPDATE},
		{"DELETE", auditv1.AuditAction_AUDIT_ACTION_DELETE},
		{"LOGIN", auditv1.AuditAction_AUDIT_ACTION_LOGIN},
		{"LOGOUT", auditv1.AuditAction_AUDIT_ACTION_LOGOUT},
		{"UNKNOWN", auditv1.AuditAction_AUDIT_ACTION_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := svc.parseAction(tt.input)
			if result != tt.expected {
				t.Errorf("parseAction(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAuditService_ParseOutcome(t *testing.T) {
	svc := &AuditService{}

	tests := []struct {
		input    string
		expected auditv1.AuditOutcome
	}{
		{"SUCCESS", auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS},
		{"AUDIT_OUTCOME_SUCCESS", auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS},
		{"FAILURE", auditv1.AuditOutcome_AUDIT_OUTCOME_FAILURE},
		{"DENIED", auditv1.AuditOutcome_AUDIT_OUTCOME_DENIED},
		{"ERROR", auditv1.AuditOutcome_AUDIT_OUTCOME_ERROR},
		{"UNKNOWN", auditv1.AuditOutcome_AUDIT_OUTCOME_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := svc.parseOutcome(tt.input)
			if result != tt.expected {
				t.Errorf("parseOutcome(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
