package passhash

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig конфигурация JWT
type JWTConfig struct {
	SecretKey          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// DefaultJWTConfig возвращает конфигурацию по умолчанию
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey:          "change-me-in-production",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "logistics-auth",
	}
}

// Claims кастомные claims для JWT
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager управляет JWT токенами
type JWTManager struct {
	config *JWTConfig
}

// NewJWTManager создаёт новый менеджер JWT
func NewJWTManager(config *JWTConfig) *JWTManager {
	if config == nil {
		config = DefaultJWTConfig()
	}
	return &JWTManager{config: config}
}

// GenerateAccessToken генерирует access token
func (m *JWTManager) GenerateAccessToken(userID, username, role string) (string, error) {
	return m.generateToken(userID, username, role, m.config.AccessTokenExpiry)
}

// GenerateRefreshToken генерирует refresh token
func (m *JWTManager) GenerateRefreshToken(userID, username, role string) (string, error) {
	return m.generateToken(userID, username, role, m.config.RefreshTokenExpiry)
}

func (m *JWTManager) generateToken(userID, username, role string, expiry time.Duration) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

// ValidateToken валидирует токен и возвращает claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GetAccessTokenExpiry возвращает время жизни access token в секундах
func (m *JWTManager) GetAccessTokenExpiry() int64 {
	return int64(m.config.AccessTokenExpiry.Seconds())
}

// RefreshAccessToken обновляет access token используя refresh token
func (m *JWTManager) RefreshAccessToken(refreshToken string) (string, *Claims, error) {
	claims, err := m.ValidateToken(refreshToken)
	if err != nil {
		return "", nil, err
	}

	newAccessToken, err := m.GenerateAccessToken(claims.UserID, claims.Username, claims.Role)
	if err != nil {
		return "", nil, err
	}

	return newAccessToken, claims, nil
}
