// services/history-svc/internal/service/history_test.go

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonv1 "logistics/gen/go/logistics/common/v1"
	historyv1 "logistics/gen/go/logistics/history/v1"
	optimizationv1 "logistics/gen/go/logistics/optimization/v1"
	"logistics/services/history-svc/internal/repository"
)

// Mock repository
type mockCalculationRepository struct {
	calculations map[string]*repository.Calculation
	nextID       int
}

func newMockRepository() *mockCalculationRepository {
	return &mockCalculationRepository{
		calculations: make(map[string]*repository.Calculation),
		nextID:       1,
	}
}

func (m *mockCalculationRepository) Create(ctx context.Context, calc *repository.Calculation) error {
	calc.ID = fmt.Sprintf("calc-%d", m.nextID)
	calc.CreatedAt = time.Now()
	calc.UpdatedAt = time.Now()
	m.nextID++
	m.calculations[calc.ID] = calc
	return nil
}

func (m *mockCalculationRepository) GetByID(ctx context.Context, id string) (*repository.Calculation, error) {
	if calc, ok := m.calculations[id]; ok {
		return calc, nil
	}
	return nil, repository.ErrCalculationNotFound
}

func (m *mockCalculationRepository) Delete(ctx context.Context, id string) error {
	if _, ok := m.calculations[id]; !ok {
		return repository.ErrCalculationNotFound
	}
	delete(m.calculations, id)
	return nil
}

func (m *mockCalculationRepository) List(ctx context.Context, userID string, opts *repository.ListOptions) ([]*repository.CalculationSummary, int64, error) {
	var results []*repository.CalculationSummary
	for _, calc := range m.calculations {
		if calc.UserID == userID {
			results = append(results, &repository.CalculationSummary{
				ID:                calc.ID,
				Name:              calc.Name,
				Algorithm:         calc.Algorithm,
				MaxFlow:           calc.MaxFlow,
				TotalCost:         calc.TotalCost,
				ComputationTimeMs: calc.ComputationTimeMs,
				NodeCount:         calc.NodeCount,
				EdgeCount:         calc.EdgeCount,
				Tags:              calc.Tags,
				CreatedAt:         calc.CreatedAt,
			})
		}
	}
	return results, int64(len(results)), nil
}

func (m *mockCalculationRepository) GetUserStatistics(ctx context.Context, userID string, startTime, endTime *time.Time) (*repository.UserStatistics, error) {
	return &repository.UserStatistics{
		TotalCalculations:        10,
		AverageMaxFlow:           100.0,
		AverageTotalCost:         500.0,
		AverageComputationTimeMs: 150.0,
		CalculationsByAlgorithm:  map[string]int{"ALGORITHM_DINIC": 7, "ALGORITHM_EDMONDS_KARP": 3},
		DailyStats:               []repository.DailyStats{},
	}, nil
}

func (m *mockCalculationRepository) Search(ctx context.Context, userID string, query string, limit int) ([]*repository.CalculationSummary, error) {
	return []*repository.CalculationSummary{}, nil
}

func TestNewHistoryService(t *testing.T) {
	repo := newMockRepository()
	svc := NewHistoryService(repo)

	if svc == nil {
		t.Fatal("NewHistoryService should not return nil")
	}
	if svc.repo == nil {
		t.Error("repo should not be nil")
	}
}

func TestHistoryService_SaveCalculation(t *testing.T) {
	repo := newMockRepository()
	svc := NewHistoryService(repo)
	ctx := context.Background()

	tests := []struct {
		name    string
		request *historyv1.SaveCalculationRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &historyv1.SaveCalculationRequest{
				UserId: "user-123",
				Name:   "Test Calculation",
				Request: &optimizationv1.SolveRequest{
					Graph: &commonv1.Graph{
						Nodes: []*commonv1.Node{{Id: 1}, {Id: 2}},
						Edges: []*commonv1.Edge{{From: 1, To: 2, Capacity: 100}},
					},
					Algorithm: commonv1.Algorithm_ALGORITHM_DINIC,
				},
				Response: &optimizationv1.SolveResponse{
					Success: true,
					Result: &commonv1.FlowResult{
						MaxFlow:   100.0,
						TotalCost: 500.0,
					},
				},
				Tags: map[string]string{"env": "test"},
			},
			wantErr: false,
		},
		{
			name: "missing user_id",
			request: &historyv1.SaveCalculationRequest{
				UserId: "",
				Request: &optimizationv1.SolveRequest{
					Graph: &commonv1.Graph{},
				},
				Response: &optimizationv1.SolveResponse{},
			},
			wantErr: true,
		},
		{
			name: "missing request",
			request: &historyv1.SaveCalculationRequest{
				UserId:   "user-123",
				Request:  nil,
				Response: &optimizationv1.SolveResponse{},
			},
			wantErr: true,
		},
		{
			name: "missing response",
			request: &historyv1.SaveCalculationRequest{
				UserId:   "user-123",
				Request:  &optimizationv1.SolveRequest{},
				Response: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.SaveCalculation(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("SaveCalculation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp.CalculationId == "" {
					t.Error("CalculationId should not be empty")
				}
				if resp.CreatedAt == nil {
					t.Error("CreatedAt should not be nil")
				}
			}
		})
	}
}

