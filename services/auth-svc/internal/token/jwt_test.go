package token

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	cfg := &Config{
		SecretKey:          "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test-issuer",
	}

	manager := NewManager(cfg)
	if manager == nil {
		t.Error("NewManager() returned nil")
	}
}

func TestManager_GenerateTokenPair(t *testing.T) {
	manager := createTestManager()

	accessToken, refreshToken, expiresIn, err := manager.GenerateTokenPair(
		"user-123",
		"testuser",
		"admin",
	)

	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	if accessToken == "" {
		t.Error("Access token should not be empty")
	}
	if refreshToken == "" {
		t.Error("Refresh token should not be empty")
	}
	if expiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}

	// Tokens should be different
	if accessToken == refreshToken {
		t.Error("Access and refresh tokens should be different")
	}
}

func TestManager_ValidateToken(t *testing.T) {
	manager := createTestManager()

	accessToken, _, _, err := manager.GenerateTokenPair("user-123", "testuser", "admin")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	claims, err := manager.ValidateToken(accessToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %v, want user-123", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("Username = %v, want testuser", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %v, want admin", claims.Role)
	}
}

func TestManager_ValidateToken_Invalid(t *testing.T) {
	manager := createTestManager()

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "not.a.valid.jwt",
		},
		{
			name:  "invalid signature",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.ValidateToken(tt.token)
			if err == nil {
				t.Error("ValidateToken() should return error for invalid token")
			}
		})
	}
}

func TestManager_RefreshAccessToken(t *testing.T) {
	manager := createTestManager()

	_, refreshToken, _, err := manager.GenerateTokenPair("user-123", "testuser", "admin")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	newAccess, newRefresh, expiresIn, err := manager.RefreshAccessToken(refreshToken)
	if err != nil {
		t.Fatalf("RefreshAccessToken() error = %v", err)
	}

	if newAccess == "" {
		t.Error("New access token should not be empty")
	}
	if newRefresh == "" {
		t.Error("New refresh token should not be empty")
	}
	if expiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}
}

func TestManager_GetExpiresIn(t *testing.T) {
	cfg := &Config{
		SecretKey:          "test-secret",
		AccessTokenExpiry:  30 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test",
	}

	manager := NewManager(cfg)
	expiresIn := manager.GetExpiresIn()

	// Should be approximately 30 minutes in seconds
	expected := int64(30 * 60)
	if expiresIn != expected {
		t.Errorf("GetExpiresIn() = %v, want %v", expiresIn, expected)
	}
}

func TestManager_TokenExpiration(t *testing.T) {
	cfg := &Config{
		SecretKey:          "test-secret",
		AccessTokenExpiry:  1 * time.Millisecond, // Very short for testing
		RefreshTokenExpiry: 1 * time.Millisecond,
		Issuer:             "test",
	}

	manager := NewManager(cfg)

	accessToken, _, _, err := manager.GenerateTokenPair("user-123", "testuser", "admin")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = manager.ValidateToken(accessToken)
	if err == nil {
		t.Error("ValidateToken() should return error for expired token")
	}
}

func createTestManager() *Manager {
	return NewManager(&Config{
		SecretKey:          "test-secret-key-for-testing-only",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test",
	})
}
