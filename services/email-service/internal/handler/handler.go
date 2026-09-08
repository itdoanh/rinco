// Package handler provides HTTP and Connect-RPC handlers for the email
// service.  The handler surfaces:
//
//   - Email send / batch send (delegating to the configured driver).
//   - Template CRUD with render / preview endpoints.
//   - Delivery logs filtered by tenant, status and date range.
//   - Aggregate stats (counts by status, by driver, by day).
//   - Provider webhook ingest (bounce / complaint / delivery).
//   - Tracking pixel (1x1 GIF) and click redirector endpoints.
//
// All endpoints are multi-tenant: requests must include an X-Tenant-ID
// header (enforced by the TenantMW middleware).  Row-level security is
// applied at the database level via the policies embedded in the
// migrations.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"github.com/rinco/services/email-service/internal/driver"
	"github.com/rinco/services/email-service/internal/platform"
	"github.com/rinco/services/email-service/internal/render"
	"github.com/rinco/services/email-service/internal/tracking"
)

// Server wires together the dependencies every handler needs.
type Server struct {
	pool       *pgxpool.Pool
	rdb        *redis.Client
	nats       *nats.Conn
	driver     driver.Driver
	cfg        *platform.Config
	renderer   func(string) (*render.Engine, error)
}

// NewServer builds the handler.Server.
func NewServer(pool *pgxpool.Pool, rdb *redis.Client, nc *nats.Conn, drv driver.Driver, cfg *platform.Config) *Server {
	return &Server{
		pool: pool, rdb: rdb, nats: nc, driver: drv, cfg: cfg,
		renderer: func(body string) (*render.Engine, error) { return render.NewEngine(body) },
	}
}

// Pool returns the underlying connection pool (used by the background worker).
func (s *Server) Pool() *pgxpool.Pool { return s.pool }

// Driver returns the configured email driver (used by the background worker).
func (s *Server) Driver() driver.Driver { return s.driver }

// DefaultFrom returns the configured default From address.
func (s *Server) DefaultFrom() string { return s.cfg.FromDefault }

// =============================================================================
// Context helpers
// =============================================================================

type ctxKey string

func (s *Server) tenantFromCtx(c echo.Context) (string, string, bool) {
	t, _ := c.Get("tenant_id").(string)
	u, _ := c.Get("user_id").(string)
	a, _ := c.Get("is_admin").(bool)
	return t, u, a
}

