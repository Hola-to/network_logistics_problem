package handlers

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "logistics/gen/go/logistics/auth/v1"
	gatewayv1 "logistics/gen/go/logistics/gateway/v1"
	"logistics/pkg/logger"
	"logistics/services/gateway-svc/internal/clients"
	"logistics/services/gateway-svc/internal/middleware"
)

// AuthHandler обработчики аутентификации
type AuthHandler struct {
	clients *clients.Manager
}

// NewAuthHandler создаёт обработчик
func NewAuthHandler(clients *clients.Manager) *AuthHandler {
	return &AuthHandler{clients: clients}
}

// Login выполняет вход
func (h *AuthHandler) Login(
	ctx context.Context,
	req *connect.Request[gatewayv1.LoginRequest],
) (*connect.Response[gatewayv1.AuthResponse], error) {
	msg := req.Msg

	resp, err := h.clients.Auth().Login(ctx, msg.Username, msg.Password)
	if err != nil {
		logger.Log.Error("Login failed", "username", msg.Username, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !resp.Success {
		return connect.NewResponse(&gatewayv1.AuthResponse{
			Success:      false,
			ErrorMessage: resp.ErrorMessage,
		}), nil
	}

	return connect.NewResponse(&gatewayv1.AuthResponse{
		Success:      true,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
		User:         h.convertUserProfile(resp.User),
	}), nil
}

// Register регистрирует пользователя
func (h *AuthHandler) Register(
	ctx context.Context,
	req *connect.Request[gatewayv1.RegisterRequest],
) (*connect.Response[gatewayv1.AuthResponse], error) {
	msg := req.Msg

	resp, err := h.clients.Auth().Register(ctx, &authv1.RegisterRequest{
		Username: msg.Username,
		Email:    msg.Email,
		Password: msg.Password,
		FullName: msg.FullName,
	})
	if err != nil {
		logger.Log.Error("Registration failed", "username", msg.Username, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !resp.Success {
		return connect.NewResponse(&gatewayv1.AuthResponse{
			Success:      false,
			ErrorMessage: resp.ErrorMessage,
		}), nil
	}

	// После успешной регистрации автоматически логиним
	loginResp, err := h.clients.Auth().Login(ctx, msg.Username, msg.Password)
	if err != nil {
		return connect.NewResponse(&gatewayv1.AuthResponse{
			Success:      true,
			ErrorMessage: "registered but login failed",
		}), nil
	}

	return connect.NewResponse(&gatewayv1.AuthResponse{
		Success:      true,
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
		ExpiresIn:    loginResp.ExpiresIn,
		User:         h.convertUserProfile(loginResp.User),
	}), nil
}

// RefreshToken обновляет токен
func (h *AuthHandler) RefreshToken(
	ctx context.Context,
	req *connect.Request[gatewayv1.RefreshTokenRequest],
) (*connect.Response[gatewayv1.AuthResponse], error) {
	msg := req.Msg

	resp, err := h.clients.Auth().RefreshToken(ctx, msg.RefreshToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if !resp.Success {
		return connect.NewResponse(&gatewayv1.AuthResponse{
			Success:      false,
			ErrorMessage: resp.ErrorMessage,
		}), nil
	}

	return connect.NewResponse(&gatewayv1.AuthResponse{
		Success:      true,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
	}), nil
}

// Logout выход из системы
func (h *AuthHandler) Logout(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[emptypb.Empty], error) {
	// Получаем токен из header
	token := req.Header().Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if token != "" {
		_, err := h.clients.Auth().Logout(ctx, token)
		if err != nil {
			logger.Log.Warn("Logout failed", "error", err)
		}
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetProfile получает профиль текущего пользователя
func (h *AuthHandler) GetProfile(
	ctx context.Context,
	req *connect.Request[emptypb.Empty],
) (*connect.Response[gatewayv1.UserProfile], error) {
	userInfo := middleware.GetUserInfo(ctx)
	if userInfo == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	return connect.NewResponse(h.convertUserProfile(userInfo)), nil
}

// ValidateToken валидирует токен
func (h *AuthHandler) ValidateToken(
	ctx context.Context,
	req *connect.Request[gatewayv1.ValidateTokenRequest],
) (*connect.Response[gatewayv1.ValidateTokenResponse], error) {
	msg := req.Msg

	resp, err := h.clients.Auth().ValidateToken(ctx, msg.Token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&gatewayv1.ValidateTokenResponse{
		Valid:     resp.Valid,
		UserId:    resp.UserId,
		ExpiresAt: resp.ExpiresAt,
	}), nil
}

// convertUserProfile конвертирует UserInfo в UserProfile
func (h *AuthHandler) convertUserProfile(user *authv1.UserInfo) *gatewayv1.UserProfile {
	if user == nil {
		return nil
	}

	return &gatewayv1.UserProfile{
		UserId:    user.UserId,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      user.Role,
		CreatedAt: timestamppb.New(timestamppb.Now().AsTime()), // TODO: из user.CreatedAt
	}
}
