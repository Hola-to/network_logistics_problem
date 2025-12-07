// services/report-svc/internal/repository/storage.go
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Ошибки
var (
	ErrNotFound      = errors.New("report not found")
	ErrAlreadyExists = errors.New("report already exists")
	ErrInvalidID     = errors.New("invalid report ID")
	ErrStorageFull   = errors.New("storage quota exceeded")
)

// Repository интерфейс хранилища отчётов
type Repository interface {
	// Create сохраняет новый отчёт
	Create(ctx context.Context, params *CreateParams) (*Report, error)

	// Get возвращает отчёт по ID (включая контент)
	Get(ctx context.Context, id uuid.UUID) (*Report, error)

	// GetContent возвращает только контент отчёта
	GetContent(ctx context.Context, id uuid.UUID) ([]byte, error)

	// List возвращает список отчётов с фильтрацией
	List(ctx context.Context, params *ListParams) (*ListResult, error)

	// Delete мягко удаляет отчёт
	Delete(ctx context.Context, id uuid.UUID) error

	// HardDelete физически удаляет отчёт
	HardDelete(ctx context.Context, id uuid.UUID) error

	// DeleteExpired удаляет устаревшие отчёты
	DeleteExpired(ctx context.Context) (int64, error)

	// UpdateTags обновляет теги отчёта
	UpdateTags(ctx context.Context, id uuid.UUID, tags []string, replace bool) ([]string, error)

	// Stats возвращает статистику хранилища
	Stats(ctx context.Context, userID string) (*Stats, error)

	// Close закрывает соединения
	Close() error

	// Ping проверяет соединение
	Ping(ctx context.Context) error
}
