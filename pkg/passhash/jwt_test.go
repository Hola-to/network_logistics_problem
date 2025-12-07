package passhash

import (
	"testing"
	"time"
)

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	manager := NewJWTManager(&JWTConfig{
		SecretKey:         "test-secret-key",
		AccessTokenExpiry: 15 * time.Minute,
		Issuer:            "test-issuer",
	})

	token, err := manager.GenerateAccessToken("user-123", "testuser", "admin")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}

	// Token should have 3 parts (header.payload.signature)
	parts := 0
	for _, c := range token {
		if c == '.' {
			parts++
		}
	}
	if parts != 2 {
		t.Errorf("expected 2 dots in JWT, got %d", parts)
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := NewJWTManager(&JWTConfig{
		SecretKey:         "test-secret-key",
		AccessTokenExpiry: 15 * time.Minute,
		Issuer:            "test-issuer",
	})

	token, _ := manager.GenerateAccessToken("user-123", "testuser", "admin")

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected userID 'user-123', got %s", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %s", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role 'admin', got %s", claims.Role)
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("expected issuer 'test-issuer', got %s", claims.Issuer)
	}
}

func TestJWTManager_ValidateToken_Invalid(t *testing.T) {
	manager := NewJWTManager(nil)

	_, err := manager.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestJWTManager_ValidateToken_Expired(t *testing.T) {
	manager := NewJWTManager(&JWTConfig{
		SecretKey:         "test-secret",
		AccessTokenExpiry: 1 * time.Millisecond, // Very short expiry
		Issuer:            "test",
	})

	token, _ := manager.GenerateAccessToken("user", "username", "role")

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	_, err := manager.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestJWTManager_ValidateToken_WrongSecret(t *testing.T) {
	manager1 := NewJWTManager(&JWTConfig{
		SecretKey:         "secret-1",
		AccessTokenExpiry: 15 * time.Minute,
	})
	manager2 := NewJWTManager(&JWTConfig{
		SecretKey:         "secret-2",
		AccessTokenExpiry: 15 * time.Minute,
	})

	token, _ := manager1.GenerateAccessToken("user", "username", "role")

	_, err := manager2.ValidateToken(token)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	manager := NewJWTManager(&JWTConfig{
		SecretKey:          "test-secret",
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	token, err := manager.GenerateRefreshToken("user-123", "testuser", "admin")
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected userID 'user-123', got %s", claims.UserID)
	}
}

func TestJWTManager_RefreshAccessToken(t *testing.T) {
	manager := NewJWTManager(&JWTConfig{
		SecretKey:          "test-secret",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})

	refreshToken, _ := manager.GenerateRefreshToken("user-123", "testuser", "admin")

	newAccessToken, claims, err := manager.RefreshAccessToken(refreshToken)
	if err != nil {
		t.Fatalf("failed to refresh: %v", err)
	}

	if newAccessToken == "" {
		t.Error("expected non-empty new access token")
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected userID 'user-123', got %s", claims.UserID)
	}
}

func TestJWTManager_RefreshAccessToken_Invalid(t *testing.T) {
	manager := NewJWTManager(nil)

	_, _, err := manager.RefreshAccessToken("invalid-refresh-token")
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}

func TestJWTManager_GetAccessTokenExpiry(t *testing.T) {
	manager := NewJWTManager(&JWTConfig{
		AccessTokenExpiry: 15 * time.Minute,
	})

	expiry := manager.GetAccessTokenExpiry()
	expected := int64(15 * 60)

	if expiry != expected {
		t.Errorf("expected %d seconds, got %d", expected, expiry)
	}
}

func TestDefaultJWTConfig(t *testing.T) {
	cfg := DefaultJWTConfig()

	if cfg.SecretKey == "" {
		t.Error("expected default secret key")
	}
	if cfg.AccessTokenExpiry != 15*time.Minute {
		t.Errorf("expected 15m, got %v", cfg.AccessTokenExpiry)
	}
	if cfg.RefreshTokenExpiry != 7*24*time.Hour {
		t.Errorf("expected 7d, got %v", cfg.RefreshTokenExpiry)
	}
	if cfg.Issuer != "logistics-auth" {
		t.Errorf("expected 'logistics-auth', got %s", cfg.Issuer)
	}
}

func TestNewJWTManager_NilConfig(t *testing.T) {
	manager := NewJWTManager(nil)

	token, err := manager.GenerateAccessToken("user", "username", "role")
	if err != nil {
		t.Fatalf("should work with nil config: %v", err)
	}

	if token == "" {
		t.Error("expected token to be generated")
	}
}
