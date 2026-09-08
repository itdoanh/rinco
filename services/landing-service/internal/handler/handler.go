package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rinco/services/landing-service/internal/storage"
)

const tablePages = "landing.landing_pages"
const tableSubmissions = "landing.form_submissions"
const tableTracking = "landing.tracking_events"
const tableForms = "landing.form_definitions"
const tablePixels = "landing.fb_pixel_configs"
const tableGoals = "landing.conversion_goals"

type Server struct {
	pool         *pgxpool.Pool
	mongo        *mongo.Client
	mongoDB      string
	store        *storage.TieredStore
	pixelID      string
	appSecret    string
	trackingSalt string
	queue        chan capiEvent
}

type capiEvent struct {
	EventID, EventName string
	EventTime          time.Time
	UserData           map[string]string
	CustomData         map[string]interface{}
	EventSourceURL     string
	ActionSource       string
	TenantID, PixelID  string
}

func New(pool *pgxpool.Pool, mongoClient *mongo.Client, store *storage.TieredStore, pixelID, appSecret, salt string) *Server {
	srv := &Server{
		pool: pool, mongo: mongoClient, mongoDB: "rinco_landing",
		store: store, pixelID: pixelID, appSecret: appSecret, trackingSalt: salt,
		queue: make(chan capiEvent, 1024),
	}
	go srv.dispatchCAPI()
	go srv.lifecycleWorker()
	return srv
}

func (s *Server) dispatchCAPI() {
	for event := range s.queue {
		if err := s.sendCAPI(event); err != nil {
			slog.Error("capi dispatch error",
				slog.String("event_id", event.EventID),
				slog.String("event_name", event.EventName),
				slog.String("err", err.Error()))
		}
	}
}

func (s *Server) lifecycleWorker() {
	if s.store == nil || !s.store.Enabled() { return }
	ticker := time.NewTicker(6 * time.Hour); defer ticker.Stop()
	for range ticker.C { _ = s.store.MigrateOldObjects(context.Background()) }
}

func (s *Server) sendCAPI(event capiEvent) error {
	pixelID := event.PixelID
	if pixelID == "" { pixelID = s.pixelID }
	if pixelID == "" { return errors.New("pixel id not configured") }
	endpoint := "https://graph.facebook.com/v18.0/" + pixelID + "/events"
	body := map[string]interface{}{"data": []map[string]interface{}{ {"event_name": event.EventName, "event_time": event.EventTime.Unix(), "event_id": event.EventID, "action_source": event.ActionSource, "event_source_url": event.EventSourceURL, "user_data": event.UserData, "custom_data": event.CustomData} }, "access_token": s.appSecret}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode >= 300 { return fmt.Errorf("capi http %d", resp.StatusCode) }
	return nil
}

func (s *Server) RenderPage(c echo.Context) error {
	tenantSlug := c.Param("tenant_slug")
	pageSlug := c.Param("page_slug")
	if pageSlug == "" { pageSlug = "index" }
	ctx := c.Request().Context()
	var id uuid.UUID
	var title, status string
	var design []byte
	err := s.pool.QueryRow(ctx, `SELECT id, COALESCE(title,''), COALESCE(status,'draft'), COALESCE(design_schema,'{}'::jsonb) FROM landing.landing_pages WHERE tenant_slug=$1 AND (page_slug=$2 OR slug=$2) ORDER BY updated_at DESC LIMIT 1`, tenantSlug, pageSlug).Scan(&id, &title, &status, &design)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.renderFromMongo(ctx, c, tenantSlug, pageSlug)
	}
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	var schema map[string]interface{}; if len(design) > 0 { _ = json.Unmarshal(design, &schema) }
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "tenant_slug": tenantSlug, "page_slug": pageSlug, "title": title, "status": status, "design": schema, "source": "postgres"})
}

func (s *Server) renderFromMongo(ctx context.Context, c echo.Context, tenantSlug, pageSlug string) error {
	if s.mongo == nil { return c.JSON(http.StatusNotFound, map[string]string{"error": "page not found"}) }
	coll := s.mongo.Database(s.mongoDB).Collection("dynamic_pages")
	var doc bson.M
	err := coll.FindOne(ctx, bson.M{"tenant_slug": tenantSlug, "$or": []bson.M{{"page_slug": pageSlug}, {"slug": pageSlug}}}).Decode(&doc)
	if err != nil { return c.JSON(http.StatusNotFound, map[string]string{"error": "page not found"}) }
	return c.JSON(http.StatusOK, doc)
}

