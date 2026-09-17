// Unit tests for meta-capi-service.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

var testSecret = []byte("rinco-meta-capi-secret-2026")

func newCtx(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// signedPayload returns a payload + the matching HMAC for testing.
func signedPayload(t *testing.T) (string, RINCOCAPIPayload) {
	t.Helper()
	p := RINCOCAPIPayload{
		TenantID:  "tid",
		LeadID:    "lid",
		EventName: "Lead",
		EventID:   "eid",
		Timestamp: 1700000000,
		Email:     "User@Example.COM",
		Phone:     "+84 90 123 4567",
		FBCLID:    "fb.1.123.456",
		FBP:       "fbp.1.123.456",
	}
	canonical, _ := json.Marshal(struct {
		LeadID, EventID, Email, Phone, FBCLID, FBP, EventName string
		Timestamp                                              int64
	}{p.LeadID, p.EventID, p.Email, p.Phone, p.FBCLID, p.FBP, p.EventName, p.Timestamp})
	p.HMAC = SignHMAC(testSecret, p.LeadID, p.FBCLID, "1700000000", string(canonical))
	body, _ := json.Marshal(p)
	return string(body), p
}

// ============================================================================
// Normalisation
// ============================================================================

func TestNormaliseEmail(t *testing.T) {
	assert.Equal(t, "user@example.com", NormaliseEmail("  User@Example.COM  "))
	assert.Equal(t, "a@b.co", NormaliseEmail("A@B.CO"))
}

func TestNormalisePhone_StripsFormatting(t *testing.T) {
	assert.Equal(t, "84901234567", NormalisePhone("+84 90 123 4567"))
	assert.Equal(t, "0901234567", NormalisePhone("(090) 123-4567"))
	assert.Equal(t, "84901234567", NormalisePhone("+84-90-123-4567"))
}

func TestHashSHA256(t *testing.T) {
	h := HashSHA256("test@example.com")
	assert.Len(t, h, 64)
	assert.Equal(t, HashSHA256("test@example.com"), h)
}

// ============================================================================
// User data hashing
// ============================================================================

func TestHashUserData_AllFields(t *testing.T) {
	ud := HashUserData("a@b.co", "+84 90 123 4567", "fb.1.x.y", "fbp.1.x.y")
	assert.Equal(t, HashSHA256("a@b.co"), ud["em"])
	assert.Equal(t, HashSHA256("84901234567"), ud["ph"])
	assert.Equal(t, HashSHA256("fb.1.x.y"), ud["fbclid"])
	assert.Equal(t, HashSHA256("fbp.1.x.y"), ud["fbp"])
}

func TestHashUserData_Empty(t *testing.T) {
	ud := HashUserData("", "", "", "")
	assert.NotEmpty(t, ud, "Meta requires at least one user_data field; we synthesise one")
}

func TestHashUserData_OnlyEmail(t *testing.T) {
	ud := HashUserData("a@b.co", "", "", "")
	assert.Contains(t, ud, "em")
	assert.NotContains(t, ud, "ph")
}

// ============================================================================
// HMAC verification
// ============================================================================

func TestVerifyHMAC_ValidSignature(t *testing.T) {
	_, p := signedPayload(t)
	assert.True(t, VerifyHMAC(testSecret, p, p.HMAC))
}

func TestVerifyHMAC_TamperedSignature(t *testing.T) {
	_, p := signedPayload(t)
	p.HMAC = "deadbeef" + p.HMAC[8:]
	assert.False(t, VerifyHMAC(testSecret, p, p.HMAC))
}

func TestVerifyHMAC_WrongSecret(t *testing.T) {
	_, p := signedPayload(t)
	assert.False(t, VerifyHMAC([]byte("wrong-secret"), p, p.HMAC))
}

func TestVerifyHMAC_TamperedPayload(t *testing.T) {
	_, p := signedPayload(t)
	// Change email after signing
	p.Email = "attacker@b.co"
	assert.False(t, VerifyHMAC(testSecret, p, p.HMAC))
}

// ============================================================================
// Event name validation
// ============================================================================

func TestValidEventName(t *testing.T) {
	good := []string{"Lead", "Purchase", "InitiateCheckout", "Contact", "Custom"}
	bad := []string{"", "click", "pageview", "submit-form"}
	for _, e := range good {
		t.Run("accept/"+e, func(t *testing.T) { assert.True(t, validEventName(e)) })
	}
	for _, e := range bad {
		t.Run("reject/"+e, func(t *testing.T) { assert.False(t, validEventName(e)) })
	}
}

// ============================================================================
// IngestEvent
// ============================================================================

func TestIngestEvent_HappyPath(t *testing.T) {
	body, _ := signedPayload(t)
	fp := &fakePublisher{}
	s := NewServerLegacy(testSecret, "pixel-123", "access-tok", fp)

	c, rec := newCtx(http.MethodPost, "/meta-capi/v1/events", body)
	err := s.IngestEvent(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Len(t, fp.events, 1, "publisher should receive one event")

	got := fp.events[0]
	assert.Equal(t, "pixel-123", got.PixelID)
	assert.Equal(t, "access-tok", got.AccessToken)
	assert.Len(t, got.Data, 1)
	assert.Equal(t, "Lead", got.Data[0].EventName)
	assert.Equal(t, "eid", got.Data[0].EventID)
	assert.Contains(t, got.Data[0].UserData, "em")
}

func TestIngestEvent_HMACFailure(t *testing.T) {
	_, p := signedPayload(t)
	p.HMAC = "0" + strings.Repeat("0", 63)
	body2, _ := json.Marshal(p)

	fp := &fakePublisher{}
	s := NewServerLegacy(testSecret, "pixel-1", "tok", fp)
	c, rec := newCtx(http.MethodPost, "/meta-capi/v1/events", string(body2))
	_ = s.IngestEvent(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, fp.events, "publisher should not receive events with bad HMAC")
}

func TestIngestEvent_InvalidEventName(t *testing.T) {
	_, p := signedPayload(t)
	p.EventName = "pageview"
	canonical, _ := json.Marshal(struct {
		LeadID, EventID, Email, Phone, FBCLID, FBP, EventName string
		Timestamp                                              int64
	}{p.LeadID, p.EventID, p.Email, p.Phone, p.FBCLID, p.FBP, p.EventName, p.Timestamp})
	p.HMAC = SignHMAC(testSecret, p.LeadID, p.FBCLID, "1700000000", string(canonical))
	body2, _ := json.Marshal(p)

	fp := &fakePublisher{}
	s := NewServerLegacy(testSecret, "pixel", "tok", fp)
	c, rec := newCtx(http.MethodPost, "/meta-capi/v1/events", string(body2))
	_ = s.IngestEvent(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIngestEvent_MissingTenant(t *testing.T) {
	_, p := signedPayload(t)
	p.TenantID = ""
	body2, _ := json.Marshal(p)
	s := NewServerLegacy(testSecret, "pixel", "tok", &fakePublisher{})
	c, rec := newCtx(http.MethodPost, "/meta-capi/v1/events", string(body2))
	_ = s.IngestEvent(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIngestEvent_MetaAPIFailureReturns502(t *testing.T) {
	body, _ := signedPayload(t)
	fp := &fakePublisher{failN: 1}
	s := NewServerLegacy(testSecret, "pixel", "tok", fp)
	c, rec := newCtx(http.MethodPost, "/meta-capi/v1/events", body)
	_ = s.IngestEvent(c)
	assert.Equal(t, http.StatusBadGateway, rec.Code)

	// And delivery status should reflect the failure.
	id := "eid"
	c2, rec2 := newCtx(http.MethodGet, "/meta-capi/v1/events/"+id, "")
	c2.SetPath("/meta-capi/v1/events/:event_id")
	c2.SetParamNames("event_id")
	c2.SetParamValues(id)
	_ = s.DeliveryStatus(c2)
	var dr DeliveryRecord
	_ = json.Unmarshal(rec2.Body.Bytes(), &dr)
	assert.Equal(t, "failed", dr.Status)
}

func TestIngestEvent_DeliveryStatusTracking(t *testing.T) {
	body, p := signedPayload(t)
	fp := &fakePublisher{}
	s := NewServerLegacy(testSecret, "pixel", "tok", fp)
	c, _ := newCtx(http.MethodPost, "/meta-capi/v1/events", body)
	_ = s.IngestEvent(c)

	c2, rec := newCtx(http.MethodGet, "/meta-capi/v1/events/"+p.EventID, "")
	c2.SetPath("/meta-capi/v1/events/:event_id")
	c2.SetParamNames("event_id")
	c2.SetParamValues(p.EventID)
	_ = s.DeliveryStatus(c2)
	assert.Equal(t, http.StatusOK, rec.Code)
	var dr DeliveryRecord
	_ = json.Unmarshal(rec.Body.Bytes(), &dr)
	assert.Equal(t, "sent", dr.Status)
	assert.Equal(t, 1, dr.Attempts)
}

func TestIngestEvent_EmailIsNormalisedBeforeHashing(t *testing.T) {
	_, p := signedPayload(t)
	p.Email = "  USER@EXAMPLE.COM  "
	// Re-sign with the new payload
	canonical, _ := json.Marshal(struct {
		LeadID, EventID, Email, Phone, FBCLID, FBP, EventName string
		Timestamp                                              int64
	}{p.LeadID, p.EventID, p.Email, p.Phone, p.FBCLID, p.FBP, p.EventName, p.Timestamp})
	p.HMAC = SignHMAC(testSecret, p.LeadID, p.FBCLID, "1700000000", string(canonical))
	body3, _ := json.Marshal(p)

	fp := &fakePublisher{}
	s := NewServerLegacy(testSecret, "p", "t", fp)
	c, _ := newCtx(http.MethodPost, "/meta-capi/v1/events", string(body3))
	_ = s.IngestEvent(c)

	assert.Len(t, fp.events, 1)
	// The hashed email field should match the normalised form.
	want := HashSHA256("user@example.com")
	assert.Equal(t, want, fp.events[0].Data[0].UserData["em"])
}

// ============================================================================
// Health
// ============================================================================

func TestHealth(t *testing.T) {
	s := NewServerLegacy(testSecret, "p", "t", &fakePublisher{})
	c, rec := newCtx(http.MethodGet, "/healthz", "")
	err := s.Health(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "meta-capi", body["service"])
}
