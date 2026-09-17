// Package middleware — RINCO WS-E audit middleware (rewritten for WS-E
// observability + security hardening milestone).
//
// Auto-records every state-changing HTTP request (POST, PUT, PATCH,
// DELETE) with actor, tenant, resource type/id and the request/response
// JSON (sensitive fields are redacted).  Writes are best-effort and
// never fail the request — even when ClickHouse or Postgres are
// unreachable the user request still completes.
//
// Two layers:
//
//  1. The hook (SetBeforeSnapshot) lets a service capture the "before"
//     snapshot of the resource.  Call SetBeforeSnapshot(c, jsonString)
//     before mutating.
//  2. The middleware itself captures the "after" snapshot from the
//     response body when the handler writes JSON.
//
// Records flow into:
//
//   - In-process channel -> background writer (non-blocking).
//   - Postgres observability.audit_logs (best-effort).
//   - ClickHouse observability.app_audit_logs (best-effort).
//
// Usage:
//
//	e.Use(middleware.Audit(middleware.AuditConfig{
//	    Logger:        logger,
//	    PGPool:        pgxPool,
//	    ClickHouseURL: os.Getenv("OBSERVABILITY_CLICKHOUSE_URL"),
//	    Service:       "auth-service",
//	}))
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// DefaultSensitiveFields is the list of body keys that are redacted
// before persisting.  Case-insensitive match.
var DefaultSensitiveFields = []string{
	"password",
	"passwd",
	"pwd",
	"secret",
	"token",
	"access_token",
	"refresh_token",
	"authorization",
	"auth",
	"api_key",
	"apikey",
	"session_id",
	"session",
	"cookie",
	"csrf_token",
	"private_key",
	"client_secret",
}

// =============================================================================
// Audit config + record
// =============================================================================

// AuditConfig bundles everything the audit middleware needs.
type AuditConfig struct {
	// Logger is the structured logger.  Required.
	Logger *slog.Logger
	// Service is the service name embedded in every audit record.
	Service string
	// Env is the deployment environment (development, staging, production).
	Env string
	// Version is the service version (free-form string).
	Version string

	// PGPool, when set, enables Postgres-backed audit persistence.
	PGPool *pgxpool.Pool

	// ClickHouseURL, when set, enables ClickHouse rollup persistence.
	ClickHouseURL string

	// ClickHouseTable overrides the default table name.
	ClickHouseTable string

	// CHBatchSize controls how many records are batched into a single
	// CH HTTP insert.  Defaults to 32.
	CHBatchSize int

	// SkipPaths are URL paths that won't be audited.
	SkipPaths []string

	// SkipMethods are HTTP methods that won't be audited.
	SkipMethods []string

	// IncludeReadMethods, when true, audits GET/HEAD in addition to
	// write methods.  Defaults to false.
	IncludeReadMethods bool

	// IncludeRequestBody, when true, captures the request body in
	// the audit record.  Default: true.
	IncludeRequestBody bool

	// IncludeResponseBody, when true, captures the response body in
	// the audit record.  Default: true (with a size limit).
	IncludeResponseBody bool

	// MaxBodySize bounds request/response bodies (bytes).  Defaults
	// to 16 KiB.  Anything larger is truncated and marked.
	MaxBodySize int64

	// SensitiveFields overrides DefaultSensitiveFields.
	SensitiveFields []string

	// BufferSize is the channel buffer for the background writer.
	BufferSize int

	// AsyncTimeout caps how long the background writer waits for
	// Postgres/ClickHouse on each flush.  Defaults to 5s.
	AsyncTimeout time.Duration

	// RedactFunc allows callers to plug a custom PII redactor.
	RedactFunc func(value any) any

	// ExtraAttrsFunc adds extra attributes to the structured log
	// line (not persisted in the audit row).
	ExtraAttrsFunc func(c echo.Context) []slog.Attr

	// Channel, when set, receives every audit record before it is
	// forwarded to the background writer.  Useful for tests and
	// in-process subscribers.  When nil a private channel is used.
	Channel chan<- *AuditRecord
}

