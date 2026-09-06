// Package db - tests.
package db

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ===== Context helpers tests =====

func TestSetTenantContext(t *testing.T) {
	ctx := SetTenantContext(context.Background(), "t1", "u1", false)
	if TenantIDFromContext(ctx) != "t1" {
		t.Fatal("tenant mismatch")
	}
	if UserIDFromContext(ctx) != "u1" {
		t.Fatal("user mismatch")
	}
	if IsSuperAdmin(ctx) {
		t.Fatal("should not be admin")
	}
	if !IsAuthenticated(ctx) {
		t.Fatal("should be authenticated")
	}

	ctxAdmin := SetTenantContext(context.Background(), "t1", "u1", true)
	if !IsSuperAdmin(ctxAdmin) {
		t.Fatal("should be admin")
	}
}

func TestBypassRLSContext(t *testing.T) {
	ctx := WithBypassRLS(context.Background())
	if !BypassRLSFromContext(ctx) {
		t.Fatal("bypass should be true")
	}
}

func TestIsPgError(t *testing.T) {
	_, ok := IsPgError(nil)
	if ok {
		t.Fatal("nil should not be pg error")
	}
	if _, ok := IsPgError(errors.New("random")); ok {
		t.Fatal("random error should not match")
	}
}

// ===== Mock-based transaction retry tests =====
// Lưu ý: Test này không cần DB thật - mock pgxpool qua interface.

type stubQuerier struct {
	failures int
	calls    int
}

func TestIsRetryable_Logic(t *testing.T) {
	// Test với error string
	cases := []struct {
		err      error
		retryable bool
	}{
		{nil, false},
		{errors.New("40001 serialization failure"), false}, // pgconn.PgError only
		{errors.New("connection refused"), false},
	}
	for _, c := range cases {
		if got := isRetryable(c.err); got != c.retryable {
			t.Fatalf("isRetryable(%v) = %v, want %v", c.err, got, c.retryable)
		}
	}
}

func TestTxOptionsDefaults(t *testing.T) {
	o := TxOptions{}.defaults()
	if o.MaxRetries != 3 {
		t.Fatalf("expected 3, got %d", o.MaxRetries)
	}
	if o.RetryDelay != 50*time.Millisecond {
		t.Fatalf("expected 50ms, got %v", o.RetryDelay)
	}
	if o.IsoLevel != "read committed" {
		t.Fatalf("expected read committed, got %v", o.IsoLevel)
	}
}

// ===== HTTP server test for splitGoose =====

func TestSplitGoose(t *testing.T) {
	body := `
-- +goose Up
-- +goose StatementBegin
CREATE TABLE foo (id INT);
-- +goose StatementEnd
INSERT INTO foo VALUES (1);

-- +goose Down
DROP TABLE foo;
`
	sql, dir := splitGoose(body)
	if dir != "Up" {
		t.Fatalf("expected Up, got %v", dir)
	}
	if !strings.Contains(sql, "CREATE TABLE foo") {
		t.Fatalf("missing CREATE: %s", sql)
	}
	if strings.Contains(sql, "DROP TABLE") {
		t.Fatalf("should not contain DROP")
	}
}

func TestExtractVersion(t *testing.T) {
	cases := []struct{ in, want string }{
		{"001_init.sql", "001"},
		{"002_create_users.sql", "002"},
		{"v1.0_init.sql", "v1.0"},
		{"init.sql", ""},
	}
	for _, c := range cases {
		if got := extractVersion(c.in); got != c.want {
			t.Fatalf("%s: got %q want %q", c.in, got, c.want)
		}
	}
}

func TestSerializeTxOptions(t *testing.T) {
	o := SerializableTxOptions()
	if o.IsoLevel != "serializable" {
		t.Fatalf("expected serializable, got %v", o.IsoLevel)
	}
	if o.MaxRetries != 5 {
		t.Fatalf("expected 5 retries, got %d", o.MaxRetries)
	}
}

// ===== Stub test for httptest (sanity check) =====

func TestHttpServerStub(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}