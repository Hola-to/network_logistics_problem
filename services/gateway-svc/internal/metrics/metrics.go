package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	once     sync.Once
	instance *GatewayMetrics
)

// GatewayMetrics метрики gateway
type GatewayMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	BackendRequests *prometheus.CounterVec
	BackendDuration *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge

	// Запросы по категориям
	RequestsByCategory *prometheus.CounterVec

	// Время ответа по категориям
	ResponseTimeByCategory *prometheus.HistogramVec

	// Активные соединения с backend сервисами
	BackendConnections *prometheus.GaugeVec

	// Здоровье backend сервисов
	BackendHealth *prometheus.GaugeVec

	// Ошибки по типам
	ErrorsByType *prometheus.CounterVec

	// Размер запросов/ответов
	RequestSize  *prometheus.HistogramVec
	ResponseSize *prometheus.HistogramVec

	// Rate limiting
	RateLimitHits   prometheus.Counter
	RateLimitPassed prometheus.Counter

	// Auth
	AuthSuccessful prometheus.Counter
	AuthFailed     prometheus.Counter

	// Кэш (если используется)
	CacheHits   prometheus.Counter
	CacheMisses prometheus.Counter
}

// Init инициализирует метрики
func Init() *GatewayMetrics {
	once.Do(func() {
		instance = &GatewayMetrics{
			RequestsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "gateway_requests_total",
					Help:      "Total gateway requests",
				},
				[]string{"method", "status"},
			),

			RequestDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gateway",
					Name:      "gateway_request_duration_seconds",
					Help:      "Gateway request duration",
					Buckets:   prometheus.DefBuckets,
				},
				[]string{"method"},
			),

			BackendRequests: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "gateway_backend_requests_total",
					Help:      "Total backend service requests",
				},
				[]string{"service", "method", "status"},
			),

			BackendDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gateway",
					Name:      "gateway_backend_duration_seconds",
					Help:      "Backend service request duration",
					Buckets:   prometheus.DefBuckets,
				},
				[]string{"service", "method"},
			),

			ActiveRequests: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: "gateway",
					Name:      "gateway_active_requests",
					Help:      "Currently active requests",
				},
			),

			RequestsByCategory: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "requests_by_category_total",
					Help:      "Total requests by category",
				},
				[]string{"category", "method", "status"},
			),

			ResponseTimeByCategory: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gateway",
					Name:      "response_time_by_category_seconds",
					Help:      "Response time by category",
					Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
				},
				[]string{"category", "method"},
			),

			BackendConnections: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "gateway",
					Name:      "backend_connections",
					Help:      "Active connections to backend services",
				},
				[]string{"service"},
			),

			BackendHealth: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "gateway",
					Name:      "backend_health",
					Help:      "Backend service health (1=healthy, 0=unhealthy)",
				},
				[]string{"service"},
			),

			ErrorsByType: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "errors_by_type_total",
					Help:      "Total errors by type",
				},
				[]string{"type", "service"},
			),

			RequestSize: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gateway",
					Name:      "request_size_bytes",
					Help:      "Request size in bytes",
					Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
				},
				[]string{"method"},
			),

			ResponseSize: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "gateway",
					Name:      "response_size_bytes",
					Help:      "Response size in bytes",
					Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
				},
				[]string{"method"},
			),

			RateLimitHits: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "rate_limit_hits_total",
					Help:      "Total rate limit hits",
				},
			),

			RateLimitPassed: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "rate_limit_passed_total",
					Help:      "Total requests passed rate limit",
				},
			),

			AuthSuccessful: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "auth_successful_total",
					Help:      "Total successful authentications",
				},
			),

			AuthFailed: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "auth_failed_total",
					Help:      "Total failed authentications",
				},
			),

			CacheHits: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "cache_hits_total",
					Help:      "Total cache hits",
				},
			),

			CacheMisses: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "gateway",
					Name:      "cache_misses_total",
					Help:      "Total cache misses",
				},
			),
		}
	})
	return instance
}

// Get возвращает инстанс метрик
func Get() *GatewayMetrics {
	if instance == nil {
		return Init()
	}
	return instance
}

// RecordRequest записывает метрики запроса
func (m *GatewayMetrics) RecordRequest(category, method, status string, duration time.Duration) {
	m.RequestsByCategory.WithLabelValues(category, method, status).Inc()
	m.ResponseTimeByCategory.WithLabelValues(category, method).Observe(duration.Seconds())
}

// RecordBackendRequest записывает метрику backend запроса
func (m *GatewayMetrics) RecordBackendRequest(service, method, status string, duration time.Duration) {
	m.BackendRequests.WithLabelValues(service, method, status).Inc()
	m.BackendDuration.WithLabelValues(service, method).Observe(duration.Seconds())
}

// IncActiveRequests увеличивает счётчик активных запросов
func (m *GatewayMetrics) IncActiveRequests() {
	m.ActiveRequests.Inc()
}

// DecActiveRequests уменьшает счётчик активных запросов
func (m *GatewayMetrics) DecActiveRequests() {
	m.ActiveRequests.Dec()
}

// RecordBackendHealth записывает здоровье backend
func (m *GatewayMetrics) RecordBackendHealth(service string, healthy bool) {
	val := 0.0
	if healthy {
		val = 1.0
	}
	m.BackendHealth.WithLabelValues(service).Set(val)
}

// RecordError записывает ошибку
func (m *GatewayMetrics) RecordError(errorType, service string) {
	m.ErrorsByType.WithLabelValues(errorType, service).Inc()
}