// AuditRecord is the canonical row written to Postgres + ClickHouse.
type AuditRecord struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	ActorUserID  string         `json:"actor_user_id"`
	ActorIP      string         `json:"actor_ip"`
	UserAgent    string         `json:"user_agent"`
	Service      string         `json:"service"`
	Env          string         `json:"env"`
	Version      string         `json:"version"`
	Method       string         `json:"method"`
	Path         string         `json:"path"`
	Route        string         `json:"route"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	StatusCode   int            `json:"status_code"`
	Outcome      string         `json:"outcome"`
	DurationMS   int64          `json:"duration_ms"`
	RequestBody  string         `json:"request_body,omitempty"`
	ResponseBody string         `json:"response_body,omitempty"`
	BeforeState  string         `json:"before_state,omitempty"`
	AfterState   string         `json:"after_state,omitempty"`
	RequestID    string         `json:"request_id,omitempty"`
	TraceID      string         `json:"trace_id,omitempty"`
	SpanID       string         `json:"span_id,omitempty"`
	Errors       []string       `json:"errors,omitempty"`
	Extra        map[string]any `json:"extra,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	CHPayload    string         `json:"-"`
}

// =============================================================================
// Snapshot hooks
// =============================================================================

const beforeSnapshotKey = "audit_before_snapshot"

// SetBeforeSnapshot stores a JSON-encoded representation of the
// resource in its pre-mutation state so the audit middleware can
// include it in the audit record.
func SetBeforeSnapshot(c echo.Context, jsonPayload string) {
	c.Set(beforeSnapshotKey, jsonPayload)
}

// GetBeforeSnapshot returns the value previously stored by
// SetBeforeSnapshot (or empty string if none).
func GetBeforeSnapshot(c echo.Context) string {
	v, _ := c.Get(beforeSnapshotKey).(string)
	return v
}

// =============================================================================
// Middleware
// =============================================================================

