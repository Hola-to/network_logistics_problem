package repository

import (
	"context"
	"errors"
	"time"
)

// Стандартные ошибки
var (
	ErrCalculationNotFound = errors.New("calculation not found")
	ErrAccessDenied        = errors.New("access denied")
)

// Calculation модель расчёта
type Calculation struct {
	ID                string
	UserID            string
	Name              string
	Algorithm         string
	MaxFlow           float64
	TotalCost         float64
	ComputationTimeMs float64
	NodeCount         int
	EdgeCount         int
	RequestData       []byte // JSON
	ResponseData      []byte // JSON
	Tags              []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CalculationSummary краткая информация о расчёте
type CalculationSummary struct {
	ID                string
	Name              string
	Algorithm         string
	MaxFlow           float64
	TotalCost         float64
	ComputationTimeMs float64
	NodeCount         int
	EdgeCount         int
	Tags              []string
	CreatedAt         time.Time
}

// ListFilter фильтры для списка
type ListFilter struct {
	Algorithm string
	Tags      []string
	MinFlow   *float64
	MaxFlow   *float64
	StartTime *time.Time
	EndTime   *time.Time
}

// SortOrder порядок сортировки
type SortOrder string

const (
	SortByCreatedDesc   SortOrder = "created_desc"
	SortByCreatedAsc    SortOrder = "created_asc"
	SortByMaxFlowDesc   SortOrder = "max_flow_desc"
	SortByTotalCostDesc SortOrder = "cost_desc"
)

// ListOptions опции для списка
type ListOptions struct {
	Limit  int
	Offset int
	Filter *ListFilter
	Sort   SortOrder
}

// UserStatistics статистика пользователя
type UserStatistics struct {
	TotalCalculations        int
	AverageMaxFlow           float64
	AverageTotalCost         float64
	AverageComputationTimeMs float64
	CalculationsByAlgorithm  map[string]int
	DailyStats               []DailyStats
}

// DailyStats статистика за день
type DailyStats struct {
	Date      string // "2024-01-15"
	Count     int
	TotalFlow float64
}

// CalculationRepository интерфейс репозитория расчётов
type CalculationRepository interface {
	// CRUD
	Create(ctx context.Context, calc *Calculation) error
	GetByID(ctx context.Context, id string) (*Calculation, error)
	Delete(ctx context.Context, id string) error

	// Списки
	List(ctx context.Context, userID string, opts *ListOptions) ([]*CalculationSummary, int64, error)

	// Статистика
	GetUserStatistics(ctx context.Context, userID string, startTime, endTime *time.Time) (*UserStatistics, error)

	// Поиск
	Search(ctx context.Context, userID string, query string, limit int) ([]*CalculationSummary, error)
}
