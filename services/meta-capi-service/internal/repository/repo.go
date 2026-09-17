package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/itdoanh/rinco/services/meta-capi-service/internal/models"
)

var ErrNotFound = errors.New("record not found")

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- CAPI Config ---

func (r *Repository) GetCAPIConfig(ctx context.Context, tenantID uuid.UUID) (*models.CAPIConfig, error) {
	const sql = `
		SELECT id, tenant_id, pixel_id, access_token, test_event_code,
			is_enabled, event_types, sample_rate, created_at, updated_at
		FROM meta_capi.configs WHERE tenant_id = $1 AND is_enabled = true LIMIT 1`
	var cfg models.CAPIConfig
	err := r.pool.QueryRow(ctx, sql, tenantID).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.PixelID, &cfg.AccessToken,
		&cfg.TestEventCode, &cfg.IsEnabled, &cfg.EventTypes,
		&cfg.SampleRate, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &cfg, err
}

func (r *Repository) UpsertCAPIConfig(ctx context.Context, cfg *models.CAPIConfig) error {
	const sql = `
		INSERT INTO meta_capi.configs (id, tenant_id, pixel_id, access_token,
			test_event_code, is_enabled, event_types, sample_rate, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (tenant_id) DO UPDATE SET
			pixel_id=EXCLUDED.pixel_id, access_token=EXCLUDED.access_token,
			test_event_code=EXCLUDED.test_event_code, is_enabled=EXCLUDED.is_enabled,
			event_types=EXCLUDED.event_types, sample_rate=EXCLUDED.sample_rate,
			updated_at=EXCLUDED.updated_at`
	_, err := r.pool.Exec(ctx, sql,
		cfg.ID, cfg.TenantID, cfg.PixelID, cfg.AccessToken,
		cfg.TestEventCode, cfg.IsEnabled, cfg.EventTypes,
		cfg.SampleRate, cfg.CreatedAt, cfg.UpdatedAt,
	)
	return err
}

// --- CAPI Events ---

func (r *Repository) CreateCAPIEvent(ctx context.Context, ev *models.CAPIEvent) error {
	const sql = `
		INSERT INTO meta_capi.events (id, tenant_id, event_id, event_name, event_time,
			event_source, email, phone, ip_address, user_agent, country, fbp_id, fbc_id,
			lead_id, deal_id, order_value, currency, custom_data, status, fb_event_id,
			error_message, retry_count, sent_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`
	_, err := r.pool.Exec(ctx, sql,
		ev.ID, ev.TenantID, ev.EventID, ev.EventName, ev.EventTime,
		ev.EventSource, ev.Email, ev.Phone, ev.IPAddress, ev.UserAgent,
		ev.Country, ev.FBPID, ev.FBCID, ev.LeadID, ev.DealID,
		ev.OrderValue, ev.Currency, ev.CustomData, ev.Status, ev.FBEventID,
		ev.ErrorMessage, ev.RetryCount, ev.SentAt, ev.CreatedAt,
	)
	return err
}

