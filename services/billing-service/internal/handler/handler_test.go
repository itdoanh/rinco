// Tests for billing-service handler.
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
	s := New(nil, nil)
	if s == nil {
		t.Fatal("New returned nil")
	}
}

func TestCreateSubscription_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/sub", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.CreateSubscription(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreateSubscription_InvalidTenantID(t *testing.T) {
	e := echo.New()
	payload := CreateSubscriptionRequest{
		TenantID:    "not-a-uuid",
		Plan:        "pro",
		Email:       "test@example.com",
		CompanyName: "Test",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/sub", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.CreateSubscription(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreateSubscription_InvalidPlan(t *testing.T) {
	e := echo.New()
	payload := CreateSubscriptionRequest{
		TenantID:    "550e8400-e29b-41d4-a716-446655440000",
		Plan:        "super-duper-enterprise",
		Email:       "test@example.com",
		CompanyName: "Test",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/sub", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.CreateSubscription(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid plan, got %d", rec.Code)
	}
}

// Free plan path requires real repo - covered in integration tests.

func TestGetSubscription_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/sub", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.GetSubscription(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetSubscription_InvalidTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/sub", nil)
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.GetSubscription(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestEnvOr_Default(t *testing.T) {
	v := envOr("billing_test_nonexistent", "default")
	if v != "default" {
		t.Errorf("expected default, got %s", v)
	}
}

func TestEnvOr_Set(t *testing.T) {
	t.Setenv("billing_test_key", "value")
	v := envOr("billing_test_key", "default")
	if v != "value" {
		t.Errorf("expected value, got %s", v)
	}
}

// UpdateSubscription validates plan after DB fetch - requires real repo for testing.

func TestCancelSubscription_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/sub", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.CancelSubscription(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestListInvoices_MissingTenantID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := New(nil, nil)
	if err := s.ListInvoices(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
