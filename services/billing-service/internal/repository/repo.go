package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
)

var ErrNotFound = errors.New("record not found")

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Subscriptions ---

func (r *Repository) CreateSubscription(ctx context.Context, sub *models.Subscription) error {
	const sql = `
		INSERT INTO subscriptions (id, tenant_id, plan, status, stripe_sub_id,
			current_period_start, current_period_end, trial_ends_at, cancelled_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	_, err := r.pool.Exec(ctx, sql,
		sub.ID, sub.TenantID, sub.Plan, sub.Status, sub.StripeSubID,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.TrialEndsAt, sub.CancelledAt,
		sub.CreatedAt, sub.UpdatedAt,
	)
	return err
}

func (r *Repository) GetSubscriptionByTenant(ctx context.Context, tenantID uuid.UUID) (*models.Subscription, error) {
	const sql = `
		SELECT id, tenant_id, plan, status, stripe_sub_id, current_period_start,
			current_period_end, trial_ends_at, cancelled_at, created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1 AND status != 'cancelled'
		ORDER BY created_at DESC LIMIT 1`
	var sub models.Subscription
	err := r.pool.QueryRow(ctx, sql, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.Plan, &sub.Status, &sub.StripeSubID,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.TrialEndsAt, &sub.CancelledAt,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &sub, err
}

func (r *Repository) UpdateSubscription(ctx context.Context, sub *models.Subscription) error {
	const sql = `
		UPDATE subscriptions SET plan=$2, status=$3, stripe_sub_id=$4,
			current_period_start=$5, current_period_end=$6, trial_ends_at=$7,
			cancelled_at=$8, updated_at=$9
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, sql,
		sub.ID, sub.Plan, sub.Status, sub.StripeSubID,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.TrialEndsAt,
		sub.CancelledAt, sub.UpdatedAt,
	)
	return err
}

// --- Invoices ---

func (r *Repository) CreateInvoice(ctx context.Context, inv *models.Invoice) error {
	const sql = `
		INSERT INTO invoices (id, tenant_id, subscription_id, number, status,
			amount, currency, tax_amount, tax_percent, stripe_invoice_id, paid_at,
			due_date, invoice_date, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`
	_, err := r.pool.Exec(ctx, sql,
		inv.ID, inv.TenantID, inv.SubscriptionID, inv.Number, inv.Status,
		inv.Amount, inv.Currency, inv.TaxAmount, inv.TaxPercent, inv.StripeInvoiceID,
		inv.PaidAt, inv.DueDate, inv.InvoiceDate, inv.CreatedAt,
	)
	return err
}

