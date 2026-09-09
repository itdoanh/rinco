// Tests for observability-service loki client.
package loki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBuildQuery_Empty returns just "{}".
func TestBuildQuery_Empty(t *testing.T) {
	q := Query{}
	got := buildQuery(q)
	if got != "{}" {
		t.Errorf("expected %q, got %q", "{}", got)
	}
}

// TestBuildQuery_ServiceOnly includes just the service label.
func TestBuildQuery_ServiceOnly(t *testing.T) {
	q := Query{Service: "auth-service"}
	got := buildQuery(q)
	if got != `{service="auth-service"}` {
		t.Errorf("got %q", got)
	}
}

// TestBuildQuery_AllLabels includes all three labels.
func TestBuildQuery_AllLabels(t *testing.T) {
	q := Query{Service: "auth", Level: "error", TenantID: "t1"}
	got := buildQuery(q)
	want := `{service="auth",level="error",tenant_id="t1"}`
	if got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}
}

// TestBuildQuery_WithPattern appends a line-filter.
func TestBuildQuery_WithPattern(t *testing.T) {
	q := Query{Service: "auth", Pattern: "null pointer"}
	got := buildQuery(q)
	if !strings.HasSuffix(got, ` |= "null pointer"`) {
		t.Errorf("expected pattern suffix, got %q", got)
	}
}

// TestBuildQuery_PatternOnly works without labels.
func TestBuildQuery_PatternOnly(t *testing.T) {
	q := Query{Pattern: "oops"}
	got := buildQuery(q)
	if !strings.Contains(got, `|= "oops"`) {
		t.Errorf("expected pattern, got %q", got)
	}
}

// TestQuery_ParsesLogs decodes a fake Loki query_range response.
func TestQuery_ParsesLogs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "streams",
				"result": [{
					"stream": {"service": "auth-service", "level": "error"},
					"values": [
						["2025-01-01T00:00:00Z", "boom"],
						["2025-01-01T00:00:01Z", "another"]
					]
				}]
			}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	logs, err := c.Query(context.Background(), Query{Service: "auth"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	if logs[0].Line != "boom" {
		t.Errorf("unexpected log line: %s", logs[0].Line)
	}
	if logs[0].Service != "auth-service" {
		t.Errorf("unexpected service: %s", logs[0].Service)
	}
	if logs[0].Level != "error" {
		t.Errorf("unexpected level: %s", logs[0].Level)
	}
	if logs[1].Line != "another" {
		t.Errorf("unexpected log line: %s", logs[1].Line)
	}
}

// TestQuery_HTTPError verifies the query fails when server returns 5xx.
func TestQuery_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`upstream error`))
	}))
	defer srv.Close()
	c := New(srv.URL)
	_, err := c.Query(context.Background(), Query{Service: "x"})
	if err == nil {
		t.Error("expected error from 500")
	}
}

// TestQuery_EmptyStream doesn't crash on a stream with no values.
func TestQuery_EmptyStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {"resultType": "streams", "result": [{
				"stream": {},
				"values": []
			}]}
		}`))
	}))
	defer srv.Close()
	c := New(srv.URL)
	logs, err := c.Query(context.Background(), Query{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected empty, got %d", len(logs))
	}
}

// TestAggregate_NoLogQL_DefaultExpr verifies the default LogQL is used.
func TestAggregate_NoLogQL_DefaultExpr(t *testing.T) {
	var hitQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitQuery = r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.Aggregate(context.Background(), Query{Service: "auth"})
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if !strings.Contains(hitQuery, "rate({service=\"auth\"}[5m])") {
		t.Errorf("expected default expr, got %q", hitQuery)
	}
}

// TestAggregate_CustomLogQL uses the explicit LogQL field.
func TestAggregate_CustomLogQL(t *testing.T) {
	var hitQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitQuery = r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := New(srv.URL)
	_, err := c.Aggregate(context.Background(), Query{LogQL: `count_over_time({service="x"}[1h])`})
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if !strings.Contains(hitQuery, "count_over_time") {
		t.Errorf("expected custom LogQL, got %q", hitQuery)
	}
}

// TestAggregate_RawJSON decodes the raw response.
func TestAggregate_RawJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"foo":"bar"},"values":[[1,"5"]]}]}}`))
	}))
	defer srv.Close()
	c := New(srv.URL)
	raw, err := c.Aggregate(context.Background(), Query{LogQL: "x"})
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := parsed["data"]; !ok {
		t.Error("raw response should have 'data' key")
	}
}

// TestNew_Defaults checks the constructor.
func TestNew_Defaults(t *testing.T) {
	c := New("http://loki:3100")
	if c == nil {
		t.Fatal("nil client")
	}
	if c.JC == nil {
		t.Error("JSONClient should be set")
	}
}

// TestLogLine_Serialization verifies the LogLine struct JSON tags.
func TestLogLine_Serialization(t *testing.T) {
	ll := LogLine{
		Line:   "hello",
		Labels: map[string]string{"a": "b"},
	}
	b, err := json.Marshal(ll)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"line":"hello"`) {
		t.Errorf("missing line: %s", s)
	}
	if !strings.Contains(s, `"a":"b"`) {
		t.Errorf("missing label: %s", s)
	}
}

// TestQuery_SendsCorrectParameters verifies query params are encoded.
func TestQuery_SendsCorrectParameters(t *testing.T) {
	var hitQuery string
	var hitLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitQuery = r.URL.RawQuery
		hitLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, _ = c.Query(context.Background(), Query{
		Service: "x",
		Level:   "info",
		Limit:   250,
	})
	// Query params are URL-encoded by net/url.Values.Encode() and embedded
	// in the ``query`` parameter itself.  Verify the encoded value contains
	// the expected substrings.
	if !strings.Contains(hitQuery, "service%3D%22x%22") {
		t.Errorf("expected encoded service param, got %q", hitQuery)
	}
	if !strings.Contains(hitQuery, "level%3D%22info%22") {
		t.Errorf("expected encoded level param, got %q", hitQuery)
	}
	if hitLimit != "250" {
		t.Errorf("expected limit=250, got %q", hitLimit)
	}
}

// TestQuery_InvalidTimestampInResponse is robust to malformed timestamps.
func TestQuery_InvalidTimestampInResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {"resultType": "streams", "result": [{
				"stream": {},
				"values": [["not-a-time", "log"]]
			}]}
		}`))
	}))
	defer srv.Close()
	c := New(srv.URL)
	logs, err := c.Query(context.Background(), Query{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if !logs[0].Time.IsZero() {
		t.Errorf("invalid timestamp should result in zero time, got %v", logs[0].Time)
	}
}
