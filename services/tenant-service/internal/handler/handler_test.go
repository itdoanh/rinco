// Unit tests for tenant-service.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
// Slug validation
// ============================================================================

func TestValidSlug_Accepts(t *testing.T) {
	good := []string{"apex", "apex-fintech", "ab", "a1b2c3", "abc-123-xyz"}
	for _, s := range good {
		t.Run(s, func(t *testing.T) { assert.NoError(t, validSlug(s)) })
	}
}

func TestValidSlug_Rejects(t *testing.T) {
	bad := []string{
		"",
		"Admin",         // uppercase
		"-leading",
		"trailing-",
		"_underscore",
		"a",             // too short
		strings.Repeat("a", 41), // too long
		"has space",
		"admin",         // reserved
		"api",           // reserved
	}
	for _, s := range bad {
		t.Run(s, func(t *testing.T) { assert.Error(t, validSlug(s)) })
	}
}

// ============================================================================
// Domain validation
// ============================================================================

func TestValidDomain_Accepts(t *testing.T) {
	good := []string{
		"apex.vn",
		"apex-fintech.vn",
		"sub.domain.example.com",
		"a.b.c.de.io",
	}
	for _, d := range good {
		t.Run(d, func(t *testing.T) { assert.NoError(t, validDomain(d)) })
	}
}

func TestValidDomain_Rejects(t *testing.T) {
	bad := []string{
		"notadomain",
		"-leading.vn",
		"trailing-.com",
		"under_score.vn",
		"noTld",
	}
	for _, d := range bad {
		t.Run(d, func(t *testing.T) { assert.Error(t, validDomain(d)) })
	}
}

func TestValidDomain_EmptyAllowed(t *testing.T) {
	assert.NoError(t, validDomain(""), "empty domain is allowed (optional field)")
}

// ============================================================================
// Plan validation
// ============================================================================

func TestValidPlan(t *testing.T) {
	assert.True(t, validPlan("starter"))
	assert.True(t, validPlan("pro"))
	assert.True(t, validPlan("enterprise"))
	assert.False(t, validPlan("free"))
	assert.False(t, validPlan(""))
	assert.False(t, validPlan("STARTER"))
}

// ============================================================================
// CreateTenant
// ============================================================================

func TestCreateTenant_HappyPath(t *testing.T) {
	s := NewServer()
	c, rec := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"apex","name":"Apex Fintech","plan":"pro","owner_email":"a@b.co"}`)
	err := s.CreateTenant(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var tnt Tenant
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tnt))
	assert.NotEqual(t, [16]byte{}, tnt.ID)
	assert.Equal(t, "apex", tnt.Slug)
	assert.Equal(t, "active", tnt.Status)
}

func TestCreateTenant_DuplicateSlug(t *testing.T) {
	s := NewServer()
	c1, _ := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"apex","name":"A","plan":"pro","owner_email":"a@b.co"}`)
	_ = s.CreateTenant(c1)
	c2, rec := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"apex","name":"B","plan":"pro","owner_email":"b@b.co"}`)
	_ = s.CreateTenant(c2)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreateTenant_InvalidSlug(t *testing.T) {
	s := NewServer()
	c, rec := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"-bad","name":"X","plan":"pro","owner_email":"a@b.co"}`)
	_ = s.CreateTenant(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateTenant_InvalidPlan(t *testing.T) {
	s := NewServer()
	c, rec := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"x","name":"X","plan":"free","owner_email":"a@b.co"}`)
	_ = s.CreateTenant(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateTenant_MissingOwner(t *testing.T) {
	s := NewServer()
	c, rec := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"x","name":"X","plan":"pro"}`)
	_ = s.CreateTenant(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateTenant_WithCustomDomain(t *testing.T) {
	s := NewServer()
	c, rec := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"apex","name":"A","plan":"pro","owner_email":"a@b.co","custom_domain":"apex.vn"}`)
	_ = s.CreateTenant(c)
	assert.Equal(t, http.StatusCreated, rec.Code)
	var tnt Tenant
	_ = json.Unmarshal(rec.Body.Bytes(), &tnt)
	assert.Equal(t, "apex.vn", tnt.CustomDomain)
}

func TestCreateTenant_DuplicateDomain(t *testing.T) {
	s := NewServer()
	c1, _ := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"a","name":"A","plan":"pro","owner_email":"a@b.co","custom_domain":"x.vn"}`)
	_ = s.CreateTenant(c1)
	c2, rec := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"b","name":"B","plan":"pro","owner_email":"b@b.co","custom_domain":"x.vn"}`)
	_ = s.CreateTenant(c2)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

// ============================================================================
// Suspend / restore
// ============================================================================

func TestSuspendTenant(t *testing.T) {
	s := NewServer()
	c1, rec1 := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"a","name":"A","plan":"pro","owner_email":"a@b.co"}`)
	_ = s.CreateTenant(c1)
	var tnt Tenant
	_ = json.Unmarshal(rec1.Body.Bytes(), &tnt)

	c2, rec2 := newCtx(http.MethodPut, "/tenant/v1/tenants/"+tnt.ID.String()+"/suspend", "")
	c2.SetPath("/tenant/v1/tenants/:id/suspend")
	c2.SetParamNames("id")
	c2.SetParamValues(tnt.ID.String())
	_ = s.SuspendTenant(c2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	var tnt2 Tenant
	_ = json.Unmarshal(rec2.Body.Bytes(), &tnt2)
	assert.Equal(t, "suspended", tnt2.Status)
}