// Audit is the Echo middleware.  It returns immediately even when
// the background writer is full (records are dropped on overflow
// rather than blocking the request).
func Audit(cfg AuditConfig) echo.MiddlewareFunc {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Service == "" {
		cfg.Service = "unknown"
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1024
	}
	if cfg.AsyncTimeout <= 0 {
		cfg.AsyncTimeout = 5 * time.Second
	}
	if cfg.CHBatchSize <= 0 {
		cfg.CHBatchSize = 32
	}
	if cfg.MaxBodySize <= 0 {
		cfg.MaxBodySize = 16 * 1024
	}
	if cfg.ClickHouseTable == "" {
		cfg.ClickHouseTable = "app_audit_logs"
	}
	if len(cfg.SensitiveFields) == 0 {
		cfg.SensitiveFields = DefaultSensitiveFields
	}

	skipPaths := map[string]bool{
		"/healthz": true, "/readyz": true, "/metrics": true,
	}
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = true
	}

	skipMethods := map[string]bool{
		http.MethodGet: true, http.MethodHead: true, http.MethodOptions: true,
	}
	if cfg.IncludeReadMethods {
		skipMethods = map[string]bool{}
	}
	for _, m := range cfg.SkipMethods {
		skipMethods[strings.ToUpper(m)] = true
	}

	w := newAuditWriter(cfg)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := strings.ToUpper(c.Request().Method)
			path := c.Request().URL.Path

			if skipMethods[method] {
				return next(c)
			}
			if skipPaths[path] {
				return next(c)
			}

			// ---- capture request body ----
			var reqBody string
			if cfg.IncludeRequestBody && c.Request().Body != nil {
				limited := io.LimitReader(c.Request().Body, cfg.MaxBodySize)
				bodyBytes, _ := io.ReadAll(limited)
				_ = c.Request().Body.Close()
				c.Request().Body = io.NopCloser(bytes.NewReader(bodyBytes))
				reqBody = string(bodyBytes)
				if int64(len(reqBody)) == cfg.MaxBodySize {
					reqBody += "...[truncated]"
				}
				reqBody = redactJSONString(reqBody, cfg.SensitiveFields, cfg.RedactFunc)
			}

			// ---- wrap response to capture status + body ----
			rw := &auditResponseWriter{Response: c.Response(), buf: &bytes.Buffer{}, cfg: &cfg}

			start := time.Now()
			err := next(c)
			duration := time.Since(start)

			statusCode := rw.statusCode()
			if err != nil {
				statusCode = httpErrorStatus(err)
			}

			// ---- build record ----
			outcome := "success"
			if statusCode >= 400 || err != nil {
				outcome = "failure"
			}

			respBody := ""
			if cfg.IncludeResponseBody && rw.buf != nil && rw.buf.Len() > 0 {
				raw := rw.buf.String()
				if int64(len(raw)) > cfg.MaxBodySize {
					raw = raw[:cfg.MaxBodySize] + "...[truncated]"
				}
				respBody = redactJSONString(raw, cfg.SensitiveFields, cfg.RedactFunc)
			}

			before := GetBeforeSnapshot(c)

			route := c.Path()
			if route == "" {
				route = path
			}

			rec := &AuditRecord{
				ID:           uuid.NewString(),
				TenantID:     headerOr(c, "X-Tenant-ID", ""),
				ActorUserID:  headerOr(c, "X-User-ID", ""),
				ActorIP:      c.RealIP(),
				UserAgent:    c.Request().UserAgent(),
				Service:      cfg.Service,
				Env:          cfg.Env,
				Version:      cfg.Version,
				Method:       method,
				Path:         path,
				Route:        route,
				Action:       deriveAction(method, path, route),
				ResourceType: deriveResourceType(path, route),
				ResourceID:   deriveResourceID(c),
				StatusCode:   statusCode,
				Outcome:      outcome,
				DurationMS:   duration.Milliseconds(),
				RequestBody:  reqBody,
				ResponseBody: respBody,
				BeforeState:  before,
				AfterState:   respBody,
				RequestID:    headerOr(c, "X-Request-ID", ""),
				TraceID:      headerOr(c, "X-Trace-ID", ""),
				SpanID:       headerOr(c, "X-Span-ID", ""),
				CreatedAt:    time.Now(),
			}
			if err != nil {
				rec.Errors = append(rec.Errors, err.Error())
			}

			rec.CHPayload = chJSONEachRow(rec)

			if cfg.Channel != nil {
				select {
				case cfg.Channel <- rec:
				default:
					// drop if subscriber is slow
				}
			}
			w.enqueue(rec)

			logAttrs := []slog.Attr{
				slog.String("audit_id", rec.ID),
				slog.String("tenant_id", rec.TenantID),
				slog.String("actor_id", rec.ActorUserID),
				slog.String("method", rec.Method),
				slog.String("path", rec.Path),
				slog.String("route", rec.Route),
				slog.String("action", rec.Action),
				slog.String("resource_type", rec.ResourceType),
				slog.String("resource_id", rec.ResourceID),
				slog.Int("status", rec.StatusCode),
				slog.String("outcome", rec.Outcome),
				slog.Int64("duration_ms", rec.DurationMS),
				slog.Bool("has_before", before != ""),
				slog.Bool("has_request_body", reqBody != ""),
				slog.Bool("has_response_body", respBody != ""),
			}
			if cfg.ExtraAttrsFunc != nil {
				logAttrs = append(logAttrs, cfg.ExtraAttrsFunc(c)...)
			}
			cfg.Logger.LogAttrs(c.Request().Context(), slog.LevelInfo, "audit", logAttrs...)

			return err
		}
	}
}

// =============================================================================
// Response body capture
// =============================================================================

type auditResponseWriter struct {
	*echo.Response
	buf  *bytes.Buffer
	cfg  *AuditConfig
	code int
}

