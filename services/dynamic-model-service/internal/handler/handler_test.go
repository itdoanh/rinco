// Tests for dynamic-model-service handler.
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestNew(t *testing.T) {
	s := New(nil)
	if s == nil {
		t.Fatal("New returned nil")
	}
}

func TestCtxTenantUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tenantID, userID := ctxTenantUser(c)
	if tenantID != "tenant-1" {
		t.Errorf("tenantID: got %s, want tenant-1", tenantID)
	}
	if userID != "user-1" {
		t.Errorf("userID: got %s, want user-1", userID)
	}
}

func TestCtxTenantUser_Empty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tenantID, userID := ctxTenantUser(c)
	if tenantID != "" {
		t.Errorf("tenantID should be empty: got %s", tenantID)
	}
	if userID != "" {
		t.Errorf("userID should be empty: got %s", userID)
	}
}

func TestCreateModel_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/models", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.CreateModel(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreateModel_MissingFields(t *testing.T) {
	e := echo.New()
	payload := createModelReq{Name: "", Slug: ""}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/models", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.CreateModel(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name/slug, got %d", rec.Code)
	}
}

func TestCreateModel_InvalidSlug(t *testing.T) {
	e := echo.New()
	payload := createModelReq{Name: "Test", Slug: "INVALID SLUG!"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/models", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.CreateModel(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid slug, got %d", rec.Code)
	}
}

func TestSlugRe_Valid(t *testing.T) {
	tests := []struct {
		slug  string
		valid bool
	}{
		{"hello-world", true},
		{"test_123", true},
		{"abc", true},
		{"Hello", false},
		{"hello world", false},
		{"hello.world", false},
		{"hello/world", false},
		{"hello@world", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got := slugRe.MatchString(tt.slug)
			if got != tt.valid {
				t.Errorf("slug %q: got %v, want %v", tt.slug, got, tt.valid)
			}
		})
	}
}

func TestModelSchema_Constant(t *testing.T) {
	if modelSchema != "model" {
		t.Errorf("modelSchema: got %s, want model", modelSchema)
	}
}

func TestCreateModelReq_JSON(t *testing.T) {
	payload := createModelReq{
		Name:        "Test Model",
		Slug:        "test-model",
		Description: "A test",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var got createModelReq
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "Test Model" {
		t.Errorf("Name: got %s", got.Name)
	}
	if got.Slug != "test-model" {
		t.Errorf("Slug: got %s", got.Slug)
	}
}

func TestModelResp_JSON(t *testing.T) {
	resp := modelResp{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		TenantID:    "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test",
		Slug:        "test",
		Version:     1,
		Status:      "draft",
		Description: "Test description",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var got modelResp
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != resp.ID {
		t.Errorf("ID: got %s", got.ID)
	}
	if got.Version != 1 {
		t.Errorf("Version: got %d", got.Version)
	}
}
