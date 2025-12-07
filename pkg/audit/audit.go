// Package audit provides components for capturing, storing, and querying audit logs.
// It defines the structure of an audit entry, actions, outcomes, and interfaces
// for different logging backends.
package audit

import (
	"context"
	"encoding/json"
	"time"
)

// Action represents the type of action performed in an audit event.
type Action string

const (
	// ActionCreate indicates a resource creation event.
	ActionCreate Action = "CREATE"
	// ActionRead indicates a resource read/retrieval event.
	ActionRead Action = "READ"
	// ActionUpdate indicates a resource modification event.
	ActionUpdate Action = "UPDATE"
	// ActionDelete indicates a resource deletion event.
	ActionDelete Action = "DELETE"
	// ActionLogin indicates a user login event.
	ActionLogin Action = "LOGIN"
	// ActionLogout indicates a user logout event.
	ActionLogout Action = "LOGOUT"
	// ActionSolve indicates an algorithm or problem-solving execution.
	ActionSolve Action = "SOLVE"
	// ActionAnalyze indicates a data analysis or reporting execution.
	ActionAnalyze Action = "ANALYZE"
)

// Outcome represents the result of an audit action.
type Outcome string

const (
	// OutcomeSuccess indicates that the action completed successfully.
	OutcomeSuccess Outcome = "SUCCESS"
	// OutcomeFailure indicates that the action failed due to an error.
	OutcomeFailure Outcome = "FAILURE"
	// OutcomeDenied indicates that the action was denied, typically due to permissions.
	OutcomeDenied Outcome = "DENIED"
)

// Entry represents a single audit log record, capturing details about an event.
type Entry struct {
	ID           string         `json:"id"`                      // Unique identifier for the audit entry.
	Timestamp    time.Time      `json:"timestamp"`               // Time when the event occurred.
	Service      string         `json:"service"`                 // Name of the service that generated the audit event.
	Method       string         `json:"method"`                  // Specific method or endpoint invoked.
	Action       Action         `json:"action"`                  // Type of action performed (e.g., CREATE, READ).
	Outcome      Outcome        `json:"outcome"`                 // Result of the action (e.g., SUCCESS, FAILURE).
	UserID       string         `json:"user_id,omitempty"`       // ID of the user who performed the action.
	Username     string         `json:"username,omitempty"`      // Username of the user who performed the action.
	ClientIP     string         `json:"client_ip,omitempty"`     // IP address of the client.
	UserAgent    string         `json:"user_agent,omitempty"`    // User-agent string of the client.
	Resource     string         `json:"resource,omitempty"`      // Type of resource affected (e.g., "graph", "user").
	ResourceID   string         `json:"resource_id,omitempty"`   // ID of the resource affected.
	RequestID    string         `json:"request_id,omitempty"`    // Unique ID of the client request, if available.
	DurationMs   int64          `json:"duration_ms"`             // Duration of the operation in milliseconds.
	ErrorCode    string         `json:"error_code,omitempty"`    // Application-specific error code if the outcome is FAILURE.
	ErrorMessage string         `json:"error_message,omitempty"` // Human-readable error message if the outcome is FAILURE.
	Metadata     map[string]any `json:"metadata,omitempty"`      // Additional arbitrary key-value metadata.
	Changes      *ChangeSet     `json:"changes,omitempty"`       // Details about changes made to a resource.
}

// ChangeSet describes changes made to a resource, useful for update actions.
type ChangeSet struct {
	Before map[string]any `json:"before,omitempty"` // State of the resource before the change.
	After  map[string]any `json:"after,omitempty"`  // State of the resource after the change.
	Fields []string       `json:"fields,omitempty"` // List of fields that were changed.
}

// Logger is the interface that audit loggers must implement.
type Logger interface {
	// Log records an audit event.
	Log(ctx context.Context, entry *Entry) error

	// Query retrieves audit logs based on a filter.
	// Not all loggers may support querying.
	Query(ctx context.Context, filter *QueryFilter) ([]*Entry, error)

	// Close shuts down the logger and releases any resources.
	Close() error
}

// QueryFilter defines criteria for querying audit log entries.
type QueryFilter struct {
	StartTime  *time.Time // Start time for the query range (inclusive).
	EndTime    *time.Time // End time for the query range (exclusive).
	Service    string     // Filter by service name.
	Method     string     // Filter by method or endpoint.
	Action     Action     // Filter by action type.
	Outcome    Outcome    // Filter by action outcome.
	UserID     string     // Filter by user ID.
	Resource   string     // Filter by resource type.
	ResourceID string     // Filter by resource ID.
	Limit      int        // Maximum number of results to return.
	Offset     int        // Number of results to skip.
}

