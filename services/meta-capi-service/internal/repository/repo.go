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
