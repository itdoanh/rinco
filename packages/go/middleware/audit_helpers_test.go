// Tests for audit helpers (parseActionResource, isUUID, isNumericID, redactSensitiveData, redactMap).
package middleware

import (
	"strings"
	"testing"
)

func TestParseActionResource_SinglePart(t *testing.T) {
	action, resource := parseActionResource("/healthz")
	if action != "access" {
		t.Errorf("action: got %s", action)
	}
	if resource != "healthz" {
		t.Errorf("resource: got %s", resource)
	}
}

func TestParseActionResource_TwoParts(t *testing.T) {
	action, resource := parseActionResource("/users")
	if action != "access" {
		t.Errorf("action: got %s", action)
	}
	if resource != "users" {
		t.Errorf("resource: got %s", resource)
	}
}

func TestParseActionResource_ActionAtEnd(t *testing.T) {
	action, resource := parseActionResource("/users/create")
	if action != "create" {
		t.Errorf("action: got %s", action)
	}
	if resource != "users" {
		t.Errorf("resource: got %s", resource)
	}
}

func TestParseActionResource_Update(t *testing.T) {
	action, _ := parseActionResource("/posts/update")
	if action != "update" {
		t.Errorf("action: got %s", action)
	}
}

func TestParseActionResource_Delete(t *testing.T) {
	action, _ := parseActionResource("/posts/delete")
	if action != "delete" {
		t.Errorf("action: got %s", action)
	}
}

func TestParseActionResource_List(t *testing.T) {
	action, _ := parseActionResource("/leads/list")
	if action != "list" {
		t.Errorf("action: got %s", action)
	}
}

func TestParseActionResource_Get(t *testing.T) {
	action, _ := parseActionResource("/leads/get")
	if action != "get" {
		t.Errorf("action: got %s", action)
	}
}

func TestParseActionResource_UUID(t *testing.T) {
	action, resource := parseActionResource("/users/550e8400-e29b-41d4-a716-446655440000")
	if action != "get" {
		t.Errorf("action: got %s", action)
	}
	if resource != "users" {
		t.Errorf("resource: got %s", resource)
	}
}

func TestParseActionResource_NumericID(t *testing.T) {
	action, resource := parseActionResource("/users/12345")
	if action != "get" {
		t.Errorf("action: got %s", action)
	}
	if resource != "users" {
		t.Errorf("resource: got %s", resource)
	}
}

func TestParseActionResource_UUIDWithAction(t *testing.T) {
	action, resource := parseActionResource("/users/550e8400-e29b-41d4-a716-446655440000/update")
	if action != "update" {
		t.Errorf("action: got %s", action)
	}
	// resource should not include the UUID
	if !strings.Contains(resource, "users") {
		t.Errorf("resource should contain users: %s", resource)
	}
}

func TestIsUUID_Valid(t *testing.T) {
	if !isUUID("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("valid uuid")
	}
}

func TestIsUUID_Invalid(t *testing.T) {
	if isUUID("not-a-uuid") {
		t.Error("invalid uuid")
	}
}

func TestIsUUID_WrongLength(t *testing.T) {
	if isUUID("12345") {
		t.Error("short string")
	}
}

func TestIsUUID_Empty(t *testing.T) {
	if isUUID("") {
		t.Error("empty")
	}
}

func TestIsUUID_NoDashes(t *testing.T) {
	if isUUID("550e8400e29b41d4a716446655440000") {
		t.Error("no dashes")
	}
}

func TestIsUUID_Uppercase(t *testing.T) {
	if !isUUID("550E8400-E29B-41D4-A716-446655440000") {
		t.Error("uppercase should be accepted")
	}
}

func TestIsNumericID_Valid(t *testing.T) {
	if !isNumericID("12345") {
		t.Error("valid")
	}
}

func TestIsNumericID_Invalid(t *testing.T) {
	if isNumericID("123abc") {
		t.Error("invalid")
	}
}

func TestIsNumericID_Empty(t *testing.T) {
	if isNumericID("") {
		t.Error("empty")
	}
}

func TestRedactSensitiveData_NoMatch(t *testing.T) {
	body := `{"name":"alice"}`
	got := redactSensitiveData(body, []string{"password"})
	if !strings.Contains(got, "alice") {
		t.Errorf("should keep non-sensitive: %s", got)
	}
}

func TestRedactSensitiveData_Match(t *testing.T) {
	body := `{"name":"alice","password":"secret"}`
	got := redactSensitiveData(body, []string{"password"})
	if strings.Contains(got, "secret") {
		t.Errorf("password should be redacted: %s", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("expected [REDACTED]: %s", got)
	}
}

func TestRedactSensitiveData_NestedMap(t *testing.T) {
	body := `{"user":{"name":"alice","password":"secret"}}`
	got := redactSensitiveData(body, []string{"password"})
	if strings.Contains(got, "secret") {
		t.Errorf("nested password should be redacted: %s", got)
	}
}

func TestRedactSensitiveData_Array(t *testing.T) {
	body := `{"items":[{"name":"a","token":"xyz"}]}`
	got := redactSensitiveData(body, []string{"token"})
	if strings.Contains(got, "xyz") {
		t.Errorf("array token should be redacted: %s", got)
	}
}

func TestRedactSensitiveData_InvalidJSON(t *testing.T) {
	body := "not json"
	got := redactSensitiveData(body, []string{"password"})
	if got != "not json" {
		t.Errorf("invalid JSON should be returned as-is")
	}
}

func TestRedactSensitiveData_MultipleFields(t *testing.T) {
	body := `{"password":"a","token":"b","name":"alice"}`
	got := redactSensitiveData(body, []string{"password", "token"})
	// Both should be [REDACTED]
	if !strings.Contains(got, `"password":"[REDACTED]"`) {
		t.Errorf("password should be redacted: %s", got)
	}
	if !strings.Contains(got, `"token":"[REDACTED]"`) {
		t.Errorf("token should be redacted: %s", got)
	}
	if !strings.Contains(got, "alice") {
		t.Errorf("name should remain: %s", got)
	}
}

func TestRedactSensitiveData_CaseInsensitive(t *testing.T) {
	body := `{"Password":"secret"}`
	got := redactSensitiveData(body, []string{"password"})
	if strings.Contains(got, "secret") {
		t.Errorf("case insensitive redaction: %s", got)
	}
}

func TestNewAuditLogChanWriter(t *testing.T) {
	w := NewAuditLogChanWriter(10)
	if w == nil {
		t.Fatal("nil")
	}
	if w.ch == nil {
		t.Error("nil channel")
	}
	if cap(w.ch) != 10 {
		t.Errorf("cap: %d", cap(w.ch))
	}
}

func TestAuditLogChanWriter_LogChan(t *testing.T) {
	w := NewAuditLogChanWriter(1)
	ch := w.LogChan()
	if ch == nil {
		t.Error("nil")
	}
}

func TestNewAuditLogSlogWriter(t *testing.T) {
	w := NewAuditLogSlogWriter(nil)
	if w == nil {
		t.Fatal("nil")
	}
	if w.logger != nil {
		t.Error("expected nil logger")
	}
}
