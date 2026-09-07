package channels

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PushChannel delivers Web Push notifications via VAPID.
type PushChannel struct {
	vapidPublic  string
	vapidPrivate string
	vapidSubject string
	client       *http.Client
}

func NewPushChannel(vapidPublic, vapidPrivate, vapidSubject string) *PushChannel {
	return &PushChannel{
		vapidPublic:  vapidPublic,
		vapidPrivate: vapidPrivate,
		vapidSubject: vapidSubject,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *PushChannel) Name() string { return "push" }

func (c *PushChannel) Send(ctx context.Context, n Notification) (DeliveryResult, error) {
	if n.Endpoint == "" || n.Token == "" {
		return DeliveryResult{Status: "failed", Provider: "push"}, fmt.Errorf("push endpoint required")
	}
	payload, _ := json.Marshal(map[string]any{
		"title": n.Title,
		"body":  n.Body,
		"icon":  n.Icon,
		"data":  n.Data,
	})
	body, _ := json.Marshal(map[string]any{
		"endpoint": n.Endpoint,
		"payload":  base64.RawURLEncoding.EncodeToString(payload),
		"auth":     n.Data["auth"],
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.vapidSubject+"/push/send", bytes.NewReader(body))
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "push"}, err
	}
	req.Header.Set("Content-Type", "application/json")
	// In production use github.com/emersion/go-webpush; this is a minimal forwarder stub.
	resp, err := c.client.Do(req)
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "push"}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return DeliveryResult{Status: "failed", Provider: "push"}, fmt.Errorf("push %d: %s", resp.StatusCode, string(buf))
	}
	return DeliveryResult{Status: "sent", Provider: "push"}, nil
}

// FCMChannel delivers via Firebase Cloud Messaging.
type FCMChannel struct {
	projectID string
	serverKey string
	client    *http.Client
}

func NewFCMChannel(projectID, serverKey string) *FCMChannel {
	return &FCMChannel{
		projectID: projectID,
		serverKey: serverKey,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *FCMChannel) Name() string { return "fcm" }

func (c *FCMChannel) Send(ctx context.Context, n Notification) (DeliveryResult, error) {
	if n.Token == "" {
		return DeliveryResult{Status: "failed", Provider: "fcm"}, fmt.Errorf("device token required")
	}
	payload := map[string]any{
		"to": n.Token,
		"notification": map[string]string{
			"title": n.Title,
			"body":  n.Body,
			"icon":  n.Icon,
		},
		"data": n.Data,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://fcm.googleapis.com/fcm/send", bytes.NewReader(body))
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "fcm"}, err
	}
	req.Header.Set("Authorization", "key="+c.serverKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "fcm"}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return DeliveryResult{Status: "failed", Provider: "fcm"}, fmt.Errorf("fcm %d: %s", resp.StatusCode, string(buf))
	}
	var result struct {
		Success int `json:"success"`
		Failure int `json:"failure"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Failure > 0 {
		return DeliveryResult{Status: "failed", Provider: "fcm"}, fmt.Errorf("fcm delivery failed")
	}
	return DeliveryResult{Status: "sent", Provider: "fcm"}, nil
}
