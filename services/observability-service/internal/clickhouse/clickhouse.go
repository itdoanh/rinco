// Package clickhouse provides a small audit-log writer backed by
// ClickHouse via its HTTP gateway (port 8123).  Failures are logged
// but never fatal: the service still serves traffic when CH is
// unreachable; the Postgres audit_logs table is the system of record.
//
// We deliberately use the HTTP gateway instead of clickhouse-go's native
// driver — the native driver couples tightly with ch-go versions and
// has historically broken under independent version bumps.  The HTTP
// gateway is sufficient for the low-volume audit traffic the
// observability service produces.
package clickhouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Writer persists audit log records to ClickHouse via its HTTP gateway.
type Writer struct {
	url    string
	client *http.Client
}

// NewWriter builds a Writer for the ClickHouse HTTP gateway (e.g.
// "http://localhost:8123").
func NewWriter(url string) *Writer {
	return &Writer{
		url:    strings.TrimRight(url, "/"),
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// AuditRecord is a single audit log row.
type AuditRecord struct {
	ID           string
	TenantID     string
	ActorUserID  string
	ActorIP      string
	Action       string
	ResourceType string
	ResourceID   string
	Payload      any
	TS           time.Time
}

// MarshalJSON produces a JSONEachRow-friendly single-line encoding.
func (r AuditRecord) MarshalJSON() ([]byte, error) {
	if r.ID == "" {
		r.ID = fmt.Sprintf("%d-%s", time.Now().UnixNano(), r.TenantID)
	}
	if r.TS.IsZero() {
		r.TS = time.Now()
	}
	return json.Marshal(struct {
		ID           string    `json:"id"`
		TenantID     string    `json:"tenant_id"`
		ActorUserID  string    `json:"actor_user_id"`
		ActorIP      string    `json:"actor_ip"`
		Action       string    `json:"action"`
		ResourceType string    `json:"resource_type"`
		ResourceID   string    `json:"resource_id"`
		Payload      string    `json:"payload"`
		TS           time.Time `json:"ts"`
	}{
		ID:           r.ID,
		TenantID:     r.TenantID,
		ActorUserID:  r.ActorUserID,
		ActorIP:      r.ActorIP,
		Action:       r.Action,
		ResourceType: r.ResourceType,
		ResourceID:   r.ResourceID,
		Payload:      encodePayload(r.Payload),
		TS:           r.TS,
	})
}

// Insert appends an audit record.  Returns an error for the caller to
// log; the service never aborts on a CH insert failure.
func (w *Writer) Insert(ctx context.Context, r AuditRecord) error {
	if w.url == "" {
		return fmt.Errorf("clickhouse url is empty")
	}
	if r.TS.IsZero() {
		r.TS = time.Now()
	}
	if r.ID == "" {
		r.ID = fmt.Sprintf("%d-%s", r.TS.UnixNano(), r.TenantID)
	}
	buf, err := r.MarshalJSON()
	if err != nil {
		return err
	}
	body := bytes.NewReader(buf)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		w.url+"/?database=observability&query=INSERT+INTO+observability.app_audit_logs+FORMAT+JSONEachRow",
		body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		// ClickHouse returns the offending line in the response body
		// — log it for easier debugging.
		var respBody bytes.Buffer
		_, _ = respBody.ReadFrom(resp.Body)
		return fmt.Errorf("clickhouse HTTP %d: %s", resp.StatusCode, respBody.String())
	}
	return nil
}

// InsertBatch is a bulk variant — one HTTP request per batch.
func (w *Writer) InsertBatch(ctx context.Context, records []AuditRecord) error {
	if w.url == "" {
		return fmt.Errorf("clickhouse url is empty")
	}
	if len(records) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for _, r := range records {
		b, err := r.MarshalJSON()
		if err != nil {
			return err
		}
		buf.Write(b)
		buf.WriteByte('\n')
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		w.url+"/?database=observability&query=INSERT+INTO+observability.app_audit_logs+FORMAT+JSONEachRow",
		&buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("clickhouse HTTP %d", resp.StatusCode)
	}
	return nil
}

// LogError is a helper that logs CH errors at warn level.
func LogError(ctx context.Context, op string, err error) {
	slog.WarnContext(ctx, "clickhouse "+op, slog.String("error", err.Error()))
}

// SafeTenant returns the tenant id with backslashes / NULs stripped.
func SafeTenant(s string) string {
	return strings.Map(func(r rune) rune {
		if r == 0 || r == '\\' || r == '\'' {
			return -1
		}
		return r
	}, s)
}

func encodePayload(p any) string {
	if p == nil {
		return ""
	}
	buf, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(buf)
}