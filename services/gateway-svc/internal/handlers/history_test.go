// services/gateway-svc/internal/handlers/history_test.go

package handlers

import (
	"testing"
)

func TestNewHistoryHandler_NotNil(t *testing.T) {
	h := NewHistoryHandler(nil)
	if h == nil {
		t.Error("NewHistoryHandler should not return nil")
	}
}

func TestAnonymousUserID(t *testing.T) {
	if anonymousUserID != "anonymous" {
		t.Errorf("anonymousUserID = %q, want %q", anonymousUserID, "anonymous")
	}
}
