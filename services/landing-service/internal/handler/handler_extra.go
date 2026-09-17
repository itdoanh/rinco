// Package handler provides HTTP handlers for the landing-service.
//
// landing-service responsibilities:
//   - Render landing pages (Next.js BFF proxy in production)
//   - Ingest lead-form submissions
//   - Compute HMAC signature for downstream services (meta-capi)
//   - Trigger NATS event for downstream processing
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// =============================================================================
// Additional handlers for landing service
// =============================================================================

// CreatePage handles POST /v1/pages — create a new landing page.
type createPageReq struct {
	TenantSlug  string                 `json:"tenant_slug"`
	PageSlug    string                 `json:"page_slug"`
	Title       string                 `json:"title"`
	Design      map[string]interface{} `json:"design_schema"`
	Status      string                 `json:"status"`
	Meta        map[string]string      `json:"meta"`
}

func (s *Server) CreatePage(c echo.Context) error {
	var req createPageReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.TenantSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_slug required"})
	}
	if req.PageSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "page_slug required"})
	}
	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title required"})
	}

	tenantID, ok := s.tenantBySlug[req.TenantSlug]
	if !ok {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	id := uuid.New()
	status := req.Status
	if status == "" {
		status = "draft"
	}
	designBytes, _ := json.Marshal(req.Design)
	metaBytes, _ := json.Marshal(req.Meta)

	var pageID uuid.UUID
	err := s.pool.QueryRow(c.Request().Context(),
		`INSERT INTO landing.landing_pages (id, tenant_id, slug, title, meta, design_schema, status)
		 VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6, $7)
		 ON CONFLICT (tenant_id, slug) DO UPDATE SET title=EXCLUDED.title, design_schema=EXCLUDED.design_schema, updated_at=NOW()
		 RETURNING id`,
		id, tenantID, req.PageSlug, req.Title, metaBytes, designBytes, status,
	).Scan(&pageID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":          pageID,
		"tenant_slug": req.TenantSlug,
		"page_slug":   req.PageSlug,
		"title":       req.Title,
		"status":      status,
	})
}

// GetPage handles GET /v1/pages/:id — get a single page by ID.
func (s *Server) GetPage(c echo.Context) error {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var tenantID uuid.UUID
	var slug, title, status string
	var designSchema, meta json.RawMessage
	var createdAt, updatedAt time.Time

	err := s.pool.QueryRow(c.Request().Context(),
		`SELECT tenant_id, slug, title, status, design_schema, meta, created_at, updated_at
		 FROM landing.landing_pages WHERE id=$1 AND status != 'deleted'`,
		id,
	).Scan(&tenantID, &slug, &title, &status, &designSchema, &meta, &createdAt, &updatedAt)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "page not found"})
	}

	var blocks []map[string]interface{}
	_ = json.Unmarshal(designSchema, &blocks)
	if blocks == nil {
		blocks = []map[string]interface{}{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":           id,
		"tenant_id":    tenantID,
		"slug":         slug,
		"title":        title,
		"status":       status,
		"blocks":       blocks,
		"meta":         meta,
		"created_at":   createdAt,
		"updated_at":   updatedAt,
	})
}

// PreviewPage handles GET /preview/:page_id — preview a page by ID (public, no auth).
func (s *Server) PreviewPage(c echo.Context) error {
	_ = c.Param("page_id")
	return s.GetPage(c)
}

// =============================================================================
// Form handlers
// =============================================================================

// SubmitForm handles POST /v1/forms/:form_slug/submit — submit a form.
type submitFormReq struct {
	Data map[string]interface{} `json:"data"`
}

