package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// =============================================================================
// In-app channel (NATS pub/sub → chat-engine)
// =============================================================================

// InAppChannel publishes a notification to NATS so the chat-engine WebSocket
// gateway can fan it out to connected clients.
type InAppChannel struct {
	nc     *nats.Conn
	prefix string
}

func NewInAppChannel(nc *nats.Conn) *InAppChannel {
	return &InAppChannel{nc: nc, prefix: "user."}
}

func (c *InAppChannel) Name() string { return "in_app" }

type inAppPayload struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Icon      string         `json:"icon,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Category  string         `json:"category,omitempty"`
	Priority  string         `json:"priority,omitempty"`
	SentAt    time.Time      `json:"sent_at"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
}

func (c *InAppChannel) Send(ctx context.Context, n Notification) (DeliveryResult, error) {
	if n.UserID == "" {
		return DeliveryResult{Status: "failed", Provider: "in_app"}, fmt.Errorf("user_id required for in_app channel")
	}
	payload := inAppPayload{
		ID:       n.Data["notif_id"].(string),
		UserID:   n.UserID,
		Type:     n.Type,
		Title:    n.Title,
		Body:     n.Body,
		Icon:     n.Icon,
		Data:     n.Data,
		SentAt:   time.Now(),
		ExpiresAt: n.ExpiresAt,
	}
	if c.nc == nil {
		slog.Warn("nats not available, in_app notification dropped", slog.String("user_id", n.UserID))
		return DeliveryResult{Status: "sent", Provider: "in_app"}, nil
	}
	body, _ := json.Marshal(payload)
	subject := c.prefix + n.UserID + ".notification"
	if err := c.nc.Publish(subject, body); err != nil {
		return DeliveryResult{Status: "failed", Provider: "in_app"}, fmt.Errorf("nats publish: %w", err)
	}
	return DeliveryResult{Status: "sent", Provider: "in_app"}, nil
}
