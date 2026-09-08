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
// Slack channel
// =============================================================================

// SlackChannel posts to a Slack Incoming Webhook.
type SlackChannel struct {
	defaultWebhook string
	client        *http.Client
}

func NewSlackChannel(webhookURL string) *SlackChannel {
	return &SlackChannel{defaultWebhook: webhookURL, client: &http.Client{Timeout: 10 * time.Second}}
}

func (c *SlackChannel) Name() string { return "slack" }

type slackPayload struct {
	Text        string           `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
	Blocks     []slackBlock     `json:"blocks,omitempty"`
}

type slackAttachment struct {
	Color string `json:"color,omitempty"`
	Text  string `json:"text,omitempty"`
	Title string `json:"title,omitempty"`
}

type slackBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func (c *SlackChannel) Send(ctx context.Context, n Notification) (DeliveryResult, error) {
	var webhook string
	if v, ok := n.Data["webhook_url"]; ok {
		webhook, _ = v.(string)
	}
	if webhook == "" {
		webhook = c.defaultWebhook
	}
	if webhook == "" {
		return DeliveryResult{Status: "failed", Provider: "slack"}, fmt.Errorf("slack webhook not configured")
	}
	payload := slackPayload{
		Text: fmt.Sprintf("*%s*\n%s", n.Title, n.Body),
		Attachments: []slackAttachment{{
			Color: "#4A90E2",
			Text:  n.Body,
		}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "slack"}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "slack"}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return DeliveryResult{Status: "failed", Provider: "slack"}, fmt.Errorf("slack %d: %s", resp.StatusCode, string(buf))
	}
	return DeliveryResult{Status: "sent", Provider: "slack"}, nil
}

// =============================================================================
// Discord channel (same interface as Slack, different format)
// =============================================================================

// DiscordChannel posts to a Discord webhook.
type DiscordChannel struct {
	defaultWebhook string
	client        *http.Client
}

func NewDiscordChannel(webhookURL string) *DiscordChannel {
	return &DiscordChannel{defaultWebhook: webhookURL, client: &http.Client{Timeout: 10 * time.Second}}
}

func (c *DiscordChannel) Name() string { return "discord" }

type discordPayload struct {
	Content   string            `json:"content,omitempty"`
	Embeds    []discordEmbed   `json:"embeds,omitempty"`
}

type discordEmbed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color,omitempty"`
}

func (c *DiscordChannel) Send(ctx context.Context, n Notification) (DeliveryResult, error) {
	var webhook string
	if v, ok := n.Data["webhook_url"]; ok {
		webhook, _ = v.(string)
	}
	if webhook == "" {
		webhook = c.defaultWebhook
	}
	if webhook == "" {
		return DeliveryResult{Status: "failed", Provider: "discord"}, fmt.Errorf("discord webhook not configured")
	}
	payload := discordPayload{
		Content: fmt.Sprintf("**%s**\n%s", n.Title, n.Body),
		Embeds: []discordEmbed{{
			Title:       n.Title,
			Description: n.Body,
			Color:       0x4A90E2,
		}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "discord"}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return DeliveryResult{Status: "failed", Provider: "discord"}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return DeliveryResult{Status: "failed", Provider: "discord"}, fmt.Errorf("discord %d: %s", resp.StatusCode, string(buf))
	}
	return DeliveryResult{Status: "sent", Provider: "discord"}, nil
}