func (w *auditResponseWriter) WriteHeader(code int) {
	w.code = code
	w.Response.WriteHeader(code)
}

func (w *auditResponseWriter) Write(b []byte) (int, error) {
	if w.code == 0 {
		w.code = http.StatusOK
	}
	w.buf.Write(b)
	return w.Response.Write(b)
}

func (w *auditResponseWriter) statusCode() int {
	if w.code == 0 {
		return w.Response.Status
	}
	return w.code
}

// =============================================================================
// Background writer (Postgres + ClickHouse)
// =============================================================================

type auditWriter struct {
	cfg      AuditConfig
	ch       chan *AuditRecord
	dropped  atomic.Uint64
	wg       sync.WaitGroup
	stopOnce sync.Once
	stopCh   chan struct{}
	closeCh  chan struct{}
	chClient *chHTTPClient
}

func newAuditWriter(cfg AuditConfig) *auditWriter {
	w := &auditWriter{
		cfg:     cfg,
		ch:      make(chan *AuditRecord, cfg.BufferSize),
		stopCh:  make(chan struct{}),
		closeCh: make(chan struct{}),
	}
	if cfg.ClickHouseURL != "" {
		w.chClient = newCHHTTPClient(cfg.ClickHouseURL, cfg.AsyncTimeout)
	}
	w.wg.Add(1)
	go w.loop()
	return w
}

func (w *auditWriter) enqueue(rec *AuditRecord) {
	select {
	case w.ch <- rec:
	default:
		w.dropped.Add(1)
		w.cfg.Logger.Warn("audit buffer full, record dropped",
			slog.String("audit_id", rec.ID),
			slog.Uint64("dropped_total", w.dropped.Load()),
		)
	}
}

func (w *auditWriter) loop() {
	defer w.wg.Done()
	batch := make([]*AuditRecord, 0, w.cfg.CHBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		w.persist(batch)
		batch = batch[:0]
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case rec, ok := <-w.ch:
			if !ok {
				flush()
				close(w.closeCh)
				return
			}
			batch = append(batch, rec)
			if len(batch) >= w.cfg.CHBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-w.stopCh:
			for {
				select {
				case rec, ok := <-w.ch:
					if !ok {
						flush()
						close(w.closeCh)
						return
					}
					batch = append(batch, rec)
				default:
					flush()
					close(w.closeCh)
					return
				}
			}
		}
	}
}

func (w *auditWriter) persist(batch []*AuditRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), w.cfg.AsyncTimeout)
	defer cancel()

	if w.cfg.PGPool != nil {
		w.writePostgres(ctx, batch)
	}
	if w.chClient != nil {
		w.writeClickHouse(ctx, batch)
	}
}

