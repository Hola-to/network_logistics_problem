// Package audit provides components for capturing, storing, and querying audit logs.
// Package audit provides tests for the gRPC client functionality.
package audit

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	auditv1 "logistics/gen/go/logistics/audit/v1"
	"logistics/pkg/client"
	"logistics/pkg/logger"
)

// GRPCClient implements the audit.Logger interface by sending audit events
// to an external audit service via gRPC. It buffers events and sends them
// in batches for efficiency.
type GRPCClient struct {
	conn   *grpc.ClientConn
	client auditv1.AuditServiceClient
	config *GRPCClientConfig
	buffer chan *Entry
	done   chan struct{}
	wg     sync.WaitGroup
}

// GRPCClientConfig holds configuration parameters for the GRPCClient.
type GRPCClientConfig struct {
	Address      string        // Address of the audit gRPC service (e.g., "localhost:50057").
	Timeout      time.Duration // Timeout for gRPC calls.
	BufferSize   int           // Size of the internal buffer for audit entries.
	BatchSize    int           // Maximum number of entries to send in a single batch.
	FlushPeriod  time.Duration // Period after which buffered entries are flushed.
	MaxRetries   int           // Maximum number of retries for connection or RPCs.
	RetryBackoff time.Duration // Time to wait between retries.
}

// DefaultGRPCClientConfig returns a GRPCClientConfig struct with default values.
func DefaultGRPCClientConfig() *GRPCClientConfig {
	return &GRPCClientConfig{
		Address:      "localhost:50057",
		Timeout:      5 * time.Second,
		BufferSize:   10000,
		BatchSize:    100,
		FlushPeriod:  5 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 100 * time.Millisecond,
	}
}

// NewGRPCClient creates and initializes a new GRPCClient.
// It establishes a gRPC connection to the audit service and starts a background
// process for buffering and sending audit events.
func NewGRPCClient(ctx context.Context, cfg *GRPCClientConfig) (*GRPCClient, error) {
	if cfg == nil {
		cfg = DefaultGRPCClientConfig()
	}

	conn, err := client.NewGRPCClient(ctx, client.ClientConfig{
		Address:      cfg.Address,
		Timeout:      cfg.Timeout,
		MaxRetries:   cfg.MaxRetries,
		RetryBackoff: cfg.RetryBackoff,
	})
	if err != nil {
		return nil, err
	}

	c := &GRPCClient{
		conn:   conn,
		client: auditv1.NewAuditServiceClient(conn),
		config: cfg,
		buffer: make(chan *Entry, cfg.BufferSize),
		done:   make(chan struct{}),
	}

	c.wg.Add(1)
	go c.processLoop()

	return c, nil
}

// Log sends an audit entry to the gRPC client's buffer. If the buffer is full,
// it attempts to send the entry synchronously.
func (c *GRPCClient) Log(ctx context.Context, entry *Entry) error {
	select {
	case c.buffer <- entry:
		return nil
	default:
		// Buffer is full, attempt to send synchronously
		return c.sendSingle(ctx, entry)
	}
}

// Query is not supported by the GRPCClient and will return a nil slice and nil error.
// The gRPC service might provide its own query functionality.
func (c *GRPCClient) Query(ctx context.Context, filter *QueryFilter) ([]*Entry, error) {
	// For query, use gRPC directly if implemented by the audit service
	return nil, nil
}

// Close shuts down the GRPCClient, stopping the background processing loop,
// flushing any remaining buffered events, and closing the gRPC connection.
func (c *GRPCClient) Close() error {
	close(c.done)
	c.wg.Wait() // Wait for processLoop to finish
	return c.conn.Close()
}

// processLoop is a goroutine that continuously reads from the buffer,
// aggregates entries into batches, and periodically flushes them to the
// audit service via gRPC.
func (c *GRPCClient) processLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.FlushPeriod)
	defer ticker.Stop()

	batch := make([]*Entry, 0, c.config.BatchSize)

	for {
		select {
		case <-c.done:
			// Drain and send any remaining entries before exiting
			if len(batch) > 0 {
				c.sendBatch(context.Background(), batch)
			}
			return

		case entry := <-c.buffer:
			batch = append(batch, entry)
			if len(batch) >= c.config.BatchSize {
				c.sendBatch(context.Background(), batch)
				batch = make([]*Entry, 0, c.config.BatchSize) // Reset batch
			}

		case <-ticker.C:
			if len(batch) > 0 {
				c.sendBatch(context.Background(), batch)
				batch = make([]*Entry, 0, c.config.BatchSize) // Reset batch
			}
		}
	}
}