func (s *Server) setRLS(ctx context.Context, tenantID, userID string, isAdmin bool) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id missing")
	}
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("invalid tenant_id: %w", err)
	}
	adminVal := "false"
	if isAdmin {
		adminVal = "true"
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		return err
	}
	if userID != "" {
		if _, err := uuid.Parse(userID); err != nil {
			return fmt.Errorf("invalid user_id: %w", err)
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.is_admin', $1, true)", adminVal); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// =============================================================================
// Common types
// =============================================================================

type errResp struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

type listResp struct {
	Data     any   `json:"data"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PerPage  int   `json:"per_page"`
	HasMore  bool  `json:"has_more"`
}

func getPagination(c echo.Context) (page, perPage, offset int) {
	page, _ = strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ = strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 200 {
		perPage = 25
	}
	offset = (page - 1) * perPage
	return
}

func (s *Server) json(c echo.Context, status int, data any) error { return c.JSON(status, data) }

func (s *Server) errorResp(c echo.Context, status int, msg string, err error) error {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return c.JSON(status, errResp{Error: msg, Details: details})
}

// =============================================================================
// Email send / batch
// =============================================================================

type sendReq struct {
	To          []string               `json:"to"`
	Cc          []string               `json:"cc,omitempty"`
	Bcc         []string               `json:"bcc,omitempty"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	BodyType    string                 `json:"body_type"` // text | html | markdown
	TemplateID  *string                `json:"template_id,omitempty"`
	TemplateData map[string]any        `json:"template_data,omitempty"`
	Attachments []attachmentReq        `json:"attachments,omitempty"`
	Priority    string                 `json:"priority"` // high | normal | low
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	From        string                 `json:"from,omitempty"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

type attachmentReq struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	DataBase64  string `json:"data_base64"`
}

// SendEmail is the canonical HTTP entry-point for a transactional send.
func (s *Server) SendEmail(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	var req sendReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if len(req.To) == 0 {
		return s.errorResp(c, http.StatusBadRequest, "to is required", nil)
	}
	if req.BodyType == "" {
		req.BodyType = "text"
	}
	msgID := uuid.NewString()
	body, err := s.resolveBody(ctx, req)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "template render failed", err)
	}
	// Apply tracking
	pixelURL := fmt.Sprintf("%s/e/%s.gif", strings.TrimRight(s.cfg.WebBaseURL, "/"), msgID)
	redirectBase := fmt.Sprintf("%s/c/%s", strings.TrimRight(s.cfg.WebBaseURL, "/"), msgID)
	if req.BodyType == "html" || req.BodyType == "markdown" {
		body = render.RewriteClickLinks(body, redirectBase)
		body = render.InjectTrackingPixel(body, pixelURL)
	}
	priority := req.Priority
	if priority == "" {
		priority = "normal"
	}
	driverMsgID, sentAt, drvName, sendErr := s.deliver(ctx, msgID, req, body)
	status := "queued"
	if sendErr == nil {
		status = "sent"
	} else if !req.ScheduledAt.IsZero() {
		status = "queued"
	} else {
		status = "failed"
	}
	scheduled := req.ScheduledAt
	if scheduled != nil && scheduled.IsZero() {
		scheduled = nil
	}
	headers := req.Headers
	if headers == nil {
		headers = map[string]string{}
	}
	unsubURL := fmt.Sprintf("%s/u/%s", strings.TrimRight(s.cfg.WebBaseURL, "/"), msgID)
	headers["List-Unsubscribe"] = render.BuildUnsubscribeHeader(unsubURL, "")
	headers["List-Unsubscribe-Post"] = "List-Unsubscribe=One-Click"

	logID, err := s.recordLog(ctx, logInput{
		MsgID: msgID, TenantID: tenantID, TemplateID: req.TemplateID,
		From: firstNonEmpty(req.From, s.cfg.FromDefault), To: req.To, Cc: req.Cc, Bcc: req.Bcc,
		Subject: req.Subject, BodyRendered: body, Status: status, Driver: drvName, ProviderID: driverMsgID,
		Priority: priority, SentAt: sentAt, ScheduledAt: scheduled, LastError: errMsg(sendErr), Tags: req.Tags,
	})
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "log failed", err)
	}
	// Push to NATS for downstream listeners
	s.publishNATS(ctx, msgID, status, driverMsgID, drvName)
	return s.json(c, http.StatusAccepted, map[string]any{
		"id": logID, "msg_id": msgID, "status": status, "driver": drvName, "driver_msg_id": driverMsgID, "sent_at": sentAt,
	})
}

// BatchSend fans out to a list of recipients. Each recipient may carry its
// own subject/body overrides (via the per-item `override` block).
func (s *Server) BatchSend(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	var batch struct {
		Default sendReq            `json:"default"`
		Items   []sendReq          `json:"items"`
	}
	if err := c.Bind(&batch); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid batch", err)
	}
	if len(batch.Items) == 0 {
		return s.errorResp(c, http.StatusBadRequest, "items required", nil)
	}
	results := make([]map[string]any, 0, len(batch.Items))
	for _, item := range batch.Items {
		merged := batch.Default
		if merged.To == nil {
			merged.To = item.To
		}
		if merged.Cc == nil {
			merged.Cc = item.Cc
		}
		if merged.Bcc == nil {
			merged.Bcc = item.Bcc
		}
		if merged.Subject == "" {
			merged.Subject = item.Subject
		}
		if merged.Body == "" {
			merged.Body = item.Body
		}
		if merged.BodyType == "" {
			merged.BodyType = item.BodyType
		}
		if merged.Priority == "" {
			merged.Priority = item.Priority
		}
		if merged.From == "" {
			merged.From = item.From
		}
		msgID := uuid.NewString()
		body, _ := s.resolveBody(ctx, merged)
		driverMsgID, sentAt, drvName, sendErr := s.deliver(ctx, msgID, merged, body)
		status := "sent"
		if sendErr != nil {
			status = "failed"
		}
		_, _ = s.recordLog(ctx, logInput{
			MsgID: msgID, TenantID: tenantID, From: firstNonEmpty(merged.From, s.cfg.FromDefault), To: merged.To, Cc: merged.Cc, Bcc: merged.Bcc,
			Subject: merged.Subject, BodyRendered: body, Status: status, Driver: drvName, ProviderID: driverMsgID,
			Priority: firstNonEmpty(merged.Priority, "normal"), SentAt: sentAt, LastError: errMsg(sendErr),
		})
		results = append(results, map[string]any{"msg_id": msgID, "status": status, "error": errMsg(sendErr)})
	}
	return s.json(c, http.StatusAccepted, map[string]any{"count": len(results), "results": results})
}

// resolveBody looks up a template (if any), renders it with the supplied
// vars, and falls back to the literal body.  Markdown is rendered to HTML.
func (s *Server) resolveBody(ctx context.Context, req sendReq) (string, error) {
	body := req.Body
	bodyType := req.BodyType
	if bodyType == "" {
		bodyType = "text"
	}
	if req.TemplateID != nil && *req.TemplateID != "" {
		var tplBody, tplType string
		var varsBytes []byte
		err := s.pool.QueryRow(ctx, `SELECT body, body_type, COALESCE(vars, '{}'::jsonb) FROM email.email_templates WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
			*req.TemplateID, currentTenant(ctx)).Scan(&tplBody, &tplType, &varsBytes)
		if err != nil {
			return "", fmt.Errorf("template: %w", err)
		}
		body = tplBody
		bodyType = tplType
	}
	if bodyType == "markdown" {
		body = renderMarkdown(body)
	}
	if req.TemplateData != nil && len(req.TemplateData) > 0 {
		eng, err := s.renderer(body)
		if err != nil {
			return "", err
		}
		rendered, err := eng.Render(req.TemplateData)
		if err != nil {
			return "", err
		}
		body = rendered
	}
	return body, nil
}

