// Package audit provides components for capturing, storing, and querying audit logs.
// This file implements various logger backends such as stdout and file, and a no-operation logger.
package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"logistics/pkg/logger"
)

// StdoutLogger implements the Logger interface by writing audit entries to standard output.
type StdoutLogger struct {
	config *Config
	mu     sync.Mutex // Mutex to ensure thread-safe writes to stdout.
}

// NewStdoutLogger creates and returns a new StdoutLogger.
func NewStdoutLogger(cfg *Config) *StdoutLogger {
	return &StdoutLogger{config: cfg}
}

// Log marshals an audit entry to JSON and prints it to stdout.
// If auditing is disabled in the config, it does nothing.
func (l *StdoutLogger) Log(_ context.Context, entry *Entry) error {
	if !l.config.Enabled {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	fmt.Println("[AUDIT]", string(data))
	return nil
}

// Query is not supported by StdoutLogger and will always return an error.
func (l *StdoutLogger) Query(_ context.Context, _ *QueryFilter) ([]*Entry, error) {
	return nil, fmt.Errorf("query not supported for stdout logger")
}

// Close for StdoutLogger does nothing as there are no resources to release.
func (l *StdoutLogger) Close() error {
	return nil
}

// FileLogger implements the Logger interface by writing audit entries to a specified file.
// It uses a buffered channel for asynchronous writing and periodic flushing.
type FileLogger struct {
	config *Config
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex    // Mutex to protect file writes and internal state.
	buffer chan *Entry   // Buffered channel for asynchronous entry logging.
	done   chan struct{} // Channel to signal shutdown of the processLoop.
}

// NewFileLogger creates and returns a new FileLogger.
// It opens the specified file (or a default 'audit.log' if not provided)
// and starts a background goroutine for processing buffered entries.
func NewFileLogger(cfg *Config) (*FileLogger, error) {
	if cfg.FilePath == "" {
		cfg.FilePath = "audit.log"
	}

	// Open file with create, append, and write-only permissions.
	file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	bufferSize := cfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 1000 // Default buffer size if not specified or invalid.
	}

	l := &FileLogger{
		config: cfg,
		file:   file,
		writer: bufio.NewWriter(file),
		buffer: make(chan *Entry, bufferSize),
		done:   make(chan struct{}),
	}

	go l.processLoop()

	return l, nil
}

// Log sends an audit entry to the internal buffer for asynchronous writing.
// If the buffer is full, it attempts to write the entry directly (synchronously).
func (l *FileLogger) Log(_ context.Context, entry *Entry) error {
	if !l.config.Enabled {
		return nil
	}

	select {
	case l.buffer <- entry:
		return nil
	default:
		// Buffer is full, write directly (synchronously)
		return l.writeEntry(entry)
	}
}

// Query is not implemented for FileLogger and will always return an error.
func (l *FileLogger) Query(_ context.Context, _ *QueryFilter) ([]*Entry, error) {
	return nil, fmt.Errorf("query not implemented for file logger")
}

// Close shuts down the FileLogger. It signals the processLoop to stop,
// drains any remaining entries from the buffer, flushes them to the file,
// and then closes the underlying file handle.
func (l *FileLogger) Close() error {
	close(l.done) // Signal the processLoop to exit.

	l.mu.Lock()
	defer l.mu.Unlock()

	// Drain and flush remaining buffered entries during shutdown.
	for {
		select {
		case entry := <-l.buffer:
			if err := l.writeEntryUnsafe(entry); err != nil {
				logger.Log.Warn("Failed to write audit entry during shutdown", "error", err)
			}
		default:
			goto flush
		}
	}

flush:
	if err := l.writer.Flush(); err != nil {
		logger.Log.Warn("Failed to flush audit writer", "error", err)
	}
	return l.file.Close()
}

// processLoop is a goroutine that continuously reads audit entries from the buffer
// and writes them to the file, or flushes the writer periodically.
func (l *FileLogger) processLoop() {
	flushPeriod := l.config.FlushPeriod
	if flushPeriod <= 0 {
		flushPeriod = 5 * time.Second // Default flush period.
	}

	ticker := time.NewTicker(flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-l.done: // Exit when shutdown is signaled.
			return
		case entry := <-l.buffer: // Write buffered entry.
			if err := l.writeEntry(entry); err != nil {
				logger.Log.Warn("Failed to write audit entry", "error", err)
			}
		case <-ticker.C: // Flush periodically.
			l.flush()
		}
	}
}

// writeEntry marshals an entry to JSON and writes it to the file, protected by a mutex.
func (l *FileLogger) writeEntry(entry *Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.writeEntryUnsafe(entry)
}

// writeEntryUnsafe marshals an entry to JSON and writes it to the file.
// This function is not thread-safe and assumes the caller holds the mutex.
func (l *FileLogger) writeEntryUnsafe(entry *Entry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = l.writer.Write(append(data, '\n'))
	return err
}

// flush flushes the buffered writer to the underlying file, protected by a mutex.
func (l *FileLogger) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.writer.Flush(); err != nil {
		logger.Log.Warn("Failed to flush audit writer", "error", err)
	}
}

// New creates and returns an appropriate Logger implementation based on the provided configuration.
// If `cfg` is nil, it uses DefaultConfig. If auditing is disabled, it returns a NoopLogger.
// It defaults to StdoutLogger if an unknown backend is specified.
func New(cfg *Config) (Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if !cfg.Enabled {
		return &NoopLogger{}, nil
	}

	switch cfg.Backend {
	case "file":
		return NewFileLogger(cfg)
	case "stdout", "": // Default backend is stdout.
		return NewStdoutLogger(cfg), nil
	default:
		logger.Log.Warn("Unknown audit backend, using stdout", "backend", cfg.Backend)
		return NewStdoutLogger(cfg), nil
	}
}

// NoopLogger is a no-operation implementation of the Logger interface.
// It performs no action and always returns nil for Log and Close, and nil for Query results.
type NoopLogger struct{}

// Log for NoopLogger does nothing.
func (l *NoopLogger) Log(_ context.Context, _ *Entry) error { return nil }

// Query for NoopLogger does nothing and returns nil.
func (l *NoopLogger) Query(_ context.Context, _ *QueryFilter) ([]*Entry, error) {
	return nil, nil
}

// Close for NoopLogger does nothing.
func (l *NoopLogger) Close() error { return nil }

// globalLogger is the package-level default audit logger, initialized as a NoopLogger.
// globalLogger is the package-level default audit logger, initialized as a NoopLogger.
var globalLogger Logger = &NoopLogger{}

// globalMu protects access to globalLogger.
var globalMu sync.RWMutex

// SetGlobal sets the global audit logger instance.
func SetGlobal(l Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = l
}

// Get returns the current global audit logger instance.
func Get() Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// Log records an audit entry using the global audit logger.
func Log(ctx context.Context, entry *Entry) error {
	return Get().Log(ctx, entry)
}