func (w *auditWriter) writePostgres(ctx context.Context, batch []*AuditRecord) {
	tx, err := w.cfg.PGPool.Begin(ctx)
	if err != nil {
		w.cfg.Logger.Warn("audit pg begin failed", slog.String("err", err.Error()))
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const stmt = `
INSERT INTO observability.audit_logs (
	id, tenant_id, actor_user_id, actor_ip, action, resource_type, resource_id, payload, ts
) VALUES (
	$1, NULLIF($2,'')::UUID, NULLIF($3,'')::UUID, $4, $5, $6, $7, $8::jsonb, $9
) ON CONFLICT (id) DO NOTHING`
	for _, r := range batch {
		pl := map[string]any{
			"service":       r.Service,
			"env":           r.Env,
			"version":       r.Version,
			"method":        r.Method,
			"path":          r.Path,
			"route":         r.Route,
			"status_code":   r.StatusCode,
			"outcome":       r.Outcome,
			"duration_ms":   r.DurationMS,
			"request_body":  r.RequestBody,
			"response_body": r.ResponseBody,
			"before_state":  r.BeforeState,
			"after_state":   r.AfterState,
			"request_id":    r.RequestID,
			"trace_id":      r.TraceID,
			"span_id":       r.SpanID,
			"user_agent":    r.UserAgent,
			"errors":        r.Errors,
		}
		buf, _ := json.Marshal(pl)
		_, err := tx.Exec(ctx, stmt,
			r.ID,
			r.TenantID,
			r.ActorUserID,
			r.ActorIP,
			r.Action,
			nullString(r.ResourceType),
			nullString(r.ResourceID),
			string(buf),
			r.CreatedAt,
		)
		if err != nil {
			w.cfg.Logger.Warn("audit pg insert failed",
				slog.String("audit_id", r.ID), slog.String("err", err.Error()))
		}
	}
	if err := tx.Commit(ctx); err != nil {
		w.cfg.Logger.Warn("audit pg commit failed", slog.String("err", err.Error()))
	}
}

func (w *auditWriter) writeClickHouse(ctx context.Context, batch []*AuditRecord) {
	lines := make([]string, 0, len(batch))
	for _, r := range batch {
		lines = append(lines, r.CHPayload)
	}
	if err := w.chClient.InsertBatch(ctx, w.cfg.ClickHouseTable, lines); err != nil {
		w.cfg.Logger.Warn("audit clickhouse insert failed",
			slog.Int("batch_size", len(batch)),
			slog.String("err", err.Error()))
	}
}

// Close drains the in-flight queue and waits for the background
// writer to exit.  Safe to call multiple times.
func (w *auditWriter) Close() error {
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
	<-w.closeCh
	return nil
}

// =============================================================================
// ClickHouse HTTP client (small, no external deps)
// =============================================================================

type chHTTPClient struct {
	baseURL string
	client  *http.Client
	db      string
	timeout time.Duration
}

func newCHHTTPClient(baseURL string, timeout time.Duration) *chHTTPClient {
	return &chHTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
		db:      "observability",
		timeout: timeout,
	}
}

func (c *chHTTPClient) InsertBatch(ctx context.Context, table string, lines []string) error {
	if c == nil || c.baseURL == "" || len(lines) == 0 {
		return nil
	}
	body := strings.Join(lines, "\n")
	q := url.Values{}
	q.Set("database", c.db)
	q.Set("query", "INSERT INTO "+table+" FORMAT JSONEachRow")
	endpoint := c.baseURL + "/?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("clickhouse HTTP %d: %s", resp.StatusCode, string(buf))
	}
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

func headerOr(c echo.Context, key, fallback string) string {
	v := strings.TrimSpace(c.Request().Header.Get(key))
	if v == "" {
		return fallback
	}
	return v
}

func nullString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func httpErrorStatus(err error) int {
	if err == nil {
		return 0
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		return he.Code
	}
	return http.StatusInternalServerError
}

func deriveAction(method, path, route string) string {
	method = strings.ToUpper(method)
	switch method {
	case http.MethodPost:
		if strings.HasSuffix(path, "/login") {
			return "auth.login"
		}
		if strings.HasSuffix(path, "/logout") {
			return "auth.logout"
		}
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	}
	return strings.ToLower(method)
}

func deriveResourceType(path, route string) string {
	src := route
	if src == "" || src == "/v1" || src == "/" {
		src = path
	}
	src = strings.TrimPrefix(src, "/v1/")
	src = strings.TrimPrefix(src, "/api/")
	src = strings.TrimPrefix(src, "/")
	if idx := strings.Index(src, "/"); idx > 0 {
		src = src[:idx]
	}
	if idx := strings.Index(src, "?"); idx > 0 {
		src = src[:idx]
	}
	if src == "" {
		return "unknown"
	}
	return src
}

func deriveResourceID(c echo.Context) string {
	if id := c.Param("id"); id != "" {
		return id
	}
	for _, name := range []string{"resource_id", "uuid", "tenant_id", "user_id", "lead_id", "deal_id"} {
		if id := c.Param(name); id != "" {
			return id
		}
	}
	return ""
}

