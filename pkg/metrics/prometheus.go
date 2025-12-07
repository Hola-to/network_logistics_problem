package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics глобальный контейнер метрик
type Metrics struct {
	// gRPC метрики
	GRPCRequestsTotal    *prometheus.CounterVec
	GRPCRequestDuration  *prometheus.HistogramVec
	GRPCRequestsInFlight prometheus.Gauge

	// Бизнес-метрики
	SolveOperationsTotal *prometheus.CounterVec
	SolveDuration        *prometheus.HistogramVec
	MaxFlowValue         *prometheus.GaugeVec
	GraphNodesTotal      *prometheus.HistogramVec
	GraphEdgesTotal      *prometheus.HistogramVec
	BottlenecksFound     *prometheus.HistogramVec

	// Системные метрики
	MemoryUsage *prometheus.GaugeVec
	Goroutines  prometheus.Gauge

	// Информация о сервисе
	ServiceInfo *prometheus.GaugeVec
}

var defaultMetrics *Metrics

// InitMetrics инициализирует метрики
func InitMetrics(namespace, subsystem string) *Metrics {
	m := &Metrics{
		// gRPC метрики
		GRPCRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "grpc_requests_total",
				Help:      "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),

		GRPCRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "grpc_request_duration_seconds",
				Help:      "Duration of gRPC requests",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method"},
		),

		GRPCRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "grpc_requests_in_flight",
				Help:      "Current number of gRPC requests being processed",
			},
		),

		// Бизнес-метрики
		SolveOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "solve_operations_total",
				Help:      "Total number of solve operations",
			},
			[]string{"algorithm", "status"},
		),

		SolveDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "solve_duration_seconds",
				Help:      "Duration of solve operations",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"algorithm"},
		),

		MaxFlowValue: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "max_flow_value",
				Help:      "Last calculated max flow value",
			},
			[]string{"algorithm"},
		),

		GraphNodesTotal: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "graph_nodes_total",
				Help:      "Number of nodes in processed graphs",
				Buckets:   []float64{10, 50, 100, 500, 1000, 5000, 10000, 50000},
			},
			[]string{"operation"},
		),

		GraphEdgesTotal: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "graph_edges_total",
				Help:      "Number of edges in processed graphs",
				Buckets:   []float64{20, 100, 500, 1000, 5000, 10000, 50000, 100000},
			},
			[]string{"operation"},
		),

		BottlenecksFound: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "bottlenecks_found",
				Help:      "Number of bottlenecks found",
				Buckets:   []float64{0, 1, 2, 5, 10, 20, 50},
			},
			[]string{"severity"},
		),

		// Системные метрики
		MemoryUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "memory_usage_bytes",
				Help:      "Current memory usage",
			},
			[]string{"type"},
		),

		Goroutines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "goroutines",
				Help:      "Current number of goroutines",
			},
		),

		ServiceInfo: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "service_info",
				Help:      "Service information",
			},
			[]string{"version", "environment"},
		),
	}

	defaultMetrics = m
	return m
}

// Get возвращает глобальные метрики
func Get() *Metrics {
	if defaultMetrics == nil {
		return InitMetrics("logistics", "")
	}
	return defaultMetrics
}

// RecordGRPCRequest записывает метрики gRPC запроса
func (m *Metrics) RecordGRPCRequest(method string, status string, duration time.Duration) {
	m.GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	m.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordSolveOperation записывает метрики операции решения
func (m *Metrics) RecordSolveOperation(algorithm string, success bool, duration time.Duration, maxFlow float64) {
	status := "success"
	if !success {
		status = "error"
	}

	m.SolveOperationsTotal.WithLabelValues(algorithm, status).Inc()
	m.SolveDuration.WithLabelValues(algorithm).Observe(duration.Seconds())
	m.MaxFlowValue.WithLabelValues(algorithm).Set(maxFlow)
}

// RecordGraphSize записывает размер графа
func (m *Metrics) RecordGraphSize(operation string, nodes, edges int) {
	m.GraphNodesTotal.WithLabelValues(operation).Observe(float64(nodes))
	m.GraphEdgesTotal.WithLabelValues(operation).Observe(float64(edges))
}

// RecordBottlenecks записывает количество найденных узких мест
func (m *Metrics) RecordBottlenecks(severity string, count int) {
	m.BottlenecksFound.WithLabelValues(severity).Observe(float64(count))
}

// SetServiceInfo устанавливает информацию о сервисе
func (m *Metrics) SetServiceInfo(version, environment string) {
	m.ServiceInfo.WithLabelValues(version, environment).Set(1)
}

// Handler возвращает HTTP handler для /metrics
func Handler() http.Handler {
	return promhttp.Handler()
}

// StartMetricsServer запускает HTTP сервер для метрик
func StartMetricsServer(port int) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Игнорируем ошибку записи - response уже отправлен
		_, _ = w.Write([]byte("OK")) //nolint:errcheck // health endpoint, ошибка записи не критична
	})

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return server.ListenAndServe()
}
