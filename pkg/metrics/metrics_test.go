package metrics

import (
	"runtime"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestInitMetrics(t *testing.T) {
	// Create fresh registry to avoid conflicts
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := InitMetrics("test", "service")

	if m == nil {
		t.Fatal("InitMetrics returned nil")
	}

	if m.GRPCRequestsTotal == nil {
		t.Error("GRPCRequestsTotal should not be nil")
	}
	if m.GRPCRequestDuration == nil {
		t.Error("GRPCRequestDuration should not be nil")
	}
	if m.SolveOperationsTotal == nil {
		t.Error("SolveOperationsTotal should not be nil")
	}
}

func TestGet(t *testing.T) {
	// Reset default metrics
	defaultMetrics = nil

	m := Get()
	if m == nil {
		t.Error("Get() should not return nil")
	}

	// Second call should return same instance
	m2 := Get()
	if m2 != m {
		t.Error("Get() should return same instance")
	}
}

func TestRecordGRPCRequest(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := InitMetrics("test", "grpc")

	// Should not panic
	m.RecordGRPCRequest("/test.Service/Method", "OK", 100*time.Millisecond)
	m.RecordGRPCRequest("/test.Service/Method", "ERROR", 50*time.Millisecond)
}

func TestRecordSolveOperation(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := InitMetrics("test", "solve")

	m.RecordSolveOperation("DINIC", true, 500*time.Millisecond, 100.5)
	m.RecordSolveOperation("EDMONDS_KARP", false, 1*time.Second, 0)
}

func TestRecordGraphSize(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := InitMetrics("test", "graph")

	m.RecordGraphSize("solve", 100, 500)
	m.RecordGraphSize("validate", 50, 200)
}

func TestRecordBottlenecks(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := InitMetrics("test", "bottleneck")

	m.RecordBottlenecks("critical", 2)
	m.RecordBottlenecks("high", 5)
}

func TestSetServiceInfo(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg

	m := InitMetrics("test", "info")

	m.SetServiceInfo("1.0.0", "production")
}

func TestRuntimeCollector(t *testing.T) {
	collector := NewRuntimeCollector("test", "runtime")

	// Test Describe
	descCh := make(chan *prometheus.Desc, 10)
	collector.Describe(descCh)
	close(descCh)

	count := 0
	for range descCh {
		count++
	}
	if count < 5 {
		t.Errorf("expected at least 5 descriptors, got %d", count)
	}

	// Test Collect
	metricCh := make(chan prometheus.Metric, 10)
	collector.Collect(metricCh)
	close(metricCh)

	count = 0
	for range metricCh {
		count++
	}
	if count < 5 {
		t.Errorf("expected at least 5 metrics, got %d", count)
	}
}

func TestRequestTracker(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_in_flight",
	})

	tracker := NewRequestTracker(gauge)

	tracker.Start("/method1")
	tracker.Start("/method1")
	tracker.Start("/method2")

	// Check active counts
	if tracker.active["/method1"] != 2 {
		t.Errorf("active[method1] = %d, want 2", tracker.active["/method1"])
	}

	tracker.End("/method1")
	if tracker.active["/method1"] != 1 {
		t.Errorf("active[method1] = %d, want 1", tracker.active["/method1"])
	}

	// End more than started should not go negative
	tracker.End("/method1")
	tracker.End("/method1")
	if tracker.active["/method1"] < 0 {
		t.Error("active count should not go negative")
	}
}

func TestTimer(t *testing.T) {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_duration",
			Buckets: []float64{.01, .1, 1},
		},
		[]string{"method"},
	)

	timer := NewTimer(histogram, "test_method")

	time.Sleep(10 * time.Millisecond)

	duration := timer.ObserveDuration()
	if duration < 10*time.Millisecond {
		t.Errorf("duration = %v, expected >= 10ms", duration)
	}
}

func TestHandler(t *testing.T) {
	handler := Handler()
	if handler == nil {
		t.Error("Handler() should not return nil")
	}
}

func TestRuntimeCollector_GCPause(t *testing.T) {
	// Force a GC to ensure we have GC data
	runtime.GC()

	collector := NewRuntimeCollector("test", "gc")
	metricCh := make(chan prometheus.Metric, 10)
	collector.Collect(metricCh)
	close(metricCh)

	// Should have collected GC pause metric
	found := false
	for range metricCh {
		found = true
	}
	if !found {
		t.Error("should have collected at least one metric")
	}
}