// redactJSONString tries to parse the body as JSON, walks the tree
// and replaces any field whose name (case-insensitive) matches a
// sensitive name with "[REDACTED]".
func redactJSONString(body string, fields []string, redactor func(any) any) string {
	if body == "" {
		return body
	}
	if redactor != nil {
		return stringify(redactor(parseJSONLoose(body)))
	}
	var m any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return body
	}
	m = redactValue(m, fields)
	return stringify(m)
}

func parseJSONLoose(body string) any {
	var m any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return body
	}
	return m
}

func stringify(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func redactValue(v any, fields []string) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if isSensitive(k, fields) {
				t[k] = "[REDACTED]"
				continue
			}
			t[k] = redactValue(val, fields)
		}
		return t
	case []any:
		for i, item := range t {
			t[i] = redactValue(item, fields)
		}
		return t
	default:
		return v
	}
}

// isSensitive returns true when name (case-insensitive) matches any
// of the configured sensitive field names.
func isSensitive(name string, fields []string) bool {
	lower := strings.ToLower(name)
	for _, f := range fields {
		if strings.EqualFold(f, name) || strings.Contains(lower, strings.ToLower(f)) {
			return true
		}
	}
	return false
}

// chJSONEachRow encodes the audit record as a one-line JSON object
// compatible with ClickHouse's JSONEachRow format.
func chJSONEachRow(r *AuditRecord) string {
	out := struct {
		ID           string `json:"id"`
		TenantID     string `json:"tenant_id"`
		ActorUserID  string `json:"actor_user_id"`
		ActorIP      string `json:"actor_ip"`
		Action       string `json:"action"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		Method       string `json:"method"`
		Path         string `json:"path"`
		StatusCode   int    `json:"status_code"`
		Outcome      string `json:"outcome"`
		Payload      string `json:"payload"`
		TS           string `json:"ts"`
	}{
		ID:           r.ID,
		TenantID:     r.TenantID,
		ActorUserID:  r.ActorUserID,
		ActorIP:      r.ActorIP,
		Action:       r.Action,
		ResourceType: r.ResourceType,
		ResourceID:   r.ResourceID,
		Method:       r.Method,
		Path:         r.Path,
		StatusCode:   r.StatusCode,
		Outcome:      r.Outcome,
		Payload:      chPayload(r),
		TS:           r.CreatedAt.UTC().Format("2006-01-02 15:04:05.000"),
	}
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}

func chPayload(r *AuditRecord) string {
	pl := map[string]any{
		"service":       r.Service,
		"env":           r.Env,
		"version":       r.Version,
		"route":         r.Route,
		"duration_ms":   r.DurationMS,
		"request_body":  r.RequestBody,
		"response_body": r.ResponseBody,
		"before_state":  r.BeforeState,
		"after_state":   r.AfterState,
		"request_id":    r.RequestID,
		"trace_id":      r.TraceID,
		"span_id":       r.SpanID,
		"user_agent":    r.UserAgent,
		"errors":        r.Errors,
	}
	b, _ := json.Marshal(pl)
	return string(b)
}

// AuditDefaultConfig keeps backwards-compatible defaults for callers
// that only want slog output.
func AuditDefaultConfig() AuditConfig {
	return AuditConfig{
		Service:             "unknown",
		BufferSize:          1024,
		AsyncTimeout:        5 * time.Second,
		CHBatchSize:         32,
		MaxBodySize:         16 * 1024,
		ClickHouseTable:     "app_audit_logs",
		SensitiveFields:     DefaultSensitiveFields,
		IncludeRequestBody:  true,
		IncludeResponseBody: true,
		Logger:              slog.Default(),
	}
}

// SetBeforeSnapshotAsJSON marshals v to JSON and stores it as the
// before-snapshot.  Errors are swallowed (best-effort).
func SetBeforeSnapshotAsJSON(c echo.Context, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	SetBeforeSnapshot(c, string(b))
}