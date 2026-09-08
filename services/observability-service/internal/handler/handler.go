// Package handler implements the HTTP + Connect-RPC API for the
// observability service.  It exposes:
//
//	GET  /v1/logs                         — Loki query proxy
//	GET  /v1/logs/aggregate               — Loki metric-style aggregate
//	GET  /v1/traces/:trace_id             — Jaeger/Tempo trace lookup
//	GET  /v1/traces                       — Jaeger trace search
//	GET  /v1/metrics                      — PromQL proxy
//	GET  /v1/services                     — service list + aggregate health
//	GET  /v1/services/:service/health     — per-service detail
//	GET  /v1/alerts/active                — firing alerts
//	GET  /v1/alerts                       — alert history
//	POST /v1/alerts/:id/ack               — acknowledge an alert
//	GET  /v1/audit/logs                   — application audit (Postgres + CH rollup)
//	POST /v1/webhook/alertmanager         — AlertManager receiver
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/rinco/services/observability-service/internal/alertmanager"
	"github.com/rinco/services/observability-service/internal/clickhouse"
	"github.com/rinco/services/observability-service/internal/jaeger"
	"github.com/rinco/services/observability-service/internal/loki"
	"github.com/rinco/services/observability-service/internal/prom"
)

// Server holds the dependencies every handler needs.
type Server struct {
	pool     *pgxpool.Pool
	loki     *loki.Client
	prom     *prom.Client
	jaeger   *jaeger.Client
	alertman *alertmanager.Receiver
	chWriter *clickhouse.Writer
}

// NewServer constructs the handler.Server.  chWriter may be nil when CH
// is not configured; audit writes still succeed, they just won't be
// replicated to ClickHouse.
func NewServer(pool *pgxpool.Pool, lokiURL, promURL, jaegerURL, tempoURL, fanoutURL string,
	chWriter *clickhouse.Writer) *Server {
	amr := alertmanager.NewReceiver(pool, fanoutURL)
	return &Server{
		pool:     pool,
		loki:     loki.New(lokiURL),
		prom:     prom.New(promURL),
		jaeger:   jaeger.New(jaegerURL, tempoURL),
		alertman: amr,
		chWriter: chWriter,
	}
}

// =============================================================================
// Logs
// =============================================================================

// QueryLogs proxies a LogQL query to Loki.
func (s *Server) QueryLogs(c echo.Context) error {
	ctx := c.Request().Context()
	q := loki.Query{
		Service:  c.QueryParam("service"),
		Level:    c.QueryParam("level"),
		TenantID: c.QueryParam("tenant_id"),
		From:     c.QueryParam("from"),
		To:       c.QueryParam("to"),
		Pattern:  c.QueryParam("query"),
	}
	q.Limit = atoiDefault(c.QueryParam("limit"), 100)
	lines, err := s.loki.Query(ctx, q)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"count": len(lines),
		"logs":  lines,
	})
}

// AggregateLogs runs a LogQL aggregate query.
func (s *Server) AggregateLogs(c echo.Context) error {
	ctx := c.Request().Context()
	q := loki.Query{
		Service:  c.QueryParam("service"),
		TenantID: c.QueryParam("tenant_id"),
		From:     c.QueryParam("from"),
		To:       c.QueryParam("to"),
		GroupBy:  c.QueryParam("group_by"),
		LogQL:    c.QueryParam("query"),
	}
	q.Limit = atoiDefault(c.QueryParam("limit"), 100)
	body, err := s.loki.Aggregate(ctx, q)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	var out any
	_ = json.Unmarshal(body, &out)
	return c.JSON(http.StatusOK, out)
}

// =============================================================================
// Traces
// =============================================================================

// GetTrace returns the spans for a trace id.
func (s *Server) GetTrace(c echo.Context) error {
	ctx := c.Request().Context()
	traceID := c.Param("trace_id")
	if traceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "trace_id required"})
	}
	trace, err := s.jaeger.GetTrace(ctx, traceID)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, trace)
}