// renderMarkdown calls the renderer package to convert Markdown to HTML.
func renderMarkdown(s string) string {
	eng, _ := render.NewEngine(s)
	out, _ := eng.Render(map[string]any{})
	return out
}

// deliver performs the actual SMTP/HTTP call.
func (s *Server) deliver(ctx context.Context, msgID string, req sendReq, body string) (string, *time.Time, string, error) {
	if !req.ScheduledAt.IsZero() && req.ScheduledAt.After(time.Now()) {
		return "", nil, s.driver.Name(), nil // scheduled, defer actual delivery
	}
	msg := driver.Message{
		From:     firstNonEmpty(req.From, s.cfg.FromDefault),
		To:       req.To, Cc: req.Cc, Bcc: req.Bcc,
		Subject: req.Subject, Body: body, BodyType: req.BodyType,
		Headers: req.Headers, ReplyTo: req.ReplyTo,
	}
	if len(req.Attachments) > 0 {
		for _, a := range req.Attachments {
			data, err := decodeBase64(a.DataBase64)
			if err != nil {
				return "", nil, s.driver.Name(), fmt.Errorf("attachment %s: %w", a.Filename, err)
			}
			msg.Attachments = append(msg.Attachments, driver.Attachment{
				Filename: a.Filename, ContentType: a.ContentType, Data: data,
			})
		}
	}
	res, err := s.driver.Send(ctx, msg)
	if err != nil {
		return "", nil, s.driver.Name(), err
	}
	t := res.SentAt
	return res.ProviderID, &t, res.Driver, nil
}

// logInput is the helper struct passed to recordLog.
type logInput struct {
	MsgID, TenantID, From, Subject, BodyRendered, Status, Driver, ProviderID, Priority string
	To, Cc, Bcc                                                                         []string
	TemplateID                                                                          *string
	SentAt, ScheduledAt                                                                 *time.Time
	LastError                                                                           string
	Tags                                                                                []string
}

