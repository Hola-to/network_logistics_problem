// services/gateway-svc/internal/middleware/auth_test.go

package middleware

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"

	authv1 "logistics/gen/go/logistics/auth/v1"
)

// Mock auth client
type mockAuthClient struct {
	validateResponse *authv1.ValidateTokenResponse
	validateError    error
}

func (m *mockAuthClient) ValidateToken(ctx context.Context, token string) (*authv1.ValidateTokenResponse, error) {
	if m.validateError != nil {
		return nil, m.validateError
	}
	return m.validateResponse, nil
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		wantToken string
		wantErr   bool
	}{
		{
			name: "valid bearer token",
			setupCtx: func() context.Context {
				md := metadata.Pairs("authorization", "Bearer test-token-123")
				return metadata.NewIncomingContext(context.Background(), md)
			},
			wantToken: "test-token-123",
			wantErr:   false,
		},
		{
			name: "token without bearer prefix",
			setupCtx: func() context.Context {
				md := metadata.Pairs("authorization", "test-token-123")
				return metadata.NewIncomingContext(context.Background(), md)
			},
			wantToken: "test-token-123",
			wantErr:   false,
		},
		{
			name: "no metadata",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "no authorization header",
			setupCtx: func() context.Context {
				md := metadata.Pairs("other-header", "value")
				return metadata.NewIncomingContext(context.Background(), md)
			},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "empty authorization header",
			setupCtx: func() context.Context {
				md := metadata.Pairs("authorization", "")
				return metadata.NewIncomingContext(context.Background(), md)
			},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "only bearer prefix",
			setupCtx: func() context.Context {
				md := metadata.Pairs("authorization", "Bearer ")
				return metadata.NewIncomingContext(context.Background(), md)
			},
			wantToken: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			token, err := extractToken(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if token != tt.wantToken {
				t.Errorf("extractToken() = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

func TestPublicMethods(t *testing.T) {
	methods := PublicMethods()

	expectedPublic := []string{
		"/logistics.gateway.v1.GatewayService/Health",
		"/logistics.gateway.v1.GatewayService/ReadinessCheck",
		"/logistics.gateway.v1.GatewayService/Info",
		"/logistics.gateway.v1.GatewayService/GetAlgorithms",
		"/logistics.gateway.v1.GatewayService/Login",
		"/logistics.gateway.v1.GatewayService/Register",
		"/logistics.gateway.v1.GatewayService/RefreshToken",
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
	}

	for _, method := range expectedPublic {
		if !methods[method] {
			t.Errorf("Method %s should be public", method)
		}
	}

	// Check that protected methods are not in the list
	protectedMethods := []string{
		"/logistics.gateway.v1.GatewayService/SolveGraph",
		"/logistics.gateway.v1.GatewayService/GetProfile",
		"/logistics.gateway.v1.GatewayService/Logout",
	}

	for _, method := range protectedMethods {
		if methods[method] {
			t.Errorf("Method %s should NOT be public", method)
		}
	}
}

func TestAuthConfig(t *testing.T) {
	client := &mockAuthClient{}
	publicMethods := PublicMethods()

	cfg := &AuthConfig{
		Client:        client,
		PublicMethods: publicMethods,
	}

	if cfg.Client == nil {
		t.Error("Client should not be nil")
	}

	if len(cfg.PublicMethods) == 0 {
		t.Error("PublicMethods should not be empty")
	}
}

func TestAuthServerStream(t *testing.T) {
	ctx := context.WithValue(context.Background(), userIDKey, "user-123")

	stream := &authServerStream{
		ctx: ctx,
	}

	resultCtx := stream.Context()
	userID := GetUserID(resultCtx)

	if userID != "user-123" {
		t.Errorf("authServerStream.Context() should preserve user ID, got %v", userID)
	}
}