// SearchTraces searches traces.
func (s *Server) SearchTraces(c echo.Context) error {
	ctx := c.Request().Context()
	q := jaeger.SearchQuery{
		Service:     c.QueryParam("service"),
		Operation:   c.QueryParam("operation"),
		Tags:        c.QueryParam("tags"),
		MinDuration: c.QueryParam("min_duration"),
		MaxDuration: c.QueryParam("max_duration"),
		Lookback:    c.QueryParam("lookback"),
		Limit:       atoiDefault(c.QueryParam("limit"), 20),
	}
	traces, err := s.jaeger.Search(ctx, q)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"count":  len(traces),
		"traces": traces,
	})
}

// =============================================================================
// Metrics
// =============================================================================

// QueryMetrics runs an instant PromQL query.
func (s *Server) QueryMetrics(c echo.Context) error {
	ctx := c.Request().Context()
	q := c.QueryParam("query")
	if q == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query required"})
	}
	res, err := s.prom.Query(ctx, q)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"query":   q,
		"results": res,
	})
}

// RangeMetrics runs a range PromQL query.
func (s *Server) RangeMetrics(c echo.Context) error {
	ctx := c.Request().Context()
	q := c.QueryParam("query")
	if q == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query required"})
	}
	from := c.QueryParam("from")
	to := c.QueryParam("to")
	step := c.QueryParam("step")
	if from == "" || to == "" {
		now := time.Now()
		to = strconv.FormatInt(now.Unix(), 10)
		from = strconv.FormatInt(now.Add(-1*time.Hour).Unix(), 10)
	}
	if step == "" {
		step = "30s"
	}
	matrix, err := s.prom.QueryRange(ctx, q, from, to, step)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"query":  q,
		"from":   from,
		"to":     to,
		"step":   step,
		"result": matrix,
	})
}

// =============================================================================
// Services
// =============================================================================

// ListServices returns the aggregated service list with health.
func (s *Server) ListServices(c echo.Context) error {
	ctx := c.Request().Context()
	window := c.QueryParam("window")
	if window == "" {
		window = "5m"
	}
	services, err := s.prom.ListServices(ctx)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	out := make([]prom.HealthSummary, 0, len(services))
	overall := "healthy"
	for _, svc := range services {
		summary := s.prom.Summarise(ctx, svc, window)
		// Add alert count for the service.
		summary.Status = composeStatus(summary.Status, s.alertCountForService(ctx, svc))
		out = append(out, summary)
		if summary.Status != "healthy" {
			overall = "degraded"
		}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"overall":    overall,
		"services":   out,
		"checked_at": time.Now(),
	})
}

// ServiceHealth returns the detailed health for a single service.
func (s *Server) ServiceHealth(c echo.Context) error {
	ctx := c.Request().Context()
	svc := c.Param("service")
	window := c.QueryParam("window")
	if window == "" {
		window = "5m"
	}
	summary := s.prom.Summarise(ctx, svc, window)
	summary.Status = composeStatus(summary.Status, s.alertCountForService(ctx, svc))
	return c.JSON(http.StatusOK, summary)
}