// Config holds configuration parameters for the audit logger.
type Config struct {
	Enabled     bool          `koanf:"enabled"`      // If true, auditing is active.
	Backend     string        `koanf:"backend"`      // The logging backend to use (e.g., "database", "file", "stdout").
	FilePath    string        `koanf:"file_path"`    // Path to the log file, if backend is "file".
	MaxSize     int           `koanf:"max_size"`     // Maximum size of the log file in MB before rotation.
	MaxAge      int           `koanf:"max_age"`      // Maximum age of log files in days before deletion.
	Compress    bool          `koanf:"compress"`     // Whether to compress old log files.
	BufferSize  int           `koanf:"buffer_size"`  // Size of the internal buffer for asynchronous logging.
	FlushPeriod time.Duration `koanf:"flush_period"` // Period to flush buffered entries to the backend.

	// Filtering and data masking settings.
	ExcludeMethods  []string `koanf:"exclude_methods"`  // List of methods to exclude from auditing.
	IncludeRequest  bool     `koanf:"include_request"`  // Whether to include request payload in metadata.
	IncludeResponse bool     `koanf:"include_response"` // Whether to include response payload in metadata.
	MaskFields      []string `koanf:"mask_fields"`      // List of fields whose values should be masked in logs.
}

// DefaultConfig returns a Config struct with default values.
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		Backend:        "stdout",
		BufferSize:     1000,
		FlushPeriod:    5 * time.Second,
		IncludeRequest: false,
		MaskFields:     []string{"password", "token", "secret", "api_key"},
	}
}

// Builder provides a fluent API for constructing an Entry object.
type Builder struct {
	entry *Entry
}

// NewEntry creates and returns a new Builder initialized with a timestamp and an empty metadata map.
func NewEntry() *Builder {
	return &Builder{
		entry: &Entry{
			Timestamp: time.Now(),
			Metadata:  make(map[string]any),
		},
	}
}

// Service sets the service name for the audit entry.
func (b *Builder) Service(s string) *Builder {
	b.entry.Service = s
	return b
}

// Method sets the method or endpoint for the audit entry.
func (b *Builder) Method(m string) *Builder {
	b.entry.Method = m
	return b
}

// Action sets the action type for the audit entry.
func (b *Builder) Action(a Action) *Builder {
	b.entry.Action = a
	return b
}

// Outcome sets the outcome for the audit entry.
func (b *Builder) Outcome(o Outcome) *Builder {
	b.entry.Outcome = o
	return b
}

// User sets the user ID and username for the audit entry.
func (b *Builder) User(id, username string) *Builder {
	b.entry.UserID = id
	b.entry.Username = username
	return b
}

// Client sets the client IP and user agent for the audit entry.
func (b *Builder) Client(ip, userAgent string) *Builder {
	b.entry.ClientIP = ip
	b.entry.UserAgent = userAgent
	return b
}

// Resource sets the resource type and ID for the audit entry.
func (b *Builder) Resource(resource, resourceID string) *Builder {
	b.entry.Resource = resource
	b.entry.ResourceID = resourceID
	return b
}

// RequestID sets the request ID for the audit entry.
func (b *Builder) RequestID(id string) *Builder {
	b.entry.RequestID = id
	return b
}

// Duration sets the duration of the operation in milliseconds for the audit entry.
func (b *Builder) Duration(d time.Duration) *Builder {
	b.entry.DurationMs = d.Milliseconds()
	return b
}

// Error sets the error code and message if the outcome was a failure.
func (b *Builder) Error(code, message string) *Builder {
	b.entry.ErrorCode = code
	b.entry.ErrorMessage = message
	return b
}

// Meta adds a key-value pair to the metadata map of the audit entry.
func (b *Builder) Meta(key string, value any) *Builder {
	b.entry.Metadata[key] = value
	return b
}

// Changes sets the ChangeSet for the audit entry, detailing resource modifications.
func (b *Builder) Changes(changes *ChangeSet) *Builder {
	b.entry.Changes = changes
	return b
}

// Build finalizes the Entry construction and returns the Entry object.
// It generates a unique ID if one is not already set.
func (b *Builder) Build() *Entry {
	if b.entry.ID == "" {
		b.entry.ID = generateID()
	}
	return b.entry
}

// MarshalJSON customizes the JSON serialization of an Entry.
func (e *Entry) MarshalJSON() ([]byte, error) {
	type Alias Entry
	return json.Marshal((*Alias)(e))
}

// generateID creates a unique ID for an audit entry, combining a timestamp and a random string.
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random alphanumeric string of a given length.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