func (s *Server) recordLog(ctx context.Context, in logInput) (uuid.UUID, error) {
	tenant := in.TenantID
	if tenant == "" {
		tenant = currentTenant(ctx)
	}
	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}
	var id uuid.UUID
	var err error
	if in.SentAt == nil {
		err = s.pool.QueryRow(ctx, `
			INSERT INTO email.email_logs (tenant_id, msg_id, template_id, from_addr, to_addrs, cc, bcc, subject, body_rendered, status, driver, driver_msg_id, priority, last_error, scheduled_at, tags)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8, $9, $10, $11, $12, $13, $14, $15, $16)
			RETURNING id`,
			tenant, in.MsgID, in.TemplateID, in.From, mustJSON(in.To), mustJSON(in.Cc), mustJSON(in.Bcc),
			in.Subject, in.BodyRendered, in.Status, in.Driver, nullableString(in.ProviderID), in.Priority,
			nullableString(in.LastError), in.ScheduledAt, mustJSON(tags),
		).Scan(&id)
	} else {
		err = s.pool.QueryRow(ctx, `
			INSERT INTO email.email_logs (tenant_id, msg_id, template_id, from_addr, to_addrs, cc, bcc, subject, body_rendered, status, driver, driver_msg_id, priority, sent_at, last_error, scheduled_at, tags)
			VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
			RETURNING id`,
			tenant, in.MsgID, in.TemplateID, in.From, mustJSON(in.To), mustJSON(in.Cc), mustJSON(in.Bcc),
			in.Subject, in.BodyRendered, in.Status, in.Driver, nullableString(in.ProviderID), in.Priority,
			in.SentAt, nullableString(in.LastError), in.ScheduledAt, mustJSON(tags),
		).Scan(&id)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert log: %w", err)
	}
	return id, nil
}

// currentTenant retrieves the tenant id from a context where the
// TenantMW middleware has set the GUC via s.setRLS.
func currentTenant(ctx context.Context) string {
	return platform.TenantFromContext(ctx)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("[]")
	}
	return b
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func errMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func decodeBase64(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("empty payload")
	}
	return driver.DecodeBase64(s)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// publishNATS emits email.sent / email.failed events.
func (s *Server) publishNATS(ctx context.Context, msgID, status, driverMsgID, driverName string) {
	if s.nats == nil {
		return
	}
	subject := "email.sent"
	if status == "failed" {
		subject = "email.failed"
	}
	payload, _ := json.Marshal(map[string]any{
		"msg_id": msgID, "status": status, "driver": driverName, "driver_msg_id": driverMsgID, "ts": time.Now().UTC(),
	})
	if err := s.nats.Publish(subject, payload); err != nil {
		slog.Warn("nats publish", slog.String("subject", subject), slog.String("error", err.Error()))
	}
	_ = ctx
}

// =============================================================================
// Template CRUD
// =============================================================================

type templateReq struct {
	Name     string            `json:"name"`
	Subject  string            `json:"subject"`
	Body     string            `json:"body"`
	BodyType string            `json:"body_type"`
	Vars     map[string]any    `json:"vars"`
	Type     string            `json:"type"`
	IsActive *bool             `json:"is_active,omitempty"`
}

