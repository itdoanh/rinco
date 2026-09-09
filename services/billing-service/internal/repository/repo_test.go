package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
)

// fakeDB is a minimal in-memory DB satisfying the DB interface.
type fakeDB struct {
	execErr  error
	queryErr error
	rows     *fakeRows
	row      *fakeRow
	rowErr   error
}

type fakeRow struct {
	values []any
	err    error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("scan arity mismatch")
	}
	for i, d := range dest {
		switch v := d.(type) {
		case *string:
			*v = r.values[i].(string)
		case *int:
			*v = r.values[i].(int)
		case *int64:
			*v = r.values[i].(int64)
		case *uuid.UUID:
			*v = r.values[i].(uuid.UUID)
		default:
			_ = v
		}
	}
	return nil
}

type fakeRows struct {
	idx   int
	total int
	items [][]any
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                 { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	if r.idx < r.total {
		r.idx++
		return true
	}
	return false
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > r.total {
		return errors.New("no row")
	}
	values := r.items[r.idx-1]
	if len(dest) != len(values) {
		return errors.New("scan arity mismatch")
	}
	for i, d := range dest {
		switch v := d.(type) {
		case *string:
			*v = values[i].(string)
		case *int:
			*v = values[i].(int)
		case *int64:
			*v = values[i].(int64)
		case *uuid.UUID:
			*v = values[i].(uuid.UUID)
		default:
			_ = v
		}
	}
	return nil
}
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }

func (db *fakeDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if db.execErr != nil {
		return pgconn.CommandTag{}, db.execErr
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func (db *fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if db.queryErr != nil {
		return nil, db.queryErr
	}
	if db.rows != nil {
		return db.rows, nil
	}
	return &fakeRows{}, nil
}
func (db *fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if db.row != nil {
		return db.row
	}
	return &fakeRow{err: pgx.ErrNoRows}
}

func TestNewRepository(t *testing.T) {
	r := New(&fakeDB{})
	if r == nil {
		t.Fatal("New returned nil")
	}
}

func TestErrNotFoundValue(t *testing.T) {
	if ErrNotFound == nil {
		t.Fatal("ErrNotFound should not be nil")
	}
	if ErrNotFound.Error() == "" {
		t.Error("ErrNotFound has empty message")
	}
}

func TestCreateSubscription(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.CreateSubscription(context.Background(), &models.Subscription{}); err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}
}

func TestCreateSubscriptionExecError(t *testing.T) {
	db := &fakeDB{execErr: errors.New("db down")}
	r := &Repository{db: db}
	if err := r.CreateSubscription(context.Background(), &models.Subscription{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetSubscriptionByTenantNotFound(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	_, err := r.GetSubscriptionByTenant(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSubscriptionByTenantSuccess(t *testing.T) {
	id := uuid.New()
	tenantID := uuid.New()
	db := &fakeDB{
		row: &fakeRow{
			values: []any{
				id, tenantID, "pro", "active", "sub_123",
				int64(0), int64(0), int64(0), int64(0),
				int64(0), int64(0),
			},
		},
	}
	r := &Repository{db: db}
	sub, err := r.GetSubscriptionByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if sub.ID != id {
		t.Errorf("expected id %s, got %s", id, sub.ID)
	}
}

func TestUpdateSubscription(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.UpdateSubscription(context.Background(), &models.Subscription{}); err != nil {
		t.Fatalf("UpdateSubscription: %v", err)
	}
}

func TestCreateInvoice(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.CreateInvoice(context.Background(), &models.Invoice{}); err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
}

func TestGetInvoiceNotFound(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	_, err := r.GetInvoice(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListInvoicesEmpty(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	invoices, err := r.ListInvoices(context.Background(), uuid.New(), 10, 0)
	if err != nil {
		t.Fatalf("ListInvoices: %v", err)
	}
	if len(invoices) != 0 {
		t.Errorf("expected 0, got %d", len(invoices))
	}
}

func TestUpdateInvoiceStatus(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.UpdateInvoiceStatus(context.Background(), uuid.New(), "paid", nil); err != nil {
		t.Fatalf("UpdateInvoiceStatus: %v", err)
	}
}

func TestUpsertUsage(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.UpsertUsage(context.Background(), &models.UsageRecord{}); err != nil {
		t.Fatalf("UpsertUsage: %v", err)
	}
}

func TestGetUsageNotFound(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	_, err := r.GetUsage(context.Background(), uuid.New(), 1, 2026)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreatePaymentMethod(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.CreatePaymentMethod(context.Background(), &models.PaymentMethod{}); err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}
}

func TestGetDefaultPaymentMethodNotFound(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	_, err := r.GetDefaultPaymentMethod(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeletePaymentMethod(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.DeletePaymentMethod(context.Background(), uuid.New()); err != nil {
		t.Fatalf("DeletePaymentMethod: %v", err)
	}
}

func TestGetDiscountByCodeNotFound(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	_, err := r.GetDiscountByCode(context.Background(), "MISSING")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestIncrementDiscountUsage(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.IncrementDiscountUsage(context.Background(), uuid.New()); err != nil {
		t.Fatalf("IncrementDiscountUsage: %v", err)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.CreateWebhookEvent(context.Background(), &models.WebhookEvent{}); err != nil {
		t.Fatalf("CreateWebhookEvent: %v", err)
	}
}

func TestGetWebhookEventNotFound(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	_, err := r.GetWebhookEvent(context.Background(), "evt_not_exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMarkWebhookProcessed(t *testing.T) {
	db := &fakeDB{}
	r := &Repository{db: db}
	if err := r.MarkWebhookProcessed(context.Background(), uuid.New(), ""); err != nil {
		t.Fatalf("MarkWebhookProcessed: %v", err)
	}
}
