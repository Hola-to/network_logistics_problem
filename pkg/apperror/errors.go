// Package apperror provides a structured way to handle application errors
// with specific codes, severity levels, and additional details. It also
// includes utilities for converting to and from gRPC status errors.
package apperror

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorCode represents a specific application error code.
type ErrorCode string

const (
	// Validation
	CodeInvalidGraph     ErrorCode = "INVALID_GRAPH"
	CodeEmptyGraph       ErrorCode = "EMPTY_GRAPH"
	CodeInvalidSource    ErrorCode = "INVALID_SOURCE"
	CodeInvalidSink      ErrorCode = "INVALID_SINK"
	CodeDuplicateNode    ErrorCode = "DUPLICATE_NODE"
	CodeDanglingEdge     ErrorCode = "DANGLING_EDGE"
	CodeSelfLoop         ErrorCode = "SELF_LOOP"
	CodeNegativeCapacity ErrorCode = "NEGATIVE_CAPACITY"
	CodeNegativeCost     ErrorCode = "NEGATIVE_COST"
	CodeSourceEqualsSink ErrorCode = "SOURCE_EQUALS_SINK"
	CodeInvalidCapacity  ErrorCode = "INVALID_CAPACITY"
	CodeNegativeLength   ErrorCode = "NEGATIVE_LENGTH"

	// Connectivity
	CodeNoPath            ErrorCode = "NO_PATH"
	CodeDisconnectedGraph ErrorCode = "DISCONNECTED_GRAPH"
	CodeIsolatedNode      ErrorCode = "ISOLATED_NODE"
	CodeUnreachableNode   ErrorCode = "UNREACHABLE_NODE"

	// Algorithms
	CodeAlgorithmError    ErrorCode = "ALGORITHM_ERROR"
	CodeTimeout           ErrorCode = "TIMEOUT"
	CodeIterationLimit    ErrorCode = "ITERATION_LIMIT"
	CodeNegativeCycle     ErrorCode = "NEGATIVE_CYCLE"
	CodeInfeasible        ErrorCode = "INFEASIBLE"
	CodeInvalidAlgorithm  ErrorCode = "INVALID_ALGORITHM"
	CodeAlgorithmMismatch ErrorCode = "ALGORITHM_MISMATCH"

	// Flow-related
	CodeFlowViolation         ErrorCode = "FLOW_VIOLATION"
	CodeCapacityOverflow      ErrorCode = "CAPACITY_OVERFLOW"
	CodeConservationViolation ErrorCode = "CONSERVATION_VIOLATION"
	CodeNegativeFlow          ErrorCode = "NEGATIVE_FLOW"
	CodeFlowImbalance         ErrorCode = "FLOW_IMBALANCE"

	// Business Logic
	CodeIsolatedWarehouse   ErrorCode = "ISOLATED_WAREHOUSE"
	CodeUnreachableDelivery ErrorCode = "UNREACHABLE_DELIVERY"
	CodeInvalidNodeType     ErrorCode = "INVALID_NODE_TYPE"
	CodeInvalidRoadType     ErrorCode = "INVALID_ROAD_TYPE"
	CodeBottleneckDetected  ErrorCode = "BOTTLENECK_DETECTED"

	// General
	CodeInternal          ErrorCode = "INTERNAL_ERROR"
	CodeNotFound          ErrorCode = "NOT_FOUND"
	CodeInvalidArgument   ErrorCode = "INVALID_ARGUMENT"
	CodeUnauthenticated   ErrorCode = "UNAUTHENTICATED"
	CodePermissionDenied  ErrorCode = "PERMISSION_DENIED"
	CodeNilInput          ErrorCode = "NIL_INPUT"
	CodeInvalidPagination ErrorCode = "INVALID_PAGINATION"
	CodeInvalidThreshold  ErrorCode = "INVALID_THRESHOLD"
	CodeUnimplemented     ErrorCode = "UNIMPLEMENTED"
)

// Severity defines the criticality level of an error.
type Severity int

const (
	// SeverityWarning indicates a non-critical issue that can be ignored or automatically resolved.
	SeverityWarning Severity = iota
	// SeverityError indicates a standard error that requires attention.
	SeverityError
	// SeverityCritical indicates a severe error that might require immediate human intervention.
	SeverityCritical
)

