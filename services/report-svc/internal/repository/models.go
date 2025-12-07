// services/report-svc/internal/repository/models.go
package repository

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

// Report модель отчёта в хранилище
type Report struct {
	ID          uuid.UUID
	Title       string
	Description string
	Author      string

	ReportType reportv1.ReportType
	Format     reportv1.ReportFormat

	Content     []byte
	ContentType string
	Filename    string
	SizeBytes   int64

	CalculationID string
	GraphID       string
	UserID        string

	GenerationTimeMs float64
	Version          string

	Tags         []string
	CustomFields map[string]string

	CreatedAt time.Time
	ExpiresAt *time.Time
	DeletedAt *time.Time
}

// ToMetadata конвертирует в protobuf metadata
func (r *Report) ToMetadata() *reportv1.ReportMetadata {
	meta := &reportv1.ReportMetadata{
		ReportId:         r.ID.String(),
		Title:            r.Title,
		Description:      r.Description,
		Type:             r.ReportType,
		Format:           r.Format,
		GeneratedAt:      timestamppb.New(r.CreatedAt),
		GeneratedBy:      r.Author,
		Version:          r.Version,
		SizeBytes:        r.SizeBytes,
		GenerationTimeMs: r.GenerationTimeMs,
		CustomFields:     r.CustomFields,
		CalculationId:    r.CalculationID,
		GraphId:          r.GraphID,
		Tags:             r.Tags,
	}

	if r.ExpiresAt != nil {
		meta.ExpiresAt = timestamppb.New(*r.ExpiresAt)
	}

	return meta
}

// ToContent конвертирует в protobuf content
func (r *Report) ToContent() *reportv1.ReportContent {
	return &reportv1.ReportContent{
		Data:        r.Content,
		ContentType: r.ContentType,
		Filename:    r.Filename,
		SizeBytes:   r.SizeBytes,
	}
}

// CreateParams параметры создания отчёта
type CreateParams struct {
	Title       string
	Description string
	Author      string

	ReportType reportv1.ReportType
	Format     reportv1.ReportFormat

	Content     []byte
	ContentType string
	Filename    string

	CalculationID string
	GraphID       string
	UserID        string

	GenerationTimeMs float64
	Version          string

	Tags         []string
	CustomFields map[string]string

	// TTL для автоудаления (0 = бессрочно)
	TTL time.Duration
}

// ListParams параметры фильтрации списка
type ListParams struct {
	Limit  int32
	Offset int32

	ReportType    *reportv1.ReportType
	Format        *reportv1.ReportFormat
	CalculationID string
	GraphID       string
	UserID        string
	Tags          []string

	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	OrderBy   string // created_at, size_bytes, title
	OrderDesc bool
}

// ListResult результат списка с пагинацией
type ListResult struct {
	Reports    []*Report
	TotalCount int64
	HasMore    bool
}

// Stats статистика хранилища
type Stats struct {
	TotalReports   int64
	TotalSizeBytes int64
	AvgSizeBytes   float64

	ReportsByType   map[string]int64
	ReportsByFormat map[string]int64
	SizeByType      map[string]int64

	OldestReportAt *time.Time
	NewestReportAt *time.Time
	ExpiredReports int64
}

// ToProto конвертирует в protobuf
func (s *Stats) ToProto() *reportv1.GetRepositoryStatsResponse {
	resp := &reportv1.GetRepositoryStatsResponse{
		TotalReports:    s.TotalReports,
		TotalSizeBytes:  s.TotalSizeBytes,
		AvgSizeBytes:    s.AvgSizeBytes,
		ReportsByType:   s.ReportsByType,
		ReportsByFormat: s.ReportsByFormat,
		SizeByType:      s.SizeByType,
		ExpiredReports:  s.ExpiredReports,
	}

	if s.OldestReportAt != nil {
		resp.OldestReportAt = timestamppb.New(*s.OldestReportAt)
	}
	if s.NewestReportAt != nil {
		resp.NewestReportAt = timestamppb.New(*s.NewestReportAt)
	}

	return resp
}
