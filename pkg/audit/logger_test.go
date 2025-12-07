// Package audit provides tests for various audit logger implementations.
package audit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"logistics/pkg/logger"
)

// init sets up the global logger for testing purposes, suppressing informational logs.
func init() {
	logger.Init("error")
}

// TestStdoutLogger verifies that StdoutLogger correctly logs entries to standard output.
func TestStdoutLogger(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Backend: "stdout",
	}

	logger := NewStdoutLogger(cfg)
	defer logger.Close()

	entry := NewEntry().
		Service("test").
		Method("/test").
		Action(ActionRead).
		Outcome(OutcomeSuccess).
		Build()

	err := logger.Log(context.Background(), entry)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestStdoutLogger_Disabled ensures that StdoutLogger does not log when disabled.
func TestStdoutLogger_Disabled(t *testing.T) {
	cfg := &Config{
		Enabled: false,
	}

	logger := NewStdoutLogger(cfg)
	defer logger.Close()

	entry := NewEntry().Build()
	err := logger.Log(context.Background(), entry)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestStdoutLogger_Query verifies that Query operations are not supported by StdoutLogger.
func TestStdoutLogger_Query(t *testing.T) {
	logger := NewStdoutLogger(&Config{Enabled: true})
	defer logger.Close()

	_, err := logger.Query(context.Background(), &QueryFilter{})
	if err == nil {
		t.Error("expected error for query on stdout logger")
	}
}

// TestFileLogger verifies that FileLogger correctly writes audit entries to a file.
func TestFileLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	cfg := &Config{
		Enabled:     true,
		Backend:     "file",
		FilePath:    logPath,
		BufferSize:  100,
		FlushPeriod: 100 * time.Millisecond,
	}

	logger, err := NewFileLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}

	entry := NewEntry().
		Service("test").
		Method("/test").
		Action(ActionCreate).
		Outcome(OutcomeSuccess).
		Build()

	err = logger.Log(context.Background(), entry)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	err = logger.Close()
	if err != nil {
		t.Errorf("failed to close logger: %v", err)
	}

	// Check file exists and has content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected log file to have content")
	}

	if !bytes.Contains(data, []byte("test")) {
		t.Error("expected log file to contain 'test'")
	}
}

// TestFileLogger_DefaultPath verifies that FileLogger uses a default path when none is provided.
func TestFileLogger_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg := &Config{
		Enabled:  true,
		Backend:  "file",
		FilePath: "", // Should use default
	}

	logger, err := NewFileLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}
	defer logger.Close()
}

// TestFileLogger_Query verifies that Query operations are not supported by FileLogger.
func TestFileLogger_Query(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Enabled:  true,
		FilePath: filepath.Join(tmpDir, "audit.log"),
	}

	logger, err := NewFileLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create file logger: %v", err)
	}
	defer logger.Close()

	_, err = logger.Query(context.Background(), &QueryFilter{})
	if err == nil {
		t.Error("expected error for query on file logger")
	}
}

// TestNew verifies that the New function correctly instantiates different logger backends.
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: false,
		},
		{
			name: "disabled",
			cfg: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "stdout backend",
			cfg: &Config{
				Enabled: true,
				Backend: "stdout",
			},
			wantErr: false,
		},
		{
			name: "unknown backend defaults to stdout",
			cfg: &Config{
				Enabled: true,
				Backend: "unknown",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if logger == nil {
				t.Error("expected logger to be non-nil")
			}
			logger.Close()
		})
	}
}

// TestNoopLogger verifies that NoopLogger correctly implements the Logger interface
// without performing any actual logging operations.
func TestNoopLogger(t *testing.T) {
	logger := &NoopLogger{}

	err := logger.Log(context.Background(), &Entry{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	entries, err := logger.Query(context.Background(), &QueryFilter{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Error("expected nil entries")
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestGlobalLogger verifies the functionality of setting and getting the global logger instance.
func TestGlobalLogger(t *testing.T) {
	// Save original global logger to restore it later
	original := Get()

	// Set a new NoopLogger as the global logger
	newLogger := &NoopLogger{}
	SetGlobal(newLogger)

	// Verify the global logger has been updated
	if Get() != newLogger {
		t.Error("expected global logger to be updated")
	}

	// Test the package-level Log function, which uses the global logger
	entry := NewEntry().Build()
	err := Log(context.Background(), entry)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Restore original
	SetGlobal(original)
}
