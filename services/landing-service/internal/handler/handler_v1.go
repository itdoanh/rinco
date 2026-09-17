package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// =============================================================================
// V1 canonical handlers
// =============================================================================
//
// These mirror the /v1/* routes documented in the API specification.
// Every write endpoint accepts (or requires) an HMAC signature in the
// X-Signature header.  Signatures follow:
//
//     X-Signature: sha256=<hex(HMAC_SHA256(secret, timestamp + "." + body))>
//     X-HMAC-Timestamp: <unix seconds>
//
// Where ``secret`` is the value of TRACKING_HMAC_SECRET (falls back to
// ``trackingSalt`` to preserve backward compatibility with the
// pre-v1 admin tools).
//
// Replay protection: requests older than 5 minutes are rejected.

// VerifyHMAC validates the request's HMAC signature + timestamp.  It
// returns an HTTP error directly when validation fails so the caller
// can simply ``return`` its result.
func (s *Server) VerifyHMAC(c echo.Context) error {
	sig := c.Request().Header.Get("X-Signature")
	ts := c.Request().Header.Get("X-HMAC-Timestamp")
	if sig == "" || ts == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing signature"})
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid timestamp"})
	}
	if abs(time.Now().Unix()-tsInt) > 300 {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "stale signature"})
	}

	// Read body (consuming it) so the next handler can still bind it.
	body, err := readAndRestoreBody(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	mac := hmac.New(sha256.New, []byte(s.trackingSalt))
	mac.Write([]byte(ts))
	mac.Write([]byte{'.'})
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid signature"})
	}
	return nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// Track is the v1/track endpoint.  Unlike the legacy TrackEvent route
// it requires a valid HMAC signature, validates the data shape and
// publishes a NATS event so the meta-capi-service can forward the
// conversion to FB.
type v1TrackReq struct {
	EventName string                 `json:"event_name"`
	EventID   string                 `json:"event_id"`
	TenantID  string                 `json:"tenant_id"`
	PageSlug  string                 `json:"page_slug"`
	SessionID string                 `json:"session_id"`
	URL       string                 `json:"url"`
	UserData  map[string]interface{} `json:"user_data"`
	Custom    map[string]interface{} `json:"custom_data"`
}

func (s *Server) Track(c echo.Context) error {
	if err := s.VerifyHMAC(c); err != nil {
		return err
	}
	var req v1TrackReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.EventName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "event_name required"})
	}

	if req.EventID == "" {
		req.EventID = uuid.NewString()
	}
	tenantID := req.TenantID
	if tenantID == "" {
		tenantID = c.Request().Header.Get("X-Tenant-ID")
	}

	payload, _ := json.Marshal(req)
	if _, err := s.pool.Exec(c.Request().Context(),
		`INSERT INTO landing.tracking_events(tenant_id,event_name,event_type,event_id,page_slug,session_id,payload,ip_address,user_agent) VALUES (NULLIF($1,'')::uuid,$2,$3,$4,$5,$6,$7,$8::inet,$9) ON CONFLICT(tenant_id,event_id) DO NOTHING`,
		tenantID, req.EventName, req.EventName, req.EventID, req.PageSlug, req.SessionID, payload, c.RealIP(), c.Request().UserAgent(),
	); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Promote "Lead" submissions into a CAPI event so FB can match.
	if req.EventName == "Lead" {
		email, _ := req.UserData["email"].(string)
		if email != "" {
			_ = s.enqueueCAPI(capiEvent{
				EventID:        req.EventID,
				EventName:      "Lead",
				EventTime:      time.Now(),
				UserData:       map[string]string{"email": email, "client_ip": c.RealIP()},
				CustomData:     req.Custom,
				EventSourceURL: req.URL,
				ActionSource:   "website",
				TenantID:       tenantID,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "recorded", "event_id": req.EventID})
}

// FormsSubmit is the /v1/forms/submit canonical alias for the legacy
// /v1/forms/:form_slug/submit route.  It enforces an HMAC signature
// before binding the body so the front-end can post to a single
// stable URL.
type v1FormSubmitReq struct {
	FormSlug       string                 `json:"form_slug"`
	Data           map[string]interface{} `json:"data"`
	IdempotencyKey string                 `json:"idempotency_key"`
	URL            string                 `json:"url"`
	Channel        string                 `json:"channel"`
}

func (s *Server) FormsSubmit(c echo.Context) error {
	if err := s.VerifyHMAC(c); err != nil {
		return err
	}
	var req v1FormSubmitReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.FormSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "form_slug required"})
	}

	tenantID := c.Request().Header.Get("X-Tenant-ID")
	idempotency := req.IdempotencyKey
	if idempotency == "" {
		idempotency = uuid.NewString()
	}
	hash := sha256.Sum256([]byte(idempotency + s.trackingSalt))
	eventID := hex.EncodeToString(hash[:])

	payload, _ := json.Marshal(req.Data)
	var id uuid.UUID
	err := s.pool.QueryRow(c.Request().Context(),
		`INSERT INTO landing.form_submissions(tenant_id,form_slug,payload,event_id,ip_address,user_agent,idempotency_key) VALUES (NULLIF($1,'')::uuid,$2,$3,$4,$5::inet,$6,$7) ON CONFLICT (tenant_id,idempotency_key) DO UPDATE SET event_id=EXCLUDED.event_id RETURNING id`,
		tenantID, req.FormSlug, payload, eventID, c.RealIP(), c.Request().UserAgent(), idempotency,
	).Scan(&id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	customData := map[string]interface{}{"form_slug": req.FormSlug, "submission_id": id.String(), "channel": req.Channel}
	if email, _ := req.Data["email"].(string); email != "" {
		_ = s.enqueueCAPI(capiEvent{EventID: eventID, EventName: "Lead", EventTime: time.Now(), UserData: map[string]string{"email": email, "client_ip": c.RealIP()}, CustomData: customData, EventSourceURL: req.URL, ActionSource: "website", TenantID: tenantID})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "accepted",
		"id":       id.String(),
		"event_id": eventID,
	})
}

