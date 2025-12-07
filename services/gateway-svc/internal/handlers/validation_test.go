// services/gateway-svc/internal/handlers/validation_test.go

package handlers

import (
	"testing"
)

func TestNewValidationHandler_NotNil(t *testing.T) {
	h := NewValidationHandler(nil)
	if h == nil {
		t.Error("NewValidationHandler should not return nil")
	}
	if h.clients != nil {
		t.Error("clients should be nil when passed nil")
	}
}
