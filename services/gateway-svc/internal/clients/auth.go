package clients

import (
	"context"

	"google.golang.org/grpc"

	authv1 "logistics/gen/go/logistics/auth/v1"
	"logistics/pkg/config"
)

// AuthClient клиент для auth-svc
type AuthClient struct {
	conn   *grpc.ClientConn
	client authv1.AuthServiceClient
}

// NewAuthClient создаёт клиент
func NewAuthClient(ctx context.Context, endpoint config.ServiceEndpoint) (*AuthClient, error) {
	conn, err := dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return &AuthClient{
		conn:   conn,
		client: authv1.NewAuthServiceClient(conn),
	}, nil
}

// Login выполняет вход
func (c *AuthClient) Login(ctx context.Context, username, password string) (*authv1.LoginResponse, error) {
	return c.client.Login(ctx, &authv1.LoginRequest{
		Username: username,
		Password: password,
	})
}

// Register регистрирует пользователя
func (c *AuthClient) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	return c.client.Register(ctx, req)
}

// ValidateToken валидирует токен
func (c *AuthClient) ValidateToken(ctx context.Context, token string) (*authv1.ValidateTokenResponse, error) {
	return c.client.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
}

// RefreshToken обновляет токен
func (c *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (*authv1.RefreshTokenResponse, error) {
	return c.client.RefreshToken(ctx, &authv1.RefreshTokenRequest{RefreshToken: refreshToken})
}

// Logout выход из системы
func (c *AuthClient) Logout(ctx context.Context, token string) (*authv1.LogoutResponse, error) {
	return c.client.Logout(ctx, &authv1.LogoutRequest{Token: token})
}

// Raw возвращает сырой gRPC клиент
func (c *AuthClient) Raw() authv1.AuthServiceClient {
	return c.client
}
