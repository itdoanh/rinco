// More tests for tenant-service: scanTenant, firstMap, isUniqueViolation, jsonErr.
package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// fakeRowScanner is a tiny rowScanner stub used to drive scanTenant() through
// every code path without needing a real pgx row.
type fakeRowScanner struct {
	values []any
	err    error
	calls  int
}

func (f *fakeRowScanner) Scan(dest ...any) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	if len(dest) != len(f.values) {
		return errors.New("dest/values length mismatch")
	}
	for i, v := range f.values {
		// Each destination is a typed pointer; we use reflection-free copy
		// via type assertion only on the supported shapes used by scanTenant.
		switch d := dest[i].(type) {
		case *string:
			s, _ := v.(string)
			*d = s
		case **time.Time:
			t, _ := v.(*time.Time)
			*d = t
		case *time.Time:
			t, _ := v.(time.Time)
			*d = t
		case *[]byte:
			b, _ := v.([]byte)
			*d = b
		default:
			return errors.New("unsupported destination type")
		}
	}
	return nil
}

func TestScanTenant_OK(t *testing.T) {
	id := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Second)
	r := &fakeRowScanner{values: []any{
		id, "acme", "Acme Inc", "active", "pro",
		[]byte(`{"theme":"dark"}`),
		[]byte(`{"primary":"#000"}`),
		now, now,
	}}
	got, err := scanTenant(r)
	if err != nil {
		t.Fatalf("scanTenant: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %q, want %q", got.ID, id)
	}
	if got.Slug != "acme" {
		t.Errorf("Slug: got %q", got.Slug)
	}
	if got.Settings["theme"] != "dark" {
		t.Errorf("Settings[theme]: got %v", got.Settings["theme"])
	}
	if got.Branding["primary"] != "#000" {
		t.Errorf("Branding[primary]: got %v", got.Branding["primary"])
	}
	if r.calls != 1 {
		t.Errorf("Scan called %d times", r.calls)
	}
}

func TestScanTenant_EmptyJSONDefaults(t *testing.T) {
	r := &fakeRowScanner{values: []any{
		"id", "slug", "name", "active", "free",
		[]byte{}, []byte{},
		time.Time{}, time.Time{},
	}}
	got, err := scanTenant(r)
	if err != nil {
		t.Fatalf("scanTenant: %v", err)
	}
	if got.Settings == nil {
		t.Error("Settings should default to non-nil empty map")
	}
	if got.Branding == nil {
		t.Error("Branding should default to non-nil empty map")
	}
	if len(got.Settings) != 0 || len(got.Branding) != 0 {
		t.Errorf("expected empty maps; got settings=%v branding=%v", got.Settings, got.Branding)
	}
}

func TestScanTenant_InvalidJSONIgnored(t *testing.T) {
	r := &fakeRowScanner{values: []any{
		"id", "slug", "name", "active", "free",
		[]byte("not-valid-json"), []byte("{also bad"),
		time.Time{}, time.Time{},
	}}
	got, err := scanTenant(r)
	if err != nil {
		t.Fatalf("scanTenant: %v", err)
	}
	if got.Settings == nil || got.Branding == nil {
		t.Errorf("should default maps on invalid JSON; got settings=%v branding=%v", got.Settings, got.Branding)
	}
}

func TestScanTenant_ScanError(t *testing.T) {
	want := errors.New("scan failed")
	r := &fakeRowScanner{err: want}
	_, err := scanTenant(r)
	if err == nil || !errors.Is(err, want) {
		t.Errorf("scanTenant should bubble Scan error; got %v", err)
	}
}

func TestFirstMap_Nil(t *testing.T) {
	got := firstMap(nil)
	if got == nil {
		t.Fatal("firstMap(nil) should return empty map (not nil)")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestFirstMap_Empty(t *testing.T) {
	got := firstMap(map[string]any{})
	if got == nil {
		t.Fatal("firstMap(empty) should return empty map")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestFirstMap_Populated(t *testing.T) {
	in := map[string]any{"a": 1, "b": "x"}
	got := firstMap(in)
	if got["a"] != 1 || got["b"] != "x" {
		t.Errorf("firstMap lost data: %v", got)
	}
	// Mutating the returned map should not mutate the original (defensive copy).
	got["c"] = true
	if _, exists := in["c"]; exists {
		t.Error("firstMap should return a copy, not share storage")
	}
}

func TestIsUniqueViolation_DuplicateKey(t *testing.T) {
	if !isUniqueViolation(errors.New(`ERROR: duplicate key value violates unique constraint "x" (SQLSTATE 23505)`)) {
		t.Error("duplicate key message should be detected")
	}
}

func TestIsUniqueViolation_OtherError(t *testing.T) {
	if isUniqueViolation(errors.New("connection reset by peer")) {
		t.Error("non-duplicate error should not be detected as unique violation")
	}
}

func TestIsUniqueViolation_Nil(t *testing.T) {
	if isUniqueViolation(nil) {
		t.Error("nil error should not be detected as unique violation")
	}
}

func TestAtoiDefault_Empty(t *testing.T) {
	if got := atoiDefault("", 7); got != 7 {
		t.Errorf("empty: got %d, want 7", got)
	}
}

func TestAtoiDefault_Invalid(t *testing.T) {
	if got := atoiDefault("not-a-number", 7); got != 7 {
		t.Errorf("invalid: got %d, want 7", got)
	}
}

func TestAtoiDefault_Negative(t *testing.T) {
	if got := atoiDefault("-12", 7); got != -12 {
		t.Errorf("negative: got %d, want -12", got)
	}
}

func TestAtoiDefault_Valid(t *testing.T) {
	if got := atoiDefault("42", 7); got != 42 {
		t.Errorf("valid: got %d, want 42", got)
	}
}

func TestNullStr_EmptyReturnsNil(t *testing.T) {
	got := nullStr("")
	if got != nil {
		t.Errorf("nullStr empty: got %v, want nil", got)
	}
}

func TestNullStr_NonEmptyReturnsString(t *testing.T) {
	got := nullStr("abc")
	s, ok := got.(string)
	if !ok {
		t.Fatalf("nullStr non-empty: got %T, want string", got)
	}
	if s != "abc" {
		t.Errorf("nullStr: got %q, want abc", s)
	}
}

func TestJsonErr_StatusAndBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := jsonErr(c, http.StatusBadRequest, "BAD_CODE", "bad input")
	if err != nil {
		t.Fatalf("jsonErr should return nil (Echo convention); got %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"BAD_CODE"`) {
		t.Errorf("body missing code: %s", body)
	}
	if !strings.Contains(body, `"message":"bad input"`) {
		t.Errorf("body missing message: %s", body)
	}
}