func (s *Server) alertCountForService(ctx context.Context, service string) int {
	var n int
	row := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM observability.alerts WHERE service = $1 AND status = 'firing'`, service)
	_ = row.Scan(&n)
	return n
}

func composeStatus(base string, alerts int) string {
	if alerts > 0 {
		if alerts >= 5 {
			return "critical"
		}
		return "degraded"
	}
	if base == "" {
		return "unknown"
	}
	return base
}

// =============================================================================
// Alerts
// =============================================================================

// ActiveAlerts returns firing alerts.
func (s *Server) ActiveAlerts(c echo.Context) error {
	ctx := c.Request().Context()
	rows, err := s.pool.Query(ctx, `
		SELECT id, fingerprint, status, severity, labels, annotations, service, title, message,
		       fired_at, resolved_at, ack_by, ack_at
		FROM observability.alerts
		WHERE status IN ('firing','ack')
		ORDER BY fired_at DESC
		LIMIT 500`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()
	out := []alertView{}
	for rows.Next() {
		var a alertView
		var labelsRaw, annRaw []byte
		var ackBy, ackAt *string
		var resolvedAt *time.Time
		if err := rows.Scan(&a.ID, &a.Fingerprint, &a.Status, &a.Severity,
			&labelsRaw, &annRaw, &a.Service, &a.Title, &a.Message,
			&a.FiredAt, &resolvedAt, &ackBy, &ackAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		_ = json.Unmarshal(labelsRaw, &a.Labels)
		_ = json.Unmarshal(annRaw, &a.Annotations)
		a.ResolvedAt = resolvedAt
		a.AckBy = ackBy
		a.AckAt = ackAt
		out = append(out, a)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"count":   len(out),
		"alerts":  out,
		"checked": time.Now(),
	})
}

type alertView struct {
	ID          string            `json:"id"`
	Fingerprint string            `json:"fingerprint"`
	Status      string            `json:"status"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Service     *string           `json:"service,omitempty"`
	Title       string            `json:"title"`
	Message     *string           `json:"message,omitempty"`
	FiredAt     time.Time         `json:"fired_at"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
	AckBy       *string           `json:"ack_by,omitempty"`
	AckAt       *string           `json:"ack_at,omitempty"`
}

// ListAlerts returns historical alerts within a window.
func (s *Server) ListAlerts(c echo.Context) error {
	ctx := c.Request().Context()
	from, to := parseWindow(c.QueryParam("from"), c.QueryParam("to"))
	service := c.QueryParam("service")
	limit := atoiDefault(c.QueryParam("limit"), 200)

	rows, err := s.pool.Query(ctx, `
		SELECT id, fingerprint, status, severity, labels, annotations, service, title, message,
		       fired_at, resolved_at, ack_by, ack_at
		FROM observability.alerts
		WHERE fired_at >= $1 AND fired_at <= $2
		  AND ($3 = '' OR service = $3)
		ORDER BY fired_at DESC
		LIMIT $4`, from, to, service, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()
	out := []alertView{}
	for rows.Next() {
		var a alertView
		var labelsRaw, annRaw []byte
		var ackBy, ackAt *string
		var resolvedAt *time.Time
		if err := rows.Scan(&a.ID, &a.Fingerprint, &a.Status, &a.Severity,
			&labelsRaw, &annRaw, &a.Service, &a.Title, &a.Message,
			&a.FiredAt, &resolvedAt, &ackBy, &ackAt); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		_ = json.Unmarshal(labelsRaw, &a.Labels)
		_ = json.Unmarshal(annRaw, &a.Annotations)
		a.ResolvedAt = resolvedAt
		a.AckBy = ackBy
		a.AckAt = ackAt
		out = append(out, a)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"count":  len(out),
		"alerts": out,
	})
}

// AckAlert marks an alert as acknowledged.
func (s *Server) AckAlert(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if strings.TrimSpace(body.UserID) == "" {
		body.UserID = c.Request().Header.Get("X-User-ID")
	}
	if body.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id required"})
	}
	ok, err := s.alertman.Acknowledge(ctx, id, body.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "alert not found"})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"status":  "acknowledged",
		"id":      id,
		"by":      body.UserID,
		"ack_at":  time.Now(),
	})
}

// WebhookAlertManager exposes the AlertManager receiver as an Echo handler.
func (s *Server) WebhookAlertManager(c echo.Context) error {
	s.alertman.HandleWebhook(c.Response().Writer, c.Request().WithContext(c.Request().Context()))
	return nil
}

// =============================================================================
// Audit logs (Postgres + ClickHouse write)
// =============================================================================

// QueryAuditLogs queries application audit from Postgres and (best
// effort) ClickHouse rollups.
func (s *Server) QueryAuditLogs(c echo.Context) error {
	ctx := c.Request().Context()
	actor := c.QueryParam("actor")
	action := c.QueryParam("action")
	tenant := c.QueryParam("tenant_id")
	from, to := parseWindow(c.QueryParam("from"), c.QueryParam("to"))
	limit := atoiDefault(c.QueryParam("limit"), 200)

	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, actor_user_id, actor_ip, action, resource_type, resource_id, payload, ts
		FROM observability.audit_logs
		WHERE ($1 = '' OR actor_user_id::text = $1)
		  AND ($2 = '' OR action = $2)
		  AND ($3 = '' OR tenant_id::text = $3)
		  AND ts >= $4 AND ts <= $5
		ORDER BY ts DESC
		LIMIT $6`, actor, action, tenant, from, to, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()
	out := []auditView{}
	for rows.Next() {
		var a auditView
		var payloadRaw []byte
		if err := rows.Scan(&a.ID, &a.TenantID, &a.ActorUserID, &a.ActorIP,
			&a.Action, &a.ResourceType, &a.ResourceID, &payloadRaw, &a.TS); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		_ = json.Unmarshal(payloadRaw, &a.Payload)
		out = append(out, a)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"count": len(out),
		"rows":  out,
	})
}

