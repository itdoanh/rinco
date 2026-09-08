// Package handler provides HTTP and Connect-RPC handlers for the notification
// service.  Each handler is a method on the Server struct which wires together
// the DB pool, Redis client, NATS connection, and all configured channels.
//
// Endpoints:
//   - POST /v1/notifications/send         — fan-out to user channels
//   - POST /v1/notifications/broadcast     — broadcast to audience
//   - GET  /v1/notifications              — list with cursor pagination
//   - GET  /v1/notifications/:id
//   - POST /v1/notifications/:id/read
//   - POST /v1/preferences/:user_id       — set user channel prefs
//   - GET  /v1/preferences/:user_id
//   - POST /v1/subscriptions/webpush      — register VAPID endpoint
//   - DELETE /v1/subscriptions/webpush/:id
//   - POST /v1/subscriptions/fcm          — register FCM device token
//   - GET  /v1/stats
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"github.com/rinco/services/notification-service/internal/channels"
	"github.com/rinco/services/notification-service/internal/platform"
	"github.com/rinco/services/notification-service/internal/preferences"
)

// Server holds all dependencies for the notification handlers.
type Server struct {
	pool   *pgxpool.Pool
	rdb    *redis.Client
	nc     *nats.Conn
	cfg    *platform.Config
	chans  map[string]channels.Driver
}

// NewServer builds the handler.Server with all channel drivers configured.
func NewServer(pool *pgxpool.Pool, rdb *redis.Client, nc *nats.Conn, cfg *platform.Config) *Server {
	chans := map[string]channels.Driver{
		"in_app":  channels.NewInAppChannel(nc),
		"email":   channels.NewEmailChannel(cfg.EmailRPCURL),
		"sms":     channels.NewSMSChannel(cfg.TwilioSID, cfg.TwilioToken, cfg.TwilioFrom),
		"push":    channels.NewPushChannel(cfg.VAPIDPublic, cfg.VAPIDPrivate, cfg.VAPIDSubject),
		"fcm":     channels.NewFCMChannel(cfg.FCMProjectID, ""),
		"slack":   channels.NewSlackChannel(cfg.SlackWebhook),
		"discord": channels.NewDiscordChannel(""),
		"telegram": channels.NewTelegramChannel(cfg.TelegramBotToken),
	}
	return &Server{pool: pool, rdb: rdb, nc: nc, cfg: cfg, chans: chans}
}

// Pool returns the underlying pool (used by cmd/main.go).
func (s *Server) Pool() *pgxpool.Pool { return s.pool }

// =============================================================================
// Context helpers
// =============================================================================

func (s *Server) tenantFromCtx(c echo.Context) (string, string, bool) {
	t, _ := c.Get("tenant_id").(string)
	u, _ := c.Get("user_id").(string)
	a, _ := c.Get("is_admin").(bool)
	return t, u, a
}

// resolveSubtreeUsers resolves a list of user_ids that are descendants of
// the given root in the crm users_tree (LTREE). Requires crm-service schema
// to be reachable via DATABASE_URL or a CRM_TREE_DSN env override.
//
// Falls back to empty slice if users_tree is not available — callers should
// surface a 422 if zero recipients resolved.
func (s *Server) resolveSubtreeUsers(ctx context.Context, tenantID, rootUserID string, maxDepth int) []string {
	var out []string
	// Look up root path first
	var rootPath string
	err := s.pool.QueryRow(ctx, `SELECT path::text FROM users_tree WHERE user_id = $1 AND tenant_id = $2`, rootUserID, tenantID).Scan(&rootPath)
	if err != nil {
		return out
	}
	return s.resolveSubtreeByPath(ctx, tenantID, rootPath, maxDepth)
}

// resolveSubtreeByPath returns user_ids whose LTREE path is a descendant of
// the given rootPath. If maxDepth > 0, depth is bounded to rootPath.NUM + maxDepth.
func (s *Server) resolveSubtreeByPath(ctx context.Context, tenantID, rootPath string, maxDepth int) []string {
	var out []string
	q := `SELECT user_id::text FROM users_tree WHERE tenant_id = $1 AND path <@ $2::ltree`
	args := []any{tenantID, rootPath}
	if maxDepth > 0 {
		// nlevel(path) <= nlevel(rootPath) + maxDepth
		q += ` AND nlevel(path) <= nlevel($2::ltree) + $3`
		args = append(args, maxDepth)
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			out = append(out, uid)
		}
	}
	return out
}

