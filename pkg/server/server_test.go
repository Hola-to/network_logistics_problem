package server

import (
	"testing"

	"logistics/pkg/config"
	"logistics/pkg/logger"

	"github.com/stretchr/testify/assert"
)

func init() {
	logger.Init("error")
}

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{Name: "test-app"},
		GRPC: config.GRPCConfig{
			Port:      50051,
			KeepAlive: config.KeepAliveConfig{},
		},
		RateLimit: config.RateLimitConfig{
			Enabled: false,
		},
		Audit: config.AuditConfig{
			Enabled: false,
		},
	}

	srv := New(cfg)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.GetEngine())

	// Audit logger должен быть nil, так как выключен
	assert.Nil(t, srv.GetAuditLogger())
}

func TestNewServer_WithOptions(t *testing.T) {
	cfg := &config.Config{
		App:   config.AppConfig{Name: "test-app"},
		GRPC:  config.GRPCConfig{Port: 50052},
		Audit: config.AuditConfig{Enabled: true}, // Включено в конфиге
	}

	// Но мы передаем nil logger явно через опции (симуляция ошибки создания)
	opts := &ServerOptions{
		AuditLogger: nil,
	}

	srv := NewWithOptions(cfg, opts)
	assert.NotNil(t, srv)
}
