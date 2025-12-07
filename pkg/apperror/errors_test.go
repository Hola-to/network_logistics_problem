// Package apperror provides tests for the custom error types and utility functions.
package apperror

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestError_Error verifies that the Error() method returns the correct string format.
func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "without field",
			err:      New(CodeInvalidGraph, "graph is invalid"),
			expected: "[INVALID_GRAPH] graph is invalid",
		},
		{
			name:     "with field",
			err:      NewWithField(CodeInvalidSource, "source not found", "source_id"),
			expected: "[INVALID_SOURCE] source not found (field: source_id)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestError_Unwrap verifies that the Unwrap() method correctly returns the underlying cause.
func TestError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(cause, CodeInternal, "wrapped error")

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

// TestError_GRPCStatus verifies that the GRPCStatus() method maps ErrorCodes to correct gRPC codes.
func TestError_GRPCStatus(t *testing.T) {
	tests := []struct {
		name         string
		code         ErrorCode
		expectedCode codes.Code
	}{
		{"invalid argument", CodeInvalidGraph, codes.InvalidArgument},
		{"not found", CodeNotFound, codes.NotFound},
		{"timeout", CodeTimeout, codes.DeadlineExceeded},
		{"unauthenticated", CodeUnauthenticated, codes.Unauthenticated},
		{"permission denied", CodePermissionDenied, codes.PermissionDenied},
		{"no path", CodeNoPath, codes.FailedPrecondition},
		{"infeasible", CodeInfeasible, codes.Aborted},
		{"internal", CodeInternal, codes.Internal},
		{"flow violation", CodeFlowViolation, codes.DataLoss},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, "test message")
			st := err.GRPCStatus()
			if st.Code() != tt.expectedCode {
				t.Errorf("GRPCStatus().Code() = %v, want %v", st.Code(), tt.expectedCode)
			}
		})
	}
}

// TestNew verifies the New function correctly initializes an Error.
func TestNew(t *testing.T) {
	err := New(CodeEmptyGraph, "graph is empty")

	if err.Code != CodeEmptyGraph {
		t.Errorf("Code = %v, want %v", err.Code, CodeEmptyGraph)
	}
	if err.Message != "graph is empty" {
		t.Errorf("Message = %v, want %v", err.Message, "graph is empty")
	}
	if err.Severity != SeverityError {
		t.Errorf("Severity = %v, want %v", err.Severity, SeverityError)
	}
}

// TestNewWarning verifies the NewWarning function correctly initializes an Error with SeverityWarning.
func TestNewWarning(t *testing.T) {
	err := NewWarning(CodeBottleneckDetected, "bottleneck found")

	if err.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", err.Severity, SeverityWarning)
	}
}

// TestNewCritical verifies the NewCritical function correctly initializes an Error with SeverityCritical.
func TestNewCritical(t *testing.T) {
	err := NewCritical(CodeInternal, "critical failure")

	if err.Severity != SeverityCritical {
		t.Errorf("Severity = %v, want %v", err.Severity, SeverityCritical)
	}
}

// TestWithDetails verifies that WithDetails adds key-value pairs to the error's details map.
func TestWithDetails(t *testing.T) {
	err := New(CodeInvalidGraph, "invalid").
		WithDetails("node_count", 5).
		WithDetails("edge_count", 10)

	if err.Details["node_count"] != 5 {
		t.Errorf("Details[node_count] = %v, want 5", err.Details["node_count"])
	}
	if err.Details["edge_count"] != 10 {
		t.Errorf("Details[edge_count] = %v, want 10", err.Details["edge_count"])
	}
}

// TestWithField verifies that WithField sets the field of the error.
func TestWithField(t *testing.T) {
	err := New(CodeInvalidSource, "invalid source").WithField("source_id")

	if err.Field != "source_id" {
		t.Errorf("Field = %v, want source_id", err.Field)
	}
}

// TestWithSeverity verifies that WithSeverity sets the severity level of the error.
func TestWithSeverity(t *testing.T) {
	err := New(CodeInvalidGraph, "invalid").WithSeverity(SeverityCritical)

	if err.Severity != SeverityCritical {
		t.Errorf("Severity = %v, want %v", err.Severity, SeverityCritical)
	}
}

// TestIs verifies the Is function correctly identifies errors by their ErrorCode.
func TestIs(t *testing.T) {
	err := New(CodeEmptyGraph, "empty graph")

	if !Is(err, CodeEmptyGraph) {
		t.Error("Is() should return true for matching code")
	}
	if Is(err, CodeInvalidGraph) {
		t.Error("Is() should return false for non-matching code")
	}
	if Is(errors.New("regular error"), CodeEmptyGraph) {
		t.Error("Is() should return false for non-Error")
	}
}

