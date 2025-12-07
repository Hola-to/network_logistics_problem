package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// RuntimeCollector собирает метрики runtime
type RuntimeCollector struct {
	goroutines *prometheus.Desc
	memAlloc   *prometheus.Desc
	memTotal   *prometheus.Desc
	memSys     *prometheus.Desc
	gcPause    *prometheus.Desc
	gcRuns     *prometheus.Desc
}

// NewRuntimeCollector создаёт новый коллектор runtime метрик
func NewRuntimeCollector(namespace, subsystem string) *RuntimeCollector {
	return &RuntimeCollector{
		goroutines: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "runtime_goroutines"),
			"Number of goroutines",
			nil, nil,
		),
		memAlloc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "runtime_memory_alloc_bytes"),
			"Bytes allocated and still in use",
			nil, nil,
		),
		memTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "runtime_memory_total_alloc_bytes"),
			"Total bytes allocated (even if freed)",
			nil, nil,
		),
		memSys: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "runtime_memory_sys_bytes"),
			"Bytes obtained from system",
			nil, nil,
		),
		gcPause: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "runtime_gc_pause_seconds"),
			"GC pause duration",
			nil, nil,
		),
		gcRuns: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "runtime_gc_runs_total"),
			"Total number of completed GC cycles",
			nil, nil,
		),
	}
}

// Describe implements prometheus.Collector
func (c *RuntimeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.goroutines
	ch <- c.memAlloc
	ch <- c.memTotal
	ch <- c.memSys
	ch <- c.gcPause
	ch <- c.gcRuns
}

// Collect implements prometheus.Collector
func (c *RuntimeCollector) Collect(ch chan<- prometheus.Metric) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	ch <- prometheus.MustNewConstMetric(c.goroutines, prometheus.GaugeValue, float64(runtime.NumGoroutine()))
	ch <- prometheus.MustNewConstMetric(c.memAlloc, prometheus.GaugeValue, float64(stats.Alloc))
	ch <- prometheus.MustNewConstMetric(c.memTotal, prometheus.CounterValue, float64(stats.TotalAlloc))
	ch <- prometheus.MustNewConstMetric(c.memSys, prometheus.GaugeValue, float64(stats.Sys))
	ch <- prometheus.MustNewConstMetric(c.gcRuns, prometheus.CounterValue, float64(stats.NumGC))

	// Последняя пауза GC
	if stats.NumGC > 0 {
		ch <- prometheus.MustNewConstMetric(c.gcPause, prometheus.GaugeValue, float64(stats.PauseNs[(stats.NumGC-1)%256])/1e9)
	}
}

// RequestTracker отслеживает активные запросы
type RequestTracker struct {
	mu       sync.Mutex
	active   map[string]int
	inFlight prometheus.Gauge
}

// NewRequestTracker создаёт новый трекер запросов
func NewRequestTracker(inFlight prometheus.Gauge) *RequestTracker {
	return &RequestTracker{
		active:   make(map[string]int),
		inFlight: inFlight,
	}
}

// Start отмечает начало запроса
func (t *RequestTracker) Start(method string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.active[method]++
	t.inFlight.Inc()
}

// End отмечает завершение запроса
func (t *RequestTracker) End(method string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.active[method] > 0 {
		t.active[method]--
		t.inFlight.Dec()
	}
}

// Timer для измерения времени выполнения
type Timer struct {
	start    time.Time
	observer prometheus.Observer
}

// NewTimer создаёт новый таймер
func NewTimer(histogram *prometheus.HistogramVec, labels ...string) *Timer {
	return &Timer{
		start:    time.Now(),
		observer: histogram.WithLabelValues(labels...),
	}
}

// ObserveDuration записывает длительность
func (t *Timer) ObserveDuration() time.Duration {
	duration := time.Since(t.start)
	t.observer.Observe(duration.Seconds())
	return duration
}