type templateResp struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Name      string         `json:"name"`
	Subject   string         `json:"subject"`
	Body      string         `json:"body"`
	BodyType  string         `json:"body_type"`
	Vars      map[string]any `json:"vars"`
	Type      string         `json:"type"`
	IsActive  bool           `json:"is_active"`
	Version   int            `json:"version"`
	CreatedBy *uuid.UUID     `json:"created_by,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (s *Server) scanTemplate(row pgx.Row, t *templateResp) error {
	var varsBytes []byte
	var createdBy *uuid.UUID
	err := row.Scan(&t.ID, &t.TenantID, &t.Name, &t.Subject, &t.Body, &t.BodyType, &varsBytes, &t.Type, &t.IsActive, &t.Version, &createdBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return err
	}
	if len(varsBytes) > 0 {
		_ = json.Unmarshal(varsBytes, &t.Vars)
	}
	t.CreatedBy = createdBy
	return nil
}

func (s *Server) ListTemplates(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	page, perPage, offset := getPagination(c)
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM email.email_templates WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT id, tenant_id, name, subject, body, body_type, COALESCE(vars, '{}'::jsonb), type, is_active, version, created_by, created_at, updated_at FROM email.email_templates WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`, perPage, offset)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()
	var items []templateResp
	for rows.Next() {
		var t templateResp
		if err := s.scanTemplate(rows, &t); err != nil {
			continue
		}
		items = append(items, t)
	}
	return s.json(c, http.StatusOK, listResp{Data: items, Total: total, Page: page, PerPage: perPage, HasMore: int64(offset+len(items)) < total})
}

func (s *Server) CreateTemplate(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	var req templateReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.Name == "" || req.Subject == "" {
		return s.errorResp(c, http.StatusBadRequest, "name and subject are required", nil)
	}
	if req.BodyType == "" {
		req.BodyType = "html"
	}
	if req.Type == "" {
		req.Type = "transactional"
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	varsBytes := []byte("{}")
	if req.Vars != nil {
		varsBytes, _ = json.Marshal(req.Vars)
	}
	var createdBy *uuid.UUID
	if userID != "" {
		u, err := uuid.Parse(userID)
		if err == nil {
			createdBy = &u
		}
	}
	var resp templateResp
	err := s.scanTemplate(s.pool.QueryRow(ctx, `
		INSERT INTO email.email_templates (tenant_id, name, subject, body, body_type, vars, type, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)
		RETURNING id, tenant_id, name, subject, body, body_type, COALESCE(vars, '{}'::jsonb), type, is_active, version, created_by, created_at, updated_at`,
		tenantID, req.Name, req.Subject, req.Body, req.BodyType, varsBytes, req.Type, active, createdBy), &resp)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "create failed", err)
	}
	return s.json(c, http.StatusCreated, resp)
}

func (s *Server) GetTemplate(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	var resp templateResp
	err := s.scanTemplate(s.pool.QueryRow(ctx, `SELECT id, tenant_id, name, subject, body, body_type, COALESCE(vars, '{}'::jsonb), type, is_active, version, created_by, created_at, updated_at FROM email.email_templates WHERE id = $1 AND deleted_at IS NULL`, id), &resp)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "template not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	return s.json(c, http.StatusOK, resp)
}

func (s *Server) UpdateTemplate(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	var req templateReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	varsBytes := []byte("{}")
	if req.Vars != nil {
		varsBytes, _ = json.Marshal(req.Vars)
	}
	bodyType := req.BodyType
	if bodyType == "" {
		bodyType = "html"
	}
	var resp templateResp
	err := s.scanTemplate(s.pool.QueryRow(ctx, `
		UPDATE email.email_templates SET
			name = COALESCE(NULLIF($2, ''), name),
			subject = COALESCE(NULLIF($3, ''), subject),
			body = COALESCE(NULLIF($4, ''), body),
			body_type = COALESCE(NULLIF($5, ''), body_type),
			vars = COALESCE($6::jsonb, vars),
			type = COALESCE(NULLIF($7, ''), type),
			is_active = COALESCE($8, is_active),
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, name, subject, body, body_type, COALESCE(vars, '{}'::jsonb), type, is_active, version, created_by, created_at, updated_at`,
		id, req.Name, req.Subject, req.Body, bodyType, varsBytes, req.Type, req.IsActive), &resp)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "template not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	return s.json(c, http.StatusOK, resp)
}