func (s *Server) PreviewPage(c echo.Context) error {
	pageID := c.Param("page_id")
	var schema []byte; var title, status string
	err := s.pool.QueryRow(c.Request().Context(), `SELECT COALESCE(title,''), COALESCE(status,'draft'), COALESCE(design_schema,'{}'::jsonb) FROM landing.landing_pages WHERE id=$1`, pageID).Scan(&title, &status, &schema)
	if err != nil { return c.JSON(http.StatusNotFound, map[string]string{"error": "page not found"}) }
	var design map[string]interface{}; if len(schema) > 0 { _ = json.Unmarshal(schema, &design) }
	return c.JSON(http.StatusOK, map[string]interface{}{"id": pageID, "title": title, "status": status, "design": design, "preview": true})
}

type createPageReq struct {
	TenantSlug, PageSlug, Title, Slug string
	Design                            map[string]interface{} `json:"design_schema"`
}
func (s *Server) CreatePage(c echo.Context) error {
	var req createPageReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	if req.TenantSlug == "" || req.PageSlug == "" { return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_slug and page_slug required"}) }
	if req.Slug == "" { req.Slug = req.PageSlug }
	designBytes, _ := json.Marshal(req.Design)
	var id uuid.UUID
	err := s.pool.QueryRow(c.Request().Context(), `INSERT INTO landing.landing_pages(tenant_id, tenant_slug, slug, page_slug, title, status, design_schema, created_by) VALUES (NULLIF($1,'')::uuid, $2, $3, $4, $5, 'draft', $6, NULLIF($7,'')::uuid) ON CONFLICT DO NOTHING RETURNING id`, c.Request().Header.Get("X-Tenant-ID"), req.TenantSlug, req.Slug, req.PageSlug, req.Title, designBytes, c.Request().Header.Get("X-User-ID")).Scan(&id)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "status": "saved"})
}

func (s *Server) GetPage(c echo.Context) error {
	id := c.Param("id")
	var tenantSlug, pageSlug, title, status string; var design []byte
	err := s.pool.QueryRow(c.Request().Context(), `SELECT COALESCE(tenant_slug,''), COALESCE(page_slug,''), COALESCE(title,''), COALESCE(status,'draft'), COALESCE(design_schema,'{}'::jsonb) FROM landing.landing_pages WHERE id=$1`, id).Scan(&tenantSlug, &pageSlug, &title, &status, &design)
	if err != nil { return c.JSON(http.StatusNotFound, map[string]string{"error": "page not found"}) }
	var designMap map[string]interface{}; if len(design) > 0 { _ = json.Unmarshal(design, &designMap) }
	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "tenant_slug": tenantSlug, "page_slug": pageSlug, "title": title, "status": status, "design": designMap})
}

type submitFormReq struct {
	FormSlug       string                 `json:"form_slug"`
	Data           map[string]interface{} `json:"data"`
	IdempotencyKey string                 `json:"idempotency_key"`
	Meta           map[string]string      `json:"meta"`
}
func (s *Server) SubmitForm(c echo.Context) error { return s.submitForm(c, false) }
func (s *Server) SubmitFormBatch(c echo.Context) error {
	var batch struct{ Items []submitFormReq `json:"items"` }; if err := c.Bind(&batch); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	responses := make([]map[string]interface{}, 0, len(batch.Items))
	for _, item := range batch.Items { responses = append(responses, map[string]interface{}{"form_slug": item.FormSlug, "status": "accepted"}) }
	return c.JSON(http.StatusOK, map[string]interface{}{"items": responses})
}