func (r *Repository) GetInvoice(ctx context.Context, id uuid.UUID) (*models.Invoice, error) {
	const sql = `
		SELECT id, tenant_id, subscription_id, number, status, amount, currency,
			tax_amount, tax_percent, stripe_invoice_id, paid_at, due_date, invoice_date, created_at
		FROM invoices WHERE id = $1`
	var inv models.Invoice
	err := r.pool.QueryRow(ctx, sql, id).Scan(
		&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.Number, &inv.Status,
		&inv.Amount, &inv.Currency, &inv.TaxAmount, &inv.TaxPercent,
		&inv.StripeInvoiceID, &inv.PaidAt, &inv.DueDate, &inv.InvoiceDate, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &inv, err
}

func (r *Repository) ListInvoices(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]models.Invoice, error) {
	const sql = `
		SELECT id, tenant_id, subscription_id, number, status, amount, currency,
			tax_amount, tax_percent, stripe_invoice_id, paid_at, due_date, invoice_date, created_at
		FROM invoices WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, sql, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		_ = rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.Number, &inv.Status,
			&inv.Amount, &inv.Currency, &inv.TaxAmount, &inv.TaxPercent,
			&inv.StripeInvoiceID, &inv.PaidAt, &inv.DueDate, &inv.InvoiceDate, &inv.CreatedAt,
		)
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

func (r *Repository) UpdateInvoiceStatus(ctx context.Context, id uuid.UUID, status string, paidAt *time.Time) error {
	const sql = `UPDATE invoices SET status=$2, paid_at=$3 WHERE id=$1`
	_, err := r.pool.Exec(ctx, sql, id, status, paidAt)
	return err
}

func (r *Repository) NextInvoiceNumber(ctx context.Context) (string, error) {
	var count int
	const sql = `SELECT COUNT(*) FROM invoices WHERE DATE(created_at) = CURRENT_DATE`
	_ = r.pool.QueryRow(ctx, sql).Scan(&count)
	year := time.Now().Year()
	return fmt.Sprintf("INV-%d-%05d", year, count+1), nil
}

// --- Usage Records ---

func (r *Repository) UpsertUsage(ctx context.Context, rec *models.UsageRecord) error {
	const sql = `
		INSERT INTO usage_records (id, tenant_id, month, year, api_requests, storage_gb, ai_calls, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (tenant_id, month, year) DO UPDATE SET
			api_requests=EXCLUDED.api_requests, storage_gb=EXCLUDED.storage_gb,
			ai_calls=EXCLUDED.ai_calls, recorded_at=EXCLUDED.recorded_at`
	_, err := r.pool.Exec(ctx, sql,
		rec.ID, rec.TenantID, rec.Month, rec.Year,
		rec.APIRequests, rec.StorageGB, rec.AIcalls, rec.RecordedAt,
	)
	return err
}

func (r *Repository) GetUsage(ctx context.Context, tenantID uuid.UUID, month, year int) (*models.UsageRecord, error) {
	const sql = `
		SELECT id, tenant_id, month, year, api_requests, storage_gb, ai_calls, recorded_at
		FROM usage_records WHERE tenant_id=$1 AND month=$2 AND year=$3`
	var rec models.UsageRecord
	err := r.pool.QueryRow(ctx, sql, tenantID, month, year).Scan(
		&rec.ID, &rec.TenantID, &rec.Month, &rec.Year,
		&rec.APIRequests, &rec.StorageGB, &rec.AIcalls, &rec.RecordedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &rec, err
}

// --- Payment Methods ---

func (r *Repository) CreatePaymentMethod(ctx context.Context, pm *models.PaymentMethod) error {
	const sql = `
		INSERT INTO payment_methods (id, tenant_id, stripe_pm_id, brand, last4,
			exp_month, exp_year, is_default, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := r.pool.Exec(ctx, sql,
		pm.ID, pm.TenantID, pm.StripePMID, pm.Brand, pm.Last4,
		pm.ExpMonth, pm.ExpYear, pm.IsDefault, pm.CreatedAt,
	)
	return err
}

func (r *Repository) GetDefaultPaymentMethod(ctx context.Context, tenantID uuid.UUID) (*models.PaymentMethod, error) {
	const sql = `
		SELECT id, tenant_id, stripe_pm_id, brand, last4, exp_month, exp_year, is_default, created_at
		FROM payment_methods WHERE tenant_id=$1 AND is_default=true LIMIT 1`
	var pm models.PaymentMethod
	err := r.pool.QueryRow(ctx, sql, tenantID).Scan(
		&pm.ID, &pm.TenantID, &pm.StripePMID, &pm.Brand, &pm.Last4,
		&pm.ExpMonth, &pm.ExpYear, &pm.IsDefault, &pm.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &pm, err
}

func (r *Repository) DeletePaymentMethod(ctx context.Context, id uuid.UUID) error {
	const sql = `DELETE FROM payment_methods WHERE id=$1`
	_, err := r.pool.Exec(ctx, sql, id)
	return err
}

// --- Discount Codes ---

func (r *Repository) GetDiscountByCode(ctx context.Context, code string) (*models.DiscountCode, error) {
	const sql = `
		SELECT id, code, discount_type, discount_value, currency, max_uses, current_uses,
			expires_at, valid_from, is_active, created_at
		FROM discount_codes WHERE code=$1`
	var dc models.DiscountCode
	err := r.pool.QueryRow(ctx, sql, code).Scan(
		&dc.ID, &dc.Code, &dc.DiscountType, &dc.DiscountValue, &dc.Currency,
		&dc.MaxUses, &dc.CurrentUses, &dc.ExpiresAt, &dc.ValidFrom, &dc.IsActive, &dc.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &dc, err
}

func (r *Repository) IncrementDiscountUsage(ctx context.Context, id uuid.UUID) error {
	const sql = `UPDATE discount_codes SET current_uses=current_uses+1 WHERE id=$1`
	_, err := r.pool.Exec(ctx, sql, id)
	return err
}

// --- Webhook Events ---

func (r *Repository) CreateWebhookEvent(ctx context.Context, ev *models.WebhookEvent) error {
	const sql = `
		INSERT INTO webhook_events (id, stripe_event_id, type, data_payload, processed_at, attempts, error, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := r.pool.Exec(ctx, sql,
		ev.ID, ev.StripeEventID, ev.Type, ev.DataPayload,
		ev.ProcessedAt, ev.Attempts, ev.Error, ev.CreatedAt,
	)
	return err
}

func (r *Repository) GetWebhookEvent(ctx context.Context, stripeEventID string) (*models.WebhookEvent, error) {
	const sql = `
		SELECT id, stripe_event_id, type, data_payload, processed_at, attempts, error, created_at
		FROM webhook_events WHERE stripe_event_id=$1`
	var ev models.WebhookEvent
	err := r.pool.QueryRow(ctx, sql, stripeEventID).Scan(
		&ev.ID, &ev.StripeEventID, &ev.Type, &ev.DataPayload,
		&ev.ProcessedAt, &ev.Attempts, &ev.Error, &ev.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &ev, err
}

func (r *Repository) MarkWebhookProcessed(ctx context.Context, id uuid.UUID, errMsg string) error {
	const sql = `UPDATE webhook_events SET processed_at=NOW(), attempts=attempts+1, error=$2 WHERE id=$1`
	_, err := r.pool.Exec(ctx, sql, id, errMsg)
	return err
}
