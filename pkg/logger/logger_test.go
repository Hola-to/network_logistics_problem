package logger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}
	for _, level := range levels {
		Init(level)
		if Log == nil {
			t.Errorf("Init(%s) should set Log", level)
		}
	}
}

func TestInitWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "json format stdout",
			config: Config{
				Level:  "info",
				Format: "json",
				Output: "stdout",
			},
		},
		{
			name: "text format stderr",
			config: Config{
				Level:  "debug",
				Format: "text",
				Output: "stderr",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitWithConfig(tt.config)
			if Log == nil {
				t.Error("Log should not be nil")
			}
		})
	}
}

func TestInitWithConfig_FileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	InitWithConfig(Config{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logPath,
	})

	if Log == nil {
		t.Fatal("Log should not be nil")
	}

	// Write a log entry
	Log.Info("test message")
}

func TestInitWithConfig_FileOutputInvalidDir(t *testing.T) {
	// Test with invalid directory - should fall back to stdout
	InitWithConfig(Config{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: "/nonexistent/deeply/nested/dir/test.log",
	})

	if Log == nil {
		t.Error("Log should not be nil even with invalid path")
	}
}

func TestLoggingFunctions(t *testing.T) {
	Init("debug")

	// These should not panic
	Debug("debug message", "key", "value")
	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Error("error message", "key", "value")
}

func TestWithContext(t *testing.T) {
	Init("info")

	logger := WithContext(context.Background(), "key1", "value1")
	if logger == nil {
		t.Error("WithContext should return logger")
	}
}

func TestWithRequestID(t *testing.T) {
	Init("info")

	logger := WithRequestID("req-123")
	if logger == nil {
		t.Error("WithRequestID should return logger")
	}
}

func TestWithService(t *testing.T) {
	Init("info")

	logger := WithService("test-service")
	if logger == nil {
		t.Error("WithService should return logger")
	}
}

func TestFatal(t *testing.T) {
	if os.Getenv("TEST_FATAL") == "1" {
		Init("info")
		Fatal("fatal message")
		return
	}

	// We can't actually test Fatal without subprocess
	// as it calls os.Exit
}
