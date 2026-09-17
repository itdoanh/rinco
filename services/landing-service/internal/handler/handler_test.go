// Unit tests for landing-service.
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

func newCtx(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// ============================================================================
// Email validation
// ============================================================================

func TestValidEmail(t *testing.T) {
	good := []string{"a@b.co", "user@example.com", "u+tag@sub.domain.io", "x@y.z"}
	bad := []string{"", "no-at-sign", "@nodomain.com", "user@", "user@.com"}
	for _, e := range good {
		t.Run("accept/"+e, func(t *testing.T) { assert.True(t, validEmail(e)) })
	}
	for _, e := range bad {
		t.Run("reject/"+e, func(t *testing.T) { assert.False(t, validEmail(e)) })
	}
}

// ============================================================================
// Phone validation
// ============================================================================

func TestValidPhone(t *testing.T) {
	good := []string{"+84901234567", "0901234567", "+1234567890", "0123456789"}
	bad := []string{"", "abc", "12", "+84-90-123", "phone"}
	for _, p := range good {
		t.Run("accept/"+p, func(t *testing.T) { assert.True(t, validPhone(p)) })
	}
	for _, p := range bad {
		t.Run("reject/"+p, func(t *testing.T) { assert.False(t, validPhone(p)) })
	}
}

// ============================================================================
// Normalisation
// ============================================================================

func TestNormaliseEmail_LowercasesAndTrims(t *testing.T) {
	assert.Equal(t, "a@b.co", normaliseEmail("  A@B.CO  "))
	assert.Equal(t, "user@example.com", normaliseEmail("USER@Example.Com"))
}

func TestNormalisePhone_StripsFormatting(t *testing.T) {
	assert.Equal(t, "+84901234567", normalisePhone("+84 90 123 4567"))
	assert.Equal(t, "0901234567", normalisePhone("(090) 123-4567"))
	assert.Equal(t, "+1234567890", normalisePhone("+1 (234) 567-890"))
}

func TestHashSHA256_Deterministic(t *testing.T) {
	h1 := HashSHA256("test")
	h2 := HashSHA256("test")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64)
}

func TestHashSHA256_DifferentInputs(t *testing.T) {
	assert.NotEqual(t, HashSHA256("a"), HashSHA256("b"))
}

// ============================================================================
// HMAC
// ============================================================================

func TestSignHMAC_Deterministic(t *testing.T) {
	secret := []byte("rinco-secret")
	s1 := SignHMAC(secret, "id1", "fb1", "1700000000", "payload")
	s2 := SignHMAC(secret, "id1", "fb1", "1700000000", "payload")
	assert.Equal(t, s1, s2)
}

func TestSignHMAC_DifferentSecretsProduceDifferentSignatures(t *testing.T) {
	s1 := SignHMAC([]byte("a"), "id", "fb", "ts", "p")
	s2 := SignHMAC([]byte("b"), "id", "fb", "ts", "p")
	assert.NotEqual(t, s1, s2)
}

func TestSignHMAC_OutputIsLowercaseHex(t *testing.T) {
	s := SignHMAC([]byte("k"), "id", "fb", "ts", "p")
	assert.Regexp(t, "^[0-9a-f]{64}$", s)
}

// ============================================================================
// SubmitLead
// ============================================================================

func TestSubmitLead_HappyPath(t *testing.T) {
	s := NewServer([]byte("test-key"))
	c, rec := newCtx(http.MethodPost, "/landing/v1/leads",
		`{"tenant_slug":"default","email":"a@b.co","full_name":"John","phone":"0901234567"}`)
	err := s.SubmitLead(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp leadResp
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.ID)
	assert.NotEmpty(t, resp.EventID)
	assert.NotEmpty(t, resp.HMAC)
	assert.Len(t, resp.HMAC, 64)
}

func TestSubmitLead_InvalidEmail(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodPost, "/landing/v1/leads",
		`{"tenant_slug":"default","email":"not-email","full_name":"John"}`)
	_ = s.SubmitLead(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSubmitLead_MissingTenantSlug(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodPost, "/landing/v1/leads",
		`{"email":"a@b.co","full_name":"John"}`)
	_ = s.SubmitLead(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSubmitLead_UnknownTenant(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodPost, "/landing/v1/leads",
		`{"tenant_slug":"unknown","email":"a@b.co","full_name":"John"}`)
	_ = s.SubmitLead(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSubmitLead_InvalidPhone(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodPost, "/landing/v1/leads",
		`{"tenant_slug":"default","email":"a@b.co","full_name":"John","phone":"abc"}`)
	_ = s.SubmitLead(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSubmitLead_EmailIsNormalised(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodPost, "/landing/v1/leads",
		`{"tenant_slug":"default","email":"  John@B.CO  ","full_name":"John"}`)
	_ = s.SubmitLead(c)
	var l Lead
	_ = json.Unmarshal(rec.Body.Bytes(), &l)
	assert.Equal(t, "john@b.co", l.Email)
}

func TestSubmitLead_DuplicateIDsAreUnique(t *testing.T) {
	s := NewServer(nil)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		c, rec := newCtx(http.MethodPost, "/landing/v1/leads",
			`{"tenant_slug":"default","email":"a@b.co","full_name":"John"}`)
		_ = s.SubmitLead(c)
		var l Lead
		_ = json.Unmarshal(rec.Body.Bytes(), &l)
		assert.False(t, seen[l.ID], "duplicate ID generated: %s", l.ID)
		seen[l.ID] = true
	}
}

func TestSubmitLead_FormFieldsPreserved(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodPost, "/landing/v1/leads",
		`{"tenant_slug":"default","email":"a@b.co","full_name":"John","form_fields":{"budget":"100k","city":"HCM"}}`)
	_ = s.SubmitLead(c)
	var l Lead
	_ = json.Unmarshal(rec.Body.Bytes(), &l)
	assert.Equal(t, "100k", l.FormFields["budget"])
	assert.Equal(t, "HCM", l.FormFields["city"])
}

// ============================================================================
// GetLead
// ============================================================================

func TestGetLead_NotFound(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodGet, "/landing/v1/leads/missing", "")
	c.SetPath("/landing/v1/leads/:id")
	c.SetParamNames("id")
	c.SetParamValues("missing")
	_ = s.GetLead(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ============================================================================
// TrackEvent
// ============================================================================

func TestTrackEvent_HappyPath(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodPost, "/landing/v1/track",
		`{"event":"page_view","url":"/summer-sale"}`)
	_ = s.TrackEvent(c)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestTrackEvent_MissingEvent(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodPost, "/landing/v1/track", `{"url":"/x"}`)
	_ = s.TrackEvent(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// RenderPage
// ============================================================================

func TestRenderPage(t *testing.T) {
	s := NewServer(nil)
	c, rec := newCtx(http.MethodGet, "/landing/v1/pages/summer-sale", "")
	c.SetPath("/landing/v1/pages/:slug")
	c.SetParamNames("slug")
	c.SetParamValues("summer-sale")
	_ = s.RenderPage(c)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "summer-sale", body["slug"])
}
