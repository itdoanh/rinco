// Tests for lead-service handler.
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestNewServer(t *testing.T) {
	s := NewServer(nil, nil, nil, "")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.pool != nil {
		t.Error("expected nil pool")
	}
	if s.scoringURL != "" {
		t.Error("scoringURL should be empty")
	}
}

func TestTenantFromCtx(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "t1")
	c.Set("user_id", "u1")
	c.Set("is_admin", true)

	s := NewServer(nil, nil, nil, "")
	tid, uid, admin := s.tenantFromCtx(c)
	if tid != "t1" {
		t.Errorf("tenantID: got %s", tid)
	}
	if uid != "u1" {
		t.Errorf("userID: got %s", uid)
	}
	if !admin {
		t.Error("isAdmin should be true")
	}
}

func TestTenantFromCtx_Empty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewServer(nil, nil, nil, "")
	tid, uid, admin := s.tenantFromCtx(c)
	if tid != "" || uid != "" || admin {
		t.Errorf("expected empty: got %s/%s/%v", tid, uid, admin)
	}
}

func TestGetPagination_Default(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 1 {
		t.Errorf("Page: got %d, want 1", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("PerPage: got %d, want 20", p.PerPage)
	}
	if p.Offset != 0 {
		t.Errorf("Offset: got %d, want 0", p.Offset)
	}
}

func TestGetPagination_Custom(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?page=3&per_page=50", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 3 {
		t.Errorf("Page: got %d, want 3", p.Page)
	}
	if p.PerPage != 50 {
		t.Errorf("PerPage: got %d, want 50", p.PerPage)
	}
	if p.Offset != 100 {
		t.Errorf("Offset: got %d, want 100", p.Offset)
	}
}

func TestGetPagination_Limits(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?per_page=999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.PerPage != 20 {
		t.Errorf("PerPage should clamp to 20, got %d", p.PerPage)
	}
}

func TestGetPagination_Invalid(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?page=abc&per_page=xyz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 1 {
		t.Errorf("Page should default to 1, got %d", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("PerPage should default to 20, got %d", p.PerPage)
	}
}

func TestRedisClient_Defaults(t *testing.T) {
	r := &RedisClient{}
	if r.Addr != "" {
		t.Error("Addr should be empty")
	}
	if r.DB != 0 {
		t.Error("DB should be 0")
	}
}

func TestRedisClient_WithValues(t *testing.T) {
	r := &RedisClient{
		Addr:     "localhost:6379",
		Password: "secret",
		DB:       1,
	}
	if r.Addr != "localhost:6379" {
		t.Errorf("Addr: got %s", r.Addr)
	}
	if r.Password != "secret" {
		t.Error("Password mismatch")
	}
	if r.DB != 1 {
		t.Errorf("DB: got %d", r.DB)
	}
}

func TestPagination_Struct(t *testing.T) {
	p := pagination{
		Page:    5,
		PerPage: 25,
		Offset:  100,
	}
	if p.Page != 5 {
		t.Errorf("Page: got %d", p.Page)
	}
	if p.PerPage != 25 {
		t.Errorf("PerPage: got %d", p.PerPage)
	}
	if p.Offset != 100 {
		t.Errorf("Offset: got %d", p.Offset)
	}
}