// String returns the string representation of the Severity.
func (s Severity) String() string {
	switch s {
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Error is a custom error type that includes an ErrorCode, message,
// an optional field, additional details, an underlying cause, and a severity level.
type Error struct {
	Code     ErrorCode      // Code is a unique identifier for the type of error.
	Message  string         // Message is a human-readable description of the error.
	Field    string         // Field indicates which input field caused the error, if applicable.
	Details  map[string]any // Details provides additional structured information about the error.
	Cause    error          // Cause is the underlying error that triggered this application error.
	Severity Severity       // Severity indicates the criticality level of the error.
}

// Error implements the error interface, returning a string representation of the error.
func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s] %s (field: %s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error, allowing for error chain introspection.
func (e *Error) Unwrap() error {
	return e.Cause
}

// GRPCStatus converts the application error into a gRPC status.Status.
func (e *Error) GRPCStatus() *status.Status {
	code := e.grpcCode()
	return status.New(code, e.Message)
}

// grpcCode maps an ErrorCode to an appropriate gRPC codes.Code.
func (e *Error) grpcCode() codes.Code {
	switch e.Code {
	case CodeInvalidGraph, CodeEmptyGraph, CodeInvalidSource, CodeInvalidSink,
		CodeDuplicateNode, CodeDanglingEdge, CodeSelfLoop, CodeNegativeCapacity,
		CodeNegativeCost, CodeSourceEqualsSink, CodeInvalidArgument, CodeInvalidCapacity,
		CodeNegativeLength, CodeNilInput, CodeInvalidPagination, CodeInvalidThreshold,
		CodeInvalidAlgorithm:
		return codes.InvalidArgument

	case CodeNoPath, CodeDisconnectedGraph, CodeIsolatedNode, CodeUnreachableNode,
		CodeNegativeCycle, CodeAlgorithmMismatch:
		return codes.FailedPrecondition

	case CodeNotFound:
		return codes.NotFound

	case CodeTimeout, CodeIterationLimit:
		return codes.DeadlineExceeded

	case CodeUnauthenticated:
		return codes.Unauthenticated

	case CodePermissionDenied:
		return codes.PermissionDenied

	case CodeInfeasible:
		return codes.Aborted

	case CodeFlowViolation, CodeCapacityOverflow, CodeConservationViolation,
		CodeNegativeFlow, CodeFlowImbalance:
		return codes.DataLoss

	default:
		return codes.Internal
	}
}

// New creates a new application error with the given code and message.
// The default severity is SeverityError.
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Details:  make(map[string]any),
		Severity: SeverityError,
	}
}

// NewWithField creates a new application error with the given code, message, and field.
// The default severity is SeverityError.
func NewWithField(code ErrorCode, message, field string) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Field:    field,
		Details:  make(map[string]any),
		Severity: SeverityError,
	}
}

// NewWarning creates a new application error with SeverityWarning.
func NewWarning(code ErrorCode, message string) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Details:  make(map[string]any),
		Severity: SeverityWarning,
	}
}

// NewCritical creates a new application error with SeverityCritical.
func NewCritical(code ErrorCode, message string) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Details:  make(map[string]any),
		Severity: SeverityCritical,
	}
}

// Wrap creates a new application error that wraps an existing error,
// providing additional context with a code and message.
// The default severity is SeverityError.
func Wrap(cause error, code ErrorCode, message string) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Cause:    cause,
		Details:  make(map[string]any),
		Severity: SeverityError,
	}
}

// WithDetails adds a key-value pair to the error's details map and returns the modified error.
func (e *Error) WithDetails(key string, value any) *Error {
	e.Details[key] = value
	return e
}

// WithField sets the field associated with the error and returns the modified error.
func (e *Error) WithField(field string) *Error {
	e.Field = field
	return e
}

// WithSeverity sets the severity level of the error and returns the modified error.
func (e *Error) WithSeverity(s Severity) *Error {
	e.Severity = s
	return e
}

// Is checks if the given error is an application error with a matching ErrorCode.
// It uses errors.As to unwrap the error chain.
func Is(err error, code ErrorCode) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// Code extracts the ErrorCode from an error. If the error is not an *Error,
// it returns CodeInternal.
func Code(err error) ErrorCode {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeInternal
}

// ToGRPC converts an application error or any other error into a gRPC error status.
// If the error is an *Error, it uses its GRPCStatus method.
// If it's already a gRPC status error, it's returned as is.
// Otherwise, it's wrapped as an internal gRPC error.
func ToGRPC(err error) error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.GRPCStatus().Err()
	}

	// If it's already a gRPC error
	if _, ok := status.FromError(err); ok {
		return err
	}

	// Wrap as an Internal error
	return status.Error(codes.Internal, err.Error())
}

