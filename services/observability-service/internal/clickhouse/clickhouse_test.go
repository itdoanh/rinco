// Tests for observability-service clickhouse writer.
package clickhouse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// AuditRecord
// =============================================================================

// TestAuditRecord_MarshalJSON_AutoID verifies an ID is generated when empty.
func TestAuditRecord_MarshalJSON_AutoID(t *testing.T) {
	r := AuditRecord{Action: "test"}
	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	id, _ := got["id"].(string)
	if !strings.Contains(id, "-") {
		t.Errorf("expected generated ID with separator, got %q", id)
	}
}

// TestAuditRecord_MarshalJSON_AutoTimestamp verifies TS is set when zero.
func TestAuditRecord_MarshalJSON_AutoTimestamp(t *testing.T) {
	r := AuditRecord{Action: "test"}
	b, _ := r.MarshalJSON()
	var got map[string]any
	_ = json.Unmarshal(b, &got)
	ts, ok := got["ts"].(string)
	if !ok {
		t.Error("ts should be present")
	}
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("ts should parse as RFC3339, got %q: %v", ts, err)
	}
}

// TestAuditRecord_MarshalJSON_PreservesExplicit verifies explicit values win.
func TestAuditRecord_MarshalJSON_PreservesExplicit(t *testing.T) {
	r := AuditRecord{
		ID:          "fixed-id",
		Action:      "test",
		TenantID:    "t1",
		ActorUserID: "u1",
		ActorIP:     "1.2.3.4",
		TS:          time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	b, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"id":"fixed-id"`) {
		t.Errorf("ID not preserved: %s", s)
	}
	if !strings.Contains(s, `"tenant_id":"t1"`) {
		t.Errorf("TenantID not preserved: %s", s)
	}
	if !strings.Contains(s, `"actor_user_id":"u1"`) {
		t.Errorf("ActorUserID not preserved: %s", s)
	}
	if !strings.Contains(s, `"actor_ip":"1.2.3.4"`) {
		t.Errorf("ActorIP not preserved: %s", s)
	}
}

// TestAuditRecord_PayloadEncodes verifies Payload is JSON-encoded.
func TestAuditRecord_PayloadEncodes(t *testing.T) {
	r := AuditRecord{Action: "x", Payload: map[string]any{"k": "v"}}
	b, _ := r.MarshalJSON()
	s := string(b)
	if !strings.Contains(s, `"payload":"{\"k\":\"v\"}"`) {
		t.Errorf("payload should be JSON-encoded string: %s", s)
	}
}

// TestAuditRecord_PayloadNil is encoded as empty string.
func TestAuditRecord_PayloadNil(t *testing.T) {
	r := AuditRecord{Action: "x"}
	b, _ := r.MarshalJSON()
	s := string(b)
	if !strings.Contains(s, `"payload":""`) {
		t.Errorf("nil payload should be empty string: %s", s)
	}
}

// =============================================================================
// Writer
// =============================================================================

// TestNewWriter_TrimsSlash verifies trailing slash is trimmed.
func TestNewWriter_TrimsSlash(t *testing.T) {
	w := NewWriter("http://clickhouse:8123/")
	if w.url != "http://clickhouse:8123" {
		t.Errorf("expected trailing slash trimmed, got %q", w.url)
	}
}

// TestNewWriter_NoSlash verifies URL is kept as-is when no trailing slash.
func TestNewWriter_NoSlash(t *testing.T) {
	w := NewWriter("http://clickhouse:8123")
	if w.url != "http://clickhouse:8123" {
		t.Errorf("expected URL unchanged, got %q", w.url)
	}
}

// TestInsert_EmptyURL fails fast without making any HTTP call.
func TestInsert_EmptyURL(t *testing.T) {
	w := NewWriter("")
	err := w.Insert(context.Background(), AuditRecord{Action: "x"})
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

// TestInsert_HTTPError verifies non-2xx is surfaced.
func TestInsert_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Syntax error`))
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	err := w.Insert(context.Background(), AuditRecord{Action: "x"})
	if err == nil {
		t.Error("expected error from 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status: %v", err)
	}
}

