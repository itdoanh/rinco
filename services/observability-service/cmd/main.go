// Observability Service — unified self-hosted observability API.
// Aggregates logs (Loki), traces (Tempo/Jaeger), metrics (Prometheus), audit
// (ClickHouse).  Listens to AlertManager webhooks → emits notifications.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "observability-service"
	version     = "1.0.0"
)

// =============================================================================
// Types
// =============================================================================

type serviceHealth struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	URL       string    `json:"url"`
	LastCheck time.Time `json:"last_check"`
	LatencyMs int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
}

type activeAlert struct {
	ID        string            `json:"id"`
	Service   string            `json:"service"`
	Severity  string            `json:"severity"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	StartedAt time.Time         `json:"started_at"`
	Labels    map[string]string `json:"labels"`
	Status    string            `json:"status"`
	AckBy     string            `json:"acknowledged_by,omitempty"`
}

type ackRequest struct {
	UserID string `json:"user_id"`
}

type incident struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Severity  string    `json:"severity"`
	Title     string    `json:"title"`
	StartedAt time.Time `json:"started_at"`
	Resolved  bool      `json:"resolved"`
}

type logQuery struct {
	Service  string `json:"service"`
	From     string `json:"from"`
	To       string `json:"to"`
	Filter   string `json:"filter"`
	Limit    int    `json:"limit"`
	TenantID string `json:"tenant_id"`
}

type traceSpan struct {
	SpanID    string `json:"span_id"`
	TraceID   string `json:"trace_id"`
	Operation string `json:"operation"`
	Service   string `json:"service"`
	Start     int64  `json:"start_ms"`
	Duration  int64  `json:"duration_ms"`
	Tags      map[string]string `json:"tags"`
}

type metricSample struct {
	Name      string                 `json:"name"`
	Labels    map[string]string     `json:"labels"`
	Value     float64               `json:"value"`
	Timestamp time.Time             `json:"timestamp"`
}

// =============================================================================
// State
// =============================================================================

var (
	muA, muI sync.RWMutex
	alerts   = map[string]*activeAlert{}
	incidents = []incident{}
)

// =============================================================================
// main
// =============================================================================

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(rincowebmw.Recovery())
	e.Use(rincowebmw.Trace())
	e.Use(rincowebmw.Logger())
	e.Use(rincowebmw.Metrics(serviceName))
	e.Use(rincowebmw.CORS([]string{"*"}))
	e.Use(rincowebmw.SecurityHeaders())

	e.GET("/health", healthHandler)
	e.GET("/ready", readyHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	o := e.Group("/v1/observability")
	o.GET("/logs", queryLogsHandler)
	o.GET("/traces/:trace_id", queryTraceHandler)
	o.GET("/metrics", queryMetricsHandler)
	o.GET("/services", listServicesHandler)
	o.GET("/services/:service", serviceHealthHandler)
	o.GET("/alerts/active", listActiveAlertsHandler)
	o.POST("/alerts/ack/:id", ackAlertHandler)
	o.GET("/audit/logs", queryAuditLogsHandler)
	o.POST("/webhook/alertmanager", alertmanagerWebhookHandler)
	o.GET("/dashboards", listDashboardsHandler)
	o.GET("/dashboards/:name", dashboardHandler)

	port := ":" + getEnv("PORT", "8089")
	logger.Info(context.Background(), "starting observability service",
		zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

// =============================================================================
// Metrics
// =============================================================================

var (
	upstreamLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rinco_observability_upstream_latency_milliseconds",
		Help:    "Latency of upstream calls (Loki, Prometheus, Jaeger)",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"backend", "op"})
	upstreamErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rinco_observability_upstream_errors_total",
		Help: "Upstream request failures",
	}, []string{"backend"})
	alertsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rinco_observability_alerts_active",
		Help: "Currently active alerts",
	})
)

// =============================================================================
// Logs
// =============================================================================

func queryLogsHandler(c echo.Context) error {
	q := logQuery{
		Service:  c.QueryParam("service"),
		From:     c.QueryParam("from"),
		To:       c.QueryParam("to"),
		Filter:   c.QueryParam("filter"),
		TenantID: c.QueryParam("tenant_id"),
	}
	q.Limit = strconvDefault(c.QueryParam("limit"), 100)

	lokiURL := getEnv("LOKI_URL", "http://loki:3100")
	if lokiURL == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{"logs": []string{}, "note": "Loki not configured"})
	}

	// Translate to Loki query_range.
	params := url.Values{}
	params.Set("limit", strconv.Itoa(q.Limit))
	if q.From != "" {
		params.Set("start", q.From)
	}
	if q.To != "" {
		params.Set("end", q.To)
	}
	query := fmt.Sprintf(`{service=%q}`, q.Service)
	if q.Filter != "" {
		query = fmt.Sprintf(`%s |= %q`, query, q.Filter)
	}
	if q.TenantID != "" {
		query = fmt.Sprintf(`%s, tenant_id=%q`, query, q.TenantID)
	}
	params.Set("query", query)

	endpoint := lokiURL + "/loki/api/v1/query_range?" + params.Encode()
	data, err := httpJSON("GET", endpoint, nil, "")
	if err != nil {
		upstreamErrors.WithLabelValues("loki").Inc()
		logger.Warn(c.Request().Context(), "loki query failed", zap.Error(err))
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(data, &resp)
	return c.JSON(http.StatusOK, resp)
}

// =============================================================================
// Traces
// =============================================================================

func queryTraceHandler(c echo.Context) error {
	traceID := c.Param("trace_id")
	jaegerURL := getEnv("JAEGER_URL", "http://jaeger:16686")
	endpoint := fmt.Sprintf("%s/api/traces/%s", jaegerURL, traceID)
	data, err := httpJSON("GET", endpoint, nil, "")
	if err != nil {
		upstreamErrors.WithLabelValues("jaeger").Inc()
		logger.Warn(c.Request().Context(), "jaeger query failed", zap.Error(err))
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.Stream(http.StatusOK, "application/json", bytes.NewReader(data))
}

// =============================================================================
// Metrics (Prometheus query_range)
// =============================================================================

func queryMetricsHandler(c echo.Context) error {
	promURL := getEnv("PROM_URL", "http://prometheus:9090")
	q := c.QueryParam("query")
	if q == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query required"})
	}
	endpoint := promURL + "/api/v1/query?query=" + url.QueryEscape(q)
	data, err := httpJSON("GET", endpoint, nil, "")
	if err != nil {
		upstreamErrors.WithLabelValues("prom").Inc()
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.Stream(http.StatusOK, "application/json", bytes.NewReader(data))
}

// =============================================================================
// Services (aggregate health)
// =============================================================================

func listServicesHandler(c echo.Context) error {
	svcs := knownServices()
	out := make([]serviceHealth, 0, len(svcs))
	overall := "healthy"
	for name, urlStr := range svcs {
		s := probe(name, urlStr)
		if s.Status != "ok" {
			overall = "degraded"
		}
		out = append(out, s)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"overall":    overall,
		"services":   out,
		"checked_at": time.Now(),
	})
}

func serviceHealthHandler(c echo.Context) error {
	name := c.Param("service")
	svcs := knownServices()
	if urlStr, ok := svcs[name]; ok {
		return c.JSON(http.StatusOK, probe(name, urlStr))
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": "unknown service"})
}

func probe(name, urlStr string) serviceHealth {
	start := time.Now()
	s := serviceHealth{Name: name, URL: urlStr, LastCheck: time.Now()}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(urlStr + "/health")
	s.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		s.Status = "down"
		s.Error = err.Error()
		upstreamErrors.WithLabelValues("probe").Inc()
		return s
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		s.Status = "degraded"
		return s
	}
	s.Status = "ok"
	return s
}

func knownServices() map[string]string {
	return map[string]string{
		"auth-service":           getEnv("AUTH_URL", "http://auth-service:8081"),
		"tenant-service":         getEnv("TENANT_URL", "http://tenant-service:8082"),
		"crm-service":            getEnv("CRM_URL", "http://crm-service:8083"),
		"dynamic-model-service":  getEnv("DMS_URL", "http://dynamic-model-service:8084"),
		"landing-service":        getEnv("LANDING_URL", "http://landing-service:8086"),
		"email-service":          getEnv("EMAIL_URL", "http://email-service:8087"),
		"notification-service":   getEnv("NOTIF_URL", "http://notification-service:8088"),
		"lead-scoring":           getEnv("LEAD_SCORE_URL", "http://lead-scoring:8092"),
		"ai-sre":                 getEnv("AI_SRE_URL", "http://ai-sre:8090"),
		"rag-chatbot":            getEnv("RAG_URL", "http://rag-chatbot:8091"),
		"stt-service":            getEnv("STT_URL", "http://stt-service:8093"),
		"recording-service":      getEnv("REC_URL", "http://recording-service:8094"),
		"chat-engine":            getEnv("CHAT_URL", "http://chat-engine:8095"),
		"webrtc-sfu":             getEnv("SFU_URL", "http://webrtc-sfu:8096"),
	}
}

// =============================================================================
// Alerts
// =============================================================================

func listActiveAlertsHandler(c echo.Context) error {
	muA.RLock()
	out := []*activeAlert{}
	for _, a := range alerts {
		out = append(out, a)
	}
	muA.RUnlock()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"count":   len(out),
		"alerts":  out,
		"checked": time.Now(),
	})
}

func ackAlertHandler(c echo.Context) error {
	id := c.Param("id")
	var req ackRequest
	_ = c.Bind(&req)
	if req.UserID == "" {
		req.UserID = "unknown"
	}
	muA.Lock()
	if a, ok := alerts[id]; ok {
		a.Status = "acknowledged"
		a.AckBy = req.UserID
	}
	muA.Unlock()
	return c.JSON(http.StatusOK, map[string]string{"status": "acknowledged", "by": req.UserID})
}

func alertmanagerWebhookHandler(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	var payload struct {
		Alerts []struct {
			Labels map[string]string `json:"labels"`
			Status string            `json:"status"`
			StartsAt time.Time       `json:"startsAt"`
			EndsAt time.Time         `json:"endsAt"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	for _, a := range payload.Alerts {
		id := fmt.Sprintf("%s/%s",
			a.Labels["alertname"], a.Labels["service"])
		muA.Lock()
		switch a.Status {
		case "firing":
			alerts[id] = &activeAlert{
				ID:        id,
				Service:   a.Labels["service"],
				Severity:  a.Labels["severity"],
				Title:     a.Labels["alertname"],
				Message:   fmt.Sprintf("%v", a.Labels),
				StartedAt: a.StartsAt,
				Labels:    a.Labels,
				Status:    "firing",
			}
			muI.Lock()
			incidents = append(incidents, incident{
				ID:        uuid.NewV7().String(),
				Service:   a.Labels["service"],
				Severity:  a.Labels["severity"],
				Title:     a.Labels["alertname"],
				StartedAt: a.StartsAt,
				Resolved: false,
			})
			muI.Unlock()
		case "resolved":
			if existing, ok := alerts[id]; ok {
				existing.Status = "resolved"
			}
			muI.Lock()
			for i := range incidents {
				if !incidents[i].Resolved && incidents[i].Service == a.Labels["service"] &&
					incidents[i].Title == a.Labels["alertname"] {
					incidents[i].Resolved = true
				}
			}
			muI.Unlock()
		}
		muA.Unlock()
	}
	alertsActive.Set(float64(countFiringAlerts()))
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func countFiringAlerts() int {
	muA.RLock()
	defer muA.RUnlock()
	c := 0
	for _, a := range alerts {
		if a.Status == "firing" {
			c++
		}
	}
	return c
}

// =============================================================================
// Audit logs (ClickHouse)
// =============================================================================

func queryAuditLogsHandler(c echo.Context) error {
	chURL := getEnv("CLICKHOUSE_URL", "http://clickhouse:8123")
	if chURL == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{"rows": []string{}})
	}
	tenantID := c.QueryParam("tenant_id")
	limit := strconvDefault(c.QueryParam("limit"), 100)
	query := fmt.Sprintf(
		"SELECT ts, tenant_id, service, user_id, action, resource, ip "+
			"FROM audit.events "+
			"WHERE (tenant_id='%s' OR '%s'='')"+
			"ORDER BY ts DESC LIMIT %d FORMAT JSON",
		escapeCH(tenantID), escapeCH(tenantID), limit)
	body := strings.NewReader(query)
	resp, err := http.Post(chURL+"/?database=audit",
		"text/plain", body)
	if err != nil {
		upstreamErrors.WithLabelValues("clickhouse").Inc()
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return c.Stream(http.StatusOK, "application/json", bytes.NewReader(b))
}

