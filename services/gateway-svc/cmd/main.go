package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"logistics/gen/go/logistics/gateway/v1/gatewayv1connect"
	"logistics/pkg/config"
	"logistics/pkg/logger"
	"logistics/pkg/metrics"
	"logistics/services/gateway-svc/internal/clients"
	"logistics/services/gateway-svc/internal/handlers"
	"logistics/services/gateway-svc/internal/middleware"
)

const (
	statusHealthy = "HEALTHY"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadWithServiceDefaults("gateway-svc", 8080)
	if err != nil {
		logger.Init("error")
		logger.Fatal("Failed to load config", "error", err)
	}

	// Инициализируем логгер
	logger.InitWithConfig(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
	})

	logger.Log.Info("Starting Gateway Service (ConnectRPC)",
		"version", cfg.App.Version,
		"environment", cfg.App.Environment,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Инициализируем клиенты к backend-сервисам (gRPC)
	clientManager, err := clients.NewManager(ctx, &clients.Config{
		Solver:     cfg.Services.Solver,
		Analytics:  cfg.Services.Analytics,
		Validation: cfg.Services.Validation,
		History:    cfg.Services.History,
		Auth:       cfg.Services.Auth,
		Simulation: cfg.Services.Simulation,
		Report:     cfg.Services.Report,
		Audit:      cfg.Services.Audit,
	})
	if err != nil {
		logger.Fatal("Failed to initialize clients", "error", err)
	}
	defer clientManager.Close()

	// Создаём handler
	gatewayHandler := handlers.NewGatewayHandler(clientManager, cfg)

	// Создаём HTTP mux
	mux := http.NewServeMux()

	// Регистрируем ConnectRPC handler с interceptors
	path, handler := gatewayv1connect.NewGatewayServiceHandler(
		gatewayHandler,
		connect.WithInterceptors(
			middleware.NewLoggingInterceptor(),
			middleware.NewAuthInterceptor(clientManager.Auth()),
			middleware.NewRateLimitInterceptor(cfg.RateLimit),
			middleware.NewMetricsInterceptor(),
		),
	)
	mux.Handle(path, handler)

	// Health endpoints (обычный HTTP для k8s probes)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady(clientManager))

	// Metrics endpoint
	if cfg.Metrics.Enabled {
		mux.Handle("/metrics", metrics.Handler())
	}

	// Применяем CORS middleware
	var httpHandler http.Handler = mux
	if cfg.HTTP.CORS.Enabled {
		httpHandler = middleware.CORS(cfg.HTTP.CORS)(mux)
	}

	// Создаём HTTP сервер с поддержкой HTTP/2
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      h2c.NewHandler(httpHandler, &http2.Server{}),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	// Запускаем сервер
	go func() {
		logger.Log.Info("Gateway listening",
			"port", cfg.HTTP.Port,
			"protocol", "HTTP/1.1 + H2C (ConnectRPC)",
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Server shutdown error", "error", err)
	}

	logger.Log.Info("Server stopped")
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		// Логировать не можем - response уже начат отправляться
		return
	}
}

func handleReady(clientManager *clients.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := clientManager.CheckHealth(r.Context())
		allHealthy := true
		for _, h := range health {
			if h.Status != statusHealthy {
				allHealthy = false
				break
			}
		}
		if allHealthy {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"ready":true}`)); err != nil {
				return
			}
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, err := w.Write([]byte(`{"ready":false}`)); err != nil {
				return
			}
		}
	}
}