func (s *Server) SubmitForm(c echo.Context) error {
	formSlug := c.Param("form_slug")
	if formSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "form_slug required"})
	}

	var req submitFormReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	tenantID := c.Request().Header.Get("X-Tenant-ID")
	eventID := uuid.New().String()

	payload, _ := json.Marshal(req.Data)
	var id uuid.UUID
	err := s.pool.QueryRow(c.Request().Context(),
		`INSERT INTO landing.form_submissions (tenant_id, form_slug, payload, event_id, ip_address, user_agent)
		 VALUES (NULLIF($1,'')::uuid, $2, $3, $4, $5::inet, $6)
		 RETURNING id`,
		tenantID, formSlug, payload, eventID, c.RealIP(), c.Request().UserAgent(),
	).Scan(&id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Enqueue CAPI event if email is present
	if email, _ := req.Data["email"].(string); email != "" {
		_ = s.enqueueCAPI(capiEvent{
			EventID:   eventID,
			EventName: "Lead",
			EventTime: time.Now(),
			UserData:  map[string]string{"email": email, "client_ip": c.RealIP()},
			CustomData: map[string]interface{}{
				"form_slug":     formSlug,
				"submission_id": id.String(),
			},
			ActionSource: "website",
			TenantID:   tenantID,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "accepted",
		"id":       id.String(),
		"event_id": eventID,
	})
}

// SubmitFormBatch handles POST /v1/forms/:form_slug/submit/batch — batch form submissions.
type batchFormReq struct {
	Submissions []map[string]interface{} `json:"submissions"`
}

func (s *Server) SubmitFormBatch(c echo.Context) error {
	formSlug := c.Param("form_slug")
	if formSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "form_slug required"})
	}

	var req batchFormReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	tenantID := c.Request().Header.Get("X-Tenant-ID")
	results := make([]map[string]interface{}, 0, len(req.Submissions))

	for _, data := range req.Submissions {
		eventID := uuid.New().String()
		payload, _ := json.Marshal(data)
		var id uuid.UUID
		err := s.pool.QueryRow(c.Request().Context(),
			`INSERT INTO landing.form_submissions (tenant_id, form_slug, payload, event_id, ip_address, user_agent)
			 VALUES (NULLIF($1,'')::uuid, $2, $3, $4, $5::inet, $6)
			 RETURNING id`,
			tenantID, formSlug, payload, eventID, c.RealIP(), c.Request().UserAgent(),
		).Scan(&id)
		if err != nil {
			results = append(results, map[string]interface{}{
				"status":  "error",
				"error":   err.Error(),
			})
			continue
		}

		// Enqueue CAPI event if email is present
		if email, _ := data["email"].(string); email != "" {
			_ = s.enqueueCAPI(capiEvent{
				EventID:   eventID,
				EventName: "Lead",
				EventTime: time.Now(),
				UserData:  map[string]string{"email": email, "client_ip": c.RealIP()},
				CustomData: map[string]interface{}{
					"form_slug":     formSlug,
					"submission_id": id.String(),
				},
				ActionSource: "website",
				TenantID:   tenantID,
			})
		}

		results = append(results, map[string]interface{}{
			"status":   "accepted",
			"id":       id.String(),
			"event_id": eventID,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"processed": len(results),
		"results":  results,
	})
}

// FormSchema handles GET /v1/forms/:form_slug/schema — get form schema.
func (s *Server) FormSchema(c echo.Context) error {
	formSlug := c.Param("form_slug")
	if formSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "form_slug required"})
	}

	// Return default schema based on form_slug pattern
	schema := map[string]interface{}{
		"form_slug": formSlug,
		"fields": []map[string]interface{}{
			{"name": "name", "type": "text", "label": "Họ và tên", "required": true},
			{"name": "phone", "type": "tel", "label": "Số điện thoại", "required": true},
			{"name": "email", "type": "email", "label": "Email", "required": false},
			{"name": "message", "type": "textarea", "label": "Nội dung", "required": false},
		},
	}

	return c.JSON(http.StatusOK, schema)
}

// =============================================================================
// Tracking handlers
// =============================================================================

// TrackPageview handles POST /v1/track/pageview — track a page view.
type pageviewReq struct {
	URL       string `json:"url"`
	SessionID string `json:"session_id"`
	Referrer  string `json:"referrer"`
}

func (s *Server) TrackPageview(c echo.Context) error {
	var req pageviewReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url required"})
	}

	tenantID := c.Request().Header.Get("X-Tenant-ID")
	eventID := uuid.New().String()

	utm := map[string]string{
		"source":   c.Request().Header.Get("X-UTM-Source"),
		"medium":   c.Request().Header.Get("X-UTM-Medium"),
		"campaign": c.Request().Header.Get("X-UTM-Campaign"),
	}
	utmBytes, _ := json.Marshal(utm)

	_, err := s.pool.Exec(c.Request().Context(),
		`INSERT INTO landing.tracking_events (tenant_id, event_type, event_id, page_slug, session_id, ip_address, user_agent, referer, utm)
		 VALUES (NULLIF($1,'')::uuid, 'page_view', $2, $3, $4, $5::inet, $6, $7, $8)`,
		tenantID, eventID, req.URL, req.SessionID, c.RealIP(), c.Request().UserAgent(), req.Referrer, utmBytes,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "recorded", "event_id": eventID})
}