func TestHistoryService_GetCalculation(t *testing.T) {
	repo := newMockRepository()
	svc := NewHistoryService(repo)
	ctx := context.Background()

	// Создаём расчёт
	calc := &repository.Calculation{
		UserID:       "user-123",
		Name:         "Test",
		Algorithm:    "ALGORITHM_DINIC",
		MaxFlow:      100.0,
		RequestData:  []byte(`{}`),
		ResponseData: []byte(`{}`),
	}
	_ = repo.Create(ctx, calc)

	tests := []struct {
		name    string
		request *historyv1.GetCalculationRequest
		wantErr bool
	}{
		{
			name: "existing calculation",
			request: &historyv1.GetCalculationRequest{
				CalculationId: calc.ID,
				UserId:        "user-123",
			},
			wantErr: false,
		},
		{
			name: "non-existing calculation",
			request: &historyv1.GetCalculationRequest{
				CalculationId: "non-existing",
				UserId:        "user-123",
			},
			wantErr: true,
		},
		{
			name: "empty calculation_id",
			request: &historyv1.GetCalculationRequest{
				CalculationId: "",
				UserId:        "user-123",
			},
			wantErr: true,
		},
		{
			name: "wrong user_id",
			request: &historyv1.GetCalculationRequest{
				CalculationId: calc.ID,
				UserId:        "other-user",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.GetCalculation(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCalculation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resp.Record == nil {
				t.Error("Record should not be nil")
			}
		})
	}
}

func TestHistoryService_ListCalculations(t *testing.T) {
	repo := newMockRepository()
	svc := NewHistoryService(repo)
	ctx := context.Background()

	// Создаём несколько расчётов
	for i := 0; i < 5; i++ {
		calc := &repository.Calculation{
			UserID:       "user-123",
			Name:         fmt.Sprintf("Calc %d", i),
			RequestData:  []byte(`{}`),
			ResponseData: []byte(`{}`),
		}
		_ = repo.Create(ctx, calc)
	}

	tests := []struct {
		name    string
		request *historyv1.ListCalculationsRequest
		wantErr bool
		minLen  int
	}{
		{
			name: "list all",
			request: &historyv1.ListCalculationsRequest{
				UserId: "user-123",
			},
			wantErr: false,
			minLen:  5,
		},
		{
			name: "empty user_id",
			request: &historyv1.ListCalculationsRequest{
				UserId: "",
			},
			wantErr: true,
		},
		{
			name: "with pagination",
			request: &historyv1.ListCalculationsRequest{
				UserId: "user-123",
				Pagination: &commonv1.PaginationRequest{
					Page:     1,
					PageSize: 2,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.ListCalculations(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListCalculations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp.Pagination == nil {
					t.Error("Pagination should not be nil")
				}
				if tt.minLen > 0 && len(resp.Calculations) < tt.minLen {
					t.Errorf("Expected at least %d calculations, got %d", tt.minLen, len(resp.Calculations))
				}
			}
		})
	}
}

func TestHistoryService_DeleteCalculation(t *testing.T) {
	repo := newMockRepository()
	svc := NewHistoryService(repo)
	ctx := context.Background()

	// Создаём расчёт
	calc := &repository.Calculation{
		UserID:       "user-123",
		Name:         "To Delete",
		RequestData:  []byte(`{}`),
		ResponseData: []byte(`{}`),
	}
	_ = repo.Create(ctx, calc)

	tests := []struct {
		name    string
		request *historyv1.DeleteCalculationRequest
		wantErr bool
	}{
		{
			name: "delete existing",
			request: &historyv1.DeleteCalculationRequest{
				CalculationId: calc.ID,
				UserId:        "user-123",
			},
			wantErr: false,
		},
		{
			name: "empty ids",
			request: &historyv1.DeleteCalculationRequest{
				CalculationId: "",
				UserId:        "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.DeleteCalculation(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteCalculation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !resp.Success {
				t.Error("Success should be true")
			}
		})
	}
}

func TestHistoryService_GetStatistics(t *testing.T) {
	repo := newMockRepository()
	svc := NewHistoryService(repo)
	ctx := context.Background()

	tests := []struct {
		name    string
		request *historyv1.GetStatisticsRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &historyv1.GetStatisticsRequest{
				UserId: "user-123",
			},
			wantErr: false,
		},
		{
			name: "with time range",
			request: &historyv1.GetStatisticsRequest{
				UserId: "user-123",
				TimeRange: &commonv1.TimeRange{
					StartTimestamp: time.Now().Add(-24 * time.Hour).Unix(),
					EndTimestamp:   time.Now().Unix(),
				},
			},
			wantErr: false,
		},
		{
			name: "empty user_id",
			request: &historyv1.GetStatisticsRequest{
				UserId: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.GetStatistics(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatistics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp.TotalCalculations < 0 {
					t.Error("TotalCalculations should be non-negative")
				}
			}
		})
	}
}

func TestSplitOnce(t *testing.T) {
	tests := []struct {
		s        string
		sep      string
		expected []string
	}{
		{"key:value", ":", []string{"key", "value"}},
		{"key:value:extra", ":", []string{"key", "value:extra"}},
		{"nodelimiter", ":", []string{"nodelimiter"}},
		{"", ":", []string{""}},
		{"key:", ":", []string{"key", ""}},
		{":value", ":", []string{"", "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := splitOnce(tt.s, tt.sep)
			if len(result) != len(tt.expected) {
				t.Errorf("splitOnce(%q, %q) length = %d, want %d", tt.s, tt.sep, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitOnce(%q, %q)[%d] = %q, want %q", tt.s, tt.sep, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestToInt32Map(t *testing.T) {
	input := map[string]int{
		"a": 1,
		"b": 100,
		"c": 999,
	}

	result := toInt32Map(input)

	if len(result) != len(input) {
		t.Errorf("Length = %d, want %d", len(result), len(input))
	}

	for k, v := range input {
		if result[k] != int32(v) {
			t.Errorf("result[%q] = %d, want %d", k, result[k], v)
		}
	}
}
