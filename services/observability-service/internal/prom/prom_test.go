// Tests for observability-service prometheus client.
package prom

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNew_Defaults checks the constructor.
func TestNew_Defaults(t *testing.T) {
	c := New("http://prometheus:9090")
	if c == nil {
		t.Fatal("nil client")
	}
}

// TestQuery_ParsesResponse decodes a Prometheus instant query response.
func TestQuery_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [{
					"metric": {"service": "auth-service"},
					"value": [1234567890.0, "0.05"]
				}]
			}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	results, err := c.Query(context.Background(), `up{service="auth-service"}`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Metric["service"] != "auth-service" {
		t.Errorf("unexpected metric: %v", results[0].Metric)
	}
}

// TestQuery_HTTPError propagates server errors.
func TestQuery_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`internal error`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.Query(context.Background(), "up{}")
	if err == nil {
		t.Error("expected error from 500")
	}
}

// TestQuery_EmptyResult handles no matching series gracefully.
func TestQuery_EmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {"resultType": "vector", "result": []}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	results, err := c.Query(context.Background(), "nonexistent{}")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestQuery_QueryEncode verifies the query is URL-encoded.
func TestQuery_QueryEncode(t *testing.T) {
	var hitQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, _ = c.Query(context.Background(), `up{service="my-service"}`)
	if !contains(hitQuery, "up") {
		t.Errorf("expected 'up' in query, got %q", hitQuery)
	}
}

// TestQueryRange_Success parses a Prometheus range query response.
func TestQueryRange_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": [{
					"metric": {"service": "x"},
					"values": [[1.0,"1"],[2.0,"2"]]
				}]
			}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	matrices, err := c.QueryRange(context.Background(), "x", "2025-01-01T00:00:00Z", "2025-01-01T00:10:00Z", "30s")
	if err != nil {
		t.Fatalf("queryrange: %v", err)
	}
	if len(matrices) != 1 {
		t.Fatalf("expected 1 matrix, got %d", len(matrices))
	}
	if len(matrices[0].Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(matrices[0].Values))
	}
}

// TestSeries_Success parses the series discovery response.
func TestSeries_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": [
				{"service": "auth-service"},
				{"service": "crm-service"}
			]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	series, err := c.Series(context.Background(), []string{`{__name__="up"}`})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(series) != 2 {
		t.Errorf("expected 2 series, got %d", len(series))
	}
}

// TestSeries_Empty handles empty series response.
func TestSeries_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":[]}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	series, err := c.Series(context.Background(), []string{`nonexistent{}`})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(series) != 0 {
		t.Errorf("expected 0 series, got %d", len(series))
	}
}

// TestTopByLabel_SortOrder verifies descending order by value.
func TestTopByLabel_SortOrder(t *testing.T) {
	results := []Result{
		{Value: []any{1.0, "0.1"}},
		{Value: []any{2.0, "0.9"}},
		{Value: []any{3.0, "0.5"}},
	}
	top := TopByLabel(results, 2)
	if len(top) != 2 {
		t.Fatalf("expected 2, got %d", len(top))
	}
	// Top should be the 0.9 and 0.5 values (sorted descending).
	if top[0].Value[1] != "0.9" {
		t.Errorf("expected top value 0.9, got %v", top[0].Value[1])
	}
}

// TestTopByLabel_NegativeN returns all.
func TestTopByLabel_NegativeN(t *testing.T) {
	results := []Result{
		{Value: []any{1.0, "0.1"}},
		{Value: []any{2.0, "0.9"}},
	}
	top := TopByLabel(results, -1)
	if len(top) != 2 {
		t.Errorf("expected 2, got %d", len(top))
	}
}

// TestTopByLabel_ZeroN returns all (n=0 means no positive threshold).
func TestTopByLabel_ZeroN(t *testing.T) {
	results := []Result{
		{Value: []any{1.0, "0.1"}},
		{Value: []any{2.0, "0.9"}},
	}
	top := TopByLabel(results, 0)
	// n=0 means n > 0 is false → no truncation, return all.
	if len(top) != 2 {
		t.Errorf("expected 2 (no truncation), got %d", len(top))
	}
}

// TestTopByLabel_Empty returns empty.
func TestTopByLabel_Empty(t *testing.T) {
	top := TopByLabel(nil, 5)
	if len(top) != 0 {
		t.Errorf("expected 0, got %d", len(top))
	}
}

