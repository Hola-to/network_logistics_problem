// services/gateway-svc/internal/handlers/audit_test.go

package handlers

import (
	"testing"

	auditv1 "logistics/gen/go/logistics/audit/v1"
)

func TestAuditHandler_ConvertEntry(t *testing.T) {
	h := &AuditHandler{}

	// Test nil input
	result := h.convertEntry(nil)
	if result != nil {
		t.Error("convertEntry(nil) should return nil")
	}

	// Test valid input
	entry := &auditv1.AuditEntry{
		Id:           "audit-123",
		Service:      "test-service",
		Method:       "TestMethod",
		Action:       auditv1.AuditAction_AUDIT_ACTION_CREATE,
		Outcome:      auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS,
		UserId:       "user-456",
		Username:     "testuser",
		ClientIp:     "192.168.1.1",
		ResourceType: "Document",
		ResourceId:   "doc-789",
		DurationMs:   150,
		Metadata:     map[string]string{"key": "value"},
	}

	result = h.convertEntry(entry)
	if result == nil {
		t.Fatal("convertEntry should not return nil for valid input")
	}

	if result.Id != "audit-123" {
		t.Errorf("Id = %v, want 'audit-123'", result.Id)
	}
	if result.Service != "test-service" {
		t.Errorf("Service = %v, want 'test-service'", result.Service)
	}
	if result.Action != "CREATE" {
		t.Errorf("Action = %v, want 'CREATE'", result.Action)
	}
	if result.Outcome != "SUCCESS" {
		t.Errorf("Outcome = %v, want 'SUCCESS'", result.Outcome)
	}
}

func TestAuditHandler_ConvertActivitySummary(t *testing.T) {
	h := &AuditHandler{}

	// Test nil input
	result := h.convertActivitySummary(nil)
	if result != nil {
		t.Error("convertActivitySummary(nil) should return nil")
	}

	// Test valid input
	summary := &auditv1.UserActivitySummary{
		TotalActions:      100,
		SuccessfulActions: 90,
		FailedActions:     8,
		DeniedActions:     2,
		ActionsByType:     map[string]int32{"CREATE": 50, "READ": 50},
		ActionsByService:  map[string]int32{"service1": 60, "service2": 40},
	}

	result = h.convertActivitySummary(summary)
	if result == nil {
		t.Fatal("convertActivitySummary should not return nil for valid input")
	}

	if result.TotalActions != 100 {
		t.Errorf("TotalActions = %d, want 100", result.TotalActions)
	}
	if result.SuccessfulActions != 90 {
		t.Errorf("SuccessfulActions = %d, want 90", result.SuccessfulActions)
	}
}

func TestAuditHandler_AllActions(t *testing.T) {
	h := &AuditHandler{}

	allActions := []string{
		"CREATE", "READ", "UPDATE", "DELETE",
		"LOGIN", "LOGOUT", "SOLVE", "ANALYZE",
		"VALIDATE", "EXPORT", "UNKNOWN",
	}

	for _, action := range allActions {
		result := h.parseAction(action)
		// Should not panic and should return a valid enum value
		if result < 0 {
			t.Errorf("parseAction(%s) returned invalid enum value", action)
		}
	}
}
