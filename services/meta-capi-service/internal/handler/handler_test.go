// Tests for meta-capi-service handler.
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSetupCAPIRequest_JSON(t *testing.T) {
	body := `{"tenant_id":"550e8400-e29b-41d4-a716-446655440000","pixel_id":"123","access_token":"secret","sample_rate":0.5}`
	var req SetupCAPIRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.TenantID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("TenantID: got %s", req.TenantID)
	}
	if req.PixelID != "123" {
		t.Errorf("PixelID: got %s", req.PixelID)
	}
	if req.AccessToken != "secret" {
		t.Errorf("AccessToken: got %s", req.AccessToken)
	}
	if req.SampleRate != 0.5 {
		t.Errorf("SampleRate: got %f", req.SampleRate)
	}
}

func TestSetupCAPIRequest_Defaults(t *testing.T) {
	body := `{"tenant_id":"550e8400-e29b-41d4-a716-446655440000","pixel_id":"123","access_token":"secret"}`
	var req SetupCAPIRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.SampleRate != 0 {
		t.Errorf("SampleRate default: want 0, got %f", req.SampleRate)
	}
	if req.EventTypes != nil {
		t.Errorf("EventTypes default: want nil, got %v", req.EventTypes)
	}
}

func TestSendEventRequest_JSON(t *testing.T) {
	body := `{"event_name":"Lead","email":"test@example.com","order_value":99.99,"currency":"USD"}`
	var req SendEventRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.EventName != "Lead" {
		t.Errorf("EventName: got %s", req.EventName)
	}
	if req.Email != "test@example.com" {
		t.Errorf("Email: got %s", req.Email)
	}
	if req.OrderValue != 99.99 {
		t.Errorf("OrderValue: got %f", req.OrderValue)
	}
	if req.Currency != "USD" {
		t.Errorf("Currency: got %s", req.Currency)
	}
}

func TestSendEventHandler_MissingTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/events", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.SendEvent(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSendEventHandler_InvalidTenantUUID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader([]byte(`{"event_name":"Lead"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.SendEvent(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetCAPIStatus_MissingTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.GetCAPIStatus(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetCAPIStatus_InvalidTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("X-Tenant-ID", "bad-uuid")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.GetCAPIStatus(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCRMBridgeRequest_JSON(t *testing.T) {
	body := `{"crm_event_type":"deal_won","tenant_id":"550e8400-e29b-41d4-a716-446655440000","deal_value":500,"currency":"USD"}`
	var req CRMBridgeRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.CRMEventType != "deal_won" {
		t.Errorf("CRMEventType: got %s", req.CRMEventType)
	}
	if req.DealValue != 500 {
		t.Errorf("DealValue: got %f", req.DealValue)
	}
	if req.Currency != "USD" {
		t.Errorf("Currency: got %s", req.Currency)
	}
}

func TestCRMBridge_MissingTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bridge", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.CRMBridge(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCRMBridge_InvalidTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bridge", bytes.NewReader([]byte(`{"crm_event_type":"deal_won","tenant_id":"not-uuid"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil)
	if err := s.CRMBridge(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestMapCRMEvents(t *testing.T) {
	tests := []struct {
		crmEvent   string
		wantCAPI   string
	}{
		{"deal_won", "Purchase"},
		{"deal_closed_won", "Purchase"},
		{"lead_created", "Lead"},
		{"lead_new", "Lead"},
		{"contact", "Contact"},
		{"contact_created", "Contact"},
		{"page_view", "ViewContent"},
		{"landing_view", "ViewContent"},
		{"form_submit", "CompleteRegistration"},
		{"form_filled", "CompleteRegistration"},
		{"add_to_cart", "AddToCart"},
		{"checkout", "InitiateCheckout"},
		{"subscribe", "Subscribe"},
		{"unknown_event", "unknown_event"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.crmEvent, func(t *testing.T) {
			got := mapCRMEvents(tt.crmEvent)
			if got != tt.wantCAPI {
				t.Errorf("mapCRMEvents(%q): want %q, got %q", tt.crmEvent, tt.wantCAPI, got)
			}
		})
	}
}
