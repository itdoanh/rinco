// Extra tests for crm-service handler helpers.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHelpersPagination_Defaults(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 1 {
		t.Errorf("default page: got %d", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("default per_page: got %d", p.PerPage)
	}
	if p.Offset != 0 {
		t.Errorf("default offset: got %d", p.Offset)
	}
}

func TestHelpersPagination_Custom(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x?page=3&per_page=50", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 3 {
		t.Errorf("page: got %d", p.Page)
	}
	if p.PerPage != 50 {
		t.Errorf("per_page: got %d", p.PerPage)
	}
	if p.Offset != 100 {
		t.Errorf("offset: got %d", p.Offset)
	}
}

func TestHelpersPagination_PerPageCapped(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x?per_page=200", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.PerPage != 20 {
		t.Errorf("per_page should be capped at 20: got %d", p.PerPage)
	}
}

func TestHelpersPagination_NegativePage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x?page=-5", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 1 {
		t.Errorf("page <1 should default to 1: got %d", p.Page)
	}
}

func TestHelpersPagination_NonNumeric(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x?page=abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	p := getPagination(c)
	if p.Page != 1 {
		t.Errorf("non-numeric page should default: got %d", p.Page)
	}
}

func TestHelpersTenantFromCtx_Empty(t *testing.T) {
	srv := &Server{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tid, uid, admin := srv.tenantFromCtx(c)
	if tid != "" || uid != "" || admin {
		t.Error("expected all empty/false")
	}
}

func TestHelpersTenantFromCtx_Populated(t *testing.T) {
	srv := &Server{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-1")
	c.Set("user_id", "user-1")
	c.Set("is_admin", true)

	tid, uid, admin := srv.tenantFromCtx(c)
	if tid != "tenant-1" {
		t.Errorf("tid: %s", tid)
	}
	if uid != "user-1" {
		t.Errorf("uid: %s", uid)
	}
	if !admin {
		t.Error("admin: false")
	}
}

func TestHelpersJsonResponse(t *testing.T) {
	srv := &Server{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := srv.json(c, http.StatusOK, map[string]string{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" {
		t.Error("body empty")
	}
}

func TestHelpersErrorResponse(t *testing.T) {
	srv := &Server{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := srv.errorResp(c, 400, "BAD", nil); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}

	var r errResp
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Error != "BAD" {
		t.Errorf("error: %s", r.Error)
	}
}

func TestHelpersErrorResponse_WithErr(t *testing.T) {
	srv := &Server{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := srv.errorResp(c, 500, "ERR", echo.NewHTTPError(404)); err != nil {
		t.Fatal(err)
	}

	var r errResp
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Details == "" {
		t.Error("details should be populated when err is given")
	}
}

func TestHelpersListResp_JSON(t *testing.T) {
	r := listResp{Data: []int{1, 2, 3}, Total: 3, Page: 1, PerPage: 3, TotalPages: 1}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(b, `"total":3`) {
		t.Error("total missing in JSON")
	}
}

func TestHelpersCompanyReq_Marshal(t *testing.T) {
	req := companyReq{
		Name: stringPtr("Acme"),
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(b, `"name":"Acme"`) {
		t.Error("name should marshal")
	}
}

func TestHelpersErrResp_JSON(t *testing.T) {
	r := errResp{Error: "BOOM"}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(b, `"error":"BOOM"`) {
		t.Error("error field should be in JSON")
	}
}

func stringPtr(s string) *string { return &s }
func contains(b []byte, sub string) bool {
	return len(b) >= len(sub) && indexOf(b, []byte(sub)) >= 0
}
func indexOf(s, sub []byte) int {
outer:
	for i := 0; i+len(sub) <= len(s); i++ {
		for j := 0; j < len(sub); j++ {
			if s[i+j] != sub[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}
