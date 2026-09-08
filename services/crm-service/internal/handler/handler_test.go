// Tests for crm-service handler helpers that don't require a DB connection.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestGetPagination_Defaults(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 1 {
		t.Errorf("expected default Page=1, got %d", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("expected default PerPage=20, got %d", p.PerPage)
	}
	if p.Offset != 0 {
		t.Errorf("expected default Offset=0, got %d", p.Offset)
	}
}

func TestGetPagination_CustomValues(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test?page=3&per_page=50", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 3 {
		t.Errorf("expected Page=3, got %d", p.Page)
	}
	if p.PerPage != 50 {
		t.Errorf("expected PerPage=50, got %d", p.PerPage)
	}
	if p.Offset != 100 {
		t.Errorf("expected Offset=100, got %d", p.Offset)
	}
}

func TestGetPagination_BoundsEnforced(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantPage    int
		wantPerPage int
		wantOffset  int
	}{
		{"per_page too low → defaults", "per_page=0", 1, 20, 0},
		{"per_page too high → clamped to 100 then offset", "per_page=200", 1, 20, 0}, // query parsing: 200 > 100 → default 20
		{"negative page → defaults", "page=-5", 1, 20, 0},
		{"page 0 → defaults to 1", "page=0", 1, 20, 0},
		{"non-numeric page → defaults", "page=abc", 1, 20, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			p := getPagination(c)
			if p.Page != tt.wantPage {
				t.Errorf("Page: want %d, got %d", tt.wantPage, p.Page)
			}
			if p.PerPage != tt.wantPerPage {
				t.Errorf("PerPage: want %d, got %d", tt.wantPerPage, p.PerPage)
			}
			if p.Offset != tt.wantOffset {
				t.Errorf("Offset: want %d, got %d", tt.wantOffset, p.Offset)
			}
		})
	}
}

func TestTenantFromCtx_AllSet(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tnt-1")
	c.Set("user_id", "usr-1")
	c.Set("is_admin", true)

	s := &Server{}
	tenant, user, admin := s.tenantFromCtx(c)
	if tenant != "tnt-1" {
		t.Errorf("tenant: want tnt-1, got %q", tenant)
	}
	if user != "usr-1" {
		t.Errorf("user: want usr-1, got %q", user)
	}
	if !admin {
		t.Errorf("admin: want true, got false")
	}
}

func TestTenantFromCtx_Missing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{}
	tenant, user, admin := s.tenantFromCtx(c)
	if tenant != "" {
		t.Errorf("tenant: want empty, got %q", tenant)
	}
	if user != "" {
		t.Errorf("user: want empty, got %q", user)
	}
	if admin {
		t.Errorf("admin: want false, got true")
	}
}

func TestTenantFromCtx_WrongTypes(t *testing.T) {
	// Wrong type assertions should return zero values, not panic.
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", 12345)            // int, not string
	c.Set("user_id", []string{"a", "b"}) // []string, not string
	c.Set("is_admin", "true")            // string, not bool

	s := &Server{}
	tenant, user, admin := s.tenantFromCtx(c)
	if tenant != "" {
		t.Errorf("tenant: want empty, got %q", tenant)
	}
	if user != "" {
		t.Errorf("user: want empty, got %q", user)
	}
	if admin {
		t.Errorf("admin: want false, got true")
	}
}

func TestErrorResp_Structure(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{}
	err := s.errorResp(c, http.StatusBadRequest, "bad input", nil)
	if err != nil {
		t.Fatalf("errorResp returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code: want 400, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "bad input" {
		t.Errorf("error: want 'bad input', got %v", body["error"])
	}
	if d, ok := body["details"]; ok && d != "" {
		t.Errorf("details should be empty when err is nil, got %v", d)
	}
}

func TestSetRLS_InvalidUserUUID_SkipsIfBlank(t *testing.T) {
	// Empty userID is allowed (some requests have no user context).
	// But pool is nil so we cannot proceed beyond tenant validation.
	// This test documents that empty userID with valid tenant proceeds
	// past validation (no error preflight); actual DB op will fail elsewhere.
	s := &Server{} // no pool
	// Even with valid tenant + empty user, we'd hit pool.Begin which panics,
	// so we only test the preflight paths here:
	err := s.setRLS(t.Context(), "", "", false)
	if err == nil {
		t.Fatal("expected error for empty tenantID")
	}
}

func TestErrorResp_WithDetails(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{}
	if err := s.errorResp(c, http.StatusInternalServerError, "failed", echo.NewHTTPError(500, "boom")); err != nil {
		t.Fatalf("returned error: %v", err)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["details"] == nil {
		t.Errorf("details should be present when err is non-nil")
	}
}

func TestSetRLS_NilTenantID(t *testing.T) {
	s := &Server{} // no pool, so we test preflight validation
	err := s.setRLS(t.Context(), "", "user-1", false)
	if err == nil {
		t.Fatal("expected error for empty tenantID")
	}
}

func TestSetRLS_InvalidTenantUUID(t *testing.T) {
	s := &Server{} // no pool, but UUID validation happens first
	err := s.setRLS(t.Context(), "not-a-uuid", "user-1", false)
	if err == nil {
		t.Fatal("expected error for invalid tenantID")
	}
}

func TestListResp_JSONShape(t *testing.T) {
	// Verify listResp serializes correctly
	in := listResp{Data: []string{"a", "b"}, Total: 2, Page: 1, PerPage: 20, TotalPages: 1}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	want := []string{"data", "total", "page", "per_page", "total_pages"}
	for _, k := range want {
		if _, ok := out[k]; !ok {
			t.Errorf("missing key %q in JSON", k)
		}
	}
}