// TrackConversion handles POST /v1/track/conversion — track a conversion event.
type conversionReq struct {
	EventName  string                 `json:"event_name"`
	EventID    string                 `json:"event_id"`
	PageSlug   string                 `json:"page_slug"`
	SessionID  string                 `json:"session_id"`
	Value      float64                `json:"value"`
	Currency   string                 `json:"currency"`
	CustomData map[string]interface{} `json:"custom_data"`
}

func (s *Server) TrackConversion(c echo.Context) error {
	var req conversionReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.EventName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "event_name required"})
	}

	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if req.EventID == "" {
		req.EventID = uuid.New().String()
	}

	customData, _ := json.Marshal(req.CustomData)
	_, err := s.pool.Exec(c.Request().Context(),
		`INSERT INTO landing.tracking_events (tenant_id, event_type, event_id, page_slug, session_id, payload, ip_address, user_agent)
		 VALUES (NULLIF($1,'')::uuid, $2, $3, $4, $5, $6, $7::inet, $8)`,
		tenantID, req.EventName, req.EventID, req.PageSlug, req.SessionID, customData, c.RealIP(), c.Request().UserAgent(),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Promote lead events to CAPI
	if req.EventName == "Lead" || req.EventName == "CompleteRegistration" {
		if email, _ := req.CustomData["email"].(string); email != "" {
			_ = s.enqueueCAPI(capiEvent{
				EventID:   req.EventID,
				EventName: "Lead",
				EventTime: time.Now(),
				UserData:  map[string]string{"email": email, "client_ip": c.RealIP()},
				CustomData: map[string]interface{}{
					"value":    req.Value,
					"currency": req.Currency,
				},
				ActionSource: "website",
				TenantID:   tenantID,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "recorded", "event_id": req.EventID})
}

// TrackingPixel handles GET /v1/track/pixel/:tenant_slug/p.gif — 1x1 tracking pixel.
func (s *Server) TrackingPixel(c echo.Context) error {
	tenantSlug := c.Param("tenant_slug")
	if tenantSlug == "" {
		tenantSlug = "default"
	}

	// Log the pageview from the tracking pixel
	tenantID, _ := s.tenantBySlug[tenantSlug]
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	eventID := uuid.New().String()
	url := c.QueryParam("url")
	sessionID := c.QueryParam("session_id")

	_, _ = s.pool.Exec(c.Request().Context(),
		`INSERT INTO landing.tracking_events (tenant_id, event_type, event_id, page_slug, session_id, ip_address, user_agent)
		 VALUES ($1::uuid, 'page_view', $2, $3, $4, $5::inet, $6)`,
		tenantID, eventID, url, sessionID, c.RealIP(), c.Request().UserAgent(),
	)

	// Return 1x1 transparent GIF
	c.Response().Header().Set("Content-Type", "image/gif")
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("Expires", "0")

	// 1x1 transparent GIF bytes
	gif := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b}
	return c.Blob(http.StatusOK, "image/gif", gif)
}

// TrackingRedirect handles GET /v1/track/redirect/:tenant_slug/:click_id — click tracking redirect.
func (s *Server) TrackingRedirect(c echo.Context) error {
	tenantSlug := c.Param("tenant_slug")
	clickID := c.Param("click_id")

	if tenantSlug == "" {
		tenantSlug = "default"
	}

	tenantID, _ := s.tenantBySlug[tenantSlug]
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	// Log the click event
	eventID := uuid.New().String()
	destURL := c.QueryParam("url")
	if destURL == "" {
		destURL = "/"
	}

	_, _ = s.pool.Exec(c.Request().Context(),
		`INSERT INTO landing.tracking_events (tenant_id, event_type, event_id, page_slug, session_id, ip_address, user_agent)
		 VALUES ($1::uuid, 'cta_click', $2, $3, $4, $5::inet, $6)`,
		tenantID, eventID, clickID, c.QueryParam("session_id"), c.RealIP(), c.Request().UserAgent(),
	)

	// Redirect to destination
	return c.Redirect(http.StatusFound, destURL)
}

// =============================================================================
// CAPI handlers
// =============================================================================

// CAPISend handles POST /v1/capi/send — send event to Meta CAPI.
type capiSendReq struct {
	EventID   string                 `json:"event_id"`
	EventName string                 `json:"event_name"`
	UserData  map[string]string      `json:"user_data"`
	CustomData map[string]interface{} `json:"custom_data"`
	URL       string                 `json:"url"`
}

func (s *Server) CAPISend(c echo.Context) error {
	var req capiSendReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.EventID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "event_id required"})
	}
	if req.EventName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "event_name required"})
	}

	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	// Enqueue CAPI event
	_ = s.enqueueCAPI(capiEvent{
		EventID:        req.EventID,
		EventName:      req.EventName,
		EventTime:      time.Now(),
		UserData:       req.UserData,
		CustomData:     req.CustomData,
		EventSourceURL: req.URL,
		ActionSource:   "website",
		TenantID:       tenantID,
	})

	return c.JSON(http.StatusAccepted, map[string]string{
		"status":   "queued",
		"event_id": req.EventID,
	})
}

