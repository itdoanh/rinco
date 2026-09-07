package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmailChannel calls the email-service HTTP API.
type EmailChannel struct {
	baseURL string
	client  *http.Client
}

func NewEmailChannel(baseURL string) *EmailChannel {
	return &EmailChannel{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *EmailChannel) Name() string { return "email" }

func (c *EmailChannel) Send(ctx context.Context, n Notification) (DeliveryResult, error) {
	if n.Email == "" {
		return DeliveryResult{Status: "failed", Provider: "email"}, fmt.Errorf("email address required")
	}
	payload := map[string]any{
		"to":        []string{n.Email},
		"subject":   n.Title,
		"body":      n.Body,
		"body_type": "html",
		"priority":  "normal",
		"headers":   map[string]string{"X-Notif-Type": n.Type},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/email/send", bytes.NewReader(body))
	if err != nil {
		return DeliveryResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "email"}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return DeliveryResult{Status: "failed", Provider: "email"}, fmt.Errorf("email-service %d: %s", resp.StatusCode, string(buf))
	}
	return DeliveryResult{Status: "sent", Provider: "email"}, nil
}
