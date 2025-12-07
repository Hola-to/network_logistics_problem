// services/report-svc/internal/repository/repository_test.go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	reportv1 "logistics/gen/go/logistics/report/v1"
)

// MockDB мок для database.DB - реализует все методы интерфейса
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	callArgs := m.Called(ctx, sql, args)
	return callArgs.Get(0).(pgconn.CommandTag), callArgs.Error(1)
}

func (m *MockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	callArgs := m.Called(ctx, sql, args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(pgx.Rows), callArgs.Error(1)
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	callArgs := m.Called(ctx, sql, args)
	return callArgs.Get(0).(pgx.Row)
}

func (m *MockDB) Begin(ctx context.Context) (pgx.Tx, error) {
	callArgs := m.Called(ctx)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(pgx.Tx), callArgs.Error(1)
}

func (m *MockDB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	callArgs := m.Called(ctx, opts)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(pgx.Tx), callArgs.Error(1)
}

func (m *MockDB) Ping(ctx context.Context) error {
	callArgs := m.Called(ctx)
	return callArgs.Error(0)
}

func (m *MockDB) Close() {
	m.Called()
}

// MockRow мок для pgx.Row
type MockRow struct {
	mock.Mock
	scanErr error
	values  []any
}

func (m *MockRow) Scan(dest ...any) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	// Копируем значения в dest
	for i, v := range m.values {
		if i < len(dest) {
			switch d := dest[i].(type) {
			case *uuid.UUID:
				if val, ok := v.(uuid.UUID); ok {
					*d = val
				}
			case *string:
				if val, ok := v.(string); ok {
					*d = val
				}
			case *sql.NullString:
				if val, ok := v.(sql.NullString); ok {
					*d = val
				}
			case *int64:
				if val, ok := v.(int64); ok {
					*d = val
				}
			case *float64:
				if val, ok := v.(float64); ok {
					*d = val
				}
			case *[]byte:
				if val, ok := v.([]byte); ok {
					*d = val
				}
			case *[]string:
				if val, ok := v.([]string); ok {
					*d = val
				}
			case *map[string]string:
				if val, ok := v.(map[string]string); ok {
					*d = val
				}
			case *time.Time:
				if val, ok := v.(time.Time); ok {
					*d = val
				}
			case *sql.NullTime:
				if val, ok := v.(sql.NullTime); ok {
					*d = val
				}
			}
		}
	}
	return nil
}

// MockRows мок для pgx.Rows
type MockRows struct {
	mock.Mock
	data    [][]any
	current int
	closed  bool
}

func (m *MockRows) Close() {
	m.closed = true
}

func (m *MockRows) Err() error {
	return nil
}

func (m *MockRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *MockRows) Next() bool {
	if m.current < len(m.data) {
		m.current++
		return true
	}
	return false
}

func (m *MockRows) Scan(dest ...any) error {
	if m.current == 0 || m.current > len(m.data) {
		return errors.New("no current row")
	}
	row := m.data[m.current-1]
	for i, v := range row {
		if i < len(dest) {
			switch d := dest[i].(type) {
			case *uuid.UUID:
				if val, ok := v.(uuid.UUID); ok {
					*d = val
				}
			case *string:
				if val, ok := v.(string); ok {
					*d = val
				}
			case *int64:
				if val, ok := v.(int64); ok {
					*d = val
				}
			case *int32:
				if val, ok := v.(int32); ok {
					*d = val
				}
			}
		}
	}
	return nil
}

func (m *MockRows) Values() ([]any, error) {
	return nil, nil
}

func (m *MockRows) RawValues() [][]byte {
	return nil
}

func (m *MockRows) Conn() *pgx.Conn {
	return nil
}

// Тесты
func TestNewPostgresRepository(t *testing.T) {
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	require.NotNil(t, repo)
	assert.Equal(t, mockDB, repo.db)
}

