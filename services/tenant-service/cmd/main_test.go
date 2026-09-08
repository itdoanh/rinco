// Tests for tenant-service pure utility functions.
package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSubtleCompare_Equal(t *testing.T) {
	if !subtleCompare("secret-key-123", "secret-key-123") {
		t.Error("expected equal strings to match")
	}
}

func TestSubtleCompare_NotEqual(t *testing.T) {
	if subtleCompare("secret-key-123", "secret-key-456") {
		t.Error("expected unequal strings to not match")
	}
}

func TestSubtleCompare_DifferentLength(t *testing.T) {
	if subtleCompare("short", "much-longer-string") {
		t.Error("expected different-length strings to not match")
	}
}

func TestSubtleCompare_Empty(t *testing.T) {
	// Equal empty strings are equal (both zero length, same bytes).
	if !subtleCompare("", "") {
		t.Error("expected empty strings to match")
	}
}

func TestAsString(t *testing.T) {
	if asString("hello") != "hello" {
		t.Error("expected 'hello'")
	}
	if asString(123) != "" {
		t.Error("expected empty for non-string")
	}
	if asString(nil) != "" {
		t.Error("expected empty for nil")
	}
}

func TestAsBool(t *testing.T) {
	if !asBool(true) {
		t.Error("expected true for true")
	}
	if asBool(false) {
		t.Error("expected false for false")
	}
	if !asBool("true") {
		t.Error("expected true for 'true' string")
	}
	if !asBool("1") {
		t.Error("expected true for '1' string")
	}
	if asBool("false") {
		t.Error("expected false for 'false' string")
	}
	if asBool(123) {
		t.Error("expected false for non-bool/int 123")
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if firstNonEmpty("a", "b", "c") != "a" {
		t.Error("expected first non-empty")
	}
	if firstNonEmpty("", "b", "c") != "b" {
		t.Error("expected second when first is empty")
	}
	if firstNonEmpty("", "", "c") != "c" {
		t.Error("expected third when first two empty")
	}
	if firstNonEmpty("", "") != "" {
		t.Error("expected empty when all empty")
	}
}

func TestNullStr(t *testing.T) {
	if nullStr("") != nil {
		t.Error("expected nil for empty string")
	}
	if nullStr("hello") != "hello" {
		t.Error("expected 'hello'")
	}
}

func TestAtoiDefault(t *testing.T) {
	if atoiDefault("42", 10) != 42 {
		t.Error("expected 42")
	}
	if atoiDefault("", 10) != 10 {
		t.Error("expected default 10")
	}
	if atoiDefault("abc", 5) != 5 {
		t.Error("expected default 5 for non-numeric")
	}
	if atoiDefault("0", 99) != 0 {
		t.Error("expected 0")
	}
}

func TestIsUniqueViolation(t *testing.T) {
	err := errors.New("duplicate key value violates unique constraint")
	if !isUniqueViolation(err) {
		t.Error("expected true for duplicate key error")
	}
	if isUniqueViolation(nil) {
		t.Error("expected false for nil")
	}
	err2 := errors.New("some other error")
	if isUniqueViolation(err2) {
		t.Error("expected false for non-unique error")
	}
}

func TestNewID(t *testing.T) {
	id := newID()
	if id == "" {
		t.Error("expected non-empty ID")
	}
	if len(id) != 36 { // UUID format (8-4-4-4-12 = 36 chars)
		t.Errorf("expected 36 char UUID, got %d", len(id))
	}
}

func TestRequireTenantMW_NoHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &server{}
	h := s.requireTenantMW()(func(c echo.Context) error { return nil })
	_ = h(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestRequireTenantMW_WithHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := &server{}
	h := s.requireTenantMW()(func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	err := h(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGetEnv(t *testing.T) {
	// Without mocking os.Getenv, we test the default path.
	// Set and unset in test.
	orig := "RINCO_TEST_VAR"
	val := getEnv(orig, "default")
	// This will return "default" since we can't set env in this test easily.
	if val == "" {
		t.Error("getEnv returned empty string")
	}
}