// setRLS applies RLS context with parameterized queries (SQL-injection-safe).
//
// KNOWN LIMITATION: SET LOCAL outside a tx has no effect — same as crm-
// service. The variables die with the tx commit below. Subsequent handler
// queries on a different pool conn see NULL settings, so RLS policies may
// not match. Architectural refactor pending.
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
	Data    any   `json:"data"`
	Total   int64 `json:"total,omitempty"`
	Cursor  string `json:"cursor,omitempty"`
	HasMore bool  `json:"has_more"`
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
// Send
// =============================================================================

type sendReq struct {
	UserID     string            `json:"user_id"`
	Type       string            `json:"type"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Icon       string            `json:"icon,omitempty"`
	Category   string            `json:"category,omitempty"`
	Priority   string            `json:"priority"` // high | normal | low
	Channels   []string          `json:"channels,omitempty"`
	Data       map[string]any   `json:"data,omitempty"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	Email      string            `json:"email,omitempty"`
	Phone      string            `json:"phone,omitempty"`
	Token      string            `json:"token,omitempty"`
	ScheduledAt *time.Time       `json:"scheduled_at,omitempty"`
}

func (s *Server) Send(c echo.Context) error {
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
	if req.UserID == "" {
		return s.errorResp(c, http.StatusBadRequest, "user_id required", nil)
	}
	if req.Title == "" || req.Body == "" {
		return s.errorResp(c, http.StatusBadRequest, "title and body required", nil)
	}
	priority := req.Priority
	if priority == "" {
		priority = "normal"
	}
	channelNames := req.Channels
	if len(channelNames) == 0 {
		channelNames = preferences.DefaultPriorityChannels[priority]
	}

	// Load user preferences to filter channels
	userPrefs, _ := s.loadUserPrefs(ctx, req.UserID)
	resolver := preferences.NewResolver(userPrefs)
	channelNames = resolver.Resolve(priority, req.Type, req.UserID)

	notifID := uuid.NewString()
	if req.Data == nil {
		req.Data = map[string]any{}
	}
	req.Data["notif_id"] = notifID

	// Insert the notification record
	var status string
	if !req.ScheduledAt.IsZero() {
		status = "pending"
	} else {
		status = "sent"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification.notifications (id, tenant_id, user_id, type, title, body, icon, category, priority, channels_resolved, data, status, expires_at, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, $13, $14)`,
		notifID, tenantID, req.UserID, req.Type, req.Title, req.Body, req.Icon, req.Category,
		priority, mustJSON(channelNames), mustJSON(req.Data), status, req.ExpiresAt,
		nilIfZero(req.ScheduledAt))
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "insert failed", err)
	}

	// Fan out to channels
	var deliveryResults []map[string]string
	for _, ch := range channelNames {
		drv, ok := s.chans[ch]
		if !ok {
			continue
		}
		notif := channels.Notification{
			TenantID: tenantID, UserID: req.UserID, Type: req.Type,
			Title: req.Title, Body: req.Body, Icon: req.Icon, Data: req.Data, ExpiresAt: req.ExpiresAt,
			Email: req.Email, Phone: req.Phone, Token: req.Token,
		}
		_, err := drv.Send(ctx, notif)
		dlStatus := "sent"
		errMsg := ""
		if err != nil {
			dlStatus = "failed"
			errMsg = err.Error()
		}
		deliveryResults = append(deliveryResults, map[string]string{"channel": ch, "status": dlStatus, "error": errMsg})
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO notification.notification_delivery_logs (notification_id, channel, status, error_msg, sent_at, delivered_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			notifID, ch, dlStatus, nullString(errMsg), tsOrNil(dlStatus == "sent"), tsOrNil(dlStatus == "delivered"))
		// Update aggregate
		go s.updateAggregate(context.Background(), tenantID, req.Type, ch, dlStatus)
	}
	return s.json(c, http.StatusAccepted, map[string]any{"id": notifID, "status": status, "channels": channelNames, "delivery": deliveryResults})
}

func (s *Server) loadUserPrefs(ctx context.Context, userID string) ([]preferences.Preference, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, notif_type, channel, enabled, quiet_start, quiet_end, digest_mode
		FROM notification.notification_preferences WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prefs []preferences.Preference
	for rows.Next() {
		var p preferences.Preference
		var quietStart, quietEnd *int
		if err := rows.Scan(&p.UserID, &p.NotifType, &p.Channel, &p.Enabled, &quietStart, &quietEnd, &p.DigestMode); err != nil {
			continue
		}
		p.QuietStart = quietStart
		p.QuietEnd = quietEnd
		prefs = append(prefs, p)
	}
	return prefs, nil
}

func (s *Server) updateAggregate(ctx context.Context, tenantID, notifType, channel, status string) {
	if tenantID == "" {
		return
	}
	colSent := "count_sent"
	if status == "delivered" {
		colSent = "count_delivered"
	} else if status == "failed" {
		colSent = "count_failed"
	}
	q := fmt.Sprintf(`INSERT INTO notification.daily_aggregates (date, tenant_id, type, channel, %s) VALUES ($1, $2, $3, $4, 1) ON CONFLICT (date, tenant_id, type, channel) DO UPDATE SET %s = daily_aggregates.%s + 1`, colSent, colSent, colSent)
	_, _ = s.pool.Exec(ctx, q, time.Now().Format("2006-01-02"), tenantID, notifType, channel)
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func nullString(s string) interface{} { if s == "" { return nil }; return s }
func tsOrNil(ok bool) interface{} { if ok { t := time.Now(); return &t }; return nil }
func nilIfZero(t *time.Time) interface{} { if t == nil || t.IsZero() { return nil }; return t }

// =============================================================================
// Broadcast
// =============================================================================

type broadcastReq struct {
	Type       string          `json:"type"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	Icon       string          `json:"icon,omitempty"`
	Category   string          `json:"category,omitempty"`
	Priority   string          `json:"priority"`
	Channels   []string        `json:"channels,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
	Audience   broadcastAudience `json:"audience"` // who to broadcast to
}

type broadcastAudience struct {
	UserIDs   []string              `json:"user_ids,omitempty"`
	Role      string                `json:"role,omitempty"`
	Department string               `json:"department,omitempty"`
	// SubtreeRootID broadcasts to all descendants of this user in the org tree
	// (giám đốc → quản lý → trưởng nhóm → nhân viên). Only users whose
	// crm users_tree.path is descendant of SubtreeRootID will receive.
	// Requires crm-schema `users_tree` table to exist (LTREE).
	SubtreeRootID string            `json:"subtree_root_id,omitempty"`
	// SubtreeRootPath bypasses the user_id lookup; must be a valid LTREE path
	// (e.g. "root.uuid"). Useful for cross-service direct broadcast.
	SubtreeRootPath string          `json:"subtree_root_path,omitempty"`
	// MaxDepth limits how many levels down the tree to send (0 = unlimited).
	MaxDepth int                   `json:"max_depth,omitempty"`
}

func (s *Server) Broadcast(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()
	tenantID, _, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, "", isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	var req broadcastReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.Title == "" {
		return s.errorResp(c, http.StatusBadRequest, "title required", nil)
	}

	// Build user list from audience
	userIDs := req.Audience.UserIDs
	if len(userIDs) == 0 && req.Audience.Role != "" {
		rows, err := s.pool.Query(ctx, `SELECT id FROM users WHERE tenant_id = $1 AND role = $2`, tenantID, req.Audience.Role)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var uid string
				_ = rows.Scan(&uid)
				userIDs = append(userIDs, uid)
			}
		}
	}
	if len(userIDs) == 0 && req.Audience.Department != "" {
		rows, err := s.pool.Query(ctx, `SELECT id FROM users WHERE tenant_id = $1 AND department = $2`, tenantID, req.Audience.Department)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var uid string
				_ = rows.Scan(&uid)
				userIDs = append(userIDs, uid)
			}
		}
	}
	if len(userIDs) == 0 && req.Audience.SubtreeRootID != "" {
		userIDs = s.resolveSubtreeUsers(ctx, tenantID, req.Audience.SubtreeRootID, req.Audience.MaxDepth)
	}
	if len(userIDs) == 0 && req.Audience.SubtreeRootPath != "" {
		userIDs = s.resolveSubtreeByPath(ctx, tenantID, req.Audience.SubtreeRootPath, req.Audience.MaxDepth)
	}

	priority := req.Priority
	if priority == "" {
		priority = "normal"
	}
	channelNames := req.Channels
	if len(channelNames) == 0 {
		channelNames = preferences.DefaultPriorityChannels[priority]
	}

	var results []map[string]any
	for _, uid := range userIDs {
		notifID := uuid.NewString()
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO notification.notifications (id, tenant_id, user_id, type, title, body, icon, category, priority, channels_resolved, data, status, sent_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, 'sent', NOW())`,
			notifID, tenantID, uid, req.Type, req.Title, req.Body, req.Icon, req.Category, priority, mustJSON(channelNames), mustJSON(req.Data))
		results = append(results, map[string]any{"user_id": uid, "notif_id": notifID})
		// Fan out
		for _, ch := range channelNames {
			drv, ok := s.chans[ch]
			if !ok {
				continue
			}
			channel := ch
			userID := uid
			_ = channel
			go func(d channels.Driver, u string) {
				_, _ = d.Send(context.Background(), channels.Notification{
					TenantID: tenantID, UserID: u, Type: req.Type,
					Title: req.Title, Body: req.Body, Icon: req.Icon, Data: req.Data,
				})
			}(drv, userID)
		}
	}
	return s.json(c, http.StatusAccepted, map[string]any{"count": len(results), "results": results})
}

