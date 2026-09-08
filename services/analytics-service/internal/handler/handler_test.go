// Tests for analytics-service handler.
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestTrackEventRequest_JSON(t *testing.T) {
	body := `{"tenant_id":"550e8400-e29b-41d4-a716-446655440000","event_type":"pageview"}`
	var tr TrackEventRequest
	if err := json.Unmarshal([]byte(body), &tr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tr.TenantID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("tenant_id: got %s", tr.TenantID)
	}
	if tr.EventType != "pageview" {
		t.Errorf("event_type: got %s", tr.EventType)
	}
}

func TestTrackEventRequest_AllFields(t *testing.T) {
	body := `{
		"tenant_id": "550e8400-e29b-41d4-a716-446655440000",
		"user_id": "user-123",
		"session_id": "sess-456",
		"event_type": "purchase",
		"source": "facebook",
		"campaign": "summer-sale",
		"url": "https://example.com/product",
		"referrer": "https://google.com",
		"user_agent": "Mozilla/5.0",
		"country": "VN",
		"device": "mobile",
		"browser": "Chrome",
		"os": "Android",
		"value": 129.99,
		"props": {"product_id": "abc"}
	}`
	var tr TrackEventRequest
	if err := json.Unmarshal([]byte(body), &tr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tr.UserID != "user-123" {
		t.Errorf("UserID: got %s", tr.UserID)
	}
	if tr.SessionID != "sess-456" {
		t.Errorf("SessionID: got %s", tr.SessionID)
	}
	if tr.Source != "facebook" {
		t.Errorf("Source: got %s", tr.Source)
	}
	if tr.Value != 129.99 {
		t.Errorf("Value: got %f", tr.Value)
	}
	if tr.Props["product_id"] != "abc" {
		t.Errorf("Props[product_id]: got %s", tr.Props["product_id"])
	}
}

func TestTrackEventRequest_InvalidJSON(t *testing.T) {
	body := `{invalid json}`
	var tr TrackEventRequest
	if err := json.Unmarshal([]byte(body), &tr); err == nil {
		t.Error("expected unmarshal error")
	}
}

func TestServer_New(t *testing.T) {
	s := New(nil)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.repo != nil {
		t.Error("expected repo to be nil")
	}
}

func TestTrackEventHandler_InvalidTenant(t *testing.T) {
	e := echo.New()
	body := `{"tenant_id":"not-a-uuid","event_type":"pageview"}`
	req := httptest.NewRequest(http.MethodPost, "/track", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.TrackEvent(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["error"] == "" {
		t.Error("expected error field in response")
	}
}

func TestTrackEventHandler_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/track", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.TrackEvent(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
