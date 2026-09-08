// Tests for email-service handler.
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestServer_NewServer(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, nil)
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.pool != nil {
		t.Error("expected nil pool")
	}
	if s.rdb != nil {
		t.Error("expected nil rdb")
	}
	if s.nats != nil {
		t.Error("expected nil nats")
	}
	if s.driver != nil {
		t.Error("expected nil driver")
	}
}

func TestServer_Pool(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, nil)
	if s.Pool() != nil {
		t.Error("expected nil pool")
	}
}

func TestServer_Driver(t *testing.T) {
	s := NewServer(nil, nil, nil, nil, nil)
	if s.Driver() != nil {
		t.Error("expected nil driver")
	}
}

func TestSendEmail_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/email/send", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.SendEmail(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Handler may return 500 if renderer init fails with nil config.
	if rec.Code < 400 || rec.Code >= 600 {
		t.Errorf("expected 4xx/5xx, got %d", rec.Code)
	}
}

func TestSendEmail_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/email/send", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "")

	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.SendEmail(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code < 400 {
		t.Errorf("expected error status, got %d", rec.Code)
	}
}

func TestBatchSend_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/email/batch", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-1")

	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.BatchSend(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code < 400 {
		t.Errorf("expected error status, got %d", rec.Code)
	}
}

func TestListTemplates_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "")

	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.ListTemplates(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code < 400 {
		t.Errorf("expected error status, got %d", rec.Code)
	}
}

func TestTrackingPixel_NoTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/track/1x1.gif", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.TrackingPixel(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Should return 1x1 GIF regardless of tenant
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestClickRedirect_NoURL(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/click", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.ClickRedirect(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Should redirect to home page when url is missing/invalid.
	if rec.Code != http.StatusFound && rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected redirect, got %d", rec.Code)
	}
}

func TestClickRedirect_InvalidURL(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/click?url=http://evil.com", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := NewServer(nil, nil, nil, nil, nil)
	if err := s.ClickRedirect(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	// Non-whitelisted URLs should redirect to home page.
	if rec.Code != http.StatusFound && rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected redirect, got %d", rec.Code)
	}
}