func TestPostgresRepository_Create(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	params := &CreateParams{
		Title:            "Test Report",
		Description:      "Test Description",
		Author:           "Test Author",
		ReportType:       reportv1.ReportType_REPORT_TYPE_FLOW,
		Format:           reportv1.ReportFormat_REPORT_FORMAT_PDF,
		Content:          []byte("test content"),
		ContentType:      "application/pdf",
		Filename:         "test.pdf",
		CalculationID:    "calc-123",
		GraphID:          "graph-456",
		UserID:           "user-789",
		GenerationTimeMs: 150.5,
		Version:          "1.0.0",
		Tags:             []string{"test", "flow"},
		CustomFields:     map[string]string{"key": "value"},
		TTL:              24 * time.Hour,
	}

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.NewCommandTag("INSERT 0 1"), nil)

	report, err := repo.Create(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, params.Title, report.Title)
	assert.Equal(t, params.Description, report.Description)
	assert.Equal(t, params.Author, report.Author)
	assert.Equal(t, params.ReportType, report.ReportType)
	assert.Equal(t, params.Format, report.Format)
	assert.Equal(t, params.Content, report.Content)
	assert.Equal(t, int64(len(params.Content)), report.SizeBytes)
	assert.NotNil(t, report.ExpiresAt)
	assert.NotEqual(t, uuid.Nil, report.ID)

	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Create_WithoutTTL(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	params := &CreateParams{
		Title:   "Test Report",
		Content: []byte("test"),
		TTL:     0, // No TTL
	}

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.NewCommandTag("INSERT 0 1"), nil)

	report, err := repo.Create(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Nil(t, report.ExpiresAt)

	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Create_Error(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	params := &CreateParams{
		Title:   "Test Report",
		Content: []byte("test"),
	}

	expectedErr := errors.New("database error")
	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.CommandTag{}, expectedErr)

	report, err := repo.Create(ctx, params)

	require.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "failed to insert report")

	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	mockRow := &MockRow{scanErr: pgx.ErrNoRows}

	mockDB.On("QueryRow", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(mockRow)

	report, err := repo.Get(ctx, id)

	require.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, report)

	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_GetContent_NotFound(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	mockRow := &MockRow{scanErr: pgx.ErrNoRows}

	mockDB.On("QueryRow", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(mockRow)

	content, err := repo.GetContent(ctx, id)

	require.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, content)

	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_GetContent_Success(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	expectedContent := []byte("test content data")
	mockRow := &MockRow{
		values: []any{expectedContent},
	}

	mockDB.On("QueryRow", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(mockRow)

	content, err := repo.GetContent(ctx, id)

	require.NoError(t, err)
	assert.Equal(t, expectedContent, content)

	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Delete(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.NewCommandTag("UPDATE 1"), nil)

	err := repo.Delete(ctx, id)

	require.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.NewCommandTag("UPDATE 0"), nil)

	err := repo.Delete(ctx, id)

	require.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Delete_Error(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	expectedErr := errors.New("db error")

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.CommandTag{}, expectedErr)

	err := repo.Delete(ctx, id)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete report")
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_HardDelete(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.NewCommandTag("DELETE 1"), nil)

	err := repo.HardDelete(ctx, id)

	require.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_HardDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.NewCommandTag("DELETE 0"), nil)

	err := repo.HardDelete(ctx, id)

	require.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_HardDelete_Error(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	expectedErr := errors.New("db error")

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.CommandTag{}, expectedErr)

	err := repo.HardDelete(ctx, id)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to hard delete report")
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_DeleteExpired(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.NewCommandTag("DELETE 5"), nil)

	count, err := repo.DeleteExpired(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_DeleteExpired_Error(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	expectedErr := errors.New("db error")
	mockDB.On("Exec", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(pgconn.CommandTag{}, expectedErr)

	count, err := repo.DeleteExpired(ctx)

	require.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Contains(t, err.Error(), "failed to delete expired")
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_UpdateTags_Replace(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	newTags := []string{"new", "tags"}

	mockRow := &MockRow{
		values: []any{newTags},
	}

	mockDB.On("QueryRow", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(mockRow)

	tags, err := repo.UpdateTags(ctx, id, newTags, true)

	require.NoError(t, err)
	assert.Equal(t, newTags, tags)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_UpdateTags_Append(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	appendTags := []string{"additional"}

	mockRow := &MockRow{
		values: []any{[]string{"existing", "additional"}},
	}

	mockDB.On("QueryRow", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(mockRow)

	tags, err := repo.UpdateTags(ctx, id, appendTags, false)

	require.NoError(t, err)
	assert.Contains(t, tags, "existing")
	assert.Contains(t, tags, "additional")
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_UpdateTags_NotFound(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	mockRow := &MockRow{scanErr: pgx.ErrNoRows}

	mockDB.On("QueryRow", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(mockRow)

	tags, err := repo.UpdateTags(ctx, id, []string{"tag"}, true)

	require.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, tags)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_UpdateTags_Error(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	id := uuid.New()
	mockRow := &MockRow{scanErr: errors.New("db error")}

	mockDB.On("QueryRow", ctx, mock.AnythingOfType("string"), mock.Anything).
		Return(mockRow)

	tags, err := repo.UpdateTags(ctx, id, []string{"tag"}, true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update tags")
	assert.Nil(t, tags)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Ping(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	mockDB.On("Ping", ctx).Return(nil)

	err := repo.Ping(ctx)

	require.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Ping_Error(t *testing.T) {
	ctx := context.Background()
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	expectedErr := errors.New("connection failed")
	mockDB.On("Ping", ctx).Return(expectedErr)

	err := repo.Ping(ctx)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockDB.AssertExpectations(t)
}

func TestPostgresRepository_Close(t *testing.T) {
	mockDB := new(MockDB)
	repo := NewPostgresRepository(mockDB)

	mockDB.On("Close").Return()

	err := repo.Close()

	require.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestBuildListConditions(t *testing.T) {
	repo := &PostgresRepository{}

	tests := []struct {
		name        string
		params      *ListParams
		expectedLen int
	}{
		{
			name:        "empty params",
			params:      &ListParams{},
			expectedLen: 1, // deleted_at IS NULL
		},
		{
			name: "with report type",
			params: &ListParams{
				ReportType: func() *reportv1.ReportType {
					t := reportv1.ReportType_REPORT_TYPE_FLOW
					return &t
				}(),
			},
			expectedLen: 2,
		},
		{
			name: "with format",
			params: &ListParams{
				Format: func() *reportv1.ReportFormat {
					f := reportv1.ReportFormat_REPORT_FORMAT_PDF
					return &f
				}(),
			},
			expectedLen: 2,
		},
		{
			name: "with calculation_id",
			params: &ListParams{
				CalculationID: "calc-123",
			},
			expectedLen: 2,
		},
		{
			name: "with graph_id",
			params: &ListParams{
				GraphID: "graph-456",
			},
			expectedLen: 2,
		},
		{
			name: "with user_id",
			params: &ListParams{
				UserID: "user-789",
			},
			expectedLen: 2,
		},
		{
			name: "with tags",
			params: &ListParams{
				Tags: []string{"tag1", "tag2"},
			},
			expectedLen: 2,
		},
		{
			name: "with created_after",
			params: &ListParams{
				CreatedAfter: func() *time.Time { t := time.Now(); return &t }(),
			},
			expectedLen: 2,
		},
		{
			name: "with created_before",
			params: &ListParams{
				CreatedBefore: func() *time.Time { t := time.Now(); return &t }(),
			},
			expectedLen: 2,
		},
		{
			name: "with all filters",
			params: &ListParams{
				ReportType: func() *reportv1.ReportType {
					t := reportv1.ReportType_REPORT_TYPE_FLOW
					return &t
				}(),
				Format: func() *reportv1.ReportFormat {
					f := reportv1.ReportFormat_REPORT_FORMAT_PDF
					return &f
				}(),
				CalculationID: "calc-123",
				GraphID:       "graph-456",
				UserID:        "user-789",
				Tags:          []string{"tag1"},
				CreatedAfter:  func() *time.Time { t := time.Now(); return &t }(),
				CreatedBefore: func() *time.Time { t := time.Now(); return &t }(),
			},
			expectedLen: 9,
		},
		{
			name: "unspecified report type ignored",
			params: &ListParams{
				ReportType: func() *reportv1.ReportType {
					t := reportv1.ReportType_REPORT_TYPE_UNSPECIFIED
					return &t
				}(),
			},
			expectedLen: 1, // только deleted_at IS NULL
		},
		{
			name: "unspecified format ignored",
			params: &ListParams{
				Format: func() *reportv1.ReportFormat {
					f := reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED
					return &f
				}(),
			},
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions, _ := repo.buildListConditions(tt.params)
			assert.Len(t, conditions, tt.expectedLen)
		})
	}
}

// Тесты для моделей
func TestReport_ToMetadata(t *testing.T) {
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)

	report := &Report{
		ID:               uuid.New(),
		Title:            "Test Report",
		Description:      "Test Description",
		Author:           "Test Author",
		ReportType:       reportv1.ReportType_REPORT_TYPE_FLOW,
		Format:           reportv1.ReportFormat_REPORT_FORMAT_PDF,
		CreatedAt:        now,
		ExpiresAt:        &expiresAt,
		GenerationTimeMs: 100.5,
		SizeBytes:        1024,
		CalculationID:    "calc-123",
		GraphID:          "graph-456",
		Tags:             []string{"tag1", "tag2"},
		CustomFields:     map[string]string{"key": "value"},
		Version:          "1.0.0",
	}

	meta := report.ToMetadata()

	require.NotNil(t, meta)
	assert.Equal(t, report.ID.String(), meta.ReportId)
	assert.Equal(t, report.Title, meta.Title)
	assert.Equal(t, report.Description, meta.Description)
	assert.Equal(t, report.Author, meta.GeneratedBy)
	assert.Equal(t, report.ReportType, meta.Type)
	assert.Equal(t, report.Format, meta.Format)
	assert.Equal(t, report.GenerationTimeMs, meta.GenerationTimeMs)
	assert.Equal(t, report.SizeBytes, meta.SizeBytes)
	assert.Equal(t, report.CalculationID, meta.CalculationId)
	assert.Equal(t, report.GraphID, meta.GraphId)
	assert.Equal(t, report.Tags, meta.Tags)
	assert.Equal(t, report.CustomFields, meta.CustomFields)
	assert.Equal(t, report.Version, meta.Version)
	assert.NotNil(t, meta.GeneratedAt)
	assert.NotNil(t, meta.ExpiresAt)
}

func TestReport_ToMetadata_WithoutExpiry(t *testing.T) {
	report := &Report{
		ID:        uuid.New(),
		Title:     "Test",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: nil,
	}

	meta := report.ToMetadata()

	require.NotNil(t, meta)
	assert.Nil(t, meta.ExpiresAt)
}

func TestReport_ToContent(t *testing.T) {
	report := &Report{
		Content:     []byte("test content"),
		ContentType: "application/pdf",
		Filename:    "test.pdf",
		SizeBytes:   12,
	}

	content := report.ToContent()

	require.NotNil(t, content)
	assert.Equal(t, report.Content, content.Data)
	assert.Equal(t, report.ContentType, content.ContentType)
	assert.Equal(t, report.Filename, content.Filename)
	assert.Equal(t, report.SizeBytes, content.SizeBytes)
}

func TestStats_ToProto(t *testing.T) {
	oldest := time.Now().Add(-24 * time.Hour)
	newest := time.Now()

	stats := &Stats{
		TotalReports:   100,
		TotalSizeBytes: 1024 * 1024,
		AvgSizeBytes:   10240,
		ReportsByType: map[string]int64{
			"REPORT_TYPE_FLOW":      50,
			"REPORT_TYPE_ANALYTICS": 30,
		},
		ReportsByFormat: map[string]int64{
			"REPORT_FORMAT_PDF": 60,
			"REPORT_FORMAT_CSV": 40,
		},
		SizeByType: map[string]int64{
			"REPORT_TYPE_FLOW": 512 * 1024,
		},
		OldestReportAt: &oldest,
		NewestReportAt: &newest,
		ExpiredReports: 5,
	}

	proto := stats.ToProto()

	require.NotNil(t, proto)
	assert.Equal(t, stats.TotalReports, proto.TotalReports)
	assert.Equal(t, stats.TotalSizeBytes, proto.TotalSizeBytes)
	assert.Equal(t, stats.AvgSizeBytes, proto.AvgSizeBytes)
	assert.Equal(t, stats.ReportsByType, proto.ReportsByType)
	assert.Equal(t, stats.ReportsByFormat, proto.ReportsByFormat)
	assert.Equal(t, stats.SizeByType, proto.SizeByType)
	assert.Equal(t, stats.ExpiredReports, proto.ExpiredReports)
	assert.NotNil(t, proto.OldestReportAt)
	assert.NotNil(t, proto.NewestReportAt)
}

func TestStats_ToProto_NilDates(t *testing.T) {
	stats := &Stats{
		TotalReports:    10,
		ReportsByType:   make(map[string]int64),
		ReportsByFormat: make(map[string]int64),
		SizeByType:      make(map[string]int64),
		OldestReportAt:  nil,
		NewestReportAt:  nil,
	}

	proto := stats.ToProto()

	require.NotNil(t, proto)
	assert.Nil(t, proto.OldestReportAt)
	assert.Nil(t, proto.NewestReportAt)
}

// Тесты для вспомогательных функций
func TestNullString(t *testing.T) {
	tests := []struct {
		input    string
		expected sql.NullString
	}{
		{
			input:    "",
			expected: sql.NullString{String: "", Valid: false},
		},
		{
			input:    "test",
			expected: sql.NullString{String: "test", Valid: true},
		},
		{
			input:    "  ",
			expected: sql.NullString{String: "  ", Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := nullString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseReportType(t *testing.T) {
	tests := []struct {
		input    string
		expected reportv1.ReportType
	}{
		{
			input:    "REPORT_TYPE_FLOW",
			expected: reportv1.ReportType_REPORT_TYPE_FLOW,
		},
		{
			input:    "REPORT_TYPE_ANALYTICS",
			expected: reportv1.ReportType_REPORT_TYPE_ANALYTICS,
		},
		{
			input:    "REPORT_TYPE_SIMULATION",
			expected: reportv1.ReportType_REPORT_TYPE_SIMULATION,
		},
		{
			input:    "REPORT_TYPE_SUMMARY",
			expected: reportv1.ReportType_REPORT_TYPE_SUMMARY,
		},
		{
			input:    "REPORT_TYPE_HISTORY",
			expected: reportv1.ReportType_REPORT_TYPE_HISTORY,
		},
		{
			input:    "REPORT_TYPE_COMPARISON",
			expected: reportv1.ReportType_REPORT_TYPE_COMPARISON,
		},
		{
			input:    "UNKNOWN",
			expected: reportv1.ReportType_REPORT_TYPE_UNSPECIFIED,
		},
		{
			input:    "",
			expected: reportv1.ReportType_REPORT_TYPE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseReportType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseReportFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected reportv1.ReportFormat
	}{
		{
			input:    "REPORT_FORMAT_PDF",
			expected: reportv1.ReportFormat_REPORT_FORMAT_PDF,
		},
		{
			input:    "REPORT_FORMAT_CSV",
			expected: reportv1.ReportFormat_REPORT_FORMAT_CSV,
		},
		{
			input:    "REPORT_FORMAT_EXCEL",
			expected: reportv1.ReportFormat_REPORT_FORMAT_EXCEL,
		},
		{
			input:    "REPORT_FORMAT_MARKDOWN",
			expected: reportv1.ReportFormat_REPORT_FORMAT_MARKDOWN,
		},
		{
			input:    "REPORT_FORMAT_HTML",
			expected: reportv1.ReportFormat_REPORT_FORMAT_HTML,
		},
		{
			input:    "REPORT_FORMAT_JSON",
			expected: reportv1.ReportFormat_REPORT_FORMAT_JSON,
		},
		{
			input:    "UNKNOWN",
			expected: reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED,
		},
		{
			input:    "",
			expected: reportv1.ReportFormat_REPORT_FORMAT_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseReportFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Тесты для ошибок
func TestErrors(t *testing.T) {
	assert.Error(t, ErrNotFound)
	assert.Error(t, ErrAlreadyExists)
	assert.Error(t, ErrInvalidID)
	assert.Error(t, ErrStorageFull)

	assert.Equal(t, "report not found", ErrNotFound.Error())
	assert.Equal(t, "report already exists", ErrAlreadyExists.Error())
	assert.Equal(t, "invalid report ID", ErrInvalidID.Error())
	assert.Equal(t, "storage quota exceeded", ErrStorageFull.Error())
}

// Тест для CreateParams
func TestCreateParams(t *testing.T) {
	params := &CreateParams{
		Title:            "Test",
		Description:      "Description",
		Author:           "Author",
		ReportType:       reportv1.ReportType_REPORT_TYPE_FLOW,
		Format:           reportv1.ReportFormat_REPORT_FORMAT_PDF,
		Content:          []byte("content"),
		ContentType:      "application/pdf",
		Filename:         "test.pdf",
		CalculationID:    "calc",
		GraphID:          "graph",
		UserID:           "user",
		GenerationTimeMs: 100,
		Version:          "1.0",
		Tags:             []string{"tag"},
		CustomFields:     map[string]string{"k": "v"},
		TTL:              time.Hour,
	}

	assert.Equal(t, "Test", params.Title)
	assert.Equal(t, time.Hour, params.TTL)
}

// Тест для ListParams
func TestListParams(t *testing.T) {
	now := time.Now()
	reportType := reportv1.ReportType_REPORT_TYPE_FLOW
	format := reportv1.ReportFormat_REPORT_FORMAT_PDF

	params := &ListParams{
		Limit:         10,
		Offset:        5,
		ReportType:    &reportType,
		Format:        &format,
		CalculationID: "calc",
		GraphID:       "graph",
		UserID:        "user",
		Tags:          []string{"tag1", "tag2"},
		CreatedAfter:  &now,
		CreatedBefore: &now,
		OrderBy:       "created_at",
		OrderDesc:     true,
	}

	assert.Equal(t, int32(10), params.Limit)
	assert.Equal(t, int32(5), params.Offset)
	assert.True(t, params.OrderDesc)
}

// Тест для ListResult
func TestListResult(t *testing.T) {
	result := &ListResult{
		Reports: []*Report{
			{ID: uuid.New(), Title: "Report 1"},
			{ID: uuid.New(), Title: "Report 2"},
		},
		TotalCount: 100,
		HasMore:    true,
	}

	assert.Len(t, result.Reports, 2)
	assert.Equal(t, int64(100), result.TotalCount)
	assert.True(t, result.HasMore)
}
