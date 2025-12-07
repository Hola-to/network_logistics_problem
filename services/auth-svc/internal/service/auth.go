package service

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/attribute"

	authv1 "logistics/gen/go/logistics/auth/v1"
	pkgerrors "logistics/pkg/apperror"
	"logistics/pkg/logger"
	"logistics/pkg/passhash"
	"logistics/pkg/telemetry"
	"logistics/services/auth-svc/internal/repository"
	"logistics/services/auth-svc/internal/token"
)

// AuthService реализация gRPC сервиса аутентификации
type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	repo      repository.UserRepository
	blacklist repository.TokenBlacklist
	tokens    *token.Manager
}

// NewAuthService создаёт новый сервис аутентификации
func NewAuthService(
	repo repository.UserRepository,
	blacklist repository.TokenBlacklist,
	tokens *token.Manager,
) *AuthService {
	return &AuthService{
		repo:      repo,
		blacklist: blacklist,
		tokens:    tokens,
	}
}

// Login аутентификация пользователя
func (s *AuthService) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuthService.Login")
	defer span.End()

	span.SetAttributes(attribute.String("username", req.Username))

	// Валидация
	if req.Username == "" || req.Password == "" {
		return &authv1.LoginResponse{
			Success:      false,
			ErrorMessage: "username and password are required",
		}, nil
	}

	// Получаем пользователя
	user, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			telemetry.AddEvent(ctx, "user_not_found")
			return &authv1.LoginResponse{
				Success:      false,
				ErrorMessage: "invalid username or password",
			}, nil
		}
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get user"))
	}

	// Проверяем пароль
	valid, err := passhash.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to verify password"))
	}

	if !valid {
		telemetry.AddEvent(ctx, "invalid_password")
		return &authv1.LoginResponse{
			Success:      false,
			ErrorMessage: "invalid username or password",
		}, nil
	}

	// Генерируем токены
	accessToken, refreshToken, expiresIn, err := s.tokens.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to generate tokens"))
	}

	telemetry.AddEvent(ctx, "login_success", attribute.String("user_id", user.ID))

	return &authv1.LoginResponse{
		Success:      true,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User:         toUserInfo(user),
	}, nil
}

// Register регистрация нового пользователя
func (s *AuthService) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuthService.Register")
	defer span.End()

	span.SetAttributes(
		attribute.String("username", req.Username),
		attribute.String("email", req.Email),
	)

	// Валидация
	if err := validateRegisterRequest(req); err != nil {
		return &authv1.RegisterResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Проверяем уникальность
	exists, err := s.repo.Exists(ctx, req.Username, req.Email)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to check user existence"))
	}
	if exists {
		return &authv1.RegisterResponse{
			Success:      false,
			ErrorMessage: "user with this username or email already exists",
		}, nil
	}

	// Хешируем пароль
	passwordHash, err := passhash.HashPassword(req.Password)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to hash password"))
	}

	// Создаём пользователя
	user := &repository.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		FullName:     req.FullName,
		Role:         "user", // Роль по умолчанию
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return &authv1.RegisterResponse{
				Success:      false,
				ErrorMessage: "user already exists",
			}, nil
		}
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to create user"))
	}

	telemetry.AddEvent(ctx, "user_registered", attribute.String("user_id", user.ID))

	return &authv1.RegisterResponse{
		Success: true,
		UserId:  user.ID,
	}, nil
}

// ValidateToken проверка токена
func (s *AuthService) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuthService.ValidateToken")
	defer span.End()

	if req.Token == "" {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

	// Проверяем blacklist
	blacklisted, err := s.blacklist.Contains(ctx, req.Token)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to check blacklist"))
	}
	if blacklisted {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

	// Валидируем токен
	claims, err := s.tokens.ValidateToken(req.Token)
	if err != nil {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

	// Получаем актуальные данные пользователя
	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return &authv1.ValidateTokenResponse{Valid: false}, nil
		}
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to get user"))
	}

	return &authv1.ValidateTokenResponse{
		Valid:     true,
		UserId:    user.ID,
		User:      toUserInfo(user),
		ExpiresAt: claims.ExpiresAt.Unix(),
	}, nil
}

// RefreshToken обновление токенов
func (s *AuthService) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuthService.RefreshToken")
	defer span.End()

	if req.RefreshToken == "" {
		return &authv1.RefreshTokenResponse{
			Success:      false,
			ErrorMessage: "refresh token is required",
		}, nil
	}

	// Проверяем blacklist
	blacklisted, err := s.blacklist.Contains(ctx, req.RefreshToken)
	if err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to check blacklist"))
	}
	if blacklisted {
		return &authv1.RefreshTokenResponse{
			Success:      false,
			ErrorMessage: "token has been revoked",
		}, nil
	}

	// Обновляем токены
	accessToken, refreshToken, expiresIn, err := s.tokens.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		return &authv1.RefreshTokenResponse{
			Success:      false,
			ErrorMessage: "invalid refresh token",
		}, nil
	}

	// Добавляем старый refresh token в blacklist
	if err := s.blacklist.Add(ctx, req.RefreshToken, 7*24*time.Hour); err != nil {
		// Логируем ошибку, но продолжаем - токен всё равно обновлён
		logger.Log.Warn("Failed to blacklist old refresh token", "error", err)
	}

	telemetry.AddEvent(ctx, "token_refreshed")

	return &authv1.RefreshTokenResponse{
		Success:      true,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// Logout выход пользователя
func (s *AuthService) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "AuthService.Logout")
	defer span.End()

	if req.Token == "" {
		return &authv1.LogoutResponse{Success: true}, nil
	}

	// Добавляем токен в blacklist
	if err := s.blacklist.Add(ctx, req.Token, 24*time.Hour); err != nil {
		telemetry.SetError(ctx, err)
		return nil, pkgerrors.ToGRPC(pkgerrors.Wrap(err, pkgerrors.CodeInternal, "failed to revoke token"))
	}

	telemetry.AddEvent(ctx, "user_logged_out")

	return &authv1.LogoutResponse{Success: true}, nil
}

// Вспомогательные функции

func validateRegisterRequest(req *authv1.RegisterRequest) error {
	if req.Username == "" {
		return errors.New("username is required")
	}
	if len(req.Username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if req.Email == "" {
		return errors.New("email is required")
	}
	// Простая проверка email
	if len(req.Email) < 3 || !contains(req.Email, "@") {
		return errors.New("invalid email format")
	}
	return nil
}

func contains(s string, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toUserInfo(user *repository.User) *authv1.UserInfo {
	return &authv1.UserInfo{
		UserId:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Unix(),
	}
}