type auditView struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	ActorUserID  *string        `json:"actor_user_id,omitempty"`
	ActorIP      *string        `json:"actor_ip,omitempty"`
	Action       string         `json:"action"`
	ResourceType *string        `json:"resource_type,omitempty"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
	TS           time.Time      `json:"ts"`
}

// WriteAudit is called by other services (and by the webhook handler)
// to write an audit record.  It writes to Postgres synchronously and
// to ClickHouse asynchronously (best-effort).
func (s *Server) WriteAudit(ctx context.Context, rec clickhouse.AuditRecord) error {
	if rec.TS.IsZero() {
		rec.TS = time.Now()
	}
	payload := rec.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	payloadJSON, _ := json.Marshal(payload)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO observability.audit_logs (tenant_id, actor_user_id, actor_ip, action, resource_type, resource_id, payload, ts)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,''), $4, NULLIF($5,''), NULLIF($6,''), $7::jsonb, $8)`,
		rec.TenantID, rec.ActorUserID, rec.ActorIP,
		rec.Action, rec.ResourceType, rec.ResourceID, payloadJSON, rec.TS)
	if err != nil {
		return fmt.Errorf("audit insert: %w", err)
	}
	if s.chWriter != nil {
		if err := s.chWriter.Insert(ctx, rec); err != nil {
			clickhouse.LogError(ctx, "audit insert", err)
		}
	}
	return nil
}

// WriteAuditFromRequest is a helper that pulls the tenant from headers.
func (s *Server) WriteAuditFromRequest(c echo.Context, action, resourceType, resourceID string, payload map[string]any) {
	rec := clickhouse.AuditRecord{
		TenantID:     c.Request().Header.Get("X-Tenant-ID"),
		ActorUserID:  c.Request().Header.Get("X-User-ID"),
		ActorIP:      c.RealIP(),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Payload:      payload,
	}
	if err := s.WriteAudit(c.Request().Context(), rec); err != nil {
		slog.Default().Error("audit write failed", slog.String("err", err.Error()))
	}
}

// =============================================================================
// Connect-RPC adapter
// =============================================================================

// ConnectRPC forwards the canonical Connect RPC paths to the same
// handler functions exposed on the REST API.
func (s *Server) ConnectRPC(e *echo.Echo) {
	g := e.Group("/internal/observability.v1.ObservabilityService")
	g.POST("/QueryLogs", s.QueryLogs)
	g.POST("/GetTrace", s.GetTrace)
	g.POST("/SearchTraces", s.SearchTraces)
	g.POST("/QueryMetrics", s.QueryMetrics)
	g.POST("/RangeMetrics", s.RangeMetrics)
	g.POST("/ListServices", s.ListServices)
	g.POST("/ServiceHealth", s.ServiceHealth)
	g.POST("/ActiveAlerts", s.ActiveAlerts)
	g.POST("/ListAlerts", s.ListAlerts)
	g.POST("/AckAlert", s.AckAlert)
	g.POST("/QueryAuditLogs", s.QueryAuditLogs)
	g.POST("/WebhookAlertManager", s.WebhookAlertManager)
}

// =============================================================================
// Helpers
// =============================================================================

// parseWindow resolves `from` and `to` query params; either may be an
// RFC3339 timestamp or an empty string ("now" / 1-hour-ago default).
func parseWindow(fromStr, toStr string) (time.Time, time.Time) {
	now := time.Now()
	if toStr == "" {
		toStr = now.Format(time.RFC3339)
	}
	if fromStr == "" {
		fromStr = now.Add(-1 * time.Hour).Format(time.RFC3339)
	}
	from, err1 := time.Parse(time.RFC3339, fromStr)
	to, err2 := time.Parse(time.RFC3339, toStr)
	if err1 != nil {
		from = now.Add(-1 * time.Hour)
	}
	if err2 != nil {
		to = now
	}
	return from, to
}

// atoiDefault is a tiny strconv alias.
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// pgErr returns true if the error is a no-rows error.
func pgErr(err error) bool { return err == pgx.ErrNoRows }
var _ = pgErr