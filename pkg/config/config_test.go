package config

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				App:  AppConfig{Name: "test-service"},
				GRPC: GRPCConfig{Port: 50051},
				Log:  LogConfig{Level: "info"},
			},
			wantErr: false,
		},
		{
			name: "missing app name",
			cfg: Config{
				GRPC: GRPCConfig{Port: 50051},
				Log:  LogConfig{Level: "info"},
			},
			wantErr: true,
		},
		{
			name: "invalid port - zero",
			cfg: Config{
				App:  AppConfig{Name: "test"},
				GRPC: GRPCConfig{Port: 0},
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: Config{
				App:  AppConfig{Name: "test"},
				GRPC: GRPCConfig{Port: 70000},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: Config{
				App:  AppConfig{Name: "test"},
				GRPC: GRPCConfig{Port: 50051},
				Log:  LogConfig{Level: "invalid"},
			},
			wantErr: true,
		},
		{
			name: "valid debug level",
			cfg: Config{
				App:  AppConfig{Name: "test"},
				GRPC: GRPCConfig{Port: 50051},
				Log:  LogConfig{Level: "debug"},
			},
			wantErr: false,
		},
		{
			name: "invalid report theme",
			cfg: Config{
				App:    AppConfig{Name: "test"},
				GRPC:   GRPCConfig{Port: 50051},
				Log:    LogConfig{Level: "info"},
				Report: ReportConfig{DefaultTheme: "invalid-theme"},
			},
			wantErr: true,
		},
		{
			name: "valid report config",
			cfg: Config{
				App:  AppConfig{Name: "test"},
				GRPC: GRPCConfig{Port: 50051},
				Log:  LogConfig{Level: "info"},
				Report: ReportConfig{
					DefaultTheme: "dark",
					PDF:          PDFConfig{PageSize: "A4", Orientation: "landscape"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"development", true},
		{"dev", true},
		{"production", false},
		{"staging", false},
	}

	for _, tt := range tests {
		cfg := &Config{App: AppConfig{Environment: tt.env}}
		if got := cfg.IsDevelopment(); got != tt.want {
			t.Errorf("IsDevelopment() for %s = %v, want %v", tt.env, got, tt.want)
		}
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"prod", true},
		{"development", false},
		{"staging", false},
	}

	for _, tt := range tests {
		cfg := &Config{App: AppConfig{Environment: tt.env}}
		if got := cfg.IsProduction(); got != tt.want {
			t.Errorf("IsProduction() for %s = %v, want %v", tt.env, got, tt.want)
		}
	}
}

func TestServiceEndpoint_Address(t *testing.T) {
	endpoint := ServiceEndpoint{
		Host: "localhost",
		Port: 50051,
	}

	addr := endpoint.Address()
	if addr != "localhost:50051" {
		t.Errorf("expected 'localhost:50051', got %s", addr)
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name   string
		cfg    DatabaseConfig
		expect string
	}{
		{
			name: "postgres",
			cfg: DatabaseConfig{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				Username: "user",
				Password: "pass",
				SSLMode:  "disable",
			},
			expect: "host=localhost port=5432 user=user password=pass dbname=testdb sslmode=disable",
		},
		{
			name: "mysql",
			cfg: DatabaseConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				Username: "user",
				Password: "pass",
			},
			expect: "user:pass@tcp(localhost:3306)/testdb?parseTime=true",
		},
		{
			name: "sqlite",
			cfg: DatabaseConfig{
				Driver:   "sqlite",
				Database: "/path/to/db.sqlite",
			},
			expect: "/path/to/db.sqlite",
		},
		{
			name: "unknown",
			cfg: DatabaseConfig{
				Driver: "unknown",
			},
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.cfg.DSN()
			if dsn != tt.expect {
				t.Errorf("expected DSN %s, got %s", tt.expect, dsn)
			}
		})
	}
}

func TestCacheConfig_Address(t *testing.T) {
	cfg := CacheConfig{
		Host: "redis.local",
		Port: 6379,
	}

	addr := cfg.Address()
	if addr != "redis.local:6379" {
		t.Errorf("expected 'redis.local:6379', got %s", addr)
	}
}

func TestKeepAliveConfig(t *testing.T) {
	cfg := KeepAliveConfig{
		MaxConnectionIdle:     15 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  5 * time.Minute,
		Timeout:               20 * time.Second,
	}

	if cfg.MaxConnectionIdle != 15*time.Minute {
		t.Errorf("unexpected MaxConnectionIdle: %v", cfg.MaxConnectionIdle)
	}
}

func TestCORSConfig(t *testing.T) {
	cfg := CORSConfig{
		Enabled:          true,
		AllowedOrigins:   []string{"http://localhost:3000", "https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}

	if !cfg.Enabled {
		t.Error("expected CORS to be enabled")
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("expected 2 origins, got %d", len(cfg.AllowedOrigins))
	}
}

func TestPDFConfig_Defaults(t *testing.T) {
	cfg := PDFConfig{
		PageSize:          "A4",
		Orientation:       "portrait",
		MarginTop:         15.0,
		MarginBottom:      15.0,
		MarginLeft:        15.0,
		MarginRight:       15.0,
		FontFamily:        "Arial",
		FontSize:          10.0,
		HeaderFontSize:    14.0,
		EnablePageNumbers: true,
	}

	if cfg.PageSize != "A4" {
		t.Errorf("expected page size A4, got %s", cfg.PageSize)
	}
	if cfg.MarginTop != 15.0 {
		t.Errorf("expected margin 15.0, got %f", cfg.MarginTop)
	}
}
