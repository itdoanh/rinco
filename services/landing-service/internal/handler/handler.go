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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/itdoanh/rinco/services/landing-service/internal/storage"
)

// =============================================================================
// Types
// =============================================================================

// Lead is the normalised representation of a form submission.
type Lead struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Email      string    `json:"email"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone,omitempty"`
	Source     string    `json:"source,omitempty"`
	UTMSource  string    `json:"utm_source,omitempty"`
	UTMMedium  string    `json:"utm_medium,omitempty"`
	UTMCampaign string   `json:"utm_campaign,omitempty"`
	FBCLID     string    `json:"fbclid,omitempty"`
	FBP        string    `json:"fbp,omitempty"`
	IP         string    `json:"ip,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	FormFields map[string]string `json:"form_fields,omitempty"`
	HMAC       string    `json:"hmac_signature"`
	EventID    string    `json:"event_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// Server holds deps. In production: ScyllaDB session, NATS publisher, etc.
type Server struct {
	mu           sync.RWMutex
	pool         *pgxpool.Pool
	mongo        *mongo.Client
	store        *storage.TieredStore
	leads        map[string]*Lead
	hmacKey      string
	trackingSalt string
	tenantBySlug map[string]string
	// CAPI dispatch queue (also exposed as `queue` for tests)
	capiQueue chan capiEvent
	queue     chan capiEvent
	closeFn   func()
}

// capiEvent is the internal event shape used to enqueue CAPI work.
type capiEvent struct {
	EventID        string
	EventName      string
	EventTime      time.Time
	UserData       map[string]string
	CustomData     map[string]interface{}
	EventSourceURL string
	ActionSource   string
	TenantID       string
}

// New returns a new Server with all dependencies wired.
func New(pool *pgxpool.Pool, mongoClient *mongo.Client, store *storage.TieredStore, hmacKey, trackingSalt string) *Server {
	if hmacKey == "" {
		hmacKey = "rinco-default-landing-key"
	}
	if trackingSalt == "" {
		trackingSalt = hmacKey
	}
	q := make(chan capiEvent, 1000)
	s := &Server{
		pool:          pool,
		mongo:         mongoClient,
		store:         store,
		leads:         map[string]*Lead{},
		hmacKey:       hmacKey,
		trackingSalt:  trackingSalt,
		tenantBySlug:  map[string]string{"default": "00000000-0000-0000-0000-000000000001"},
		capiQueue:     q,
		queue:         q,
	}
	// Start background CAPI dispatcher
	s.startCAPIDispatcher()
	return s
}

// NewServer is the legacy constructor kept for backward compatibility with
// existing tests and callers.
func NewServer(hmacKey []byte) *Server {
	key := string(hmacKey)
	if key == "" {
		key = "rinco-default-landing-key"
	}
	return New(nil, nil, nil, key, key)
}

// startCAPIDispatcher drains the CAPI queue and forwards events to meta-capi-service.
func (s *Server) startCAPIDispatcher() {
	ctx, cancel := context.WithCancel(context.Background())
	s.closeFn = cancel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-s.capiQueue:
				s.dispatchCAPI(ctx, ev)
			}
		}
	}()
}

// enqueueCAPI adds an event to the CAPI dispatch queue.
func (s *Server) enqueueCAPI(ev capiEvent) bool {
	select {
	case s.capiQueue <- ev:
		return true
	default:
		// Queue full — log and drop
		return false
	}
}