func escapeCH(s string) string { return strings.ReplaceAll(s, "'", "''") }

// =============================================================================
// Dashboards (generated Grafana JSON)
// =============================================================================

func listDashboardsHandler(c echo.Context) error {
	dbs := []map[string]interface{}{}
	for _, name := range []string{
		"system-overview", "service-latency", "error-budget",
		"tenant-activity", "ai-sre",
	} {
		dbs = append(dbs, map[string]interface{}{
			"name":      name,
			"url":       fmt.Sprintf("/v1/observability/dashboards/%s", name),
			"format":    "grafana-json",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"count":      len(dbs),
		"dashboards": dbs,
	})
}

func dashboardHandler(c echo.Context) error {
	name := c.Param("name")
	uid := uuid.NewV7().String()
	db := buildDashboard(name, uid)
	return c.JSON(http.StatusOK, db)
}

func buildDashboard(name, uid string) map[string]interface{} {
	switch name {
	case "service-latency":
		return latencyDashboard(uid)
	case "ai-sre":
		return aiSREDashboard(uid)
	}
	return overviewDashboard(uid)
}

func overviewDashboard(uid string) map[string]interface{} {
	return map[string]interface{}{
		"title":   "RINCO — System Overview",
		"uid":     uid,
		"schemaVersion": 39,
		"version": 1,
		"panels": []map[string]interface{}{
			{
				"title": "HTTP rate per service",
				"type":  "timeseries",
				"targets": []map[string]interface{}{
					{"expr": "sum by (service) (rate(rinco_http_requests_total[1m]))"},
				},
				"gridPos": map[string]int{"x": 0, "y": 0, "w": 12, "h": 8},
			},
			{
				"title": "P95 latency",
				"type":  "timeseries",
				"targets": []map[string]interface{}{
					{"expr": "histogram_quantile(0.95, sum by (le, service) (rate(rinco_http_request_duration_seconds_bucket[5m])))"},
				},
				"gridPos": map[string]int{"x": 12, "y": 0, "w": 12, "h": 8},
			},
			{
				"title": "Errors",
				"type":  "timeseries",
				"targets": []map[string]interface{}{
					{"expr": "sum by (service) (rate(rinco_http_requests_total{status=~\"5..\"}[1m]))"},
				},
				"gridPos": map[string]int{"x": 0, "y": 8, "w": 12, "h": 8},
			},
		},
	}
}

func latencyDashboard(uid string) map[string]interface{} {
	return map[string]interface{}{
		"title": "Service Latency", "uid": uid,
		"panels": []map[string]interface{}{
			{
				"title": "P50 / P95 / P99",
				"type":  "timeseries",
				"targets": []map[string]interface{}{
					{"expr": "histogram_quantile(0.50, sum by (le, service) (rate(rinco_http_request_duration_seconds_bucket[1m])))", "legendFormat": "p50 {{service}}"},
					{"expr": "histogram_quantile(0.95, sum by (le, service) (rate(rinco_http_request_duration_seconds_bucket[1m])))", "legendFormat": "p95 {{service}}"},
					{"expr": "histogram_quantile(0.99, sum by (le, service) (rate(rinco_http_request_duration_seconds_bucket[1m])))", "legendFormat": "p99 {{service}}"},
				},
			},
		},
	}
}

func aiSREDashboard(uid string) map[string]interface{} {
	return map[string]interface{}{
		"title": "AI SRE",
		"panels": []map[string]interface{}{
			{
				"title": "Inference latency",
				"targets": []map[string]interface{}{
					{"expr": "rate(rinco_observability_upstream_latency_milliseconds_sum[1m]) / rate(rinco_observability_upstream_latency_milliseconds_count[1m])"},
				},
			},
		},
	}
}

// =============================================================================
// Helpers
// =============================================================================

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func readyHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func strconvDefault(v string, def int) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func httpJSON(method, urlStr string, body io.Reader, apiKey string) ([]byte, error) {
	start := time.Now()
	req, _ := http.NewRequest(method, urlStr, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	upstreamLatency.WithLabelValues(extractBackend(urlStr), method).
		Observe(float64(time.Since(start).Milliseconds()))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s -> %d", method, urlStr, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func extractBackend(u string) string {
	if strings.Contains(u, "loki") {
		return "loki"
	}
	if strings.Contains(u, "jaeger") {
		return "jaeger"
	}
	if strings.Contains(u, "prom") {
		return "prom"
	}
	if strings.Contains(u, "clickhouse") {
		return "clickhouse"
	}
	return "other"
}