// =============================================================================
// List notifications
// =============================================================================

type notifResp struct {
	ID              uuid.UUID         `json:"id"`
	TenantID       uuid.UUID         `json:"tenant_id"`
	UserID         string            `json:"user_id"`
	Type           string            `json:"type"`
	Title          string            `json:"title"`
	Body           string            `json:"body"`
	Icon           string            `json:"icon,omitempty"`
	Category       string            `json:"category,omitempty"`
	Priority       string            `json:"priority"`
	ChannelsResolved []string        `json:"channels_resolved"`
	Data           map[string]any    `json:"data,omitempty"`
	Status         string            `json:"status"`
	ReadAt         *time.Time        `json:"read_at,omitempty"`
	SentAt         *time.Time        `json:"sent_at,omitempty"`
	ExpiresAt      *time.Time        `json:"expires_at,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

func (s *Server) scanNotif(row pgx.Row, n *notifResp) error {
	var channelsJSON, dataJSON []byte
	err := row.Scan(&n.ID, &n.TenantID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Icon, &n.Category, &n.Priority, &channelsJSON, &dataJSON, &n.Status, &n.ReadAt, &n.SentAt, &n.ExpiresAt, &n.CreatedAt)
	if err != nil {
		return err
	}
	_ = json.Unmarshal(channelsJSON, &n.ChannelsResolved)
	_ = json.Unmarshal(dataJSON, &n.Data)
	return nil
}

func (s *Server) List(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, userID, isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	filterUser := c.QueryParam("user_id")
	status := c.QueryParam("status")
	notifType := c.QueryParam("type")
	from := c.QueryParam("from")
	cursor := c.QueryParam("cursor")
	limit := 25
	if l, _ := strconv.Atoi(c.QueryParam("limit")); l > 0 && l <= 200 {
		limit = l
	}
	where := []string{"1=1"}
	args := []any{}
	idx := 1
	if filterUser != "" {
		where = append(where, fmt.Sprintf("user_id = $%d", idx))
		args = append(args, filterUser)
		idx++
	} else if !isAdmin {
		where = append(where, fmt.Sprintf("user_id = $%d", idx))
		args = append(args, userID)
		idx++
	}
	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, status)
		idx++
	}
	if notifType != "" {
		where = append(where, fmt.Sprintf("type = $%d", idx))
		args = append(args, notifType)
		idx++
	}
	if cursor != "" {
		where = append(where, fmt.Sprintf("created_at < $%d", idx))
		args = append(args, cursor)
		idx++
	}
	if from != "" {
		where = append(where, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, from)
		idx++
	}
	args = append(args, limit+1)
	q := `SELECT id, tenant_id, user_id, type, title, body, icon, category, priority, channels_resolved, data, status, read_at, sent_at, expires_at, created_at FROM notification.notifications WHERE ` + strings.Join(where, " AND ") + " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(idx)
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()
	var items []notifResp
	var lastCreated *time.Time
	for rows.Next() {
		var n notifResp
		if err := s.scanNotif(rows, &n); err != nil {
			continue
		}
		items = append(items, n)
		lastCreated = &n.CreatedAt
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	nextCursor := ""
	if hasMore && lastCreated != nil {
		nextCursor = lastCreated.Format(time.RFC3339Nano)
	}
	return s.json(c, http.StatusOK, listResp{Data: items, Cursor: nextCursor, HasMore: hasMore})
}

func (s *Server) Get(c echo.Context) error {
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
	var n notifResp
	err := s.scanNotif(s.pool.QueryRow(ctx, `SELECT id, tenant_id, user_id, type, title, body, icon, category, priority, channels_resolved, data, status, read_at, sent_at, expires_at, created_at FROM notification.notifications WHERE id = $1`, id), &n)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.errorResp(c, http.StatusNotFound, "notification not found", nil)
	}
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	return s.json(c, http.StatusOK, n)
}

func (s *Server) MarkRead(c echo.Context) error {
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
	res, err := s.pool.Exec(ctx, `UPDATE notification.notifications SET status = 'read', read_at = NOW() WHERE id = $1 AND (status != 'read' OR read_at IS NULL)`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "update failed", err)
	}
	if res.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "notification not found or already read", nil)
	}
	return s.json(c, http.StatusOK, map[string]string{"status": "read"})
}

// =============================================================================
// Preferences
// =============================================================================

type prefsReq struct {
	Channels   map[string]prefChannel `json:"channels"`
	QuietHours []int                  `json:"quiet_hours"`
	DigestMode string                `json:"digest_mode"`
	Types      map[string]bool        `json:"types,omitempty"`
}

type prefChannel struct {
	Enabled bool `json:"enabled"`
}

type prefResp struct {
	UserID      string               `json:"user_id"`
	Channels    map[string]bool     `json:"channels"`
	QuietHours []int                `json:"quiet_hours"`
	DigestMode string               `json:"digest_mode"`
	Types      map[string]bool      `json:"types,omitempty"`
}

func (s *Server) SetPreferences(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, _, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, "", isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	userID := c.Param("user_id")
	if userID == "" {
		return s.errorResp(c, http.StatusBadRequest, "user_id required", nil)
	}
	var req prefsReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}

	// Upsert each channel preference
	quietStart, quietEnd := preferences.ParseQuietHours(req.QuietHours)
	for ch, p := range req.Channels {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO notification.notification_preferences (user_id, notif_type, channel, enabled, quiet_start, quiet_end, digest_mode)
			VALUES ($1, '*', $2, $3, $4, $5, $6)
			ON CONFLICT (user_id, notif_type, channel) DO UPDATE SET enabled = $3, quiet_start = $4, quiet_end = $5, digest_mode = $6`,
			userID, ch, p.Enabled, quietStart, quietEnd, req.DigestMode)
		if err != nil {
			slog.Warn("set pref failed", slog.String("error", err.Error()))
		}
	}
	return s.json(c, http.StatusOK, map[string]any{"status": "saved", "user_id": userID})
}