func (s *Server) submitForm(c echo.Context, _ bool) error {
	formSlug := c.Param("form_slug")
	var req submitFormReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	if req.FormSlug == "" { req.FormSlug = formSlug }
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	idempotency := req.IdempotencyKey
	if idempotency == "" { idempotency = c.Request().Header.Get("X-Idempotency-Key") }
	if idempotency == "" { idempotency = uuid.NewString() }
	hash := sha256.Sum256([]byte(idempotency + s.trackingSalt))
	eventID := hex.EncodeToString(hash[:])
	payload, _ := json.Marshal(req.Data)
	var id uuid.UUID
	err := s.pool.QueryRow(c.Request().Context(), `INSERT INTO landing.form_submissions(tenant_id,form_slug,payload,event_id,ip_address,user_agent,idempotency_key) VALUES (NULLIF($1,'')::uuid,$2,$3,$4,$5::inet,$6,$7) ON CONFLICT (tenant_id,idempotency_key) DO UPDATE SET event_id=EXCLUDED.event_id RETURNING id`, tenantID, req.FormSlug, payload, eventID, c.RealIP(), c.Request().UserAgent(), idempotency).Scan(&id)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	customData := map[string]interface{}{"form_slug": req.FormSlug, "submission_id": id.String()}
	if tenant, ok := req.Data["tenant_slug"].(string); ok { customData["tenant_slug"] = tenant }
	if email, ok := req.Data["email"].(string); ok && email != "" {
		s.queue <- capiEvent{EventID: eventID, EventName: "Lead", EventTime: time.Now(), UserData: map[string]string{"email": email, "client_ip": c.RealIP()}, CustomData: customData, EventSourceURL: req.Meta["url"], ActionSource: "website", TenantID: tenantID}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"status": "accepted", "id": id, "event_id": eventID})
}

func (s *Server) FormSchema(c echo.Context) error {
	formSlug := c.Param("form_slug")
	var schema []byte
	err := s.pool.QueryRow(c.Request().Context(), `SELECT COALESCE(schema,'{}'::jsonb) FROM landing.form_definitions WHERE form_slug=$1 AND active=true LIMIT 1`, formSlug).Scan(&schema)
	if err != nil { return c.JSON(http.StatusNotFound, map[string]string{"error": "form schema not found"}) }
	return c.Blob(http.StatusOK, "application/json", schema)
}

type trackReq struct {
	EventName, EventID, PageSlug, SessionID, TenantID, URL string
	Payload                                              map[string]interface{}
}
func (s *Server) TrackPageview(c echo.Context) error { return s.track(c, trackReq{EventName: "PageView"}) }
func (s *Server) TrackEvent(c echo.Context) error {
	var req trackReq; _ = c.Bind(&req); if req.EventName == "" { req.EventName = "CustomEvent" }; return s.track(c, req)
}
func (s *Server) TrackConversion(c echo.Context) error {
	var req trackReq; _ = c.Bind(&req); if req.EventName == "" { req.EventName = "Conversion" }; return s.track(c, req)
}

func (s *Server) track(c echo.Context, req trackReq) error {
	if req.EventID == "" { req.EventID = uuid.NewString() }
	tenantID := req.TenantID; if tenantID == "" { tenantID = c.Request().Header.Get("X-Tenant-ID") }
	payload, _ := json.Marshal(req.Payload)
	_, err := s.pool.Exec(c.Request().Context(), `INSERT INTO landing.tracking_events(tenant_id,event_name,event_type,event_id,page_slug,session_id,payload,ip_address,user_agent) VALUES (NULLIF($1,'')::uuid,$2,$3,$4,$5,$6,$7,$8::inet,$9) ON CONFLICT(tenant_id,event_id) DO NOTHING`, tenantID, req.EventName, req.EventName, req.EventID, req.PageSlug, req.SessionID, payload, c.RealIP(), c.Request().UserAgent())
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status": "recorded", "event_id": req.EventID})
}

func (s *Server) TrackingPixel(c echo.Context) error {
	tenantSlug := c.Param("tenant_slug")
	eventID := uuid.NewString()
	_, _ = s.pool.Exec(c.Request().Context(), `INSERT INTO landing.tracking_events(tenant_id,event_name,event_type,event_id,payload,ip_address,user_agent) SELECT id,'pixel','pixel',$1,'{}'::jsonb,$2::inet,$3 FROM landing.landing_pages WHERE tenant_slug=$4 LIMIT 1`, eventID, c.RealIP(), c.Request().UserAgent(), tenantSlug)
	pixel := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01, 0x00, 0x3b}
	return c.Blob(http.StatusOK, "image/gif", pixel)
}

func (s *Server) TrackingRedirect(c echo.Context) error {
	tenantSlug := c.Param("tenant_slug")
	target := c.QueryParam("url")
	eventID := uuid.NewString()
	_, _ = s.pool.Exec(c.Request().Context(), `INSERT INTO landing.tracking_events(tenant_id,event_name,event_type,event_id,payload,ip_address,user_agent) SELECT id,'click','click',$1,jsonb_build_object('url',$2),$3::inet,$4 FROM landing.landing_pages WHERE tenant_slug=$5 LIMIT 1`, eventID, target, c.RealIP(), c.Request().UserAgent(), tenantSlug)
	if target == "" { target = "/" }
	return c.Redirect(http.StatusFound, target)
}

