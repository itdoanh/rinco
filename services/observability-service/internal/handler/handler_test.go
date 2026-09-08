// Tests for observability-service handler.
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestServer_NewServer(t *testing.T) {
	s := NewServer(nil, "", "", "", "", "", nil)
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.pool != nil {
		t.Error("expected nil pool")
	}
}

func TestQueryLogs_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "")

	s := NewServer(nil, "", "", "", "", "", nil)
	if err := s.QueryLogs(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code < 400 {
		t.Errorf("expected error status, got %d", rec.Code)
	}
}

func TestQueryMetrics_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "")

	s := NewServer(nil, "", "", "", "", "", nil)
	if err := s.QueryMetrics(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code < 400 {
		t.Errorf("expected error status, got %d", rec.Code)
	}
}

func TestSearchTraces_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/traces", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "")

	s := NewServer(nil, "", "", "", "", "", nil)
	if err := s.SearchTraces(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code < 400 {
		t.Errorf("expected error status, got %d", rec.Code)
	}
}

func TestListServices_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "")

	s := NewServer(nil, "", "", "", "", "", nil)
	if err := s.ListServices(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code < 400 {
		t.Errorf("expected error status, got %d", rec.Code)
	}
}
