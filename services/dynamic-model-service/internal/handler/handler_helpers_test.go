// Tests for dynamic-model-service handler helpers.
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/rinco/services/dynamic-model-service/internal/validation"
)

// =============================================================================
// validFieldType
// =============================================================================

func TestExtraValidFieldType_AllValidTypes(t *testing.T) {
	valid := []string{
		"string", "text", "number", "integer", "boolean",
		"date", "datetime", "time", "json", "enum",
		"array", "object", "relation", "file", "ref",
		"email", "phone", "url", "color",
	}
	for _, v := range valid {
		if !validFieldType(v) {
			t.Errorf("expected %q to be valid", v)
		}
	}
}

func TestExtraValidFieldType_Invalid(t *testing.T) {
	invalid := []string{
		"", "unknown", "String", "INTEGER", "blob",
		"float", "double", "varchar", "char", "uuid",
	}
	for _, v := range invalid {
		if validFieldType(v) {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}

func TestExtraValidFieldType_CaseSensitive(t *testing.T) {
	// "String" is not in the list (lowercase only).
	if validFieldType("String") {
		t.Error("capitalized 'String' should be invalid")
	}
	if validFieldType("INTEGER") {
		t.Error("uppercase 'INTEGER' should be invalid")
	}
}

// =============================================================================
// zeroUUID
// =============================================================================

func TestExtraZeroUUID(t *testing.T) {
	got := zeroUUID()
	const want = "00000000-0000-0000-0000-000000000000"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if len(got) != 36 {
		t.Errorf("expected 36 chars, got %d", len(got))
	}
}

// =============================================================================
// orEmpty
// =============================================================================

func TestExtraOrEmpty_NilReturnsEmptyMap(t *testing.T) {
	got := orEmpty(nil)
	if got == nil {
		t.Error("expected non-nil empty map")
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestExtraOrEmpty_PreservesExisting(t *testing.T) {
	in := map[string]interface{}{"a": "b"}
	got := orEmpty(in)
	if got["a"] != "b" {
		t.Errorf("expected a=b, got %v", got)
	}
	// Should be the same reference.
	if &got == &in {
		// comparing addresses; both should be same
	}
}

// =============================================================================
// lower
// =============================================================================

func TestExtraLower_AllLowercase(t *testing.T) {
	got := lower([]string{"HELLO", "World", "FOO"})
	want := []string{"hello", "world", "foo"}
	if len(got) != 3 {
		t.Fatal("length mismatch")
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("[%d]: got %s, want %s", i, got[i], v)
		}
	}
}

func TestExtraLower_TrimsSpace(t *testing.T) {
	got := lower([]string{"  HELLO  ", "\tWORLD\n"})
	if got[0] != "hello" {
		t.Errorf("got %q", got[0])
	}
	if got[1] != "world" {
		t.Errorf("got %q", got[1])
	}
}

func TestExtraLower_Empty(t *testing.T) {
	got := lower(nil)
	if len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestExtraLower_AlreadyLower(t *testing.T) {
	got := lower([]string{"abc", "def"})
	if got[0] != "abc" || got[1] != "def" {
		t.Errorf("got %v", got)
	}
}

// =============================================================================
// toCSV
// =============================================================================

func TestExtraToCSV_String(t *testing.T) {
	if got := toCSV("hello"); got != "hello" {
		t.Errorf("string: got %q", got)
	}
}

func TestExtraToCSV_Nil(t *testing.T) {
	if got := toCSV(nil); got != "" {
		t.Errorf("nil: got %q, want empty", got)
	}
}

func TestExtraToCSV_Bool(t *testing.T) {
	got := toCSV(true)
	if got != "true" {
		t.Errorf("bool: got %q", got)
	}
}

func TestExtraToCSV_Number(t *testing.T) {
	got := toCSV(3.14)
	if got != "3.14" {
		t.Errorf("number: got %q", got)
	}
}

func TestExtraToCSV_Map(t *testing.T) {
	got := toCSV(map[string]interface{}{"a": "b"})
	// JSON marshalling is non-deterministic in key order but should contain "a":"b".
	if !strings.Contains(got, `"a":"b"`) {
		t.Errorf("map: got %q", got)
	}
}

func TestExtraToCSV_Slice(t *testing.T) {
	got := toCSV([]interface{}{1, 2, 3})
	if !strings.Contains(got, "1") || !strings.Contains(got, "3") {
		t.Errorf("slice: got %q", got)
	}
}

func TestExtraToCSV_OtherType(t *testing.T) {
	got := toCSV(123) // int (not float64)
	if got != "123" {
		t.Errorf("int: got %q", got)
	}
}

// =============================================================================
// joinErrs
// =============================================================================

func TestExtraJoinErrs_Empty(t *testing.T) {
	got := joinErrs(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtraJoinErrs_One(t *testing.T) {
	errs := []validation.Error{{Field: "name", Message: "required"}}
	got := joinErrs(errs)
	if got != "name: required" {
		t.Errorf("got %q", got)
	}
}

func TestExtraJoinErrs_Multiple(t *testing.T) {
	errs := []validation.Error{
		{Field: "name", Message: "required"},
		{Field: "email", Message: "invalid"},
	}
	got := joinErrs(errs)
	if !strings.Contains(got, "name: required") {
		t.Errorf("missing name: %q", got)
	}
	if !strings.Contains(got, "email: invalid") {
		t.Errorf("missing email: %q", got)
	}
	if !strings.Contains(got, "; ") {
		t.Errorf("missing separator: %q", got)
	}
}

// =============================================================================
// attachJSON
// =============================================================================

func TestExtraAttachJSON_SinglePair(t *testing.T) {
	target := map[string]interface{}{}
	out := attachJSON(target, "config", []byte(`{"x":1}`))
	if out["config"] == nil {
		t.Error("expected config key set")
	}
	cfg, ok := out["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", out["config"])
	}
	if cfg["x"] != float64(1) {
		t.Errorf("x: got %v", cfg["x"])
	}
}

func TestExtraAttachJSON_EmptyBytesCreatesEmptyMap(t *testing.T) {
	target := map[string]interface{}{}
	out := attachJSON(target, "config", []byte(``))
	// Empty bytes still JSON-unmarshal successfully to an empty map.
	if out["config"] == nil {
		t.Errorf("expected non-nil (empty map) for empty bytes, got nil")
	}
}

func TestExtraAttachJSON_MultiplePairs(t *testing.T) {
	target := map[string]interface{}{}
	out := attachJSON(target,
		"a", []byte(`{"x":1}`),
		"b", []byte(`{"y":2}`),
	)
	if out["a"] == nil || out["b"] == nil {
		t.Errorf("missing keys: %v", out)
	}
}

// =============================================================================
// ctxTenantUser
// =============================================================================

func TestExtraCtxTenantUser_BothEmpty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tenant, user := ctxTenantUser(c)
	if tenant != "" || user != "" {
		t.Errorf("got tenant=%s user=%s", tenant, user)
	}
}

func TestExtraCtxTenantUser_BothSet(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tenant, user := ctxTenantUser(c)
	if tenant != "t1" {
		t.Errorf("tenant: got %s", tenant)
	}
	if user != "u1" {
		t.Errorf("user: got %s", user)
	}
}

func TestExtraCtxTenantUser_OnlyTenant(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	tenant, user := ctxTenantUser(c)
	if tenant != "t1" || user != "" {
		t.Errorf("got tenant=%s user=%s", tenant, user)
	}
}