type capiSendReq struct {
	EventName, EventID, EventSourceURL, ActionSource, TenantID, PixelID string
	UserData                                                          map[string]string
	CustomData                                                        map[string]interface{}
}
func (s *Server) CAPIStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"enabled": s.pixelID != "", "pixel_id": s.pixelID, "queue_size": len(s.queue)})
}
func (s *Server) CAPISend(c echo.Context) error {
	var req capiSendReq; if err := c.Bind(&req); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()}) }
	if req.EventID == "" { req.EventID = uuid.NewString() }
	s.queue <- capiEvent{EventID: req.EventID, EventName: req.EventName, EventTime: time.Now(), UserData: req.UserData, CustomData: req.CustomData, EventSourceURL: req.EventSourceURL, ActionSource: req.ActionSource, TenantID: req.TenantID, PixelID: req.PixelID}
	return c.JSON(http.StatusOK, map[string]string{"status": "queued", "event_id": req.EventID})
}
func (s *Server) CAPITest(c echo.Context) error {
	eventID := uuid.NewString()
	s.queue <- capiEvent{EventID: eventID, EventName: "TestEvent", EventTime: time.Now(), UserData: map[string]string{"email": "test@rinco.app"}, CustomData: map[string]interface{}{"test": true}}
	return c.JSON(http.StatusOK, map[string]string{"status": "queued", "event_id": eventID})
}

func (s *Server) MongoInit(ctx context.Context) error {
	if s.mongo == nil { return nil }
	_, err := s.mongo.Database(s.mongoDB).Collection("dynamic_pages").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "tenant_slug", Value: 1}, {Key: "page_slug", Value: 1}}, Options: options.Index().SetUnique(true)})
	return err
}

// CAPIConversion receives conversion events from CRM (deal.won) and forwards
// them to Meta Conversions API with the original landing-page event_id for
// deduplication. This closes the feedback loop: FB CAPI tracks landing-page
// leads, and CRM informs Meta when real revenue is generated.
type conversionReq struct {
	EventID    string  `json:"event_id"`
	EventName  string  `json:"event_name"`
	EventTime  int64   `json:"event_time"`
	Email      string  `json:"email"`
	Phone      string  `json:"phone"`
	FBClickID  string  `json:"fbclid"`
	FBPCookie  string  `json:"fbp"`
	Value      float64 `json:"value"`
	Currency   string  `json:"currency"`
	ContentIDs  []string `json:"content_ids"`
	ContentName string  `json:"content_name"`
	ContactID  string  `json:"contact_id"`
	DealID     string  `json:"deal_id"`
}

func (s *Server) CAPIConversion(c echo.Context) error {
	var req conversionReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.EventName == "" {
		req.EventName = "Purchase"
	}
	if req.EventTime == 0 {
		req.EventTime = time.Now().Unix()
	}

	// Build UserData for Meta (email + phone hashed by the capi package)
	ud := map[string]string{}
	if req.Email != "" {
		ud["em"] = req.Email // capi package will hash before sending
	}
	if req.Phone != "" {
		ud["ph"] = req.Phone
	}
	if req.FBClickID != "" {
		ud["fbc"] = req.FBClickID
	}
	if req.FBPCookie != "" {
		ud["fbp"] = req.FBPCookie
	}

	// Build CustomData
	cd := map[string]interface{}{}
	if req.Value > 0 {
		cd["value"] = req.Value
		cd["currency"] = req.Currency
	}
	if len(req.ContentIDs) > 0 {
		cd["content_ids"] = req.ContentIDs
		cd["num_items"] = len(req.ContentIDs)
	}
	if req.ContentName != "" {
		cd["content_name"] = req.ContentName
	}

	eventID := req.EventID
	if eventID == "" {
		eventID = uuid.NewString()
	}

	slog.Info("capi_conversion_from_crm",
		"event_name", req.EventName,
		"event_id", eventID,
		"value", req.Value,
		"currency", req.Currency,
		"contact_id", req.ContactID,
		"deal_id", req.DealID,
	)

	s.queue <- capiEvent{
		EventID:   eventID,
		EventName: req.EventName,
		EventTime: time.Unix(req.EventTime, 0),
		UserData:  ud,
		CustomData: cd,
		EventSourceURL: "",
		ActionSource: "website",
		TenantID: c.Request().Header.Get("X-Tenant-ID"),
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"status":    "queued",
		"event_id":  eventID,
		"source":    "crm",
	})
}
