package token

import (
	"logistics/pkg/passhash"
	"time"
)

// Manager обёртка над passhash.JWTManager для auth-svc
type Manager struct {
	jwt *passhash.JWTManager
}

// Config конфигурация токенов
type Config struct {
	SecretKey          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// NewManager создаёт новый менеджер токенов
func NewManager(cfg *Config) *Manager {
	jwtCfg := &passhash.JWTConfig{
		SecretKey:          cfg.SecretKey,
		AccessTokenExpiry:  cfg.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.RefreshTokenExpiry,
		Issuer:             cfg.Issuer,
	}

	return &Manager{
		jwt: passhash.NewJWTManager(jwtCfg),
	}
}

// GenerateTokenPair генерирует пару access + refresh токенов
func (m *Manager) GenerateTokenPair(userID, username, role string) (accessToken, refreshToken string, expiresIn int64, err error) {
	accessToken, err = m.jwt.GenerateAccessToken(userID, username, role)
	if err != nil {
		return "", "", 0, err
	}

	refreshToken, err = m.jwt.GenerateRefreshToken(userID, username, role)
	if err != nil {
		return "", "", 0, err
	}

	expiresIn = m.jwt.GetAccessTokenExpiry()
	return accessToken, refreshToken, expiresIn, nil
}

// ValidateToken валидирует токен
func (m *Manager) ValidateToken(tokenString string) (*passhash.Claims, error) {
	return m.jwt.ValidateToken(tokenString)
}

// RefreshAccessToken обновляет access token
func (m *Manager) RefreshAccessToken(refreshToken string) (newAccessToken, newRefreshToken string, expiresIn int64, err error) {
	claims, err := m.jwt.ValidateToken(refreshToken)
	if err != nil {
		return "", "", 0, err
	}

	return m.GenerateTokenPair(claims.UserID, claims.Username, claims.Role)
}

// GetExpiresIn возвращает время жизни access token
func (m *Manager) GetExpiresIn() int64 {
	return m.jwt.GetAccessTokenExpiry()
}
