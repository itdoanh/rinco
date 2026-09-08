// Tests for notification-service channels.
package channels

import (
	"context"
	"testing"
)

func TestInAppChannel_Name(t *testing.T) {
	c := NewInAppChannel(nil)
	if c.Name() != "in_app" {
		t.Errorf("expected 'in_app', got %s", c.Name())
	}
}

func TestInAppChannel_NoUserID(t *testing.T) {
	c := NewInAppChannel(nil)
	_, err := c.Send(context.Background(), Notification{UserID: ""})
	if err == nil {
		t.Error("expected error for missing user_id")
	}
}

func TestInAppChannel_NoNATS(t *testing.T) {
	// With nil nc, should still return "sent" (just drop the message).
	c := NewInAppChannel(nil)
	n := Notification{
		UserID: "u1",
		Title:  "test",
		Body:   "body",
		Data:   map[string]any{"notif_id": "n1"},
	}
	res, err := c.Send(context.Background(), n)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if res.Status != "sent" {
		t.Errorf("expected sent status, got %s", res.Status)
	}
	if res.Provider != "in_app" {
		t.Errorf("expected in_app provider, got %s", res.Provider)
	}
}

func TestInAppChannel_NilData(t *testing.T) {
	// Should not panic with nil Data.
	c := NewInAppChannel(nil)
	n := Notification{
		UserID: "u1",
		Title:  "test",
		Body:   "body",
		Data:   nil,
	}
	res, err := c.Send(context.Background(), n)
	if err != nil {
		t.Errorf("expected no error with nil data, got %v", err)
	}
	if res.Status != "sent" {
		t.Errorf("expected sent, got %s", res.Status)
	}
}

func TestInAppChannel_NonStringNotifID(t *testing.T) {
	// notif_id as non-string type should not panic.
	c := NewInAppChannel(nil)
	n := Notification{
		UserID: "u1",
		Data:   map[string]any{"notif_id": 12345},
	}
	res, err := c.Send(context.Background(), n)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if res.Status != "sent" {
		t.Errorf("expected sent, got %s", res.Status)
	}
}

func TestTelegramChannel_NoBotToken(t *testing.T) {
	c := &TelegramChannel{}
	_, err := c.Send(context.Background(), Notification{
		Data: map[string]any{"chat_id": "123"},
	})
	if err == nil {
		t.Error("expected error when bot token missing")
	}
}

func TestTelegramChannel_NoChatID(t *testing.T) {
	c := &TelegramChannel{botToken: "test-token"}
	_, err := c.Send(context.Background(), Notification{
		Data: map[string]any{},
	})
	if err == nil {
		t.Error("expected error when chat_id missing")
	}
}

func TestTelegramChannel_NilData(t *testing.T) {
	c := &TelegramChannel{botToken: "test-token"}
	_, err := c.Send(context.Background(), Notification{Data: nil})
	if err == nil {
		t.Error("expected error with nil data")
	}
}

func TestTelegramChannel_NonStringChatID(t *testing.T) {
	c := &TelegramChannel{botToken: "test-token"}
	n := Notification{Data: map[string]any{"chat_id": 12345}}
	// Should not panic; should fail at chat_id check.
	_, err := c.Send(context.Background(), n)
	if err == nil {
		t.Error("expected error for non-string chat_id")
	}
}

func TestSlackChannel_NoWebhook(t *testing.T) {
	c := NewSlackChannel("")
	_, err := c.Send(context.Background(), Notification{
		Data: map[string]any{},
	})
	if err == nil {
		t.Error("expected error when webhook missing")
	}
}

func TestSlackChannel_DefaultWebhook(t *testing.T) {
	c := NewSlackChannel("https://hooks.slack.test.invalid/test")
	n := Notification{
		Title: "Test",
		Body:  "Body",
		Data:  map[string]any{}, // no override
	}
	// Should fail at DNS resolution (since webhook is fake), but not at validation.
	_, err := c.Send(context.Background(), n)
	if err == nil {
		t.Error("expected error from fake webhook")
	}
}

func TestSlackChannel_NilData(t *testing.T) {
	// Should not panic with nil Data.
	c := NewSlackChannel("https://example.com")
	_, _ = c.Send(context.Background(), Notification{Data: nil})
	// No panic = success.
}

func TestSlackChannel_NonStringWebhook(t *testing.T) {
	c := NewSlackChannel("https://default.example.com")
	n := Notification{Data: map[string]any{"webhook_url": 12345}}
	// Should not panic; uses default webhook.
	_, _ = c.Send(context.Background(), n)
}

func TestDiscordChannel_NoWebhook(t *testing.T) {
	c := NewDiscordChannel("")
	_, err := c.Send(context.Background(), Notification{
		Data: map[string]any{},
	})
	if err == nil {
		t.Error("expected error when webhook missing")
	}
}

func TestDiscordChannel_DefaultWebhook(t *testing.T) {
	c := NewDiscordChannel("https://discord.example.com/webhook")
	n := Notification{Title: "T", Body: "B", Data: map[string]any{}}
	_, err := c.Send(context.Background(), n)
	if err == nil {
		t.Error("expected HTTP error from fake webhook")
	}
}

func TestNotificationStruct(t *testing.T) {
	n := Notification{
		TenantID:  "t1",
		UserID:    "u1",
		Type:      "test",
		Title:     "Hi",
		Body:      "Hello",
		Email:     "a@b.com",
		Phone:     "+1234567890",
		Token:     "tok",
		Endpoint:  "https://example.com",
		Data:      map[string]any{"k": "v"},
	}
	if n.UserID != "u1" {
		t.Errorf("UserID: got %s", n.UserID)
	}
	if n.Email != "a@b.com" {
		t.Errorf("Email: got %s", n.Email)
	}
	if n.Data["k"] != "v" {
		t.Errorf("Data: got %v", n.Data)
	}
}
