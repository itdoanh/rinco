// Package prom is a thin client over the Prometheus HTTP API.  We
// support query (instant), query_range (range), and series (label
// discovery) — enough to power the /v1/metrics endpoint and the service
// health aggregator.
package prom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/rinco/services/observability-service/internal/platform"
)

// Client wraps the Prometheus HTTP API.
type Client struct {
	JC *platform.JSONClient
}

// New constructs a Prom client.
func New(baseURL string) *Client {
	return &Client{JC: platform.NewJSONClient(baseURL, "", 15*time.Second)}
}

// Matrix is a single time series in range queries.
type Matrix struct {
	Metric map[string]string `json:"metric"`
	Values [][]any           `json:"values"`
}

// Result is a single instant query sample.
type Result struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}

// QueryResponse is a Prometheus instant query response.
type QueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     []json.RawMessage `json:"result"`
	} `json:"data"`
}

// RangeResponse is a Prometheus range query response.
type RangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string   `json:"resultType"`
		Result     []Matrix `json:"result"`
	} `json:"data"`
}

// Query issues an instant query.
func (c *Client) Query(ctx context.Context, query string) ([]Result, error) {
	var resp QueryResponse
	if err := c.JC.Get(ctx, "/api/v1/query?query="+url.QueryEscape(query), &resp); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(resp.Data.Result))
	for _, raw := range resp.Data.Result {
		var v struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			continue
		}
		out = append(out, Result{Metric: v.Metric, Value: v.Value})
	}
	return out, nil
}

// QueryRange issues a range query.
func (c *Client) QueryRange(ctx context.Context, query, from, to, step string) ([]Matrix, error) {
	u := fmt.Sprintf("/api/v1/query_range?query=%s&start=%s&end=%s&step=%s",
		url.QueryEscape(query), url.QueryEscape(from), url.QueryEscape(to), url.QueryEscape(step))
	var resp RangeResponse
	if err := c.JC.Get(ctx, u, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Result, nil
}

// Series returns all series matching the label selector.
func (c *Client) Series(ctx context.Context, match []string) ([]map[string]string, error) {
	u := "/api/v1/series?match[]=" + url.QueryEscape(match[0])
	for _, m := range match[1:] {
		u += "&match[]=" + url.QueryEscape(m)
	}
	var resp struct {
		Status string              `json:"status"`
		Data   []map[string]string `json:"data"`
	}
	if err := c.JC.Get(ctx, u, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// TopByLabel returns the top series ordered by the value column, where
// value is the numeric float.
func TopByLabel(results []Result, n int) []Result {
	sorted := append([]Result{}, results...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if valueAt(sorted[i].Value) < valueAt(sorted[j].Value) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	if n > 0 && len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

func valueAt(v []any) float64 {
	if len(v) < 2 {
		return 0
	}
	switch x := v[1].(type) {
	case string:
		var f float64
		_, _ = fmt.Sscanf(x, "%g", &f)
		return f
	case float64:
		return x
	}
	return 0
}

// Convenience wrappers for the most common queries.

// Up returns the up{} sample for `service` (1 == healthy).
func (c *Client) Up(ctx context.Context, service string) (float64, error) {
	expr := fmt.Sprintf("up{service=%q}", service)
	res, err := c.Query(ctx, expr)
	if err != nil || len(res) == 0 {
		return 0, err
	}
	return valueAt(res[0].Value), nil
}

// ErrorRate returns the 5xx rate for a service over `window`.
func (c *Client) ErrorRate(ctx context.Context, service, window string) (float64, error) {
	expr := fmt.Sprintf(`sum(rate(http_requests_total{service=%q,status=~"5.."}[%s]))`, service, window)
	res, err := c.Query(ctx, expr)
	if err != nil || len(res) == 0 {
		return 0, err
	}
	return valueAt(res[0].Value), nil
}

// P99Latency returns p99 latency in seconds for a service over `window`.
func (c *Client) P99Latency(ctx context.Context, service, window string) (float64, error) {
	expr := fmt.Sprintf(`histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=%q}[%s])))`, service, window)
	res, err := c.Query(ctx, expr)
	if err != nil || len(res) == 0 {
		return 0, err
	}
	return valueAt(res[0].Value), nil
}

// HealthSummary is the rolled-up service health record.
type HealthSummary struct {
	Service    string  `json:"service"`
	Status     string  `json:"status"`
	ErrorRate  float64 `json:"error_rate"`
	P99Latency float64 `json:"p99_latency"`
	Up         float64 `json:"up"`
}

// Summarise fetches up + p99 + 5xx rate for a service in a single call.
func (c *Client) Summarise(ctx context.Context, service, window string) HealthSummary {
	up, _ := c.Up(ctx, service)
	errRate, _ := c.ErrorRate(ctx, service, window)
	p99, _ := c.P99Latency(ctx, service, window)
	status := "healthy"
	if up == 0 || errRate > 0.05 {
		status = "degraded"
	}
	if up == 0 && errRate > 0.5 {
		status = "down"
	}
	return HealthSummary{
		Service:    service,
		Status:     status,
		ErrorRate:  errRate,
		P99Latency: p99,
		Up:         up,
	}
}

// ListServices uses up{} discovery to find running services.
func (c *Client) ListServices(ctx context.Context) ([]string, error) {
	res, err := c.Series(ctx, []string{`{__name__="up"}`})
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, r := range res {
		if svc, ok := r["service"]; ok && svc != "" {
			seen[svc] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for svc := range seen {
		out = append(out, svc)
	}
	return out, nil
}

// FormatStep converts a step string ("30s", "1m", …) into a Duration.
func FormatStep(step string) (time.Duration, error) {
	if step == "" {
		return 30 * time.Second, nil
	}
	return time.ParseDuration(step)
}