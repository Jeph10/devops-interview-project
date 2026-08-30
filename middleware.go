package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMiddleware wraps HTTP handlers with Prometheus metrics.
type PrometheusMiddleware struct {
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	inFlight        prometheus.Gauge
	tasksTotal      prometheus.Gauge
	tasksDone       prometheus.Gauge
}

// NewPrometheusMiddleware creates a new middleware instance with registered metrics.
func NewPrometheusMiddleware() *PrometheusMiddleware {
	return NewPrometheusMiddlewareWithRegistry(prometheus.DefaultRegisterer)
}

// NewPrometheusMiddlewareWithRegistry creates middleware using the given registerer.
// Useful for tests to avoid duplicate registration on the global registry.
func NewPrometheusMiddlewareWithRegistry(reg prometheus.Registerer) *PrometheusMiddleware {
	return &PrometheusMiddleware{
		requestCounter: promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: promauto.With(reg).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency distribution.",
				Buckets: []float64{0.001, 0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"method", "path"},
		),
		inFlight: promauto.With(reg).NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently in flight.",
			},
		),
		tasksTotal: promauto.With(reg).NewGauge(
			prometheus.GaugeOpts{
				Name: "task_api_tasks_total",
				Help: "Total number of tasks.",
			},
		),
		tasksDone: promauto.With(reg).NewGauge(
			prometheus.GaugeOpts{
				Name: "task_api_tasks_done",
				Help: "Number of completed tasks.",
			},
		),
	}
}

// UpdateTaskStats updates the task gauge metrics from the store.
func (pm *PrometheusMiddleware) UpdateTaskStats(total, done int) {
	pm.tasksTotal.Set(float64(total))
	pm.tasksDone.Set(float64(done))
}

// WrapHandler wraps an HTTP handler with Prometheus metrics collection.
func (pm *PrometheusMiddleware) WrapHandler(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pm.IncInFlight()
		defer pm.DecInFlight()

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next(rec, r)
		duration := time.Since(start).Seconds()

		route := normalizePath(r.Method, path, r)
		pm.RecordRequest(r.Method, route, strconv.Itoa(rec.statusCode), duration)
	}
}

// IncInFlight increments the in-flight request gauge.
func (pm *PrometheusMiddleware) IncInFlight() {
	pm.inFlight.Inc()
}

// DecInFlight decrements the in-flight request gauge.
func (pm *PrometheusMiddleware) DecInFlight() {
	pm.inFlight.Dec()
}

// RecordRequest records a single request's metrics.
func (pm *PrometheusMiddleware) RecordRequest(method, path, status string, duration float64) {
	pm.requestCounter.WithLabelValues(method, path, status).Inc()
	pm.requestDuration.WithLabelValues(method, path).Observe(duration)
}

// MetricsHandler returns the Prometheus /metrics HTTP handler.
func (pm *PrometheusMiddleware) MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsHandlerForRegistry returns a /metrics handler for a specific registry.
func (pm *PrometheusMiddleware) MetricsHandlerForRegistry(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

// statusRecorder captures the status code written by a handler.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// normalizePath returns a route pattern suitable for metric labels.
// It strips the numeric ID from /tasks/{id} to avoid high cardinality.
func normalizePath(method, pattern string, r *http.Request) string {
	if pattern != "" {
		return pattern
	}
	// Fallback: use the route pattern from the request.
	return r.URL.Path
}
