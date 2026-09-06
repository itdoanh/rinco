// Notification Service - multi-channel notifications (Telegram, Slack, Discord, SMS, Webhook).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "notification-service"
	version     = "1.0.0"
)

type notification struct {
	Channel   string                 `json:"channel"` // telegram, slack, discord, sms, webhook, push
	Recipient string                 `json:"recipient"`
	Subject   string                 `json:"subject,omitempty"`
	Message   string                 `json:"message" validate:"required"`
	Metadata  map[string]interface{} `json:"metadata"`
	Priority  string                 `json:"priority"` // low, normal, high, urgent
}

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

	e := echo.New()
	e.HideBanner = true
	e.Use(rincowebmw.Recovery())
	e.Use(rincowebmw.Trace())
	e.Use(rincowebmw.Logger())
	e.Use(rincowebmw.Metrics(serviceName))
	e.Use(rincowebmw.CORS([]string{"*"}))
	e.Use(rincowebmw.SecurityHeaders())

	e.GET("/health", healthHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.POST("/v1/notify", sendHandler)
	e.POST("/v1/notify/batch", batchSendHandler)

	port := ":" + getEnv("PORT", "8088")
	logger.Info(context.Background(), "starting notification service", zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func sendHandler(c echo.Context) error {
	var n notification
	if err := c.Bind(&n); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if n.Message == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "message required"})
	}
	if n.Channel == "" {
		n.Channel = "telegram"
	}
	if n.Priority == "" {
		n.Priority = "normal"
	}

	if err := sendNotification(c.Request().Context(), n); err != nil {
		logger.Error(c.Request().Context(), "send notification failed", err,
			zap.String("channel", n.Channel),
			zap.String("recipient", n.Recipient),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "sent",
		"notification_id": uuid.NewV7().String(),
		"sent_at":    time.Now(),
	})
}

func batchSendHandler(c echo.Context) error {
	var ns []notification
	if err := c.Bind(&ns); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	results := make([]map[string]interface{}, len(ns))
	for i, n := range ns {
		if err := sendNotification(c.Request().Context(), n); err != nil {
			results[i] = map[string]interface{}{"status": "failed", "error": err.Error()}
		} else {
			results[i] = map[string]interface{}{"status": "sent"}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"results": results})
}

func sendNotification(ctx context.Context, n notification) error {
	switch n.Channel {
	case "telegram":
		return sendTelegram(ctx, n)
	case "slack":
		return sendSlack(ctx, n)
	case "discord":
		return sendDiscord(ctx, n)
	case "webhook":
		return sendWebhook(ctx, n)
	case "sms":
		return sendSMS(ctx, n)
	case "push":
		return sendPush(ctx, n)
	default:
		return fmt.Errorf("unsupported channel: %s", n.Channel)
	}
}

func sendTelegram(ctx context.Context, n notification) error {
	token := getEnv("TELEGRAM_BOT_TOKEN", "")
	if token == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	chatID := n.Recipient
	if chatID == "" {
		chatID = getEnv("TELEGRAM_DEFAULT_CHAT_ID", "")
		if chatID == "" {
			return fmt.Errorf("recipient (chat_id) required")
		}
	}

	body := map[string]interface{}{
		"chat_id":    chatID,
		"text":       formatMessage(n),
		"parse_mode": "Markdown",
	}
	bodyJSON, _ := json.Marshal(body)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(body))
	}
	return nil
}

func sendSlack(ctx context.Context, n notification) error {
	webhook := getEnv("SLACK_WEBHOOK_URL", "")
	if webhook == "" {
		return fmt.Errorf("SLACK_WEBHOOK_URL not set")
	}

	body := map[string]interface{}{
		"channel": n.Recipient,
		"text":    n.Subject,
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{"type": "mrkdwn", "text": formatMessage(n)},
			},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", webhook, bytes.NewReader(bodyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func sendDiscord(ctx context.Context, n notification) error {
	webhook := n.Recipient
	if webhook == "" {
		webhook = getEnv("DISCORD_WEBHOOK_URL", "")
	}
	if webhook == "" {
		return fmt.Errorf("DISCORD_WEBHOOK_URL not set")
	}

	body := map[string]interface{}{
		"content": formatMessage(n),
	}
	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", webhook, bytes.NewReader(bodyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func sendWebhook(ctx context.Context, n notification) error {
	bodyJSON, _ := json.Marshal(n)
	req, err := http.NewRequestWithContext(ctx, "POST", n.Recipient, bytes.NewReader(bodyJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func sendSMS(ctx context.Context, n notification) error {
	// Twilio stub - in production use twilio-go SDK
	logger.Info(ctx, "sms_send_stub", zap.String("to", n.Recipient))
	return nil
}

func sendPush(ctx context.Context, n notification) error {
	// FCM stub - in production use firebase-admin
	logger.Info(ctx, "push_send_stub", zap.String("to", n.Recipient))
	return nil
}

func formatMessage(n notification) string {
	var sb strings.Builder
	if n.Priority == "urgent" || n.Priority == "high" {
		sb.WriteString(fmt.Sprintf("🚨 *Priority: %s*\n\n", strings.ToUpper(n.Priority)))
	}
	if n.Subject != "" {
		sb.WriteString(fmt.Sprintf("*%s*\n\n", n.Subject))
	}
	sb.WriteString(n.Message)
	if len(n.Metadata) > 0 {
		sb.WriteString("\n\n_Metadata:_\n")
		for k, v := range n.Metadata {
			sb.WriteString(fmt.Sprintf("- `%s`: %v\n", k, v))
		}
	}
	return sb.String()
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
