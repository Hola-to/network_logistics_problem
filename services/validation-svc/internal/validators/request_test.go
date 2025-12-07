package validators

import (
	"testing"

	commonv1 "logistics/gen/go/logistics/common/v1"
)

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name       string
		graph      *commonv1.Graph
		wantErrors int
	}{
		{
			name:       "nil_graph",
			graph:      nil,
			wantErrors: 1,
		},
		{
			name:       "valid_graph",
			graph:      createValidGraph(),
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateRequest(tt.graph)

			if len(errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d", len(errors), tt.wantErrors)
			}
		})
	}
}

func TestValidateAlgorithmChoice(t *testing.T) {
	tests := []struct {
		name       string
		algorithm  commonv1.Algorithm
		wantErrors int
	}{
		{
			name:       "edmonds_karp",
			algorithm:  commonv1.Algorithm_ALGORITHM_EDMONDS_KARP,
			wantErrors: 0,
		},
		{
			name:       "dinic",
			algorithm:  commonv1.Algorithm_ALGORITHM_DINIC,
			wantErrors: 0,
		},
		{
			name:       "min_cost",
			algorithm:  commonv1.Algorithm_ALGORITHM_MIN_COST,
			wantErrors: 0,
		},
		{
			name:       "push_relabel",
			algorithm:  commonv1.Algorithm_ALGORITHM_PUSH_RELABEL,
			wantErrors: 0,
		},
		{
			name:       "ford_fulkerson",
			algorithm:  commonv1.Algorithm_ALGORITHM_FORD_FULKERSON,
			wantErrors: 0,
		},
		{
			name:       "unspecified",
			algorithm:  commonv1.Algorithm_ALGORITHM_UNSPECIFIED,
			wantErrors: 0, // Unspecified is allowed
		},
		{
			name:       "invalid",
			algorithm:  commonv1.Algorithm(999),
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateAlgorithmChoice(tt.algorithm)

			if len(errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d", len(errors), tt.wantErrors)
			}
		})
	}
}

func TestValidateThreshold(t *testing.T) {
	tests := []struct {
		name       string
		value      float64
		fieldName  string
		min        float64
		max        float64
		wantErrors int
	}{
		{
			name:       "within_range",
			value:      0.5,
			fieldName:  "utilization",
			min:        0.0,
			max:        1.0,
			wantErrors: 0,
		},
		{
			name:       "at_min",
			value:      0.0,
			fieldName:  "utilization",
			min:        0.0,
			max:        1.0,
			wantErrors: 0,
		},
		{
			name:       "at_max",
			value:      1.0,
			fieldName:  "utilization",
			min:        0.0,
			max:        1.0,
			wantErrors: 0,
		},
		{
			name:       "below_min",
			value:      -0.1,
			fieldName:  "utilization",
			min:        0.0,
			max:        1.0,
			wantErrors: 1,
		},
		{
			name:       "above_max",
			value:      1.1,
			fieldName:  "utilization",
			min:        0.0,
			max:        1.0,
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateThreshold(tt.value, tt.fieldName, tt.min, tt.max)

			if len(errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d", len(errors), tt.wantErrors)
			}
		})
	}
}

func TestValidatePagination(t *testing.T) {
	tests := []struct {
		name       string
		page       int32
		pageSize   int32
		wantErrors int
	}{
		{
			name:       "valid",
			page:       0,
			pageSize:   20,
			wantErrors: 0,
		},
		{
			name:       "negative_page",
			page:       -1,
			pageSize:   20,
			wantErrors: 1,
		},
		{
			name:       "negative_page_size",
			page:       0,
			pageSize:   -1,
			wantErrors: 1,
		},
		{
			name:       "page_size_too_large",
			page:       0,
			pageSize:   1001,
			wantErrors: 1,
		},
		{
			name:       "max_valid_page_size",
			page:       0,
			pageSize:   1000,
			wantErrors: 0,
		},
		{
			name:       "all_invalid",
			page:       -1,
			pageSize:   -1,
			wantErrors: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidatePagination(tt.page, tt.pageSize)

			if len(errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d: %+v", len(errors), tt.wantErrors, errors)
			}
		})
	}
}
