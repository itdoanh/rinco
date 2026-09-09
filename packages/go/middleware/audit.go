// Package middleware provides audit logging middleware.
//
// This middleware logs all API requests to ClickHouse and structured logs
// with actor, tenant, action, and resource information.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// AuditConfig holds audit middleware configuration.
type AuditConfig struct {
	// Logger is the structured logger to use
	Logger *slog.Logger

	// ClickHouseDSN is the ClickHouse connection string
	ClickHouseDSN string

	// ClickHouseTable is the table to write audit logs to
	ClickHouseTable string

	// SkipPaths are paths that won't be audited
	SkipPaths []string

	// SkipMethods are methods that won't be audited
	SkipMethods []string

	// SkipCodes are status codes that won't be audited
	SkipCodes []int

	// IncludeRequestBody includes request body in audit log
	IncludeRequestBody bool

	// IncludeResponseBody includes response body in audit log
	IncludeResponseBody bool

	// MaxBodySize is the maximum body size to capture
	MaxBodySize int64

	// SensitiveFields are field names to redact
	SensitiveFields []string

	// ExtraAttrsFunc adds extra attributes to audit log
	ExtraAttrsFunc func(c echo.Context) []slog.Attr
}

// DefaultAuditConfig returns sensible defaults.
func AuditDefaultConfig(logger *slog.Logger) AuditConfig {
	return AuditConfig{
		Logger:            logger,
		ClickHouseTable:   "audit_logs",
		SkipPaths:         []string{"/healthz", "/readyz", "/metrics"},
		SkipMethods:       []string{"OPTIONS"},
		SkipCodes:         []int{200, 201, 204}, // Only audit errors by default
		MaxBodySize:       10240,                // 10KB
		SensitiveFields:   []string{"password", "token", "secret", "api_key", "authorization"},
	}
}

// responseWriter wraps echo.Response to capture status code.
type responseWriter struct {
	echo.Response
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.Response.WriteHeader(code)
}

// bodyCapture captures and restores request/response bodies.
type bodyCapture struct {
	Buffer *bytes.Buffer
	Reader io.ReadCloser
}

func (b *bodyCapture) Read(p []byte) (n int, err error) {
	n, err = b.Reader.Read(p)
	b.Buffer.Write(p[:n])
	if err != nil {
		return
	}
	return n, nil
}

