// services/gateway-svc/internal/middleware/context_test.go

package middleware

import (
	"context"
	"testing"

	authv1 "logistics/gen/go/logistics/auth/v1"
)

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "with user id",
			ctx:      context.WithValue(context.Background(), userIDKey, "user-123"),
			expected: "user-123",
		},
		{
			name:     "with wrong type",
			ctx:      context.WithValue(context.Background(), userIDKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserID(tt.ctx)
			if result != tt.expected {
				t.Errorf("GetUserID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetUserInfo(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		expectNil bool
	}{
		{
			name:      "empty context",
			ctx:       context.Background(),
			expectNil: true,
		},
		{
			name: "with user info",
			ctx: context.WithValue(context.Background(), userInfoKey, &authv1.UserInfo{
				UserId:   "user-123",
				Username: "testuser",
			}),
			expectNil: false,
		},
		{
			name:      "with wrong type",
			ctx:       context.WithValue(context.Background(), userInfoKey, "not a user info"),
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserInfo(tt.ctx)
			if (result == nil) != tt.expectNil {
				t.Errorf("GetUserInfo() nil = %v, want nil = %v", result == nil, tt.expectNil)
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "with request id",
			ctx:      context.WithValue(context.Background(), requestIDKey, "req-456"),
			expected: "req-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRequestID(tt.ctx)
			if result != tt.expected {
				t.Errorf("GetRequestID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user-123"

	newCtx := WithUserID(ctx, userID)

	result := GetUserID(newCtx)
	if result != userID {
		t.Errorf("WithUserID() -> GetUserID() = %v, want %v", result, userID)
	}

	// Original context should not be modified
	if GetUserID(ctx) != "" {
		t.Error("Original context should not be modified")
	}
}

func TestWithUserInfo(t *testing.T) {
	ctx := context.Background()
	userInfo := &authv1.UserInfo{
		UserId:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "admin",
	}

	newCtx := WithUserInfo(ctx, userInfo)

	result := GetUserInfo(newCtx)
	if result == nil {
		t.Fatal("WithUserInfo() -> GetUserInfo() returned nil")
	}
	if result.UserId != userInfo.UserId {
		t.Errorf("UserId = %v, want %v", result.UserId, userInfo.UserId)
	}
	if result.Username != userInfo.Username {
		t.Errorf("Username = %v, want %v", result.Username, userInfo.Username)
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "req-789"

	newCtx := WithRequestID(ctx, requestID)

	result := GetRequestID(newCtx)
	if result != requestID {
		t.Errorf("WithRequestID() -> GetRequestID() = %v, want %v", result, requestID)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if id1 == "" {
		t.Error("GenerateRequestID() should not return empty string")
	}

	if id2 == "" {
		t.Error("GenerateRequestID() should not return empty string")
	}

	if id1 == id2 {
		t.Error("GenerateRequestID() should return unique IDs")
	}

	// Should be 16 hex characters (8 bytes)
	if len(id1) != 16 {
		t.Errorf("GenerateRequestID() length = %d, want 16", len(id1))
	}
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()

	userInfo := &authv1.UserInfo{
		UserId:   "user-123",
		Username: "testuser",
		Role:     "admin",
	}

	// Chain multiple context values
	ctx = WithUserID(ctx, "user-123")
	ctx = WithUserInfo(ctx, userInfo)
	ctx = WithRequestID(ctx, "req-456")

	// All values should be retrievable
	if GetUserID(ctx) != "user-123" {
		t.Error("UserID not preserved in chain")
	}
	if GetUserInfo(ctx) == nil {
		t.Error("UserInfo not preserved in chain")
	}
	if GetRequestID(ctx) != "req-456" {
		t.Error("RequestID not preserved in chain")
	}
}
