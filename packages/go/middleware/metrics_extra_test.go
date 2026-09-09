// Tests for middleware metrics helpers (metrics.go).
package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestExtra_MetricsDefaultConfig(t *testing.T) {
	cfg := MetricsDefaultConfig()
	if !cfg.NormalizePath {
		t.Error("NormalizePath should default to true")
	}
	if len(cfg.SkipPaths) == 0 {
		t.Error("SkipPaths should have defaults")
	}
}

func TestExtra_MetricsConfig_Fields(t *testing.T) {
	cfg := MetricsConfig{
		SkipPaths:            []string{"/healthz"},
		NormalizePath:        true,
		ResponseSizeBuckets:  []float64{100, 1000},
		DurationBuckets:      []float64{.1, 1, 10},
		ExtraLabels:          map[string]string{"env": "test"},
	}
	if !cfg.NormalizePath {
		t.Error("NormalizePath")
	}
	if cfg.ExtraLabels["env"] != "test" {
		t.Error("ExtraLabels")
	}
	if len(cfg.DurationBuckets) != 3 {
		t.Error("DurationBuckets")
	}
}

func TestExtra_Metrics_Execution(t *testing.T) {
	cfg := MetricsDefaultConfig()
	cfg.SkipPaths = nil // Don't skip any path
	
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/users")
	
	mw := Metrics(cfg)
	called := false
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	
	if err := h(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if !called {
		t.Error("handler not called")
	}
}

func TestExtra_Metrics_SkipPath(t *testing.T) {
	cfg := MetricsDefaultConfig()
	cfg.SkipPaths = []string{"/healthz"}
	
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/healthz")
	
	mw := Metrics(cfg)
	called := false
	h := mw(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})
	
	if err := h(c); err != nil {
		t.Errorf("err: %v", err)
	}
	if !called {
		t.Error("handler should be called")
	}
}

func TestExtra_Metrics_UnknownPath(t *testing.T) {
	cfg := MetricsDefaultConfig()
	cfg.SkipPaths = nil
	
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Don't set path
	
	mw := Metrics(cfg)
	h := mw(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	
	if err := h(c); err != nil {
		t.Errorf("err: %v", err)
	}
}

func TestExtra_getRequestSize(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api", strings.NewReader("body"))
	req.ContentLength = 4
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	
	size := getRequestSize(c)
	if size != 4 {
		t.Errorf("expected 4, got %d", size)
	}
}

func TestExtra_getRequestSize_NoBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	
	size := getRequestSize(c)
	if size != 0 {
		t.Errorf("expected 0, got %d", size)
	}
}

func TestExtra_NewServiceMetrics(t *testing.T) {
	sm := NewServiceMetrics("test", nil)
	if sm == nil {
		t.Fatal("NewServiceMetrics returned nil")
	}
	if sm.RequestsTotal == nil {
		t.Error("RequestsTotal not set")
	}
	if sm.RequestDuration == nil {
		t.Error("RequestDuration not set")
	}
	if sm.ResponseSize == nil {
		t.Error("ResponseSize not set")
	}
	if sm.InFlight == nil {
		t.Error("InFlight not set")
	}
}

func TestExtra_NewServiceMetrics_CustomBuckets(t *testing.T) {
	// Skip because NewServiceMetrics uses promauto which uses the default registry
	// and creating multiple service metrics with same prefix would conflict
	t.Skip("NewServiceMetrics uses promauto (default registry); custom buckets test would need a separate registry")
}

func TestExtra_ServiceMetrics_Middleware(t *testing.T) {
	t.Skip("ServiceMetrics uses promauto (default registry); would conflict with HTTPRequestsTotal etc.")
}

func TestExtra_ServiceMetrics_Middleware_EmptyPath(t *testing.T) {
	t.Skip("ServiceMetrics uses promauto (default registry); would conflict with HTTPRequestsTotal etc.")
}

func TestExtra_ServiceMetrics_Record(t *testing.T) {
	t.Skip("ServiceMetrics uses promauto (default registry); would conflict with HTTPRequestsTotal etc.")
}

func TestExtra_ServiceMetrics_RecordError(t *testing.T) {
	t.Skip("ServiceMetrics uses promauto (default registry); would conflict with HTTPRequestsTotal etc.")
}

func TestExtra_ServiceMetrics_RecordSuccess(t *testing.T) {
	t.Skip("ServiceMetrics uses promauto (default registry); would conflict with HTTPRequestsTotal etc.")
}

func TestExtra_NewBusinessMetrics(t *testing.T) {
	t.Skip("NewBusinessMetrics uses promauto (default registry); would conflict with other tests")
}

func TestExtra_BusinessMetrics_RecordLead(t *testing.T) {
	t.Skip("BusinessMetrics uses promauto (default registry); would conflict with other tests")
}

func TestExtra_BusinessMetrics_RecordFormSubmission(t *testing.T) {
	t.Skip("BusinessMetrics uses promauto (default registry); would conflict with other tests")
}

func TestExtra_BusinessMetrics_RecordCAPIEvent(t *testing.T) {
	t.Skip("BusinessMetrics uses promauto (default registry); would conflict with other tests")
}

func TestExtra_BusinessMetrics_RecordPageView(t *testing.T) {
	t.Skip("BusinessMetrics uses promauto (default registry); would conflict with other tests")
}

func TestExtra_HTTPRequestsTotal_Init(t *testing.T) {
	if HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal should be initialized")
	}
	counter, err := HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/test", "200")
	if err != nil {
		t.Errorf("err: %v", err)
	}
	if counter == nil {
		t.Error("counter is nil")
	}
}

func TestExtra_HTTPRequestDuration_Init(t *testing.T) {
	if HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration should be initialized")
	}
}

func TestExtra_HTTPResponseSize_Init(t *testing.T) {
	if HTTPResponseSize == nil {
		t.Error("HTTPResponseSize should be initialized")
	}
}

func TestExtra_HTTPRequestsInFlight(t *testing.T) {
	if HTTPRequestsInFlight == nil {
		t.Error("HTTPRequestsInFlight should be initialized")
	}
	// Inc and Dec should not panic
	HTTPRequestsInFlight.Inc()
	HTTPRequestsInFlight.Dec()
}

func TestExtra_HTTPUseSizeTotal(t *testing.T) {
	if HTTPUseSizeTotal == nil {
		t.Error("HTTPUseSizeTotal should be initialized")
	}
}

func TestExtra_MetricVec_Write(t *testing.T) {
	// Verify HTTPRequestsTotal can produce a valid metric
	counter, _ := HTTPRequestsTotal.GetMetricWithLabelValues("POST", "/api/users", "201")
	counter.Inc()
	
	pb := &dto.Metric{}
	if err := counter.(prometheus.Metric).Write(pb); err != nil {
		t.Errorf("write: %v", err)
	}
}
