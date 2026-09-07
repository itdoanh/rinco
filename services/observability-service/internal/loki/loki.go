// Package loki is a thin client over the Loki HTTP API.
//
// The service uses two endpoints:
//
//	GET /loki/api/v1/query_range   — return log lines
//	GET /loki/api/v1/query         — return metric-style aggregate
//
// The aggregate endpoint is reached via QueryRange with a LogQL query
// that includes a quantisation function (`sum by (...) (... over time)`).
package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rinco/services/observability-service/internal/platform"
)

// Client wraps the Loki HTTP API.
type Client struct {
	JC *platform.JSONClient
}

// New constructs a Loki client.
func New(baseURL string) *Client {
	return &Client{JC: platform.NewJSONClient(baseURL, "", 15*time.Second)}
}

// LogLine is a single log entry from Loki.
type LogLine struct {
	Time    time.Time         `json:"time"`
	Line    string            `json:"line"`
	Labels  map[string]string `json:"labels,omitempty"`
	Tenant  string            `json:"tenant_id,omitempty"`
	Service string            `json:"service,omitempty"`
	Level   string            `json:"level,omitempty"`
}

// QueryRangeResponse is the Loki query_range response.
type QueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]any           `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// Query logs with LogQL.
func (c *Client) Query(ctx context.Context, q Query) ([]LogLine, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", q.Limit))
	if q.From != "" {
		params.Set("start", q.From)
	}
	if q.To != "" {
		params.Set("end", q.To)
	}
	params.Set("query", buildQuery(q))

	var resp QueryRangeResponse
	if err := c.JC.Get(ctx, "/loki/api/v1/query_range?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	out := make([]LogLine, 0)
	for _, stream := range resp.Data.Result {
		for _, vals := range stream.Values {
			if len(vals) < 2 {
				continue
			}
			ts, _ := vals[0].(string)
			line, _ := vals[1].(string)
			t, _ := time.Parse(time.RFC3339Nano, ts)
			out = append(out, LogLine{
				Time:    t,
				Line:    line,
				Labels:  stream.Stream,
				Tenant:  stream.Stream["tenant_id"],
				Service: stream.Stream["service"],
				Level:   stream.Stream["level"],
			})
		}
	}
	return out, nil
}

// Aggregate runs a metric-style LogQL query and returns the raw Loki
// response (matrix/vector form).
func (c *Client) Aggregate(ctx context.Context, q Query) (json.RawMessage, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", q.Limit))
	if q.From != "" {
		params.Set("start", q.From)
	}
	if q.To != "" {
		params.Set("end", q.To)
	}
	expr := q.LogQL
	if expr == "" {
		expr = "sum by (level) (rate({service=\"" + q.Service + "\"}[5m]))"
	}
	params.Set("query", expr)
	return c.JC.GetRaw(ctx, "/loki/api/v1/query_range?"+params.Encode())
}

// Query is a LogQL filter.
type Query struct {
	Service  string
	Level    string
	TenantID string
	From     string // RFC3339Nano or unix
	To       string
	Limit    int
	Pattern  string // free-text filter (`|=`)
	GroupBy  string // aggregation group-by for Aggregate
	LogQL    string // full LogQL for Aggregate; overrides pattern/GroupBy
}

// buildQuery assembles a LogQL line filter.
func buildQuery(q Query) string {
	var b strings.Builder
	b.WriteString("{")
	pairs := []string{}
	if q.Service != "" {
		pairs = append(pairs, fmt.Sprintf("service=%q", q.Service))
	}
	if q.Level != "" {
		pairs = append(pairs, fmt.Sprintf("level=%q", q.Level))
	}
	if q.TenantID != "" {
		pairs = append(pairs, fmt.Sprintf("tenant_id=%q", q.TenantID))
	}
	b.WriteString(strings.Join(pairs, ","))
	b.WriteString("}")
	if q.Pattern != "" {
		b.WriteString(fmt.Sprintf(" |= %q", q.Pattern))
	}
	return b.String()
}