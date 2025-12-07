package repository

import (
	"context"
	"errors"
	"time"
)

// Стандартные ошибки репозитория
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// User модель пользователя
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	FullName     string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserRepository интерфейс репозитория пользователей
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, username, email string) (bool, error)
}

// TokenBlacklist интерфейс для хранения отозванных токенов
type TokenBlacklist interface {
	Add(ctx context.Context, token string, expiry time.Duration) error
	Contains(ctx context.Context, token string) (bool, error)
}
