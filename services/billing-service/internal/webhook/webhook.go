package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/billing-service/internal/models"
	"github.com/itdoanh/rinco/services/billing-service/internal/payment"
	"github.com/itdoanh/rinco/services/billing-service/internal/repository"
)

// Handler processes Stripe webhook events
type Handler struct {
	repo      *repository.Repository
	driver    payment.Driver
	secret    string
	log       *slog.Logger
}

func NewHandler(repo *repository.Repository, driver payment.Driver, secret string) *Handler {
	return &Handler{
		repo:   repo,
		driver: driver,
		secret: secret,
		log:    slog.Default(),
	}
}// StripeWebhook handles incoming Stripe webhook events
func (h *Handler) StripeWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot read body"})
	}

	// Idempotency: check if event already processed
	// We'll parse the event ID from raw JSON first
	var rawEvent struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &rawEvent); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
	}

	existing, _ := h.repo.GetWebhookEvent(context.Background(), rawEvent.ID)
	if existing != nil && !existing.ProcessedAt.IsZero() {
		// Already processed, return 200 to avoid Stripe retry
		return c.JSON(http.StatusOK, map[string]string{"status": "already processed"})
	}

	payload, err := h.driver.ConstructWebhookEvent(c.Request().Context(), body, c.Request().Header.Get("Stripe-Signature"), h.secret)
	if err != nil {
		h.log.Error("webhook signature verification failed", "err", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid signature"})
	}

	// Record webhook event (unprocessed)
	webhookEv := &models.WebhookEvent{
		ID:            uuid.New(),
		StripeEventID: rawEvent.ID,
		Type:          payload.Type,
		DataPayload:   body,
		Attempts:      1,
		CreatedAt:     time.Now(),
	}
	if err := h.repo.CreateWebhookEvent(context.Background(), webhookEv); err != nil {
		h.log.Warn("failed to record webhook event", "err", err)
	}

	// Process event
	if err := h.processEvent(context.Background(), payload); err != nil {
		h.log.Error("webhook event processing failed", "type", payload.Type, "err", err)
		// Still return 200 to prevent Stripe retry loop; error is logged
	}

	// Mark as processed
	_ = h.repo.MarkWebhookProcessed(context.Background(), webhookEv.ID, "")

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) processEvent(ctx context.Context, p *payment.WebhookPayload) error {
	switch p.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		return h.handleSubscriptionChange(ctx, p)
	case "customer.subscription.deleted":
		return h.handleSubscriptionCancelled(ctx, p)
	case "invoice.paid":
		return h.handleInvoicePaid(ctx, p)
	case "invoice.payment_failed":
		return h.handlePaymentFailed(ctx, p)
	case "checkout.session.completed":
		return h.handleCheckoutCompleted(ctx, p)
	default:
		h.log.Info("unhandled webhook event type", "type", p.Type)
		return nil
	}
}

func (h *Handler) handleCheckoutCompleted(ctx context.Context, p *payment.WebhookPayload) error {
	if p.SubID == "" {
		return nil
	}
	// Update subscription status to active
	h.log.Info("checkout completed", "sub_id", p.SubID)
	return nil
}

func (h *Handler) handleSubscriptionChange(ctx context.Context, p *payment.WebhookPayload) error {
	if p.SubID == "" {
		return nil
	}
	// Find subscription by Stripe ID and update status
	h.log.Info("subscription changed", "sub_id", p.SubID, "status", p.Status)
	// In production, would lookup by stripe_sub_id and update local DB
	return nil
}

func (h *Handler) handleSubscriptionCancelled(ctx context.Context, p *payment.WebhookPayload) error {
	if p.SubID == "" {
		return nil
	}
	// Mark subscription as cancelled
	h.log.Info("subscription cancelled", "sub_id", p.SubID)
	// Publish NATS event for other services
	return nil
}

func (h *Handler) handleInvoicePaid(ctx context.Context, p *payment.WebhookPayload) error {
	if p.InvoiceID == "" {
		return nil
	}
	// Update invoice status to paid
	h.log.Info("invoice paid", "invoice_id", p.InvoiceID, "amount", p.AmountPaid, "currency", p.Currency)
	return nil
}

func (h *Handler) handlePaymentFailed(ctx context.Context, p *payment.WebhookPayload) error {
	if p.SubID == "" {
		return nil
	}
	// Mark subscription as past_due
	h.log.Warn("payment failed", "sub_id", p.SubID)
	// Publish alert notification
	return nil
}