// ============================================================================
// 2-of-3 Quorum
// ============================================================================

func TestDeletion_Requires2Of3Quorum(t *testing.T) {
	s := NewServer()
	c1, rec1 := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"a","name":"A","plan":"pro","owner_email":"a@b.co"}`)
	_ = s.CreateTenant(c1)
	var tnt Tenant
	_ = json.Unmarshal(rec1.Body.Bytes(), &tnt)

	// Request deletion
	c2, rec2 := newCtx(http.MethodPost, "/tenant/v1/tenants/"+tnt.ID.String()+"/delete", "")
	c2.SetPath("/tenant/v1/tenants/:id/delete")
	c2.SetParamNames("id")
	c2.SetParamValues(tnt.ID.String())
	_ = s.RequestDeletion(c2)
	assert.Equal(t, http.StatusCreated, rec2.Code)

	var q QuorumRequest
	_ = json.Unmarshal(rec2.Body.Bytes(), &q)
	assert.Equal(t, 2, q.RequiredCount)
	assert.False(t, q.Executed)

	// First signature — not enough
	c3, _ := newCtx(http.MethodPost, "/q/"+q.ID.String(),
		`{"admin_id":"admin-1"}`)
	c3.SetPath("/q/:qid")
	c3.SetParamNames("qid")
	c3.SetParamValues(q.ID.String())
	_ = s.SignDeletion(c3)
	var q2 QuorumRequest
	_ = json.Unmarshal(_(c3), &q2)
	assert.Len(t, q2.Signatures, 1)
	assert.False(t, q2.Executed, "1/2 signatures must not execute")

	// Second signature — quorum reached, deletion executes
	c4, _ := newCtx(http.MethodPost, "/q/"+q.ID.String(),
		`{"admin_id":"admin-2"}`)
	c4.SetPath("/q/:qid")
	c4.SetParamNames("qid")
	c4.SetParamValues(q.ID.String())
	_ = s.SignDeletion(c4)
	var q3 QuorumRequest
	_ = json.Unmarshal(_(c4), &q3)
	assert.True(t, q3.Executed, "2/2 signatures must execute")
}

func TestDeletion_DuplicateSignatureRejected(t *testing.T) {
	s := NewServer()
	c1, _ := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"a","name":"A","plan":"pro","owner_email":"a@b.co"}`)
	_ = s.CreateTenant(c1)
	var tnt Tenant
	_ = json.Unmarshal(c1.Response().Body.Bytes(), &tnt)

	c2, rec2 := newCtx(http.MethodPost, "/tenant/v1/tenants/"+tnt.ID.String()+"/delete", "")
	c2.SetPath("/tenant/v1/tenants/:id/delete")
	c2.SetParamNames("id")
	c2.SetParamValues(tnt.ID.String())
	_ = s.RequestDeletion(c2)
	var q QuorumRequest
	_ = json.Unmarshal(rec2.Body.Bytes(), &q)

	c3, _ := newCtx(http.MethodPost, "/q/"+q.ID.String(), `{"admin_id":"a"}`)
	c3.SetPath("/q/:qid")
	c3.SetParamNames("qid")
	c3.SetParamValues(q.ID.String())
	_ = s.SignDeletion(c3)

	c4, rec4 := newCtx(http.MethodPost, "/q/"+q.ID.String(), `{"admin_id":"a"}`)
	c4.SetPath("/q/:qid")
	c4.SetParamNames("qid")
	c4.SetParamValues(q.ID.String())
	_ = s.SignDeletion(c4)
	assert.Equal(t, http.StatusConflict, rec4.Code)
}

func TestDeletion_ExpiredQuorumRejected(t *testing.T) {
	s := NewServer()
	c1, _ := newCtx(http.MethodPost, "/tenant/v1/tenants",
		`{"slug":"a","name":"A","plan":"pro","owner_email":"a@b.co"}`)
	_ = s.CreateTenant(c1)
	var tnt Tenant
	_ = json.Unmarshal(c1.Response().Body.Bytes(), &tnt)

	c2, rec2 := newCtx(http.MethodPost, "/tenant/v1/tenants/"+tnt.ID.String()+"/delete", "")
	c2.SetPath("/tenant/v1/tenants/:id/delete")
	c2.SetParamNames("id")
	c2.SetParamValues(tnt.ID.String())
	_ = s.RequestDeletion(c2)
	var q QuorumRequest
	_ = json.Unmarshal(rec2.Body.Bytes(), &q)

	// Manually expire
	s.mu.Lock()
	q.ExpiresAt = q.ExpiresAt.Add(-1 * time.Hour)
	s.mu.Unlock()

	c3, rec3 := newCtx(http.MethodPost, "/q/"+q.ID.String(), `{"admin_id":"a"}`)
	c3.SetPath("/q/:qid")
	c3.SetParamNames("qid")
	c3.SetParamValues(q.ID.String())
	_ = s.SignDeletion(c3)
	assert.Equal(t, http.StatusGone, rec3.Code)
}

// helper to make the test more readable (echo's recorder is reachable via Response())
func _(c echo.Context) []byte {
	return c.Response().Body.Bytes()
}