func (s *Server) DeleteTemplate(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	res, err := s.pool.Exec(ctx, `UPDATE email.email_templates SET deleted_at = NOW(), is_active = false WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if res.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "template not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

// RenderTemplate returns the rendered body/subject for a template without
// sending anything.
func (s *Server) RenderTemplate(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	var req struct {
		Data map[string]any `json:"data"`
	}
	_ = c.Bind(&req)
	var body, subject, bodyType string
	err := s.pool.QueryRow(ctx, `SELECT body, subject, body_type FROM email.email_templates WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, id, tenantID).Scan(&body, &subject, &bodyType)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "template not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	eng, err := s.renderer(body)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "template parse failed", err)
	}
	renderedBody, err := eng.Render(req.Data)
	if err != nil {
		return s.errorResp(c, http.StatusBadRequest, "render failed", err)
	}
	return s.json(c, http.StatusOK, map[string]any{"subject": subject, "body": renderedBody, "body_type": bodyType})
}

// PreviewTemplate returns the template with sample data applied.
func (s *Server) PreviewTemplate(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	var body, subject, bodyType string
	err := s.pool.QueryRow(ctx, `SELECT body, subject, body_type FROM email.email_templates WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, id, tenantID).Scan(&body, &subject, &bodyType)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "template not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	eng, err := s.renderer(body)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "template parse failed", err)
	}
	sample := map[string]any{
		"name":        "Alice",
		"order_id":    "ORD-001",
		"amount":      "1.000.000 VND",
		"reset_link":  "https://example.com/reset?token=abc",
		"verify_code": "123456",
	}
	rendered, _ := eng.Render(sample)
	return s.json(c, http.StatusOK, map[string]any{
		"subject":  subject,
		"body":     rendered,
		"raw":      body,
		"sample":   sample,
		"body_type": bodyType,
	})
}

// =============================================================================
// Logs
// =============================================================================

type logResp struct {
	ID          uuid.UUID  `json:"id"`
	MsgID       string     `json:"msg_id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	TemplateID  *uuid.UUID `json:"template_id,omitempty"`
	From        string     `json:"from"`
	To          []string   `json:"to"`
	Cc          []string   `json:"cc"`
	Bcc         []string   `json:"bcc"`
	Subject     string     `json:"subject"`
	Status      string     `json:"status"`
	Driver      string     `json:"driver"`
	DriverMsgID string     `json:"driver_msg_id"`
	Priority    string     `json:"priority"`
	RetryCount  int        `json:"retry_count"`
	LastError   string     `json:"last_error"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	OpenedAt    *time.Time `json:"opened_at,omitempty"`
	ClickedAt   *time.Time `json:"clicked_at,omitempty"`
	BouncedAt   *time.Time `json:"bounced_at,omitempty"`
	ComplainedAt *time.Time `json:"complained_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (s *Server) scanLog(row pgx.Row, l *logResp) error {
	var tmplID *uuid.UUID
	var toB, ccB, bccB []byte
	var driverMsg, lastErr *string
	err := row.Scan(&l.ID, &l.MsgID, &l.TenantID, &tmplID, &l.From, &toB, &ccB, &bccB, &l.Subject, &l.Status, &l.Driver, &driverMsg, &l.Priority, &l.RetryCount, &lastErr, &l.SentAt, &l.DeliveredAt, &l.OpenedAt, &l.ClickedAt, &l.BouncedAt, &l.ComplainedAt, &l.FailedAt, &l.CreatedAt)
	if err != nil {
		return err
	}
	l.TemplateID = tmplID
	if driverMsg != nil {
		l.DriverMsgID = *driverMsg
	}
	if lastErr != nil {
		l.LastError = *lastErr
	}
	_ = json.Unmarshal(toB, &l.To)
	_ = json.Unmarshal(ccB, &l.Cc)
	_ = json.Unmarshal(bccB, &l.Bcc)
	return nil
}

func (s *Server) ListLogs(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	status := c.QueryParam("status")
	from := c.QueryParam("from")
	to := c.QueryParam("to")
	page, perPage, offset := getPagination(c)
	where := []string{"1=1"}
	args := []any{}
	idx := 1
	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, status)
		idx++
	}
	if from != "" {
		where = append(where, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, from)
		idx++
	}
	if to != "" {
		where = append(where, fmt.Sprintf("created_at <= $%d", idx))
		args = append(args, to)
		idx++
	}
	args = append(args, perPage, offset)
	q := "SELECT id, msg_id, tenant_id, template_id, from_addr, to_addrs, cc, bcc, subject, status, driver, driver_msg_id, priority, retry_count, last_error, sent_at, delivered_at, opened_at, clicked_at, bounced_at, complained_at, failed_at, created_at FROM email.email_logs WHERE " + strings.Join(where, " AND ") + " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(idx) + " OFFSET $" + strconv.Itoa(idx+1)
	var total int64
	countQ := "SELECT COUNT(*) FROM email.email_logs WHERE " + strings.Join(where, " AND ")
	countArgs := args[:idx-1]
	if err := s.pool.QueryRow(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "count failed", err)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()
	var items []logResp
	for rows.Next() {
		var l logResp
		if err := s.scanLog(rows, &l); err == nil {
			items = append(items, l)
		}
	}
	return s.json(c, http.StatusOK, listResp{Data: items, Total: total, Page: page, PerPage: perPage, HasMore: int64(offset+len(items)) < total})
}