// dispatchCAPI forwards a single CAPI event to meta-capi-service via HTTP.
func (s *Server) dispatchCAPI(ctx context.Context, ev capiEvent) {
	metaCAPIURL := os.Getenv("META_CAPI_SERVICE_URL")
	if metaCAPIURL == "" {
		metaCAPIURL = "http://localhost:8093"
	}

	payload := map[string]interface{}{
		"tenant_id":  ev.TenantID,
		"event_id":   ev.EventID,
		"event_name": ev.EventName,
		"timestamp":  ev.EventTime.Unix(),
		"user_data":  ev.UserData,
		"custom_data": ev.CustomData,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metaCAPIURL+"/v1/capi/lead", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("CAPI: failed to create request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", ev.TenantID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("CAPI: failed to send event %s: %v\n", ev.EventID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("CAPI: server returned %d for event %s: %s\n", resp.StatusCode, ev.EventID, string(body))
	}
}

// Close gracefully shuts down background workers.
func (s *Server) Close() {
	if s.closeFn != nil {
		s.closeFn()
	}
}

// MongoInit is a no-op stub kept for backward compat with existing callers.
// Real implementations can perform MongoDB setup here.
func (s *Server) MongoInit(ctx context.Context) error {
	return nil
}

// =============================================================================
// Validation helpers
// =============================================================================

var phoneRegex = regexp.MustCompile(`^\+?[0-9]{8,15}$`)

func validEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil && strings.Contains(s, "@")
}

func validPhone(s string) bool {
	return phoneRegex.MatchString(s)
}

// normaliseEmail trims and lowercases an email per Meta's CAPI spec.
func normaliseEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// normalisePhone strips all non-digit chars except leading '+'.
func normalisePhone(s string) string {
	s = strings.TrimSpace(s)
	hasPlus := strings.HasPrefix(s, "+")
	digits := regexp.MustCompile(`[0-9]+`).FindAllString(s, -1)
	joined := strings.Join(digits, "")
	if hasPlus {
		return "+" + joined
	}
	return joined
}

// HashSHA256 returns the hex SHA-256 of s (used for Meta CAPI user data).
func HashSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// SignHMAC computes the signature over lead_id + fbclid + timestamp + payload.
func SignHMAC(secret []byte, leadID, fbclid, timestamp, payload string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(leadID + fbclid + timestamp + payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateEventID returns a UUIDv7-shaped string. We use UUIDv4 here for
// simplicity (real implementation uses google/uuid with the v7 option).
func GenerateEventID() string {
	return uuid.NewString()
}

// =============================================================================
// HTTP handlers
// =============================================================================

type leadSubmitReq struct {
	TenantSlug  string            `json:"tenant_slug"`
	Email       string            `json:"email"`
	FullName    string            `json:"full_name"`
	Phone       string            `json:"phone,omitempty"`
	Source      string            `json:"source,omitempty"`
	UTMSource   string            `json:"utm_source,omitempty"`
	UTMMedium   string            `json:"utm_medium,omitempty"`
	UTMCampaign string            `json:"utm_campaign,omitempty"`
	FBCLID      string            `json:"fbclid,omitempty"`
	FBP         string            `json:"fbp,omitempty"`
	FormFields  map[string]string `json:"form_fields,omitempty"`
}

type leadResp struct {
	ID         string    `json:"id"`
	EventID    string    `json:"event_id"`
	HMAC       string    `json:"hmac_signature"`
	ReceivedAt time.Time `json:"received_at"`
}

// SubmitLead handles POST /landing/v1/leads.
func (s *Server) SubmitLead(c echo.Context) error {
	var req leadSubmitReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json", "details": err.Error()})
	}

	// Preflight validation
	if req.TenantSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_slug required"})
	}
	if !validEmail(req.Email) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email"})
	}
	if req.FullName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "full_name required"})
	}
	if req.Phone != "" && !validPhone(req.Phone) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid phone"})
	}

	tenantID, ok := s.tenantBySlug[req.TenantSlug]
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "unknown tenant"})
	}

	now := time.Now().UTC()
	lead := &Lead{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		Email:      normaliseEmail(req.Email),
		FullName:   strings.TrimSpace(req.FullName),
		Phone:      normalisePhone(req.Phone),
		Source:     req.Source,
		UTMSource:  req.UTMSource,
		UTMMedium:  req.UTMMedium,
		UTMCampaign: req.UTMCampaign,
		FBCLID:     req.FBCLID,
		FBP:        req.FBP,
		FormFields: req.FormFields,
		EventID:    GenerateEventID(),
		CreatedAt:  now,
	}

	// Compute HMAC over a canonical payload representation.
	canonical, _ := json.Marshal(struct {
		ID, Email, FullName, Phone, FBCLID string
	}{lead.ID, lead.Email, lead.FullName, lead.Phone, lead.FBCLID})
	lead.HMAC = SignHMAC([]byte(s.hmacKey), lead.ID, lead.FBCLID, fmt.Sprintf("%d", now.Unix()), string(canonical))

	s.mu.Lock()
	s.leads[lead.ID] = lead
	s.mu.Unlock()

	return c.JSON(http.StatusCreated, lead)
}

// GetLead handles GET /landing/v1/leads/:id (admin/debug).
func (s *Server) GetLead(c echo.Context) error {
	id := c.Param("id")
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.leads[id]
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	return c.JSON(http.StatusOK, l)
}

// RenderPage handles GET /landing/v1/pages/:slug (returns JSON placeholder;
// real impl proxies to Next.js renderer).
func (s *Server) RenderPage(c echo.Context) error {
	slug := c.Param("slug")
	if slug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "slug required"})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"slug":    slug,
		"blocks":  []any{},
		"version": 1,
	})
}

// TrackEvent handles POST /landing/v1/track (clickstream / analytics).
type trackReq struct {
	Event   string `json:"event"`
	URL     string `json:"url"`
	AnonID  string `json:"anonymous_id,omitempty"`
}

func (s *Server) TrackEvent(c echo.Context) error {
	var req trackReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
	}
	if req.Event == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "event required"})
	}
	if req.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url required"})
	}
	return c.NoContent(http.StatusNoContent)
}