// FromGRPC converts a gRPC error into an *Error.
// If the input error is nil, it returns nil.
// If the gRPC status code cannot be mapped to a specific ErrorCode,
// it defaults to CodeInternal.
func FromGRPC(err error) *Error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return New(CodeInternal, err.Error())
	}

	var code ErrorCode
	switch st.Code() {
	case codes.InvalidArgument:
		code = CodeInvalidArgument
	case codes.NotFound:
		code = CodeNotFound
	case codes.DeadlineExceeded:
		code = CodeTimeout
	case codes.Unauthenticated:
		code = CodeUnauthenticated
	case codes.PermissionDenied:
		code = CodePermissionDenied
	case codes.FailedPrecondition:
		code = CodeNoPath
	default:
		code = CodeInternal
	}

	return New(code, st.Message())
}

// IsWarning checks if the given error is an application error with SeverityWarning.
func IsWarning(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Severity == SeverityWarning
	}
	return false
}

// IsCritical checks if the given error is an application error with SeverityCritical.
func IsCritical(err error) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Severity == SeverityCritical
	}
	return false
}

// Predefined errors for common scenarios.
var (
	ErrEmptyGraph       = New(CodeEmptyGraph, "graph is empty")
	ErrInvalidSource    = New(CodeInvalidSource, "source node not found")
	ErrInvalidSink      = New(CodeInvalidSink, "sink node not found")
	ErrSourceEqualsSink = New(CodeSourceEqualsSink, "source and sink cannot be the same")
	ErrNoPath           = New(CodeNoPath, "no path from source to sink")
	ErrNegativeCycle    = New(CodeNegativeCycle, "graph contains negative cycle")
	ErrTimeout          = New(CodeTimeout, "operation timed out")
	ErrNilGraph         = New(CodeNilInput, "graph is nil")
	ErrIterationLimit   = New(CodeIterationLimit, "iteration limit exceeded")
)

// ValidationErrors is a collection of application errors and warnings,
// typically used for aggregating results of multiple validation checks.
type ValidationErrors struct {
	Errors   []*Error // Errors contains all collected errors (SeverityError and SeverityCritical).
	Warnings []*Error // Warnings contains all collected warnings (SeverityWarning).
}

// NewValidationErrors creates and returns a new empty ValidationErrors collection.
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors:   make([]*Error, 0),
		Warnings: make([]*Error, 0),
	}
}

// Add appends an *Error to the appropriate slice (Errors or Warnings)
// based on its Severity.
func (v *ValidationErrors) Add(err *Error) {
	if err.Severity == SeverityWarning {
		v.Warnings = append(v.Warnings, err)
	} else {
		v.Errors = append(v.Errors, err)
	}
}

// AddError creates and adds a new application error with SeverityError.
func (v *ValidationErrors) AddError(code ErrorCode, message string) {
	v.Errors = append(v.Errors, New(code, message))
}

// AddWarning creates and adds a new application error with SeverityWarning.
func (v *ValidationErrors) AddWarning(code ErrorCode, message string) {
	v.Warnings = append(v.Warnings, NewWarning(code, message))
}

// AddErrorWithField creates and adds a new application error with a specific field.
func (v *ValidationErrors) AddErrorWithField(code ErrorCode, message, field string) {
	v.Errors = append(v.Errors, NewWithField(code, message, field))
}

// HasErrors returns true if the collection contains any errors (non-warning severity).
func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}

// HasWarnings returns true if the collection contains any warnings.
func (v *ValidationErrors) HasWarnings() bool {
	return len(v.Warnings) > 0
}

// IsValid returns true if the collection contains no errors (warnings do not affect validity).
func (v *ValidationErrors) IsValid() bool {
	return !v.HasErrors()
}

// Merge combines the current ValidationErrors collection with another one.
// All errors and warnings from the 'other' collection are appended to the current one.
func (v *ValidationErrors) Merge(other *ValidationErrors) {
	if other == nil {
		return
	}
	v.Errors = append(v.Errors, other.Errors...)
	v.Warnings = append(v.Warnings, other.Warnings...)
}

// ErrorMessages returns a slice of string messages for all collected errors.
func (v *ValidationErrors) ErrorMessages() []string {
	messages := make([]string, len(v.Errors))
	for i, err := range v.Errors {
		messages[i] = err.Error()
	}
	return messages
}

// WarningMessages returns a slice of string messages for all collected warnings.
func (v *ValidationErrors) WarningMessages() []string {
	messages := make([]string, len(v.Warnings))
	for i, warn := range v.Warnings {
		messages[i] = warn.Message
	}
	return messages
}