func (s *Server) GetLog(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	var l logResp
	err := s.scanLog(s.pool.QueryRow(ctx, `SELECT id, msg_id, tenant_id, template_id, from_addr, to_addrs, cc, bcc, subject, status, driver, driver_msg_id, priority, retry_count, last_error, sent_at, delivered_at, opened_at, clicked_at, bounced_at, complained_at, failed_at, created_at FROM email.email_logs WHERE id = $1`, id), &l)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "log not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	return s.json(c, http.StatusOK, l)
}

// =============================================================================
// Stats
// =============================================================================

func (s *Server) Stats(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	out := map[string]any{}
	rows, err := s.pool.Query(ctx, `SELECT status, COUNT(*) FROM email.email_logs GROUP BY status`)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "stats failed", err)
	}
	defer rows.Close()
	byStatus := map[string]int64{}
	for rows.Next() {
		var s string
		var n int64
		_ = rows.Scan(&s, &n)
		byStatus[s] = n
	}
	out["by_status"] = byStatus
	rows2, err := s.pool.Query(ctx, `SELECT driver, COUNT(*) FROM email.email_logs GROUP BY driver`)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "stats failed", err)
	}
	defer rows2.Close()
	byDriver := map[string]int64{}
	for rows2.Next() {
		var s string
		var n int64
		_ = rows2.Scan(&s, &n)
		byDriver[s] = n
	}
	out["by_driver"] = byDriver
	rows3, err := s.pool.Query(ctx, `SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day, COUNT(*) FROM email.email_logs WHERE created_at > NOW() - INTERVAL '14 days' GROUP BY day ORDER BY day`)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "stats failed", err)
	}
	defer rows3.Close()
	type dayCount struct {
		Day   string `json:"day"`
		Count int64  `json:"count"`
	}
	var byDay []dayCount
	for rows3.Next() {
		var dc dayCount
		_ = rows3.Scan(&dc.Day, &dc.Count)
		byDay = append(byDay, dc)
	}
	out["by_day"] = byDay
	return s.json(c, http.StatusOK, out)
}

// =============================================================================
// Webhook ingest (provider-agnostic)
// =============================================================================

type webhookPayload struct {
	EventType string         `json:"event_type"`
	MsgID     string         `json:"msg_id"`
	Payload   map[string]any `json:"payload"`
}

func (s *Server) Webhook(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	provider := c.Param("provider")
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = c.QueryParam("tenant_id")
	}
	if tenantID == "" {
		return s.errorResp(c, http.StatusBadRequest, "tenant required", nil)
	}
	raw, _ := readAll(c.Request().Body)
	if _, err := s.pool.Exec(ctx, `INSERT INTO email.email_webhooks (tenant_id, provider, event_type, payload) VALUES ($1, $2, $3, $4)`,
		tenantID, provider, c.QueryParam("event_type"), json.RawMessage(raw)); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "webhook record failed", err)
	}
	var event, msgID string
	var wp webhookPayload
	_ = json.Unmarshal(raw, &wp)
	// Best-effort update of the log
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if mp, ok := readMsgID(raw); ok {
		event = guessEventType(raw)
		msgID = mp
		_ = updateLogStatus(ctx, s.pool, tenantID, msgID, event)
	}
	return s.json(c, http.StatusOK, map[string]any{"status": "received", "provider": provider, "msg_id": msgID, "event": event})
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 1024)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}

