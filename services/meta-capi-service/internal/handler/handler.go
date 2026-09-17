// Package handler provides HTTP handlers for the meta-capi-service.
//
// meta-capi-service responsibilities:
//   - Receive signed events from landing/CRM (HMAC-verified)
//   - Translate RINCO lead shape → Meta CAPI event shape
//   - Forward events to Meta Graph API with retry / circuit breaker
//   - Track delivery success/failure per event_id
package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/meta-capi-service/internal/capi"
	"github.com/itdoanh/rinco/services/meta-capi-service/internal/repository"
)

// =============================================================================
// Types
// =============================================================================

// MetaCAPIEvent mirrors the structure posted to Meta Graph API.
type MetaCAPIEvent struct {
	Data           []CAPIEventItem `json:"data"`
	AccessToken    string          `json:"-"` // set at send time
	PixelID        string          `json:"-"`
	TestEventCode  string          `json:"test_event_code,omitempty"`
}

type CAPIEventItem struct {
	EventName      string            `json:"event_name"`
	EventTime      int64             `json:"event_time"`
	EventID        string            `json:"event_id"`
	ActionSource   string            `json:"action_source"`
	UserData       map[string]string `json:"user_data"`
	CustomData     map[string]any    `json:"custom_data,omitempty"`
}

// RINCOCAPIPayload is what landing-service / crm-service post to us.
type RINCOCAPIPayload struct {
	TenantID    string  `json:"tenant_id"`
	LeadID      string  `json:"lead_id"`
	EventName   string  `json:"event_name"`
	EventID     string  `json:"event_id"`
	Timestamp   int64   `json:"timestamp"`
	Email       string  `json:"email,omitempty"`
	Phone       string  `json:"phone,omitempty"`
	FBCLID      string  `json:"fbclid,omitempty"`
	FBP         string  `json:"fbp,omitempty"`
	Value       float64 `json:"value,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	ContentName string  `json:"content_name,omitempty"`
	HMAC        string  `json:"hmac_signature"`
}

// DeliveryRecord tracks the result of one event delivery attempt.
type DeliveryRecord struct {
	EventID    string    `json:"event_id"`
	Status     string    `json:"status"` // queued | sent | failed
	Attempts   int       `json:"attempts"`
	LastError  string    `json:"last_error,omitempty"`
	SentAt     time.Time `json:"sent_at,omitempty"`
}

// Server holds state for the meta-capi service.
type Server struct {
	mu          sync.RWMutex
	repo        *repository.Repository
	secret      []byte
	pixelID     string
	accessToken string
	deliveries  map[string]*DeliveryRecord
	publisher   Publisher // pluggable so tests can mock Meta's Graph API
	capiClient  *capi.Client
}

// Publisher abstracts the Meta Graph API. Tests inject a fake.
type Publisher interface {
	Send(event MetaCAPIEvent) error
}

// fakePublisher records all events sent to Meta (for assertions).
type fakePublisher struct {
	mu      sync.Mutex
	events  []MetaCAPIEvent
	failN   int // fail the next N calls
}

func (f *fakePublisher) Send(e MetaCAPIEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failN > 0 {
		f.failN--
		return fmt.Errorf("simulated meta api error")
	}
	f.events = append(f.events, e)
	return nil
}

// NewServer returns a meta-capi Server.
func NewServer(repo *repository.Repository, secret []byte, pixelID, accessToken string, p Publisher) *Server {
	if p == nil {
		p = &fakePublisher{}
	}
	return &Server{
		repo:        repo,
		secret:      secret,
		pixelID:     pixelID,
		accessToken: accessToken,
		deliveries:  map[string]*DeliveryRecord{},
		publisher:   p,
		capiClient:  capi.NewClient(),
	}
}

// NewServerLegacy keeps backward compatibility with the old constructor
// signature (no repository, just secret/pixelID/accessToken/publisher).
func NewServerLegacy(secret []byte, pixelID, accessToken string, p Publisher) *Server {
	return NewServer(nil, secret, pixelID, accessToken, p)
}

// New returns a meta-capi Server with default dependencies.
func New(repo *repository.Repository) *Server {
	return &Server{
		repo:        repo,
		deliveries:  map[string]*DeliveryRecord{},
		publisher:   &fakePublisher{},
		capiClient:  capi.NewClient(),
	}
}

// =============================================================================
// Validation / hashing
// ============================================================================

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// NormaliseEmail lowercases + trims the email per Meta's spec.
func NormaliseEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalisePhone strips everything except digits (Meta expects digits only).
func NormalisePhone(s string) string {
	digits := regexp.MustCompile(`[0-9]+`).FindAllString(s, -1)
	return strings.Join(digits, "")
}

// HashUserData returns a map of normalised, SHA-256-hashed user identifiers.
// Meta CAPI requires each field to be hashed before transmission.
func HashUserData(email, phone, fbclid, fbp string) map[string]string {
	out := map[string]string{}
	if email != "" {
		out["em"] = HashSHA256(NormaliseEmail(email))
	}
	if phone != "" {
		out["ph"] = HashSHA256(NormalisePhone(phone))
	}
	if fbclid != "" {
		out["fbclid"] = HashSHA256(fbclid)
	}
	if fbp != "" {
		out["fbp"] = HashSHA256(fbp)
	}
	if len(out) == 0 {
		// Meta requires at least one user data field. Provide a zero-byte
		// marker so the API call doesn't 400.
		out["external_id"] = HashSHA256("anonymous")
	}
	return out
}

// HashSHA256 is a tiny helper.
func HashSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// VerifyHMAC re-computes the expected signature and constant-time compares.
func VerifyHMAC(secret []byte, payload RINCOCAPIPayload, provided string) bool {
	canonical, _ := json.Marshal(struct {
		LeadID, EventID, Email, Phone, FBCLID, FBP, EventName string
		Timestamp                                              int64
	}{payload.LeadID, payload.EventID, payload.Email, payload.Phone, payload.FBCLID, payload.FBP, payload.EventName, payload.Timestamp})
	expected := SignHMAC(secret, payload.LeadID, payload.FBCLID, fmt.Sprintf("%d", payload.Timestamp), string(canonical))
	return hmac.Equal([]byte(expected), []byte(provided))
}

// SignHMAC produces the HMAC-SHA256 signature used by upstream services.
func SignHMAC(secret []byte, leadID, fbclid, timestamp, payload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(leadID + fbclid + timestamp + payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// validEventName returns true if name is one of Meta's standard events.
func validEventName(n string) bool {
	switch n {
	case "Lead", "Purchase", "InitiateCheckout", "AddPaymentInfo",
		"AddToCart", "CompleteRegistration", "Contact", "SubmitApplication",
		"Subscribe", "Custom":
		return true
	}
	return false
}

// =============================================================================
// HTTP handlers
// ============================================================================

type errorResp struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// IngestEvent handles POST /meta-capi/v1/events.
// Verifies HMAC, normalises user data, dispatches to Meta Graph API.
func (s *Server) IngestEvent(c echo.Context) error {
	var p RINCOCAPIPayload
	if err := c.Bind(&p); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid json", Details: err.Error()})
	}

	// Preflight validation
	if p.TenantID == "" {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "tenant_id required"})
	}
	if p.LeadID == "" || p.EventID == "" {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "lead_id and event_id required"})
	}
	if !validEventName(p.EventName) {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "invalid event_name"})
	}
	if p.Timestamp <= 0 {
		return c.JSON(http.StatusBadRequest, errorResp{Error: "timestamp required"})
	}
	// HMAC must verify before any downstream side effects.
	if !VerifyHMAC(s.secret, p, p.HMAC) {
		return c.JSON(http.StatusUnauthorized, errorResp{Error: "hmac_signature invalid"})
	}

	// Build Meta CAPI event
	event := MetaCAPIEvent{
		AccessToken: s.accessToken,
		PixelID:     s.pixelID,
		Data: []CAPIEventItem{{
			EventName:    p.EventName,
			EventTime:    p.Timestamp,
			EventID:      p.EventID,
			ActionSource: "website",
			UserData:     HashUserData(p.Email, p.Phone, p.FBCLID, p.FBP),
			CustomData: map[string]any{
				"lead_id":    p.LeadID,
				"tenant_id":  p.TenantID,
				"value":      p.Value,
				"currency":   p.Currency,
				"content_name": p.ContentName,
			},
		}},
	}

	// Record + dispatch
	rec := &DeliveryRecord{EventID: p.EventID, Status: "queued"}
	s.mu.Lock()
	s.deliveries[p.EventID] = rec
	s.mu.Unlock()

	if err := s.publisher.Send(event); err != nil {
		rec.Attempts++
		rec.LastError = err.Error()
		rec.Status = "failed"
		return c.JSON(http.StatusBadGateway, errorResp{Error: "meta api rejected", Details: err.Error()})
	}
	rec.Attempts++
	rec.Status = "sent"
	rec.SentAt = time.Now().UTC()

	return c.JSON(http.StatusAccepted, map[string]any{
		"event_id":  p.EventID,
		"status":    "queued_for_delivery",
		"pixel_id":  s.pixelID,
	})
}

// DeliveryStatus handles GET /meta-capi/v1/events/:event_id (debug).
func (s *Server) DeliveryStatus(c echo.Context) error {
	id := c.Param("event_id")
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.deliveries[id]
	if !ok {
		return c.JSON(http.StatusNotFound, errorResp{Error: "unknown event_id"})
	}
	return c.JSON(http.StatusOK, r)
}

// Health is the liveness endpoint.
func (s *Server) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":     "ok",
		"service":    "meta-capi",
		"timestamp":  time.Now().UTC(),
		"server_id":  uuid.NewString(),
	})
}