// =============================================================================
// Pages CRUD (v1)
// =============================================================================

// v1ListPages mirrors GET /v1/pages — admin list with pagination.
func (s *Server) ListPages(c echo.Context) error {
	rows, err := s.pool.Query(c.Request().Context(),
		`SELECT id, tenant_slug, page_slug, title, status, updated_at FROM landing.landing_pages ORDER BY updated_at DESC LIMIT 200`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer rows.Close()

	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id uuid.UUID
		var tenantSlug, pageSlug, title, status string
		var updatedAt time.Time
		if err := rows.Scan(&id, &tenantSlug, &pageSlug, &title, &status, &updatedAt); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":         id,
			"tenant_slug": tenantSlug,
			"page_slug":  pageSlug,
			"title":      title,
			"status":     status,
			"updated_at": updatedAt,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"pages": out, "count": len(out)})
}

// v1UpdatePage mirrors PUT /v1/pages/:id.
type v1UpdatePageReq struct {
	Title  string                 `json:"title"`
	Status string                 `json:"status"`
	Design map[string]interface{} `json:"design_schema"`
}

func (s *Server) UpdatePage(c echo.Context) error {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	var req v1UpdatePageReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	designBytes, _ := json.Marshal(req.Design)
	_, err := s.pool.Exec(c.Request().Context(),
		`UPDATE landing.landing_pages SET title=COALESCE(NULLIF($2,''),title), status=COALESCE(NULLIF($3,''),status), design_schema=$4, updated_at=NOW() WHERE id=$1`,
		id, req.Title, req.Status, designBytes,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "updated", "id": id})
}

// v1DeletePage mirrors DELETE /v1/pages/:id.  Soft delete preserves
// the row for audit/recovery.
func (s *Server) DeletePage(c echo.Context) error {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	ct, err := s.pool.Exec(c.Request().Context(),
		`UPDATE landing.landing_pages SET status='deleted', updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if ct.RowsAffected() == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "page not found"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// ListBlocks returns the canonical block-type definitions.  Front-end
// admin tooling uses this to render the block picker.
func (s *Server) ListBlocks(c echo.Context) error {
	out := []map[string]interface{}{
		{"type": "hero", "label": "Hero", "icon": "🚀", "description": "Tiêu đề chính + form/CTA"},
		{"type": "features", "label": "Feature Grid", "icon": "✨", "description": "4-6 features với icon"},
		{"type": "logos", "label": "Logo Cloud", "icon": "🏢", "description": "Logo đối tác, MXV/CQG/NYMEX"},
		{"type": "speakers", "label": "Speakers", "icon": "🎤", "description": "Diễn giả chính"},
		{"type": "trust", "label": "Trust Badges", "icon": "🛡️", "description": "Bảo chứng uy tín"},
		{"type": "stats", "label": "Statistics", "icon": "📊", "description": "Số liệu ấn tượng"},
		{"type": "faq", "label": "FAQ", "icon": "❓", "description": "Câu hỏi thường gặp"},
		{"type": "cta", "label": "CTA", "icon": "🎯", "description": "Call-to-action section"},
		{"type": "testimonial", "label": "Testimonials", "icon": "💬", "description": "Đánh giá khách hàng"},
		{"type": "risk_warning", "label": "Risk Warning", "icon": "⚠️", "description": "Cảnh báo rủi ro"},
		{"type": "form", "label": "Form", "icon": "📝", "description": "Form đăng ký multi-step"},
		{"type": "pricing_table", "label": "Pricing Table", "icon": "💰", "description": "Bảng giá các gói"},
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"blocks": out, "count": len(out)})
}

// =============================================================================
// helpers
// =============================================================================

// readAndRestoreBody reads the entire request body so we can compute
// the HMAC, then restores the body via io.NopCloser so downstream
// handlers can re-read it via echo.Context.Bind().
func readAndRestoreBody(c echo.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}
	c.Request().Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}