// GetCAPIEventByEventID looks up an event by its dedup event_id.  This
// is used by the handler to short-circuit duplicate events that the
// caller supplied an explicit event_id for.
func (r *Repository) GetCAPIEventByEventID(ctx context.Context, tenantID uuid.UUID, eventID string) (*models.CAPIEvent, error) {
	const sql = `
		SELECT id, tenant_id, event_id, event_name, event_time, event_source,
			email, phone, ip_address, user_agent, country, fbp_id, fbc_id,
			lead_id, deal_id, order_value, currency, custom_data, status,
			fb_event_id, error_message, retry_count, sent_at, created_at
		FROM meta_capi.events
		WHERE tenant_id = $1 AND event_id = $2
		LIMIT 1`
	var ev models.CAPIEvent
	err := r.pool.QueryRow(ctx, sql, tenantID, eventID).Scan(
		&ev.ID, &ev.TenantID, &ev.EventID, &ev.EventName, &ev.EventTime,
		&ev.EventSource, &ev.Email, &ev.Phone, &ev.IPAddress, &ev.UserAgent,
		&ev.Country, &ev.FBPID, &ev.FBCID, &ev.LeadID, &ev.DealID,
		&ev.OrderValue, &ev.Currency, &ev.CustomData, &ev.Status,
		&ev.FBEventID, &ev.ErrorMessage, &ev.RetryCount, &ev.SentAt, &ev.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

// ListCAPIEvents returns the most recent events for the tenant
// (optionally filtered by status).
func (r *Repository) ListCAPIEvents(ctx context.Context, tenantID uuid.UUID, status string, limit int) ([]models.CAPIEvent, error) {
	args := []any{tenantID}
	q := `
		SELECT id, tenant_id, event_id, event_name, event_time, event_source,
			email, phone, ip_address, user_agent, country, fbp_id, fbc_id,
			lead_id, deal_id, order_value, currency, custom_data, status,
			fb_event_id, error_message, retry_count, sent_at, created_at
		FROM meta_capi.events
		WHERE tenant_id = $1`
	if status != "" {
		args = append(args, status)
		q += ` AND status = $2`
	}
	args = append(args, limit)
	q += ` ORDER BY created_at DESC LIMIT $` + intToStr(len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.CAPIEvent
	for rows.Next() {
		var ev models.CAPIEvent
		if err := rows.Scan(
			&ev.ID, &ev.TenantID, &ev.EventID, &ev.EventName, &ev.EventTime,
			&ev.EventSource, &ev.Email, &ev.Phone, &ev.IPAddress, &ev.UserAgent,
			&ev.Country, &ev.FBPID, &ev.FBCID, &ev.LeadID, &ev.DealID,
			&ev.OrderValue, &ev.Currency, &ev.CustomData, &ev.Status,
			&ev.FBEventID, &ev.ErrorMessage, &ev.RetryCount, &ev.SentAt, &ev.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

// CAPIStats is the aggregate used by /v1/capi/stats.
type CAPIStats struct {
	TenantID        string    `json:"tenant_id"`
	TotalEvents     int64     `json:"total_events"`
	Sent            int64     `json:"sent"`
	Failed          int64     `json:"failed"`
	Pending         int64     `json:"pending"`
	Suppressed      int64     `json:"suppressed"`
	MatchRate       float64   `json:"match_rate"`
	DedupRate       float64   `json:"dedup_rate"`
	LastEventAt     *time.Time `json:"last_event_at,omitempty"`
	LastSuccessAt   *time.Time `json:"last_success_at,omitempty"`
	WindowStart     time.Time `json:"window_start"`
}

// GetCAPIStats returns aggregate counters over the last 24h window.
func (r *Repository) GetCAPIStats(ctx context.Context, tenantID uuid.UUID) (*CAPIStats, error) {
	stats := &CAPIStats{
		TenantID:    tenantID.String(),
		WindowStart: time.Now().Add(-24 * time.Hour),
	}

	// Aggregate counts
	const countSQL = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'sent') AS sent,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE status = 'pending') AS pending,
			COUNT(*) FILTER (WHERE status = 'suppressed') AS suppressed
		FROM meta_capi.events
		WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'`
	if err := r.pool.QueryRow(ctx, countSQL, tenantID).Scan(
		&stats.TotalEvents, &stats.Sent, &stats.Failed, &stats.Pending, &stats.Suppressed,
	); err != nil {
		return nil, err
	}

	// Last event timestamp
	const lastSQL = `
		SELECT MAX(created_at) FROM meta_capi.events WHERE tenant_id = $1`
	_ = r.pool.QueryRow(ctx, lastSQL, tenantID).Scan(&stats.LastEventAt)

	// Last success timestamp
	const lastSuccessSQL = `
		SELECT MAX(sent_at) FROM meta_capi.events WHERE tenant_id = $1 AND status = 'sent'`
	_ = r.pool.QueryRow(ctx, lastSuccessSQL, tenantID).Scan(&stats.LastSuccessAt)

	// Match rate approximation: share of sent events that have a fb_event_id
	// returned by Facebook.  A returned fb_event_id means FB accepted
	// the event.  In production this would be cross-referenced with
	// match_score from the CAPI Insights API; here we just report the
	// share of accepted events.
	if stats.Sent > 0 {
		var matched int64
		_ = r.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM meta_capi.events
			WHERE tenant_id = $1 AND status = 'sent' AND fb_event_id IS NOT NULL AND fb_event_id <> ''
			AND created_at >= NOW() - INTERVAL '24 hours'`, tenantID).Scan(&matched)
		stats.MatchRate = float64(matched) / float64(stats.Sent)
	}

	// Dedup rate: events whose event_id appears multiple times
	if stats.TotalEvents > 0 {
		const dupSQL = `
			SELECT COUNT(*) FROM (
				SELECT event_id FROM meta_capi.events
				WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'
				GROUP BY event_id HAVING COUNT(*) > 1
			) t`
		var dup int64
		_ = r.pool.QueryRow(ctx, dupSQL, tenantID).Scan(&dup)
		stats.DedupRate = float64(dup) / float64(stats.TotalEvents)
	}

	return stats, nil
}

// --- (legacy helpers kept for backward compatibility) ---

func (r *Repository) GetPendingCAPIEvents(ctx context.Context, tenantID uuid.UUID, limit int) ([]models.CAPIEvent, error) {
	const sql = `
		SELECT id, tenant_id, event_id, event_name, event_time, event_source,
			email, phone, ip_address, user_agent, country, fbp_id, fbc_id,
			lead_id, deal_id, order_value, currency, custom_data, status,
			fb_event_id, error_message, retry_count, sent_at, created_at
		FROM meta_capi.events
		WHERE tenant_id = $1 AND status = 'pending' AND retry_count < 5
		ORDER BY created_at ASC LIMIT $2`
	rows, err := r.pool.Query(ctx, sql, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.CAPIEvent
	for rows.Next() {
		var ev models.CAPIEvent
		_ = rows.Scan(
			&ev.ID, &ev.TenantID, &ev.EventID, &ev.EventName, &ev.EventTime,
			&ev.EventSource, &ev.Email, &ev.Phone, &ev.IPAddress, &ev.UserAgent,
			&ev.Country, &ev.FBPID, &ev.FBCID, &ev.LeadID, &ev.DealID,
			&ev.OrderValue, &ev.Currency, &ev.CustomData, &ev.Status,
			&ev.FBEventID, &ev.ErrorMessage, &ev.RetryCount, &ev.SentAt, &ev.CreatedAt,
		)
		events = append(events, ev)
	}
	return events, rows.Err()
}

func (r *Repository) UpdateCAPIEventStatus(ctx context.Context, id uuid.UUID, status, fbEventID, errorMsg string) error {
	var sentAt *time.Time
	if status == "sent" {
		now := time.Now()
		sentAt = &now
	}
	const sql = `
		UPDATE meta_capi.events
		SET status=$2, fb_event_id=$3, error_message=$4, sent_at=$5,
			retry_count=retry_count+1
		WHERE id=$1`
	_, err := r.pool.Exec(ctx, sql, id, status, fbEventID, errorMsg, sentAt)
	return err
}

// --- Feedback Events ---

func (r *Repository) CreateFeedbackEvent(ctx context.Context, fb *models.FeedbackEvent) error {
	const sql = `
		INSERT INTO meta_capi.feedback_events (id, tenant_id, capi_event_id, fb_event_id,
			fb_event_name, fb_event_time, fb_partner_name, fb_partner_id,
			fb_artist_id, fb_claim_code, fb_disaggregate, processed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, sql,
		fb.ID, fb.TenantID, fb.CAPIEventID, fb.FBEventID,
		fb.FBEventName, fb.FBEventTime, fb.FBPartnerName, fb.FBPartnerID,
		fb.FBArtistID, fb.FBClaimCode, fb.FBDisaggregate, fb.ProcessedAt,
	)
	return err
}

// --- Conversion Mappings ---

func (r *Repository) GetActiveMappings(ctx context.Context, tenantID uuid.UUID) ([]models.ConversionMapping, error) {
	const sql = `
		SELECT id, tenant_id, crm_event_type, capi_event_name, is_active, value_field, created_at
		FROM meta_capi.mappings WHERE tenant_id = $1 AND is_active = true`
	rows, err := r.pool.Query(ctx, sql, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []models.ConversionMapping
	for rows.Next() {
		var m models.ConversionMapping
		if err := rows.Scan(&m.ID, &m.TenantID, &m.CRMEventType, &m.CAPIEventName, &m.IsActive, &m.ValueField, &m.CreatedAt); err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

// intToStr formats a positive int for use in dynamic SQL.  We only
// call it with values we control (argument indexes), so this is safe.
func intToStr(n int) string {
	if n <= 0 {
		return "0"
	}
	const digits = "0123456789"
	out := make([]byte, 0, 4)
	for n > 0 {
		out = append([]byte{digits[n%10]}, out...)
		n /= 10
	}
	return string(out)
}