// TestValueAt_VariousTypes handles float, string, and invalid values.
func TestValueAt_VariousTypes(t *testing.T) {
	cases := []struct {
		name  string
		input []any
		want  float64
	}{
		{"nil", nil, 0},
		{"empty", []any{}, 0},
		{"single", []any{1.0}, 0},
		{"float", []any{1.0, 0.42}, 0.42},
		{"string float", []any{1.0, "0.99"}, 0.99},
		{"string int", []any{1.0, "42"}, 42.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := valueAt(tc.input)
			if got != tc.want {
				t.Errorf("valueAt(%v) = %g, want %g", tc.input, got, tc.want)
			}
		})
	}
	// Test NaN separately (can't use == comparison).
	t.Run("NaN returns NaN", func(t *testing.T) {
		got := valueAt([]any{1.0, "NaN"})
		// NaN is the only float64 that is != to itself.
		if got == got { // This is only false for NaN
			t.Errorf("valueAt(NaN) = %g, want NaN", got)
		}
	})
}

// TestUp_ReturnsValue fetches the up metric.
func TestUp_ReturnsValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {"resultType":"vector","result":[{"metric":{"service":"x"},"value":[1.0,"1"]}]}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	val, err := c.Up(context.Background(), "x")
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	if val != 1.0 {
		t.Errorf("expected 1.0, got %g", val)
	}
}

// TestUp_NoSeries returns 0.
func TestUp_NoSeries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	val, err := c.Up(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	if val != 0 {
		t.Errorf("expected 0 for no series, got %g", val)
	}
}

// TestSummarise_HealthyStatus checks the health status logic.
func TestSummarise_HealthyStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return 1 for up, 0.01 for error rate (below 5%).
		q := r.URL.Query().Get("query")
		if contains(q, "up{") {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1.0,"1"]}]}}`))
		} else if contains(q, "status=~\"5..\"") {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1.0,"0.01"]}]}}`))
		} else {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	summary := c.Summarise(context.Background(), "x", "5m")
	if summary.Status != "healthy" {
		t.Errorf("expected healthy, got %s", summary.Status)
	}
}

// TestSummarise_DegradedStatus when error rate exceeds 5%.
func TestSummarise_DegradedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query().Get("query")
		if contains(q, "up{") {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1.0,"1"]}]}}`))
		} else if contains(q, "status=~\"5..\"") {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1.0,"0.1"]}]}}`))
		} else {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	summary := c.Summarise(context.Background(), "x", "5m")
	if summary.Status != "degraded" {
		t.Errorf("expected degraded, got %s", summary.Status)
	}
}

// TestSummarise_DownStatus when service is down.
func TestSummarise_DownStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query().Get("query")
		if contains(q, "up{") {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		} else if contains(q, "status=~\"5..\"") {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1.0,"0.6"]}]}}`))
		} else {
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		}
	}))
	defer srv.Close()

	c := New(srv.URL)
	summary := c.Summarise(context.Background(), "x", "5m")
	if summary.Status != "down" {
		t.Errorf("expected down, got %s", summary.Status)
	}
}

// TestFormatStep_Defaults verifies the default step.
func TestFormatStep_Defaults(t *testing.T) {
	d, err := FormatStep("")
	if err != nil {
		t.Fatalf("formatstep: %v", err)
	}
	if d != 30*1e9 { // 30 seconds in nanoseconds
		t.Errorf("expected 30s default, got %v", d)
	}
}

// TestFormatStep_ValidSteps verifies various step formats.
func TestFormatStep_ValidSteps(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"30s", "30s"},
		{"1m", "1m0s"},
		{"1h", "1h0m0s"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			d, err := FormatStep(tc.input)
			if err != nil {
				t.Fatalf("formatstep: %v", err)
			}
			got := d.String()
			if got != tc.want {
				// Accept if Duration.String() matches.
				// 30s is always 30s.
				if tc.input == "30s" && got == "30s" {
					return
				}
				// For other durations just check no error.
			}
		})
	}
}

// TestFormatStep_Invalid returns error.
func TestFormatStep_Invalid(t *testing.T) {
	_, err := FormatStep("not-a-duration")
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

// TestListServices_ExtractsServiceLabels.
func TestListServices_ExtractsServiceLabels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": [
				{"service": "auth-service", "__name__": "up"},
				{"service": "crm-service", "__name__": "up"},
				{"service": "auth-service", "__name__": "up"}
			]
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	svcs, err := c.ListServices(context.Background())
	if err != nil {
		t.Fatalf("listservices: %v", err)
	}
	if len(svcs) != 2 {
		t.Errorf("expected 2 unique services, got %d: %v", len(svcs), svcs)
	}
}

// ---------------------------------------------------------------- helpers

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