// TestCode verifies the Code function correctly extracts the ErrorCode.
func TestCode(t *testing.T) {
	err := New(CodeNoPath, "no path")

	if Code(err) != CodeNoPath {
		t.Errorf("Code() = %v, want %v", Code(err), CodeNoPath)
	}

	regularErr := errors.New("regular error")
	if Code(regularErr) != CodeInternal {
		t.Errorf("Code() for regular error = %v, want %v", Code(regularErr), CodeInternal)
	}
}

// TestToGRPC verifies the ToGRPC function's behavior with different error types.
func TestToGRPC(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if ToGRPC(nil) != nil {
			t.Error("ToGRPC(nil) should return nil")
		}
	})

	t.Run("app error", func(t *testing.T) {
		err := New(CodeInvalidGraph, "invalid graph")
		grpcErr := ToGRPC(err)
		st, _ := status.FromError(grpcErr)
		if st.Code() != codes.InvalidArgument {
			t.Errorf("ToGRPC() code = %v, want %v", st.Code(), codes.InvalidArgument)
		}
	})

	t.Run("regular error", func(t *testing.T) {
		err := errors.New("regular error")
		grpcErr := ToGRPC(err)
		st, _ := status.FromError(grpcErr)
		if st.Code() != codes.Internal {
			t.Errorf("ToGRPC() code = %v, want %v", st.Code(), codes.Internal)
		}
	})

	t.Run("already grpc error", func(t *testing.T) {
		grpcErr := status.Error(codes.NotFound, "not found")
		result := ToGRPC(grpcErr)
		st, _ := status.FromError(result)
		if st.Code() != codes.NotFound {
			t.Errorf("ToGRPC() should preserve grpc error code")
		}
	})
}

// TestFromGRPC verifies the FromGRPC function's behavior when converting gRPC errors.
func TestFromGRPC(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := FromGRPC(nil)
		if result != nil {
			t.Error("FromGRPC(nil) should return nil")
		}
	})

	t.Run("grpc error", func(t *testing.T) {
		grpcErr := status.Error(codes.NotFound, "resource not found")
		err := FromGRPC(grpcErr)
		assertErrorNotNil(t, err, "grpc error")
		assertErrorCode(t, err, CodeNotFound)
		assertErrorHasMessage(t, err)
	})

	t.Run("regular error", func(t *testing.T) {
		regularErr := errors.New("regular")
		err := FromGRPC(regularErr)
		assertErrorNotNil(t, err, "regular error")
		assertErrorCode(t, err, CodeInternal)
		assertErrorHasMessage(t, err)
	})
}

// assertErrorNotNil is a helper to check if an error is not nil.
func assertErrorNotNil(t *testing.T, err *Error, desc string) {
	t.Helper()
	if err == nil {
		t.Fatalf("FromGRPC() should not return nil for %s", desc)
	}
}

// assertErrorCode is a helper to check if an error has the expected ErrorCode.
func assertErrorCode(t *testing.T, err *Error, expected ErrorCode) {
	t.Helper()
	if err == nil {
		return
	}
	if err.Code != expected {
		t.Errorf("FromGRPC() code = %v, want %v", err.Code, expected)
	}
}

// assertErrorHasMessage is a helper to check if an error has a non-empty message.
func assertErrorHasMessage(t *testing.T, err *Error) {
	t.Helper()
	if err == nil {
		return
	}
	if err.Message == "" {
		t.Error("FromGRPC() message should not be empty")
	}
}

// TestIsWarning verifies the IsWarning function correctly identifies warning errors.
func TestIsWarning(t *testing.T) {
	warning := NewWarning(CodeBottleneckDetected, "bottleneck")
	err := New(CodeInvalidGraph, "invalid")

	if !IsWarning(warning) {
		t.Error("IsWarning() should return true for warning")
	}
	if IsWarning(err) {
		t.Error("IsWarning() should return false for error")
	}
}

// TestIsCritical verifies the IsCritical function correctly identifies critical errors.
func TestIsCritical(t *testing.T) {
	critical := NewCritical(CodeInternal, "critical")
	err := New(CodeInvalidGraph, "invalid")

	if !IsCritical(critical) {
		t.Error("IsCritical() should return true for critical")
	}
	if IsCritical(err) {
		t.Error("IsCritical() should return false for error")
	}
}

