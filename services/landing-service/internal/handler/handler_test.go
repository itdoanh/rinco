// Tests for landing-service handler.
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestTableConstants(t *testing.T) {
	if tablePages != "landing.landing_pages" {
		t.Errorf("tablePages: got %s", tablePages)
	}
	if tableSubmissions != "landing.form_submissions" {
		t.Errorf("tableSubmissions: got %s", tableSubmissions)
	}
	if tableTracking != "landing.tracking_events" {
		t.Errorf("tableTracking: got %s", tableTracking)
	}
	if tableForms != "landing.form_definitions" {
		t.Errorf("tableForms: got %s", tableForms)
	}
	if tablePixels != "landing.fb_pixel_configs" {
		t.Errorf("tablePixels: got %s", tablePixels)
	}
	if tableGoals != "landing.conversion_goals" {
		t.Errorf("tableGoals: got %s", tableGoals)
	}
}

func TestCAPIStatus_NoPixel(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/capi/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{queue: make(chan capiEvent, 10)}
	if err := s.CAPIStatus(c); err != nil {
		t.Fatalf("CAPIStatus: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["enabled"] != false {
		t.Error("expected enabled=false when no pixel configured")
	}
	if body["pixel_id"] != "" {
		t.Errorf("expected empty pixel_id, got %v", body["pixel_id"])
	}
}

func TestCAPIStatus_WithPixel(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/capi/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &Server{queue: make(chan capiEvent, 10), pixelID: "123456789"}
	if err := s.CAPIStatus(c); err != nil {
		t.Fatalf("CAPIStatus: %v", err)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["enabled"] != true {
		t.Error("expected enabled=true when pixel configured")
	}
	if body["pixel_id"] != "123456789" {
		t.Errorf("pixel_id: got %v", body["pixel_id"])
	}
}

func TestNew_StartsWorkers(t *testing.T) {
	// Smoke test: New doesn't panic with nil deps.
	s := New(nil, nil, nil, "", "", "")
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.queue == nil {
		t.Error("queue should be initialized")
	}
	// Close queue to stop dispatchCAPI goroutine.
	close(s.queue)
}

func TestCAPISendRequest_Binds(t *testing.T) {
	// Note: capiSendReq has no json tags, so it uses the Go field name directly.
	body := `{"EventName":"Lead","EventID":"evt-1","ActionSource":"website","TenantID":"tnt-1"}`
	var req capiSendReq
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.EventName != "Lead" {
		t.Errorf("EventName: got %s", req.EventName)
	}
	if req.ActionSource != "website" {
		t.Errorf("ActionSource: got %s", req.ActionSource)
	}
	if req.TenantID != "tnt-1" {
		t.Errorf("TenantID: got %s", req.TenantID)
	}
}

func TestSubmitFormRequest_Binds(t *testing.T) {
	body := `{"form_slug":"contact","idempotency_key":"key-123","data":{"name":"John"}}`
	var req submitFormReq
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.FormSlug != "contact" {
		t.Errorf("FormSlug: got %s", req.FormSlug)
	}
	if req.IdempotencyKey != "key-123" {
		t.Errorf("IdempotencyKey: got %s", req.IdempotencyKey)
	}
	if req.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestTrackRequest_Binds(t *testing.T) {
	// trackReq has no json tags, uses Go field name.
	body := `{"EventName":"PageView","PageSlug":"home","SessionID":"sess-1"}`
	var req trackReq
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.EventName != "PageView" {
		t.Errorf("EventName: got %s", req.EventName)
	}
	if req.PageSlug != "home" {
		t.Errorf("PageSlug: got %s", req.PageSlug)
	}
	if req.SessionID != "sess-1" {
		t.Errorf("SessionID: got %s", req.SessionID)
	}
}

func TestConversionRequest_Binds(t *testing.T) {
	body := `{"event_id":"evt-1","event_name":"Purchase","event_time":1700000000,"email":"test@example.com"}`
	var req conversionReq
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.EventName != "Purchase" {
		t.Errorf("EventName: got %s", req.EventName)
	}
	if req.EventID != "evt-1" {
		t.Errorf("EventID: got %s", req.EventID)
	}
	if req.EventTime != 1700000000 {
		t.Errorf("EventTime: got %d", req.EventTime)
	}
	if req.Email != "test@example.com" {
		t.Errorf("Email: got %s", req.Email)
	}
}
