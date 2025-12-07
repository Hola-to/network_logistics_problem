// services/simulation-svc/internal/repository/repository.go
package repository

import (
	"context"
	"errors"
	"time"
)

// Стандартные ошибки
var (
	ErrSimulationNotFound = errors.New("simulation not found")
	ErrAccessDenied       = errors.New("access denied")
)

// Simulation модель симуляции
type Simulation struct {
	ID                string
	UserID            string
	Name              string
	Description       string
	SimulationType    string
	NodeCount         int
	EdgeCount         int
	ComputationTimeMs float64
	BaselineFlow      *float64
	ResultFlow        *float64
	FlowChangePercent *float64
	GraphData         []byte // JSON
	RequestData       []byte // JSON
	ResponseData      []byte // JSON
	Tags              []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// SimulationSummary краткая информация
type SimulationSummary struct {
	ID             string
	Name           string
	SimulationType string
	CreatedAt      time.Time
	Tags           []string
}

// ListOptions опции для списка
type ListOptions struct {
	Limit  int
	Offset int
}

// SimulationRepository интерфейс репозитория
type SimulationRepository interface {
	// CRUD
	Create(ctx context.Context, sim *Simulation) error
	GetByID(ctx context.Context, id string) (*Simulation, error)
	Delete(ctx context.Context, id string) error

	// Списки
	List(ctx context.Context, userID string, simType string, opts *ListOptions) ([]*SimulationSummary, int64, error)
	ListByUser(ctx context.Context, userID string, opts *ListOptions) ([]*SimulationSummary, int64, error)

	// Проверка доступа
	GetByUserAndID(ctx context.Context, userID, id string) (*Simulation, error)
}
