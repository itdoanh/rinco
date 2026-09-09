package clickhouse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewWriterTrimSlash(t *testing.T) {
	w := NewWriter("http://localhost:8123/")
	if !strings.HasSuffix(w.url, ":8123") {
		t.Errorf("trailing slash not trimmed: %q", w.url)
	}
	if strings.HasSuffix(w.url, "/") {
		t.Errorf("trailing slash still present: %q", w.url)
	}
}

func TestNewWriterNoSlash(t *testing.T) {
	w := NewWriter("http://localhost:8123")
	if w.url != "http://localhost:8123" {
		t.Errorf("url = %q", w.url)
	}
}

func TestSafeTenantStripsBackslash(t *testing.T) {
	got := SafeTenant(`tenant\bad`)
	if got != "tenantbad" {
		t.Errorf("backslash not stripped: %q", got)
	}
}

func TestSafeTenantStripsSingleQuote(t *testing.T) {
	got := SafeTenant("tena'nt")
	if got != "tenant" {
		t.Errorf("quote not stripped: %q", got)
	}
}

func TestSafeTenantStripsNUL(t *testing.T) {
	got := SafeTenant("tena\x00nt")
	if got != "tenant" {
		t.Errorf("NUL not stripped: %q", got)
	}
}

func TestSafeTenantKeepsOthers(t *testing.T) {
	got := SafeTenant("tenant-x.y_1")
	if got != "tenant-x.y_1" {
		t.Errorf("normal chars modified: %q", got)
	}
}

func TestEncodePayloadNil(t *testing.T) {
	if got := encodePayload(nil); got != "" {
		t.Errorf("nil should encode empty, got %q", got)
	}
}

func TestEncodePayloadString(t *testing.T) {
	got := encodePayload("hello")
	if got != `"hello"` {
		t.Errorf("got %q", got)
	}
}

func TestEncodePayloadMap(t *testing.T) {
	m := map[string]any{"a": 1, "b": "two"}
	got := encodePayload(m)
	if !strings.Contains(got, `"a":1`) || !strings.Contains(got, `"b":"two"`) {
		t.Errorf("map not encoded: %s", got)
	}
}

func TestAuditRecordMarshalDefaultsID(t *testing.T) {
	r := AuditRecord{
		TenantID: "t1",
		Action:   "create",
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	if id, _ := m["id"].(string); id == "" {
		t.Error("id empty")
	}
	if ts, _ := m["ts"].(string); ts == "" {
		t.Error("ts empty")
	}
}

func TestAuditRecordMarshalPreservesID(t *testing.T) {
	r := AuditRecord{
		ID:       "fixed-id",
		TenantID: "t1",
	}
	data, _ := json.Marshal(r)
	if !strings.Contains(string(data), `"id":"fixed-id"`) {
		t.Error("explicit id lost")
	}
}

func TestInsertEmptyURL(t *testing.T) {
	w := NewWriter("")
	w.url = ""
	err := w.Insert(t.Context(), AuditRecord{TenantID: "t1"})
	if err == nil {
		t.Error("expected error for empty url")
	}
}

func TestInsertSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "JSONEachRow") {
			t.Error("query missing JSONEachRow")
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	if err := w.Insert(t.Context(), AuditRecord{ID: "x", TenantID: "t"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInsertHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("syntax error"))
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	err := w.Insert(t.Context(), AuditRecord{ID: "x", TenantID: "t"})
	if err == nil {
		t.Error("expected error for HTTP 400")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error should mention status: %v", err)
	}
}

func TestInsertServerUnreachable(t *testing.T) {
	w := NewWriter("http://localhost:1")
	ctx, cancel := contextWithTimeout()
	defer cancel()
	err := w.Insert(ctx, AuditRecord{TenantID: "t"})
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestInsertBatchEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for empty batch")
		w.WriteHeader(200)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	if err := w.InsertBatch(t.Context(), nil); err != nil {
		t.Errorf("empty batch: %v", err)
	}
}

func TestInsertBatchSuccess(t *testing.T) {
	called := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(200)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	records := []AuditRecord{
		{ID: "1", TenantID: "t1"},
		{ID: "2", TenantID: "t2"},
	}
	if err := w.InsertBatch(t.Context(), records); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 server call, got %d", called)
	}
}

func TestInsertBatchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	w := NewWriter(srv.URL)
	err := w.InsertBatch(t.Context(), []AuditRecord{{ID: "1"}})
	if err == nil {
		t.Error("expected error for 500")
	}
}

func TestInsertBatchEmptyURL(t *testing.T) {
	w := NewWriter("")
	w.url = ""
	if err := w.InsertBatch(t.Context(), []AuditRecord{{ID: "1"}}); err == nil {
		t.Error("expected error for empty url")
	}
}

func TestLogErrorNoCrash(t *testing.T) {
	// Just ensure no panic; slog captures into default
	LogError(t.Context(), "insert", err("boom"))
}

func TestAuditRecordPayloadEncoded(t *testing.T) {
	r := AuditRecord{
		ID:       "x",
		TenantID: "t",
		Payload:  map[string]string{"k": "v"},
	}
	data, _ := json.Marshal(r)
	// Payload field is encoded as a string (JSON)
	if !strings.Contains(string(data), `"payload":"{\"k\":\"v\"}"`) {
		t.Errorf("payload encoding wrong: %s", data)
	}
}

func TestAuditRecordPayloadNilEmpty(t *testing.T) {
	r := AuditRecord{ID: "x", TenantID: "t"}
	data, _ := json.Marshal(r)
	if !strings.Contains(string(data), `"payload":""`) {
		t.Errorf("payload nil should be empty: %s", data)
	}
}

// helpers
type strErr string

func (s strErr) Error() string { return string(s) }

func err(s string) error { return strErr(s) }

func contextWithTimeout() (ctx ctxT, cancel func()) {
	c := make(chan struct{})
	return ctxT{ch: c}, func() { close(c) }
}

type ctxT struct{ ch chan struct{} }

func (c ctxT) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c ctxT) Done() <-chan struct{}       { return c.ch }
func (c ctxT) Err() error                   { return nil }
func (c ctxT) Value(key any) any            { return nil }