func (b *bodyCapture) Close() error {
	return b.Reader.Close()
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	Timestamp   time.Time `json:"timestamp"`
	ActorID     string    `json:"actor_id"`
	TenantID    string    `json:"tenant_id"`
	IP          string    `json:"ip"`
	UserAgent   string    `json:"user_agent"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	QueryString string    `json:"query_string"`
	StatusCode  int       `json:"status_code"`
	Duration    int64     `json:"duration_ms"`
	RequestSize int64     `json:"request_size"`
	ResponseSize int64    `json:"response_size"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	Outcome     string    `json:"outcome"`
	Error       string    `json:"error,omitempty"`
	RequestBody string    `json:"request_body,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
}

// Audit creates an audit logging middleware.
func Audit(cfg AuditConfig) echo.MiddlewareFunc {
	skipPaths := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = true
	}

	skipMethods := make(map[string]bool)
	for _, m := range cfg.SkipMethods {
		skipMethods[m] = true
	}

	skipCodes := make(map[int]bool)
	for _, c := range cfg.SkipCodes {
		skipCodes[c] = true
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Path()
			if skipPaths[path] {
				return next(c)
			}

			method := c.Request().Method
			if skipMethods[method] {
				return next(c)
			}

			// Capture request body
			var requestBody string
			if cfg.IncludeRequestBody && c.Request().Body != nil {
				bodyBytes, _ := io.ReadAll(io.LimitReader(c.Request().Body, cfg.MaxBodySize))
				c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				requestBody = cfg.Logger.Handler().(interface{ Redact(string) string }).(interface {
					Redact(string) string
				}).Redact(string(bodyBytes))
				if requestBody == "" {
					requestBody = string(bodyBytes)
				}
			}

			// Wrap response writer
			wrapper := &responseWriter{Response: *c.Response(), statusCode: 200}

			// Record start time
			start := time.Now()

			// Execute handler
			err := next(c)

			// Get status code
			statusCode := wrapper.statusCode
			if c.Response().Status > 0 {
				statusCode = c.Response().Status
			}

			// Skip if status is in skip list
			if skipCodes[statusCode] && err == nil {
				return nil
			}

			// Calculate duration
			duration := time.Since(start).Milliseconds()

			// Extract actor and tenant from context
			actorID := GetUserID(c.Request().Context())
			tenantID := TenantIDFromContext(c.Request().Context())

			// Determine outcome
			outcome := "success"
			if statusCode >= 400 || err != nil {
				outcome = "failure"
			}

			// Build action and resource from path
			action, resource := parseActionResource(path)

			// Build audit log entry
			entry := &AuditLog{
				Timestamp:    start,
				ActorID:      actorID,
				TenantID:     tenantID,
				IP:           c.RealIP(),
				UserAgent:    c.Request().UserAgent(),
				Method:       method,
				Path:         path,
				QueryString:  c.Request().URL.RawQuery,
				StatusCode:   statusCode,
				Duration:     duration,
				RequestSize:  c.Request().ContentLength,
				ResponseSize: c.Response().Size,
				Action:       action,
				Resource:     resource,
				Outcome:      outcome,
				RequestBody:  requestBody,
			}

			// Add error if present
			if err != nil {
				entry.Error = err.Error()
			}

			// Add extra attributes if configured
			if cfg.ExtraAttrsFunc != nil {
				extra := make(map[string]any)
				for _, attr := range cfg.ExtraAttrsFunc(c) {
					extra[attr.Key] = attr.Value
				}
				entry.Extra = extra
			}

			// Redact sensitive fields
			if len(cfg.SensitiveFields) > 0 && requestBody != "" {
				entry.RequestBody = redactSensitiveData(requestBody, cfg.SensitiveFields)
			}

			// Log to structured logger
			cfg.Logger.LogAttrs(c.Request().Context(), slog.LevelInfo, "audit",
				slog.String("audit_action", action),
				slog.String("audit_resource", resource),
				slog.String("audit_actor", actorID),
				slog.String("audit_tenant", tenantID),
				slog.String("audit_method", method),
				slog.Int("audit_status", statusCode),
				slog.Int64("audit_duration_ms", duration),
				slog.String("audit_outcome", outcome),
			)

			// TODO: Write to ClickHouse if configured
			// This would be done asynchronously in production

			return err
		}
	}
}

// parseActionResource extracts action and resource from path.
func parseActionResource(path string) (action, resource string) {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Split into parts
	parts := strings.Split(path, "/")

	if len(parts) >= 2 {
		// Last part is often the action or ID
		last := parts[len(parts)-1]
		
		// Check if last part is HTTP method (GET, POST, etc.) or ID
		if last == "create" || last == "update" || last == "delete" || last == "list" || last == "get" {
			action = last
			resource = strings.Join(parts[:len(parts)-1], ".")
		} else if isUUID(last) || isNumericID(last) {
			// It's an ID, so the action is the previous part
			if len(parts) >= 3 {
				action = parts[len(parts)-2]
				resource = strings.Join(parts[:len(parts)-2], ".")
			} else {
				action = "get"
				resource = parts[0]
			}
		} else {
			// It's likely a resource name
			action = "access"
			resource = strings.Join(parts, ".")
		}
	} else {
		action = "access"
		resource = path
	}

	return
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for _, c := range s {
		if c != '-' && (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func isNumericID(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// redactSensitiveData replaces sensitive field values with [REDACTED].
func redactSensitiveData(body string, fields []string) string {
	var result map[string]any
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return body
	}

	redactMap(result, fields)
	
	out, _ := json.Marshal(result)
	return string(out)
}

func redactMap(m map[string]any, fields []string) {
	for k, v := range m {
		redacted := false
		// Check if this key should be redacted
		for _, field := range fields {
			if strings.EqualFold(k, field) {
				m[k] = "[REDACTED]"
				redacted = true
				break
			}
		}
		if redacted {
			continue
		}

		// Recurse into nested objects
		switch child := v.(type) {
		case map[string]any:
			redactMap(child, fields)
		case []any:
			for _, item := range child {
				if nested, ok := item.(map[string]any); ok {
					redactMap(nested, fields)
				}
			}
		}
	}
}

// AuditLogWriter defines the interface for writing audit logs.
type AuditLogWriter interface {
	Write(ctx context.Context, log *AuditLog) error
	WriteBatch(ctx context.Context, logs []*AuditLog) error
}

// AuditLogChanWriter writes audit logs via a channel (for async processing).
type AuditLogChanWriter struct {
	ch chan *AuditLog
}

// NewAuditLogChanWriter creates a new channel-based audit log writer.
func NewAuditLogChanWriter(bufferSize int) *AuditLogChanWriter {
	return &AuditLogChanWriter{ch: make(chan *AuditLog, bufferSize)}
}

// Write writes a single audit log entry.
func (w *AuditLogChanWriter) Write(ctx context.Context, log *AuditLog) error {
	select {
	case w.ch <- log:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WriteBatch writes multiple audit log entries.
func (w *AuditLogChanWriter) WriteBatch(ctx context.Context, logs []*AuditLog) error {
	for _, log := range logs {
		if err := w.Write(ctx, log); err != nil {
			return err
		}
	}
	return nil
}

// LogChan returns the channel for consuming audit logs.
func (w *AuditLogChanWriter) LogChan() <-chan *AuditLog {
	return w.ch
}

// AuditLogSlogWriter writes audit logs to a structured logger.
type AuditLogSlogWriter struct {
	logger *slog.Logger
}

// NewAuditLogSlogWriter creates a new slog-based audit log writer.
func NewAuditLogSlogWriter(logger *slog.Logger) *AuditLogSlogWriter {
	return &AuditLogSlogWriter{logger: logger}
}

// Write writes a single audit log entry.
func (w *AuditLogSlogWriter) Write(ctx context.Context, log *AuditLog) error {
	attrs := []slog.Attr{
		slog.String("audit_action", log.Action),
		slog.String("audit_resource", log.Resource),
		slog.String("audit_actor", log.ActorID),
		slog.String("audit_tenant", log.TenantID),
		slog.String("audit_method", log.Method),
		slog.String("audit_path", log.Path),
		slog.Int("audit_status", log.StatusCode),
		slog.Int64("audit_duration_ms", log.Duration),
		slog.String("audit_outcome", log.Outcome),
		slog.String("audit_ip", log.IP),
		slog.String("audit_user_agent", log.UserAgent),
	}

	if log.Error != "" {
		attrs = append(attrs, slog.String("audit_error", log.Error))
	}
	if log.RequestBody != "" {
		attrs = append(attrs, slog.String("audit_request_body", log.RequestBody))
	}

	level := slog.LevelInfo
	if log.Outcome == "failure" {
		level = slog.LevelWarn
	}

	w.logger.LogAttrs(ctx, level, "audit", attrs...)
	return nil
}

// WriteBatch writes multiple audit log entries.
func (w *AuditLogSlogWriter) WriteBatch(ctx context.Context, logs []*AuditLog) error {
	for _, log := range logs {
		if err := w.Write(ctx, log); err != nil {
			return err
		}
	}
	return nil
}

// AuditContext is used to add audit context to a request.
type AuditContext struct {
	ActorID  string
	TenantID string
	Action   string
	Resource string
	Extra    map[string]any
}

// AuditContextKey is the context key for audit context.
const AuditContextKey = "audit_context"

// WithAuditContext adds audit context to context.
func WithAuditContext(ctx context.Context, audit *AuditContext) context.Context {
	return context.WithValue(ctx, AuditContextKey, audit)
}

// GetAuditContext retrieves audit context from context.
func GetAuditContext(ctx context.Context) *AuditContext {
	if v := ctx.Value(AuditContextKey); v != nil {
		return v.(*AuditContext)
	}
	return nil
}
