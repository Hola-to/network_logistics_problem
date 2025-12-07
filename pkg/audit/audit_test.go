// Package audit provides tests for the audit logging components.
package audit

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNewEntry verifies that the Builder correctly constructs an Entry with all fields set.
func TestNewEntry(t *testing.T) {
	entry := NewEntry().
		Service("test-service").
		Method("/test.Method").
		Action(ActionCreate).
		Outcome(OutcomeSuccess).
		User("user-123", "testuser").
		Client("127.0.0.1", "test-agent").
		Resource("graph", "graph-456").
		RequestID("req-789").
		Duration(100*time.Millisecond).
		Meta("key1", "value1").
		Build()

	if entry.Service != "test-service" {
		t.Errorf("expected service 'test-service', got %s", entry.Service)
	}
	if entry.Method != "/test.Method" {
		t.Errorf("expected method '/test.Method', got %s", entry.Method)
	}
	if entry.Action != ActionCreate {
		t.Errorf("expected action CREATE, got %s", entry.Action)
	}
	if entry.Outcome != OutcomeSuccess {
		t.Errorf("expected outcome SUCCESS, got %s", entry.Outcome)
	}
	if entry.UserID != "user-123" {
		t.Errorf("expected userID 'user-123', got %s", entry.UserID)
	}
	if entry.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %s", entry.Username)
	}
	if entry.ClientIP != "127.0.0.1" {
		t.Errorf("expected clientIP '127.0.0.1', got %s", entry.ClientIP)
	}
	if entry.Resource != "graph" {
		t.Errorf("expected resource 'graph', got %s", entry.Resource)
	}
	if entry.ResourceID != "graph-456" {
		t.Errorf("expected resourceID 'graph-456', got %s", entry.ResourceID)
	}
	if entry.RequestID != "req-789" {
		t.Errorf("expected requestID 'req-789', got %s", entry.RequestID)
	}
	if entry.DurationMs != 100 {
		t.Errorf("expected durationMs 100, got %d", entry.DurationMs)
	}
	if entry.Metadata["key1"] != "value1" {
		t.Errorf("expected metadata key1='value1', got %v", entry.Metadata["key1"])
	}
	if entry.ID == "" {
		t.Error("expected ID to be generated")
	}
}

// TestBuilder_Error verifies that the Error method correctly sets error fields on an Entry.
func TestBuilder_Error(t *testing.T) {
	entry := NewEntry().
		Service("test").
		Method("/test").
		Action(ActionRead).
		Outcome(OutcomeFailure).
		Error("NOT_FOUND", "resource not found").
		Build()

	if entry.ErrorCode != "NOT_FOUND" {
		t.Errorf("expected errorCode 'NOT_FOUND', got %s", entry.ErrorCode)
	}
	if entry.ErrorMessage != "resource not found" {
		t.Errorf("expected errorMessage 'resource not found', got %s", entry.ErrorMessage)
	}
}

// TestBuilder_Changes verifies that the Changes method correctly sets the ChangeSet on an Entry.
func TestBuilder_Changes(t *testing.T) {
	changes := &ChangeSet{
		Before: map[string]any{"status": "pending"},
		After:  map[string]any{"status": "completed"},
		Fields: []string{"status"},
	}

	entry := NewEntry().
		Service("test").
		Changes(changes).
		Build()

	if entry.Changes == nil {
		t.Fatal("expected changes to be set")
	}
	if entry.Changes.Before["status"] != "pending" {
		t.Errorf("expected before status 'pending', got %v", entry.Changes.Before["status"])
	}
	if entry.Changes.After["status"] != "completed" {
		t.Errorf("expected after status 'completed', got %v", entry.Changes.After["status"])
	}
}

// TestEntry_MarshalJSON verifies that Entry can be marshaled and unmarshaled to/from JSON correctly.
func TestEntry_MarshalJSON(t *testing.T) {
	entry := NewEntry().
		Service("test-service").
		Method("/test.Method").
		Action(ActionSolve).
		Outcome(OutcomeSuccess).
		Build()

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal entry: %v", err)
	}

	var decoded Entry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal entry: %v", err)
	}

	if decoded.Service != entry.Service {
		t.Errorf("expected service %s, got %s", entry.Service, decoded.Service)
	}
	if decoded.Action != entry.Action {
		t.Errorf("expected action %s, got %s", entry.Action, decoded.Action)
	}
}

// TestDefaultConfig verifies that DefaultConfig returns a Config with expected default values.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("expected enabled to be true by default")
	}
	if cfg.Backend != "stdout" {
		t.Errorf("expected backend 'stdout', got %s", cfg.Backend)
	}
	if cfg.BufferSize != 1000 {
		t.Errorf("expected buffer size 1000, got %d", cfg.BufferSize)
	}
	if cfg.FlushPeriod != 5*time.Second {
		t.Errorf("expected flush period 5s, got %v", cfg.FlushPeriod)
	}
	if len(cfg.MaskFields) == 0 {
		t.Error("expected mask fields to be set")
	}
}

// TestAction_Constants verifies the string representation of Action constants.
func TestAction_Constants(t *testing.T) {
	actions := []struct {
		action   Action
		expected string
	}{
		{ActionCreate, "CREATE"},
		{ActionRead, "READ"},
		{ActionUpdate, "UPDATE"},
		{ActionDelete, "DELETE"},
		{ActionLogin, "LOGIN"},
		{ActionLogout, "LOGOUT"},
		{ActionSolve, "SOLVE"},
		{ActionAnalyze, "ANALYZE"},
	}

	for _, tc := range actions {
		if string(tc.action) != tc.expected {
			t.Errorf("expected action %s, got %s", tc.expected, tc.action)
		}
	}
}

// TestOutcome_Constants verifies the string representation of Outcome constants.
func TestOutcome_Constants(t *testing.T) {
	outcomes := []struct {
		outcome  Outcome
		expected string
	}{
		{OutcomeSuccess, "SUCCESS"},
		{OutcomeFailure, "FAILURE"},
		{OutcomeDenied, "DENIED"},
	}

	for _, tc := range outcomes {
		if string(tc.outcome) != tc.expected {
			t.Errorf("expected outcome %s, got %s", tc.expected, tc.outcome)
		}
	}
}

// TestQueryFilter verifies the initialization and basic fields of QueryFilter.
func TestQueryFilter(t *testing.T) {
	now := time.Now()
	filter := &QueryFilter{
		StartTime:  &now,
		EndTime:    &now,
		Service:    "test",
		Method:     "/test.Method",
		Action:     ActionCreate,
		Outcome:    OutcomeSuccess,
		UserID:     "user-123",
		Resource:   "graph",
		ResourceID: "graph-456",
		Limit:      100,
		Offset:     0,
	}

	if filter.Service != "test" {
		t.Errorf("expected service 'test', got %s", filter.Service)
	}
	if filter.Limit != 100 {
		t.Errorf("expected limit 100, got %d", filter.Limit)
	}
}

// TestGenerateID verifies that generateID produces a non-empty and reasonably structured ID.
func TestGenerateID(t *testing.T) {
	id1 := generateID()

	if id1 == "" {
		t.Error("expected non-empty ID")
	}
	if len(id1) < 10 {
		t.Error("expected ID to have reasonable length")
	}

	// IDs should contain timestamp prefix
	if len(id1) < 14 {
		t.Error("expected ID to contain timestamp")
	}
}
