// Package middleware provides HTTP metrics middleware for Prometheus.
//
// This middleware records:
//   - Request duration (histogram)
//   - Request count (counter)
//   - Response size (histogram)
//   - Requests in flight (gauge)
package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts total HTTP requests.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks request duration.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// HTTPResponseSize tracks response sizes.
	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
		},
		[]string{"method", "path"},
	)

	// HTTPRequestsInFlight tracks concurrent requests.
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// HTTPUseSizeTotal tracks request body sizes.
	HTTPUseSizeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_size_bytes_total",
			Help: "Total size of HTTP request bodies in bytes",
		},
		[]string{"method", "path"},
	)
)

// MetricsConfig holds metrics middleware configuration.
type MetricsConfig struct {
	// SkipPaths are paths that won't be tracked
	SkipPaths []string

	// NormalizePath normalizes paths (e.g., /users/123 -> /users/:id)
	NormalizePath bool

	// Bucket sizes for response size histogram
	ResponseSizeBuckets []float64

	// Bucket sizes for duration histogram
	DurationBuckets []float64

	// ExtraLabels are additional labels to add to all metrics
	ExtraLabels map[string]string
}

// DefaultMetricsConfig returns sensible defaults.
func MetricsDefaultConfig() MetricsConfig {
	return MetricsConfig{
		SkipPaths:     []string{"/healthz", "/readyz", "/metrics"},
		NormalizePath: true,
	}
}

// Metrics creates a Prometheus metrics middleware for Echo.
func Metrics(cfg MetricsConfig) echo.MiddlewareFunc {
	skipPaths := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if skipPaths[path] {
				return next(c)
			}

			// Normalize path if enabled
			if cfg.NormalizePath {
				path = c.Path()
			}

			// Track in-flight requests
			HTTPRequestsInFlight.Inc()
			defer HTTPRequestsInFlight.Dec()

			// Record start time
			start := time.Now()

			// Get request size
			requestSize := getRequestSize(c)

			// Execute handler
			err := next(c)

			// Record metrics
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(c.Response().Status)
			method := c.Request().Method
			responseSize := c.Response().Size

			// Skip if path is still empty
			if path == "" {
				path = "unknown"
			}

			// Update counters
			HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
			HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
			HTTPResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
			HTTPUseSizeTotal.WithLabelValues(method, path).Add(float64(requestSize))

			return err
		}
	}
}

// getRequestSize returns the size of the request body.
func getRequestSize(c echo.Context) int64 {
	if c.Request().Body != nil {
		return c.Request().ContentLength
	}
	return 0
}

// ServiceMetrics holds service-specific metrics.
type ServiceMetrics struct {
	// RequestsTotal counts requests by endpoint
	RequestsTotal *prometheus.CounterVec

	// RequestDuration tracks request duration
	RequestDuration *prometheus.HistogramVec

	// ResponseSize tracks response sizes
	ResponseSize *prometheus.HistogramVec

	// InFlight tracks concurrent requests
	InFlight prometheus.Gauge

	prefix string
}

// NewServiceMetrics creates metrics for a specific service.
func NewServiceMetrics(prefix string, buckets []float64) *ServiceMetrics {
	if buckets == nil {
		buckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
	}

	sm := &ServiceMetrics{prefix: prefix}

	sm.RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_http_requests_total",
			Help: "Total number of HTTP requests for " + prefix,
		},
		[]string{"method", "path", "status"},
	)

	sm.RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds for " + prefix,
			Buckets: buckets,
		},
		[]string{"method", "path"},
	)

	sm.ResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_http_response_size_bytes",
			Help:    "HTTP response size in bytes for " + prefix,
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
		},
		[]string{"method", "path"},
	)

	sm.InFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: prefix + "_http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed for " + prefix,
		},
	)

	return sm
}

// Middleware returns an Echo middleware for these metrics.
func (m *ServiceMetrics) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if path == "" {
				path = "unknown"
			}

			m.InFlight.Inc()
			defer m.InFlight.Dec()

			start := time.Now()
			err := next(c)
			duration := time.Since(start).Seconds()

			status := strconv.Itoa(c.Response().Status)
			method := c.Request().Method

			m.RequestsTotal.WithLabelValues(method, path, status).Inc()
			m.RequestDuration.WithLabelValues(method, path).Observe(duration)
			m.ResponseSize.WithLabelValues(method, path).Observe(float64(c.Response().Size))

			return err
		}
	}
}

// Record records metrics for a request.
func (m *ServiceMetrics) Record(method, path, status string, duration time.Duration, responseSize int64) {
	m.RequestsTotal.WithLabelValues(method, path, status).Inc()
	m.RequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	m.ResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// RecordError records a request with error status.
func (m *ServiceMetrics) RecordError(method, path string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(method, path, "error").Inc()
	m.RequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordSuccess records a successful request.
func (m *ServiceMetrics) RecordSuccess(method, path string, duration time.Duration, responseSize int64) {
	m.RequestsTotal.WithLabelValues(method, path, "200").Inc()
	m.RequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	m.ResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// BusinessMetrics holds business-level metrics.
type BusinessMetrics struct {
	// LeadsCreated counts leads created
	LeadsCreated *prometheus.CounterVec

	// FormsSubmitted counts form submissions
	FormsSubmitted *prometheus.CounterVec

	// CAPIEventsSent counts CAPI events sent
	CAPIEventsSent *prometheus.CounterVec

	// PageViews counts page views
	PageViews *prometheus.CounterVec
}

// NewBusinessMetrics creates business-level metrics.
func NewBusinessMetrics() *BusinessMetrics {
	bm := &BusinessMetrics{}

	bm.LeadsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "leads_created_total",
			Help: "Total number of leads created",
		},
		[]string{"tenant_id", "form_slug"},
	)

	bm.FormsSubmitted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "forms_submitted_total",
			Help: "Total number of form submissions",
		},
		[]string{"tenant_id", "form_slug", "status"},
	)

	bm.CAPIEventsSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "capi_events_sent_total",
			Help: "Total number of CAPI events sent",
		},
		[]string{"tenant_id", "event_name", "status"},
	)

	bm.PageViews = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "page_views_total",
			Help: "Total number of page views",
		},
		[]string{"tenant_id", "page_slug"},
	)

	return bm
}

// RecordLead records a lead creation.
func (m *BusinessMetrics) RecordLead(tenantID, formSlug string) {
	m.LeadsCreated.WithLabelValues(tenantID, formSlug).Inc()
}

// RecordFormSubmission records a form submission.
func (m *BusinessMetrics) RecordFormSubmission(tenantID, formSlug, status string) {
	m.FormsSubmitted.WithLabelValues(tenantID, formSlug, status).Inc()
}

// RecordCAPIEvent records a CAPI event.
func (m *BusinessMetrics) RecordCAPIEvent(tenantID, eventName, status string) {
	m.CAPIEventsSent.WithLabelValues(tenantID, eventName, status).Inc()
}

// RecordPageView records a page view.
func (m *BusinessMetrics) RecordPageView(tenantID, pageSlug string) {
	m.PageViews.WithLabelValues(tenantID, pageSlug).Inc()
}
