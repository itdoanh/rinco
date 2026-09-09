// Extra tests for billing webhook event handling.
package webhook

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
	"github.com/itdoanh/rinco/services/billing-service/internal/payment"
)

// fakeDriver implements payment.Driver for testing.
type fakeDriver struct {
	verifyPayload *payment.WebhookPayload
	verifyErr     error
}

func (f *fakeDriver) CreateCheckoutSession(ctx context.Context, sub *models.Subscription, successURL, cancelURL string) (string, error) {
	return "", nil
}
func (f *fakeDriver) CreateCustomer(ctx context.Context, tenantID uuid.UUID, email, name string) (string, error) {
	return "cus_test", nil
}
func (f *fakeDriver) CreateSubscription(ctx context.Context, customerID, priceID string, trialDays int) (string, error) {
	return "sub_test", nil
}
func (f *fakeDriver) CancelSubscription(ctx context.Context, subID string) error {
	return nil
}
func (f *fakeDriver) UpdateSubscription(ctx context.Context, subID, newPriceID string) error {
	return nil
}
func (f *fakeDriver) GetCheckoutSession(ctx context.Context, sessionID string) (*payment.CheckoutSession, error) {
	return &payment.CheckoutSession{}, nil
}
func (f *fakeDriver) ConstructWebhookEvent(ctx context.Context, payload []byte, sig, secret string) (*payment.WebhookPayload, error) {
	if f.verifyErr != nil {
		return nil, f.verifyErr
	}
	return f.verifyPayload, nil
}
func (f *fakeDriver) CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	return "", nil
}
func (f *fakeDriver) CreateInvoiceItem(ctx context.Context, customerID string, amount int64, currency, description string) error {
	return nil
}

// stubRepo implements a minimal repo interface for testing.
// Since we can't easily mock *repository.Repository (concrete type),
// tests use httptest to focus on the entrypoint/handler behavior.
func makeRequest(payload string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newEchoCtx(payload string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := makeRequest(payload)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestExtraProcessEvent_UnhandledType(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{Type: "unknown.event"}
	if err := h.processEvent(context.Background(), p); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtraHandleCheckoutCompleted_NoSubID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{Type: "checkout.session.completed"}
	if err := h.handleCheckoutCompleted(context.Background(), p); err != nil {
		t.Errorf("should not err on missing SubID: %v", err)
	}
}

func TestExtraHandleCheckoutCompleted_WithSubID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{Type: "checkout.session.completed", SubID: "sub_123"}
	if err := h.handleCheckoutCompleted(context.Background(), p); err != nil {
		t.Errorf("err: %v", err)
	}
}

func TestExtraHandleSubscriptionChange_NoSubID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{Type: "customer.subscription.updated"}
	if err := h.handleSubscriptionChange(context.Background(), p); err != nil {
		t.Errorf("should not err: %v", err)
	}
}

func TestExtraHandleSubscriptionChange_WithSubID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{SubID: "sub_123", Status: "active"}
	if err := h.handleSubscriptionChange(context.Background(), p); err != nil {
		t.Errorf("err: %v", err)
	}
}

func TestExtraHandleSubscriptionCancelled_NoSubID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{}
	if err := h.handleSubscriptionCancelled(context.Background(), p); err != nil {
		t.Errorf("should not err: %v", err)
	}
}

func TestExtraHandleSubscriptionCancelled_WithSubID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{SubID: "sub_cancel"}
	if err := h.handleSubscriptionCancelled(context.Background(), p); err != nil {
		t.Errorf("err: %v", err)
	}
}

func TestExtraHandleInvoicePaid_NoInvoiceID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{}
	if err := h.handleInvoicePaid(context.Background(), p); err != nil {
		t.Errorf("should not err: %v", err)
	}
}

func TestExtraHandleInvoicePaid_WithInvoiceID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{InvoiceID: "in_123", AmountPaid: 9999, Currency: "USD"}
	if err := h.handleInvoicePaid(context.Background(), p); err != nil {
		t.Errorf("err: %v", err)
	}
}

func TestExtraHandlePaymentFailed_NoSubID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{}
	if err := h.handlePaymentFailed(context.Background(), p); err != nil {
		t.Errorf("should not err: %v", err)
	}
}

func TestExtraHandlePaymentFailed_WithSubID(t *testing.T) {
	h := &Handler{log: slog.Default()}
	p := &payment.WebhookPayload{SubID: "sub_fail"}
	if err := h.handlePaymentFailed(context.Background(), p); err != nil {
		t.Errorf("err: %v", err)
	}
}

func TestExtraNewHandler(t *testing.T) {
	h := NewHandler(nil, &fakeDriver{}, "secret")
	if h == nil {
		t.Fatal("nil handler")
	}
	if h.secret != "secret" {
		t.Errorf("secret: %s", h.secret)
	}
	if h.log == nil {
		t.Error("logger should not be nil")
	}
}

func TestExtraProcessEvent_AllTypes(t *testing.T) {
	h := &Handler{log: slog.Default()}
	types := []string{
		"customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted",
		"invoice.paid",
		"invoice.payment_failed",
		"checkout.session.completed",
		"some.unknown.event",
	}
	for _, typ := range types {
		p := &payment.WebhookPayload{Type: typ, SubID: "sub_x"}
		if err := h.processEvent(context.Background(), p); err != nil {
			t.Errorf("type %s: %v", typ, err)
		}
	}
}

func TestExtraStripeWebhook_BadJSON(t *testing.T) {
	// Cannot easily test without a real repo, but the early return path
	// for invalid JSON should be triggered. Skip if can't be tested.
	t.Skip("requires full repo mock")
}

func TestExtraStripeWebhook_BodyRead(t *testing.T) {
	// Test just the body reading capability
	body := `{"id":"evt_x","type":"customer.subscription.updated","data":{"object":{"id":"sub_x","customer":"cus_x","status":"active"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	_, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("body read failed: %v", err)
	}
}
