// Package alertmanager contains the receiver that ingests AlertManager
// webhook payloads, normalises them into our `observability.alerts`
// table, and fans out notifications to the configured webhook (Slack,
// generic HTTP, …).
//
// The receiver is intentionally tolerant of partially-formed payloads:
// AlertManager sometimes sends resolved alerts with no `endsAt`, and the
// fingerprint we use (alertname + service) is intentionally loose so
// duplicates collapse.
package alertmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Webhook is the AlertManager v4 payload root.
type Webhook struct {
	Version  string  `json:"version"`
	GroupKey string  `json:"groupKey"`
	Status   string  `json:"status"` // firing | resolved
	Receiver string  `json:"receiver"`
	Alerts   []Alert `json:"alerts"`
}

// Alert is a single notification from AlertManager.
type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// Receiver persists incoming webhook payloads to Postgres and optionally
// fans out to a configured external webhook (e.g. Slack).
type Receiver struct {
	pool        *pgxpool.Pool
	fanoutURL   string
	httpClient  *http.Client
}

// NewReceiver constructs a webhook Receiver.
func NewReceiver(pool *pgxpool.Pool, fanoutURL string) *Receiver {
	return &Receiver{
		pool:       pool,
		fanoutURL:  fanoutURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// HandleWebhook processes an AlertManager payload.  The endpoint is
// mounted at POST /v1/webhook/alertmanager.
func (r *Receiver) HandleWebhook(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var payload Webhook
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	processed := 0
	for _, a := range payload.Alerts {
		if err := r.upsertAlert(ctx, a); err != nil {
			slog.Error("alertmanager upsert failed",
				slog.String("fingerprint", a.Fingerprint),
				slog.String("error", err.Error()))
			continue
		}
		processed++
		if r.fanoutURL != "" {
			r.fanout(ctx, a)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"processed": processed,
		"received":  len(payload.Alerts),
	})
}

func (r *Receiver) upsertAlert(ctx context.Context, a Alert) error {
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}
	fp := a.Fingerprint
	if fp == "" {
		fp = computeFingerprint(a.Labels)
	}
	status := mapStatus(a.Status)
	service := a.Labels["service"]
	title := firstNonEmpty(a.Labels["alertname"], a.Annotations["summary"], "unknown")
	message := a.Annotations["description"]
	severity := firstNonEmpty(a.Labels["severity"], "warning")

	labelsJSON, _ := json.Marshal(a.Labels)
	annJSON, _ := json.Marshal(a.Annotations)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Upsert by fingerprint.  When the alert already exists we only
	// update status / annotations / fired_at / resolved_at; we keep
	// the original fired_at and ack metadata.
	var alertID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO observability.alerts (fingerprint, status, severity, labels, annotations, service, title, message, fired_at, resolved_at)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7, $8, $9, $10)
		ON CONFLICT (fingerprint) DO UPDATE SET
			status = EXCLUDED.status,
			severity = EXCLUDED.severity,
			annotations = EXCLUDED.annotations,
			resolved_at = CASE WHEN EXCLUDED.status = 'resolved' THEN NOW() ELSE observability.alerts.resolved_at END,
			updated_at = NOW()
		RETURNING id`,
		fp, status, severity, string(labelsJSON), string(annJSON), service, title, message,
		a.StartsAt, nullableTime(a.EndsAt),
	).Scan(&alertID)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}

	payload := map[string]any{
		"labels":      a.Labels,
		"annotations": a.Annotations,
		"startsAt":    a.StartsAt,
		"endsAt":      a.EndsAt,
	}
	payloadJSON, _ := json.Marshal(payload)
	if _, err := tx.Exec(ctx, `
		INSERT INTO observability.alert_history (alert_id, event, payload)
		VALUES ($1, $2, $3::jsonb)`,
		alertID, "ingest:"+status, string(payloadJSON)); err != nil {
		return fmt.Errorf("history: %w", err)
	}

	return tx.Commit(ctx)
}

// fanout posts the alert to a configured webhook (Slack, custom …).
// Best-effort; errors are logged but not surfaced.
func (r *Receiver) fanout(ctx context.Context, a Alert) {
	body, _ := json.Marshal(map[string]any{
		"status": a.Status,
		"labels": a.Labels,
		"annotations": a.Annotations,
		"startsAt": a.StartsAt,
		"endsAt": a.EndsAt,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.fanoutURL, strings.NewReader(string(body)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		slog.Warn("fanout failed", slog.String("error", err.Error()))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Warn("fanout non-2xx", slog.Int("status", resp.StatusCode))
	}
}

// Acknowledge marks an alert as acknowledged in Postgres.  Returns true
// if the alert existed and was updated.
func (r *Receiver) Acknowledge(ctx context.Context, alertID, userID string) (bool, error) {
	tag, err := uuid.Parse(alertID)
	if err != nil {
		return false, fmt.Errorf("invalid id: %w", err)
	}
	cmd, err := r.pool.Exec(ctx, `
		UPDATE observability.alerts
		SET status = 'ack', ack_by = $2::uuid, ack_at = NOW(), updated_at = NOW()
		WHERE id = $1`, tag, userID)
	if err != nil {
		return false, err
	}
	if cmd.RowsAffected() == 0 {
		return false, nil
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO observability.alert_history (alert_id, event, payload)
		VALUES ($1, 'ack', jsonb_build_object('user_id', $2::uuid))`, tag, userID)
	return true, err
}

func mapStatus(in string) string {
	switch in {
	case "firing":
		return "firing"
	case "resolved":
		return "resolved"
	default:
		return "firing"
	}
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// computeFingerprint derives a stable fingerprint from labels even when
// AlertManager does not provide one (older configs).
func computeFingerprint(labels map[string]string) string {
	if v := labels["fingerprint"]; v != "" {
		return v
	}
	keys := []string{"alertname", "service", "severity"}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, ",")
}