func (s *Server) GetPreferences(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, _, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, "", isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	userID := c.Param("user_id")
	if userID == "" {
		return s.errorResp(c, http.StatusBadRequest, "user_id required", nil)
	}
	rows, err := s.pool.Query(ctx, `SELECT user_id, notif_type, channel, enabled, quiet_start, quiet_end, digest_mode FROM notification.notification_preferences WHERE user_id = $1`, userID)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "query failed", err)
	}
	defer rows.Close()
	channels := map[string]bool{}
	var quietStart, quietEnd *int
	digestMode := "none"
	for rows.Next() {
		var p preferences.Preference
		var qs, qe *int
		_ = rows.Scan(&p.UserID, &p.NotifType, &p.Channel, &p.Enabled, &qs, &qe, &p.DigestMode)
		channels[p.Channel] = p.Enabled
		if qs != nil {
			quietStart = qs
		}
		if qe != nil {
			quietEnd = qe
		}
		digestMode = p.DigestMode
	}
	var quietHours []int
	if quietStart != nil && quietEnd != nil {
		quietHours = []int{*quietStart, *quietEnd}
	}
	return s.json(c, http.StatusOK, prefResp{UserID: userID, Channels: channels, QuietHours: quietHours, DigestMode: digestMode})
}

// =============================================================================
// Subscriptions
// =============================================================================