// TestSeverity_String verifies the String method of Severity returns the correct string representation.
func TestSeverity_String(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{SeverityCritical, "critical"},
		{Severity(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.severity.String(); got != tt.expected {
			t.Errorf("Severity.String() = %v, want %v", got, tt.expected)
		}
	}
}

// TestValidationErrors verifies the functionality of the ValidationErrors collection.
func TestValidationErrors(t *testing.T) {
	t.Run("new validation errors", func(t *testing.T) {
		ve := NewValidationErrors()
		if ve.HasErrors() {
			t.Error("new ValidationErrors should not have errors")
		}
		if ve.HasWarnings() {
			t.Error("new ValidationErrors should not have warnings")
		}
		if !ve.IsValid() {
			t.Error("new ValidationErrors should be valid")
		}
	})

	t.Run("add error", func(t *testing.T) {
		ve := NewValidationErrors()
		ve.AddError(CodeInvalidGraph, "invalid graph")

		if !ve.HasErrors() {
			t.Error("should have errors")
		}
		if ve.IsValid() {
			t.Error("should not be valid")
		}
		if len(ve.Errors) != 1 {
			t.Errorf("errors count = %d, want 1", len(ve.Errors))
		}
	})

	t.Run("add warning", func(t *testing.T) {
		ve := NewValidationErrors()
		ve.AddWarning(CodeBottleneckDetected, "bottleneck")

		if !ve.HasWarnings() {
			t.Error("should have warnings")
		}
		if !ve.IsValid() {
			t.Error("should be valid (warnings don't affect validity)")
		}
	})

	t.Run("add error with field", func(t *testing.T) {
		ve := NewValidationErrors()
		ve.AddErrorWithField(CodeInvalidSource, "invalid", "source_id")

		if ve.Errors[0].Field != "source_id" {
			t.Errorf("Field = %v, want source_id", ve.Errors[0].Field)
		}
	})

	t.Run("add via Add method", func(t *testing.T) {
		ve := NewValidationErrors()
		ve.Add(NewWarning(CodeBottleneckDetected, "warning"))
		ve.Add(New(CodeInvalidGraph, "error"))

		if len(ve.Warnings) != 1 {
			t.Errorf("warnings count = %d, want 1", len(ve.Warnings))
		}
		if len(ve.Errors) != 1 {
			t.Errorf("errors count = %d, want 1", len(ve.Errors))
		}
	})

	t.Run("merge", func(t *testing.T) {
		ve1 := NewValidationErrors()
		ve1.AddError(CodeInvalidGraph, "error1")

		ve2 := NewValidationErrors()
		ve2.AddError(CodeInvalidSource, "error2")
		ve2.AddWarning(CodeBottleneckDetected, "warning")

		ve1.Merge(ve2)

		if len(ve1.Errors) != 2 {
			t.Errorf("errors count = %d, want 2", len(ve1.Errors))
		}
		if len(ve1.Warnings) != 1 {
			t.Errorf("warnings count = %d, want 1", len(ve1.Warnings))
		}
	})

	t.Run("merge nil", func(t *testing.T) {
		ve := NewValidationErrors()
		ve.Merge(nil) // should not panic
	})

	t.Run("error messages", func(t *testing.T) {
		ve := NewValidationErrors()
		ve.AddError(CodeInvalidGraph, "error1")
		ve.AddError(CodeInvalidSource, "error2")

		messages := ve.ErrorMessages()
		if len(messages) != 2 {
			t.Errorf("messages count = %d, want 2", len(messages))
		}
	})

	t.Run("warning messages", func(t *testing.T) {
		ve := NewValidationErrors()
		ve.AddWarning(CodeBottleneckDetected, "warning1")

		messages := ve.WarningMessages()
		if len(messages) != 1 {
			t.Errorf("messages count = %d, want 1", len(messages))
		}
		if messages[0] != "warning1" {
			t.Errorf("message = %v, want warning1", messages[0])
		}
	})
}

// TestPredefinedErrors verifies that all predefined errors are correctly initialized.
func TestPredefinedErrors(t *testing.T) {
	predefinedErrors := []*Error{
		ErrEmptyGraph,
		ErrInvalidSource,
		ErrInvalidSink,
		ErrSourceEqualsSink,
		ErrNoPath,
		ErrNegativeCycle,
		ErrTimeout,
		ErrNilGraph,
		ErrIterationLimit,
	}

	for _, err := range predefinedErrors {
		if err == nil {
			t.Error("predefined error should not be nil")
			continue
		}
		if err.Code == "" {
			t.Error("predefined error should have a code")
		}
		if err.Message == "" {
			t.Error("predefined error should have a message")
		}
	}
}
