package repository

import (
	"context"
	"errors"
	"time"
)

// Стандартные ошибки
var (
	ErrAuditNotFound = errors.New("audit entry not found")
)

// AuditEntry модель аудит записи
type AuditEntry struct {
	ID            string
	Timestamp     time.Time
	Service       string
	Method        string
	RequestID     string
	Action        string
	Outcome       string
	UserID        string
	Username      string
	UserRole      string
	ClientIP      string
	UserAgent     string
	ResourceType  string
	ResourceID    string
	ResourceName  string
	DurationMs    int64
	ErrorCode     string
	ErrorMessage  string
	ChangesBefore []byte // JSON
	ChangesAfter  []byte // JSON
	ChangedFields []string
	Metadata      map[string]string
}

// AuditFilter фильтр для запросов
type AuditFilter struct {
	TimeRange    *TimeRange
	Services     []string
	Methods      []string
	Actions      []string
	Outcomes     []string
	UserID       string
	ResourceType string
	ResourceID   string
	ClientIP     string
	SearchQuery  string
}

// TimeRange временной диапазон
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// ListOptions опции списка
type ListOptions struct {
	Limit     int
	Offset    int
	SortOrder string // "timestamp_desc", "timestamp_asc"
}

// UserActivitySummary сводка активности пользователя
type UserActivitySummary struct {
	TotalActions      int
	SuccessfulActions int
	FailedActions     int
	DeniedActions     int
	ActionsByType     map[string]int
	ActionsByService  map[string]int
	FirstActivity     time.Time
	LastActivity      time.Time
}

// ResourceSummary сводка по ресурсу
type ResourceSummary struct {
	CreatedAt      time.Time
	CreatedBy      string
	LastModifiedAt time.Time
	LastModifiedBy string
	TotalChanges   int
}

// AuditStats статистика аудита
type AuditStats struct {
	TotalEvents      int64
	SuccessfulEvents int64
	FailedEvents     int64
	DeniedEvents     int64
	UniqueUsers      int64
	UniqueResources  int64
	AvgDurationMs    float64
	ByService        map[string]int64
	ByAction         map[string]int64
	ByOutcome        map[string]int64
	Timeline         []TimelinePoint
	TopUsers         []TopUser
	TopResources     []TopResource
}

// TimelinePoint точка на графике
type TimelinePoint struct {
	Timestamp    time.Time
	Count        int64
	SuccessCount int64
	FailureCount int64
}

// TopUser топ пользователь
type TopUser struct {
	UserID      string
	Username    string
	ActionCount int64
}

// TopResource топ ресурс
type TopResource struct {
	ResourceType string
	ResourceID   string
	ActionCount  int64
}

// AuditRepository интерфейс репозитория
type AuditRepository interface {
	// Запись
	Create(ctx context.Context, entry *AuditEntry) error
	CreateBatch(ctx context.Context, entries []*AuditEntry) (int, error)

	// Чтение
	GetByID(ctx context.Context, id string) (*AuditEntry, error)
	List(ctx context.Context, filter *AuditFilter, opts *ListOptions) ([]*AuditEntry, int64, error)

	// История ресурса
	GetResourceHistory(ctx context.Context, resourceType, resourceID string, opts *ListOptions) ([]*AuditEntry, *ResourceSummary, int64, error)

	// Активность пользователя
	GetUserActivity(ctx context.Context, userID string, timeRange *TimeRange, opts *ListOptions) ([]*AuditEntry, *UserActivitySummary, int64, error)

	// Статистика
	GetStats(ctx context.Context, timeRange *TimeRange, groupBy string) (*AuditStats, error)

	// Подсчёт
	Count(ctx context.Context) (int64, error)

	// Очистка старых записей
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}