// sendSingle sends a single audit entry to the audit service via gRPC.
// It uses a timeout specified in the client configuration.
func (c *GRPCClient) sendSingle(ctx context.Context, entry *Entry) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	_, err := c.client.LogEvent(ctx, &auditv1.LogEventRequest{
		Entry: c.entryToProto(entry),
	})
	if err != nil {
		logger.Log.Warn("Failed to send audit event", "error", err)
	}
	return err
}

// sendBatch sends a batch of audit entries to the audit service via gRPC.
// It converts the internal Entry slice to a protobuf message slice.
func (c *GRPCClient) sendBatch(ctx context.Context, entries []*Entry) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	protoEntries := make([]*auditv1.AuditEntry, 0, len(entries))
	for _, e := range entries {
		protoEntries = append(protoEntries, c.entryToProto(e))
	}

	resp, err := c.client.LogEventBatch(ctx, &auditv1.LogEventBatchRequest{
		Entries: protoEntries,
	})
	if err != nil {
		logger.Log.Warn("Failed to send audit batch", "error", err, "count", len(entries))
		return
	}

	if resp.FailedCount > 0 {
		logger.Log.Warn("Some audit events failed",
			"logged", resp.LoggedCount,
			"failed", resp.FailedCount,
		)
	}
}

// entryToProto converts an internal audit.Entry object to its protobuf equivalent
// auditv1.AuditEntry, including mapping Action and Outcome enums.
func (c *GRPCClient) entryToProto(e *Entry) *auditv1.AuditEntry {
	entry := &auditv1.AuditEntry{
		Id:           e.ID,
		Timestamp:    timestamppb.New(e.Timestamp),
		Service:      e.Service,
		Method:       e.Method,
		RequestId:    e.RequestID,
		UserId:       e.UserID,
		Username:     e.Username,
		ClientIp:     e.ClientIP,
		UserAgent:    e.UserAgent,
		ResourceType: e.Resource,
		ResourceId:   e.ResourceID,
		DurationMs:   e.DurationMs,
		ErrorCode:    e.ErrorCode,
		ErrorMessage: e.ErrorMessage,
	}

	// Map Action
	switch e.Action {
	case ActionCreate:
		entry.Action = auditv1.AuditAction_AUDIT_ACTION_CREATE
	case ActionRead:
		entry.Action = auditv1.AuditAction_AUDIT_ACTION_READ
	case ActionUpdate:
		entry.Action = auditv1.AuditAction_AUDIT_ACTION_UPDATE
	case ActionDelete:
		entry.Action = auditv1.AuditAction_AUDIT_ACTION_DELETE
	case ActionLogin:
		entry.Action = auditv1.AuditAction_AUDIT_ACTION_LOGIN
	case ActionLogout:
		entry.Action = auditv1.AuditAction_AUDIT_ACTION_LOGOUT
	case ActionSolve:
		entry.Action = auditv1.AuditAction_AUDIT_ACTION_SOLVE
	case ActionAnalyze:
		entry.Action = auditv1.AuditAction_AUDIT_ACTION_ANALYZE
	}

	// Map Outcome
	switch e.Outcome {
	case OutcomeSuccess:
		entry.Outcome = auditv1.AuditOutcome_AUDIT_OUTCOME_SUCCESS
	case OutcomeFailure:
		entry.Outcome = auditv1.AuditOutcome_AUDIT_OUTCOME_FAILURE
	case OutcomeDenied:
		entry.Outcome = auditv1.AuditOutcome_AUDIT_OUTCOME_DENIED
	}

	// Metadata
	if len(e.Metadata) > 0 {
		entry.Metadata = make(map[string]string)
		for k, v := range e.Metadata {
			if s, ok := v.(string); ok {
				entry.Metadata[k] = s
			}
		}
	}

	return entry
}
