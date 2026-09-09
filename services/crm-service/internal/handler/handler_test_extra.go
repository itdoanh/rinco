// Tests for crm-service handler helpers (pagination + tenant context)
// that don't require a database connection.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// =============================================================================
// getPagination
// =============================================================================

func TestExtraGetPagination_Defaults(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
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

func TestExtraGetPagination_Custom(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x?page=3&per_page=50", nil)
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

func TestExtraGetPagination_InvalidValues(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x?page=-5&per_page=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 1 {
		t.Errorf("negative page should default to 1, got %d", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("invalid per_page should default to 20, got %d", p.PerPage)
	}
}

func TestExtraGetPagination_PerPageOver100(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x?per_page=9999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.PerPage != 20 {
		t.Errorf("per_page > 100 should be clamped to 20, got %d", p.PerPage)
	}
}

func TestExtraGetPagination_Boundary100(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x?per_page=100", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	// 100 is allowed (<=).
	if p.PerPage != 100 {
		t.Errorf("per_page=100 should be allowed, got %d", p.PerPage)
	}
}

func TestExtraGetPagination_OffsetArithmetic(t *testing.T) {
	e := echo.New()
	// page=5, per_page=10 → offset = 40
	req := httptest.NewRequest(http.MethodGet, "/x?page=5&per_page=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Offset != 40 {
		t.Errorf("Offset: got %d, want 40", p.Offset)
	}
}

// =============================================================================
// tenantFromCtx
// =============================================================================

func TestExtraTenantFromCtx_Empty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{}
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if tenantID != "" || userID != "" || isAdmin {
		t.Errorf("expected empty defaults, got tenant=%s user=%s admin=%v", tenantID, userID, isAdmin)
	}
}

func TestExtraTenantFromCtx_AllSet(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")
	c.Set("is_admin", true)

	s := &Server{}
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if tenantID != "tenant-1" {
		t.Errorf("tenant_id: got %s", tenantID)
	}
	if userID != "user-1" {
		t.Errorf("user_id: got %s", userID)
	}
	if !isAdmin {
		t.Error("is_admin should be true")
	}
}

func TestExtraTenantFromCtx_Partial(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "t1")

	s := &Server{}
	tenantID, userID, isAdmin := s.tenantFromCtx(c)
	if tenantID != "t1" {
		t.Errorf("tenant_id: got %s", tenantID)
	}
	if userID != "" {
		t.Errorf("user_id: got %s", userID)
	}
	if isAdmin {
		t.Error("is_admin should default to false")
	}
}

// =============================================================================
// json / errorResp helpers
// =============================================================================

func TestExtraJson_HelperWrapsEchoJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{}
	if err := s.json(c, http.StatusOK, map[string]string{"hello": "world"}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d", rec.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["hello"] != "world" {
		t.Errorf("body: %v", resp)
	}
}

func TestExtraErrorResp_Helper(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{}
	if err := s.errorResp(c, http.StatusBadRequest, "invalid input", nil); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d", rec.Code)
	}
	var resp errResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != "invalid input" {
		t.Errorf("Error: %s", resp.Error)
	}
}

func TestExtraErrorResp_WithDetails(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{}
	if err := s.errorResp(c, http.StatusInternalServerError, "boom", &simpleErr{"database down"}); err != nil {
		t.Fatal(err)
	}
	var resp errResp
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	details, _ := resp.Details.(string)
	if details != "database down" {
		t.Errorf("Details: got %v", resp.Details)
	}
}

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }

// =============================================================================
// ListResp JSON shape
// =============================================================================

func TestExtraListResp_JSONShape(t *testing.T) {
	resp := listResp{
		Data:       []int{1, 2, 3},
		Total:      3,
		Page:       1,
		PerPage:    20,
		TotalPages: 1,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var back listResp
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Total != 3 || back.Page != 1 {
		t.Errorf("round trip mismatch: %+v", back)
	}
}