// CAPIStatus handles GET /v1/capi/status — get CAPI status.
func (s *Server) CAPIStatus(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	var totalEvents int64
	if s.pool != nil {
		_ = s.pool.QueryRow(c.Request().Context(),
			`SELECT COUNT(*) FROM landing.tracking_events WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'`,
			tenantID,
		).Scan(&totalEvents)
	}

	queueLen := 0
	if s.capiQueue != nil {
		queueLen = len(s.capiQueue)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":           "active",
		"tenant_id":        tenantID,
		"total_events_24h": totalEvents,
		"queue_depth":      queueLen,
	})
}

// CAPITest handles POST /v1/capi/test — test CAPI connectivity.
func (s *Server) CAPITest(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	// Send a test event
	testEventID := uuid.New().String()
	_ = s.enqueueCAPI(capiEvent{
		EventID:   testEventID,
		EventName: "TestEvent",
		EventTime: time.Now(),
		UserData:  map[string]string{"test": "ping"},
		CustomData: map[string]interface{}{
			"test": true,
			"service": "landing-service",
		},
		ActionSource: "website",
		TenantID:   tenantID,
	})

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"test_event_id":  testEventID,
		"queue_depth":    len(s.capiQueue),
	})
}

// CAPIConversion handles POST /internal/landing.v1.LandingService/Conversion — CRM conversion event.
type conversionEventReq struct {
	LeadID     string  `json:"lead_id"`
	EventID    string  `json:"event_id"`
	EventName  string  `json:"event_name"`
	DealID     string  `json:"deal_id"`
	Value      float64 `json:"value"`
	Currency   string  `json:"currency"`
	Email      string  `json:"email"`
	Phone      string  `json:"phone"`
	FBCLID     string  `json:"fbclid"`
}

func (s *Server) CAPIConversion(c echo.Context) error {
	var req conversionEventReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	// Determine event name
	eventName := req.EventName
	if eventName == "" {
		eventName = "Purchase"
	}

	if req.EventID == "" {
		req.EventID = uuid.New().String()
	}

	// Enqueue CAPI conversion event
	userData := map[string]string{}
	if req.Email != "" {
		userData["email"] = req.Email
	}
	if req.Phone != "" {
		userData["phone"] = req.Phone
	}
	if req.FBCLID != "" {
		userData["fbclid"] = req.FBCLID
	}

	_ = s.enqueueCAPI(capiEvent{
		EventID:   req.EventID,
		EventName: eventName,
		EventTime: time.Now(),
		UserData:  userData,
		CustomData: map[string]interface{}{
			"lead_id":  req.LeadID,
			"deal_id":  req.DealID,
			"value":    req.Value,
			"currency": req.Currency,
		},
		ActionSource: "website",
		TenantID:   tenantID,
	})

	return c.JSON(http.StatusOK, map[string]string{
		"status":   "queued",
		"event_id": req.EventID,
	})
}

// =============================================================================
// Health check handler
// =============================================================================

// Health handles GET /health — service health check.
func (s *Server) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"service":   "landing-service",
		"timestamp": time.Now().UTC(),
		"pool":      s.pool != nil,
		"mongo":     s.mongo != nil,
		"store":     s.store != nil,
	})
}

// =============================================================================
// Utility functions
// =============================================================================

// readBody reads and restores request body for HMAC verification.
func readBody(c echo.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}
	c.Request().Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// helper to extract string from map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
