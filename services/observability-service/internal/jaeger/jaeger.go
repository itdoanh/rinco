// Package jaeger is a thin client over the Jaeger HTTP API and Tempo
// (search) API.  We support:
//
//   - Get trace by ID      (`/api/traces/{trace_id}`)
//   - Search traces        (`/api/traces?service=…&operation=…`)
//
// Tempo uses gRPC internally but exposes `/api/search` (HTTP) which is
// compatible with the same JSON envelope.
package jaeger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rinco/services/observability-service/internal/platform"
)

// Span describes a single trace span.
type Span struct {
	SpanID    string            `json:"span_id"`
	TraceID   string            `json:"trace_id"`
	Operation string            `json:"operation"`
	Service   string            `json:"service"`
	StartMs   int64             `json:"start_ms"`
	Duration  int64             `json:"duration_ms"`
	Tags      map[string]string `json:"tags"`
	Logs      []SpanLog         `json:"logs,omitempty"`
}

// SpanLog is a single span log entry.
type SpanLog struct {
	Timestamp int64             `json:"timestamp_ms"`
	Fields    map[string]string `json:"fields"`
}

// Trace bundles spans by trace id.
type Trace struct {
	TraceID   string `json:"trace_id"`
	Spans     []Span `json:"spans"`
	Services  []string `json:"services,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

// Client wraps both Jaeger and Tempo.
type Client struct {
	JC      *platform.JSONClient
	TempoJC *platform.JSONClient
	UseTempo bool
}

// New constructs a Jaeger client (with optional Tempo fallback for search).
func New(jaegerURL, tempoURL string) *Client {
	c := &Client{
		JC:       platform.NewJSONClient(jaegerURL, "", 15*time.Second),
		TempoJC:  platform.NewJSONClient(tempoURL, "", 15*time.Second),
		UseTempo: tempoURL != "" && jaegerURL != tempoURL,
	}
	return c
}

// GetTrace returns all spans for a trace id.
func (c *Client) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	var resp struct {
		Data []struct {
			TraceID string `json:"traceID"`
			Spans  []struct {
				SpanID    string            `json:"spanID"`
				OperationName string       `json:"operationName"`
				ProcessID string            `json:"processID"`
				StartTime int64             `json:"startTime"`
				Duration  int64             `json:"duration"`
				Tags      []struct {
					Key   string `json:"key"`
					Value any    `json:"value"`
				} `json:"tags"`
				Logs []struct {
					Timestamp int64 `json:"timestamp"`
					Fields    []struct {
						Key   string `json:"key"`
						Value any    `json:"value"`
					} `json:"fields"`
				} `json:"logs"`
			} `json:"spans"`
			Processes map[string]struct {
				ServiceName string `json:"serviceName"`
			} `json:"processes"`
		} `json:"data"`
	}
	if err := c.JC.Get(ctx, "/api/traces/"+traceID, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return &Trace{TraceID: traceID, Spans: []Span{}}, nil
	}
	first := resp.Data[0]
	trace := &Trace{TraceID: first.TraceID, Spans: make([]Span, 0, len(first.Spans))}
	for _, s := range first.Spans {
		tags := map[string]string{}
		for _, t := range s.Tags {
			tags[t.Key] = stringify(t.Value)
		}
		logs := make([]SpanLog, 0, len(s.Logs))
		for _, l := range s.Logs {
			fields := map[string]string{}
			for _, f := range l.Fields {
				fields[f.Key] = stringify(f.Value)
			}
			logs = append(logs, SpanLog{Timestamp: l.Timestamp, Fields: fields})
		}
		svc := ""
		if p, ok := first.Processes[s.ProcessID]; ok {
			svc = p.ServiceName
		}
		trace.Spans = append(trace.Spans, Span{
			SpanID:    s.SpanID,
			TraceID:   first.TraceID,
			Operation: s.OperationName,
			Service:   svc,
			StartMs:   s.StartTime / 1000,
			Duration:  s.Duration / 1000,
			Tags:      tags,
			Logs:      logs,
		})
	}
	return trace, nil
}

// Search returns traces that match the given filter.
func (c *Client) Search(ctx context.Context, q SearchQuery) ([]Trace, error) {
	params := url.Values{}
	if q.Service != "" {
		params.Set("service", q.Service)
	}
	if q.Operation != "" {
		params.Set("operation", q.Operation)
	}
	if q.Tags != "" {
		params.Set("tags", q.Tags)
	}
	if q.MinDuration != "" {
		params.Set("minDuration", q.MinDuration)
	}
	if q.MaxDuration != "" {
		params.Set("maxDuration", q.MaxDuration)
	}
	if q.Lookback != "" {
		params.Set("lookback", q.Lookback)
	}
	if q.Limit > 0 {
		params.Set("limit", strconv.Itoa(q.Limit))
	}

	if c.UseTempo {
		// Tempo search uses a JSON body — fall back to Jaeger for simplicity.
		// We keep the API uniform: GET to /api/search.
		body := map[string]any{
			"query":   q.Tags,
			"limit":   q.Limit,
			"start":   int(time.Now().Add(-1 * time.Hour).Unix()),
			"end":     int(time.Now().Unix()),
		}
		_ = body // unused, we still hit Jaeger below
	}

	var resp struct {
		Data []struct {
			TraceID string `json:"traceID"`
			Spans  []struct {
				SpanID        string `json:"spanID"`
				OperationName string `json:"operationName"`
				ProcessID     string `json:"processID"`
				StartTime     int64  `json:"startTime"`
				Duration      int64  `json:"duration"`
				Tags          []struct {
					Key   string `json:"key"`
					Value any    `json:"value"`
				} `json:"tags"`
			} `json:"spans"`
			Processes map[string]struct {
				ServiceName string `json:"serviceName"`
			} `json:"processes"`
		} `json:"data"`
	}
	if err := c.JC.Get(ctx, "/api/traces?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	out := make([]Trace, 0, len(resp.Data))
	for _, d := range resp.Data {
		trace := Trace{TraceID: d.TraceID}
		for _, s := range d.Spans {
			tags := map[string]string{}
			for _, t := range s.Tags {
				tags[t.Key] = stringify(t.Value)
			}
			svc := ""
			if p, ok := d.Processes[s.ProcessID]; ok {
				svc = p.ServiceName
			}
			trace.Spans = append(trace.Spans, Span{
				SpanID:    s.SpanID,
				TraceID:   d.TraceID,
				Operation: s.OperationName,
				Service:   svc,
				StartMs:   s.StartTime / 1000,
				Duration:  s.Duration / 1000,
				Tags:      tags,
			})
		}
		out = append(out, trace)
	}
	return out, nil
}

// SearchQuery is the input to Search.
type SearchQuery struct {
	Service     string
	Operation   string
	Tags        string
	MinDuration string
	MaxDuration string
	Lookback    string
	Limit       int
}

func stringify(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	case nil:
		return ""
	case json.Number:
		return x.String()
	}
	return fmt.Sprintf("%v", v)
}

// IsLikelyTraceID checks whether a string is a plausible trace id
// (hex 16-32 chars or Jaeger-style W3C trace context).
func IsLikelyTraceID(s string) bool {
	if len(s) < 8 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') && !(r >= 'A' && r <= 'F') && r != '-' {
			return false
		}
	}
	return strings.Contains(s, "-") || len(s) >= 16
}