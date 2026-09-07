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

// =============================================================================
// Telegram channel
// =============================================================================

// TelegramChannel sends messages via the Telegram Bot API.
type TelegramChannel struct {
	botToken string
	client   *http.Client
}

func NewTelegramChannel(botToken string) *TelegramChannel {
	return &TelegramChannel{botToken: botToken, client: &http.Client{Timeout: 10 * time.Second}}
}

func (c *TelegramChannel) Name() string { return "telegram" }

type tgSendMessageReq struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool `json:"disable_web_page_preview,omitempty"`
}

func (c *TelegramChannel) Send(ctx context.Context, n Notification) (DeliveryResult, error) {
	if c.botToken == "" {
		return DeliveryResult{Status: "failed", Provider: "telegram"}, fmt.Errorf("telegram bot token not configured")
	}
	chatID := n.Data["chat_id"].(string)
	if chatID == "" {
		return DeliveryResult{Status: "failed", Provider: "telegram"}, fmt.Errorf("chat_id required")
	}
	text := fmt.Sprintf("*%s*\n%s", n.Title, n.Body)
	payload := tgSendMessageReq{ChatID: chatID, Text: text, ParseMode: "Markdown"}
	body, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "telegram"}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "telegram"}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return DeliveryResult{Status: "failed", Provider: "telegram"}, fmt.Errorf("telegram %d: %s", resp.StatusCode, string(buf))
	}
	return DeliveryResult{Status: "sent", Provider: "telegram"}, nil
}
