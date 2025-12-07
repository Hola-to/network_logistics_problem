package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_LoadDefaults(t *testing.T) {
	cfg, err := NewLoader().Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Check defaults
	if cfg.App.Name != "logistics-service" {
		t.Errorf("expected app name 'logistics-service', got %s", cfg.App.Name)
	}
	if cfg.GRPC.Port != 50051 {
		t.Errorf("expected gRPC port 50051, got %d", cfg.GRPC.Port)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected log level 'info', got %s", cfg.Log.Level)
	}
	if cfg.Metrics.Port != 9090 {
		t.Errorf("expected metrics port 9090, got %d", cfg.Metrics.Port)
	}
}

func TestLoader_LoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
app:
  name: custom-service
  version: 2.0.0
  environment: staging
grpc:
  port: 50052
log:
  level: debug
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loader := NewLoader(WithConfigPaths(configPath))
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.App.Name != "custom-service" {
		t.Errorf("expected app name 'custom-service', got %s", cfg.App.Name)
	}
	if cfg.App.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %s", cfg.App.Version)
	}
	if cfg.GRPC.Port != 50052 {
		t.Errorf("expected port 50052, got %d", cfg.GRPC.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level 'debug', got %s", cfg.Log.Level)
	}
}

func TestLoader_LoadFromEnv(t *testing.T) {
	// Set env vars
	os.Setenv("LOGISTICS_APP_NAME", "env-service")
	os.Setenv("LOGISTICS_GRPC_PORT", "50053")
	defer func() {
		os.Unsetenv("LOGISTICS_APP_NAME")
		os.Unsetenv("LOGISTICS_GRPC_PORT")
	}()

	cfg, err := NewLoader().Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.App.Name != "env-service" {
		t.Errorf("expected app name 'env-service', got %s", cfg.App.Name)
	}
	if cfg.GRPC.Port != 50053 {
		t.Errorf("expected port 50053, got %d", cfg.GRPC.Port)
	}
}

func TestLoader_EnvOverridesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
app:
  name: file-service
grpc:
  port: 50054
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	// Env should override file
	os.Setenv("LOGISTICS_APP_NAME", "env-override")
	defer os.Unsetenv("LOGISTICS_APP_NAME")

	cfg, err := NewLoader(WithConfigPaths(configPath)).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.App.Name != "env-override" {
		t.Errorf("expected env override, got %s", cfg.App.Name)
	}
	// Port should come from file
	if cfg.GRPC.Port != 50054 {
		t.Errorf("expected port from file 50054, got %d", cfg.GRPC.Port)
	}
}

func TestLoader_WithEnvPrefix(t *testing.T) {
	os.Setenv("CUSTOM_APP_NAME", "custom-prefix-service")
	defer os.Unsetenv("CUSTOM_APP_NAME")

	cfg, err := NewLoader(WithEnvPrefix("CUSTOM_")).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.App.Name != "custom-prefix-service" {
		t.Errorf("expected 'custom-prefix-service', got %s", cfg.App.Name)
	}
}

func TestMustLoad_Success(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustLoad should not panic with valid config")
		}
	}()

	cfg := MustLoad()
	if cfg == nil {
		t.Error("expected non-nil config")
	}
}

func TestLoad_Simple(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg == nil {
		t.Error("expected non-nil config")
	}
}

func TestLoadWithServiceDefaults(t *testing.T) {
	cfg, err := LoadWithServiceDefaults("test-svc", 60000)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Should use service defaults since no explicit config
	if cfg.App.Name != "test-svc" {
		t.Errorf("expected app name 'test-svc', got %s", cfg.App.Name)
	}
	if cfg.GRPC.Port != 60000 {
		t.Errorf("expected port 60000, got %d", cfg.GRPC.Port)
	}
}

func TestLoader_ConfigEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom-config.yaml")

	configContent := `
app:
  name: config-env-var-service
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	os.Setenv("CONFIG_PATH", configPath)
	defer os.Unsetenv("CONFIG_PATH")

	cfg, err := NewLoader().Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.App.Name != "config-env-var-service" {
		t.Errorf("expected 'config-env-var-service', got %s", cfg.App.Name)
	}
}