type webpushSubReq struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
	UserID   string `json:"user_id"`
}

func (s *Server) RegisterWebPush(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, _, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, "", isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	var req webpushSubReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.Endpoint == "" || req.P256dh == "" || req.Auth == "" {
		return s.errorResp(c, http.StatusBadRequest, "endpoint, p256dh, and auth required", nil)
	}
	keys := map[string]string{"p256dh": req.P256dh, "auth": req.Auth}
	keysJSON, _ := json.Marshal(keys)
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO notification.push_subscriptions (user_id, endpoint, p256dh, auth, keys, vapid_public_key)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		ON CONFLICT (endpoint) DO UPDATE SET user_id = $1, p256dh = $3, auth = $4, last_seen = NOW()
		RETURNING id`,
		firstNonEmpty(req.UserID, c.Request().Header.Get("X-User-ID")),
		req.Endpoint, req.P256dh, req.Auth, keysJSON, s.cfg.VAPIDPublic).Scan(&id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "insert failed", err)
	}
	return s.json(c, http.StatusCreated, map[string]any{"id": id, "status": "registered"})
}

func (s *Server) DeleteWebPush(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, _, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, "", isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid id", err)
	}
	res, err := s.pool.Exec(ctx, `DELETE FROM notification.push_subscriptions WHERE id = $1`, id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "delete failed", err)
	}
	if res.RowsAffected() == 0 {
		return s.errorResp(c, http.StatusNotFound, "subscription not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

type fcmSubReq struct {
	DeviceToken string `json:"device_token"`
	Platform    string `json:"platform"` // android | ios | web
	AppVersion  string `json:"app_version,omitempty"`
	UserID      string `json:"user_id"`
}

func (s *Server) RegisterFCM(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, _, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, "", isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	var req fcmSubReq
	if err := c.Bind(&req); err != nil {
		return s.errorResp(c, http.StatusBadRequest, "invalid request", err)
	}
	if req.DeviceToken == "" {
		return s.errorResp(c, http.StatusBadRequest, "device_token required", nil)
	}
	platform := req.Platform
	if platform == "" {
		platform = "web"
	}
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO notification.fcm_subscriptions (user_id, device_token, platform, app_version)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (device_token) DO UPDATE SET user_id = $1, platform = $3, app_version = $4, last_seen = NOW()
		RETURNING id`,
		firstNonEmpty(req.UserID, c.Request().Header.Get("X-User-ID")),
		req.DeviceToken, platform, req.AppVersion).Scan(&id)
	if err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "insert failed", err)
	}
	return s.json(c, http.StatusCreated, map[string]any{"id": id, "status": "registered"})
}