func readMsgID(raw []byte) (string, bool) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", false
	}
	for _, k := range []string{"msg_id", "MessageID", "message_id", "id"} {
		if v, ok := m[k].(string); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

func guessEventType(raw []byte) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, k := range []string{"event_type", "event", "type", "notificationType"} {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func updateLogStatus(ctx context.Context, pool *pgxpool.Pool, tenantID, msgID, event string) error {
	var column string
	switch event {
	case "delivered", "delivery":
		column = "delivered_at"
	case "open", "opened":
		column = "opened_at"
	case "click", "clicked":
		column = "clicked_at"
	case "bounce", "bounced":
		column = "bounced_at"
	case "complaint", "complained", "spamreport":
		column = "complained_at"
	case "failed", "failure":
		column = "failed_at"
	default:
		return nil
	}
	q := fmt.Sprintf(`UPDATE email.email_logs SET %s = NOW(), status = $1, updated_at = NOW() WHERE msg_id = $2 AND tenant_id = $3`, column)
	res, err := pool.Exec(ctx, q, event, msgID, tenantID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return nil
	}
	return nil
}

// =============================================================================
// Tracking endpoints
// =============================================================================

// TrackingPixel returns a 1x1 transparent GIF and records an open event.
func (s *Server) TrackingPixel(c echo.Context) error {
	msgID := strings.TrimSuffix(c.Param("msg_id"), ".gif")
	ctx := c.Request().Context()
	if msgID != "" {
		_, _ = s.pool.Exec(ctx, `UPDATE email.email_logs SET opened_at = COALESCE(opened_at, NOW()), status = 'opened', updated_at = NOW() WHERE msg_id = $1`, msgID)
	}
	c.Response().Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Response().Header().Set("Pragma", "no-cache")
	return c.Blob(http.StatusOK, "image/gif", tracking.PixelGif)
}

// ClickRedirect records a click and 302-redirects to the original target.
func (s *Server) ClickRedirect(c echo.Context) error {
	msgID := c.Param("msg_id")
	target := c.QueryParam("url")
	if target == "" {
		target = "/"
	}
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}
	ctx := c.Request().Context()
	if msgID != "" {
		_, _ = s.pool.Exec(ctx, `UPDATE email.email_logs SET clicked_at = COALESCE(clicked_at, NOW()), status = 'clicked', updated_at = NOW() WHERE msg_id = $1`, msgID)
	}
	return c.Redirect(http.StatusFound, target)
}

// Unsubscribe handles one-click unsubscribe (RFC 8058).
func (s *Server) Unsubscribe(c echo.Context) error {
	msgID := c.Param("msg_id")
	ctx := c.Request().Context()
	if msgID != "" {
		_, _ = s.pool.Exec(ctx, `UPDATE email.email_logs SET status = 'complained', complained_at = NOW(), updated_at = NOW() WHERE msg_id = $1`, msgID)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "unsubscribed", "msg_id": msgID})
}

// =============================================================================
// Connect-RPC adapter
// =============================================================================

// ConnectRPC forwards the standard Connect RPC paths to the same handler
// functions exposed on the REST API.  The path layout mirrors what
// bufbuild/connect-go would generate.
func (s *Server) ConnectRPC(e *echo.Echo) {
	g := e.Group("/internal/email.v1.EmailService")
	g.POST("/SendEmail", s.SendEmail)
	g.POST("/BatchSend", s.BatchSend)
	g.POST("/RenderTemplate", s.RenderTemplate)
	g.POST("/GetDeliveryLog", s.GetLog)
	g.POST("/ListTemplates", s.ListTemplates)
	g.POST("/GetTemplateStats", s.Stats)
}

// ensureQueryEscape exists so other packages can reuse the helper without
// pulling net/url directly.
func ensureQueryEscape(v string) string { return url.QueryEscape(v) }