// TestInsert_Success sends a record.
func TestInsert_Success(t *testing.T) {
	var hitBody string
	var hitQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		hitBody = string(buf[:n])
		hitQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	err := w.Insert(context.Background(), AuditRecord{
		Action:   "user.created",
		TenantID: "t1",
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if !strings.Contains(hitQuery, "INSERT") {
		t.Errorf("expected INSERT query, got %q", hitQuery)
	}
	if !strings.Contains(hitBody, "user.created") {
		t.Errorf("body missing action: %s", hitBody)
	}
}

// TestInsert_SetsTimestamp verifies TS is set when zero.
func TestInsert_SetsTimestamp(t *testing.T) {
	var hitBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		hitBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	_ = w.Insert(context.Background(), AuditRecord{Action: "x"})
	if !strings.Contains(hitBody, `"ts":`) {
		t.Errorf("body should have ts: %s", hitBody)
	}
}

// TestInsert_SetsID verifies ID is generated when empty.
func TestInsert_SetsID(t *testing.T) {
	var hitBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		hitBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	_ = w.Insert(context.Background(), AuditRecord{Action: "x", TenantID: "t1"})
	if !strings.Contains(hitBody, `"id":`) {
		t.Errorf("body should have id: %s", hitBody)
	}
}

// TestInsertBatch_Empty is a no-op for empty input.
func TestInsertBatch_Empty(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	err := w.InsertBatch(context.Background(), nil)
	if err != nil {
		t.Errorf("empty batch: %v", err)
	}
	if hits != 0 {
		t.Errorf("no HTTP call should be made for empty batch, got %d", hits)
	}
}

// TestInsertBatch_EmptyURL fails fast.
func TestInsertBatch_EmptyURL(t *testing.T) {
	w := NewWriter("")
	err := w.InsertBatch(context.Background(), []AuditRecord{{Action: "x"}})
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

// TestInsertBatch_HTTPError surfaces non-2xx.
func TestInsertBatch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	err := w.InsertBatch(context.Background(), []AuditRecord{{Action: "x"}})
	if err == nil {
		t.Error("expected error from 400 response")
	}
}

// TestInsertBatch_Success sends multiple records in one request.
func TestInsertBatch_Success(t *testing.T) {
	var hitBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 8192)
		n, _ := r.Body.Read(buf)
		hitBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	err := w.InsertBatch(context.Background(), []AuditRecord{
		{Action: "a"},
		{Action: "b"},
		{Action: "c"},
	})
	if err != nil {
		t.Fatalf("insertbatch: %v", err)
	}
	// Each record should be a separate JSON line.
	lines := strings.Split(strings.TrimSpace(hitBody), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %s", len(lines), hitBody)
	}
}

// =============================================================================
// SafeTenant
// =============================================================================

// TestSafeTenant_Strips verifies dangerous characters are removed.
func TestSafeTenant_Strips(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"abc", "abc"},
		{"abc'xyz", "abcxyz"}, // single quote
		{"abc\\xyz", "abcxyz"}, // backslash
		{"abc\000def", "abcdef"}, // NUL
		{"", ""},
		{"safe_tenant-1", "safe_tenant-1"},
	}
	for _, tc := range cases {
		got := SafeTenant(tc.in)
		if got != tc.want {
			t.Errorf("SafeTenant(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// =============================================================================
// LogError
// =============================================================================

// TestLogError_NonNilError verifies the helper handles non-nil errors gracefully.
// Note: LogError panics on nil error (calls err.Error() without nil check) —
// callers must guard against nil before calling.
func TestLogError_NonNilError(t *testing.T) {
	// Non-nil errors are handled gracefully (no panic path).
	LogError(context.Background(), "insert", context.Canceled)
	// This would panic if uncommented: LogError(ctx, "x", nil)
}

// =============================================================================
// encodePayload
// =============================================================================

// TestEncodePayload_VariousTypes verifies the payload encoding for different Go types.
func TestEncodePayload_VariousTypes(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", `"hello"`},
		{"map", map[string]any{"k": "v"}, `{"k":"v"}`},
		{"int", 42, "42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := encodePayload(tc.input)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestEncodePayload_MarshalError covers a payload that can't be marshalled.
func TestEncodePayload_MarshalError(t *testing.T) {
	// Channels can't be marshalled as JSON.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("should not panic: %v", r)
		}
	}()
	got := encodePayload(make(chan int))
	if got != "" {
		t.Errorf("unmarshallable should return empty string, got %q", got)
	}
}