// =============================================================================
// Stats
// =============================================================================

func (s *Server) Stats(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()
	tenantID, _, isAdmin := s.tenantFromCtx(c)
	if err := s.setRLS(ctx, tenantID, "", isAdmin); err != nil {
		return s.errorResp(c, http.StatusInternalServerError, "context setup failed", err)
	}
	out := map[string]any{}

	rows, _ := s.pool.Query(ctx, `SELECT status, COUNT(*) FROM notification.notifications WHERE tenant_id = $1 GROUP BY status`, tenantID)
	if rows != nil {
		defer rows.Close()
		byStatus := map[string]int64{}
		for rows.Next() {
			var st string
			var n int64
			_ = rows.Scan(&st, &n)
			byStatus[st] = n
		}
		out["by_status"] = byStatus
	}

	rows2, _ := s.pool.Query(ctx, `SELECT channel, COUNT(*) FROM notification.notification_delivery_logs l JOIN notification.notifications n ON l.notification_id = n.id WHERE n.tenant_id = $1 GROUP BY channel`, tenantID)
	if rows2 != nil {
		defer rows2.Close()
		byChannel := map[string]int64{}
		for rows2.Next() {
			var ch string
			var n int64
			_ = rows2.Scan(&ch, &n)
			byChannel[ch] = n
		}
		out["by_channel"] = byChannel
	}

	rows3, _ := s.pool.Query(ctx, `SELECT type, COUNT(*) FROM notification.notifications WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '7 days' GROUP BY type ORDER BY count DESC LIMIT 20`, tenantID)
	if rows3 != nil {
		defer rows3.Close()
		type topType struct {
			Type  string `json:"type"`
			Count int64 `json:"count"`
		}
		var topTypes []topType
		for rows3.Next() {
			var t topType
			_ = rows3.Scan(&t.Type, &t.Count)
			topTypes = append(topTypes, t)
		}
		out["top_types_7d"] = topTypes
	}

	return s.json(c, http.StatusOK, out)
}

// =============================================================================
// Connect-RPC adapter
// =============================================================================

func (s *Server) ConnectRPC(e *echo.Echo) {
	g := e.Group("/internal/notification.v1.NotificationService")
	g.POST("/SendNotification", s.Send)
	g.POST("/BroadcastNotification", s.Broadcast)
	g.POST("/MarkAsRead", s.MarkRead)
	g.POST("/GetUserPreferences", s.GetPreferences)
	g.POST("/SetUserPreferences", s.SetPreferences)
	g.POST("/RegisterPushSubscription", s.RegisterWebPush)
	g.POST("/GetNotificationFeed", s.List)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}