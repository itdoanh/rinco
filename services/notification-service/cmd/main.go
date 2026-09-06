// Notification Service — multi-channel (in-app, email, SMS, web push,
// FCM, Slack, Discord, Telegram).  Priority-based fanout theo channel
// matrix (high / normal / low).  User preferences: per-type enable,
// quiet hours, daily digest.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "notification-service"
	version     = "1.0.0"
)

// Channels
const (
	ChannelInApp  = "in_app"
	ChannelEmail  = "email"
	ChannelSMS    = "sms"
	ChannelPush   = "push"
	ChannelFCM    = "fcm"
	ChannelSlack  = "slack"
	ChannelDiscord = "discord"
	ChannelTelegram = "telegram"
	ChannelWebhook = "webhook"
)

// Priorities
const (
	PriorityHigh   = "high"
	PriorityNormal = "normal"
	PriorityLow    = "low"
)

// =============================================================================
// Types
// =============================================================================

type notificationRequest struct {
	UserID  string                 `json:"user_id" validate:"required"`
	Type    string                 `json:"type" validate:"required"` // task_assigned | lead_scored | ...
	Title   string                 `json:"title"`
	Message string                 `json:"message" validate:"required"`
	Data    map[string]interface{} `json:"data"`
	Priority string                `json:"priority"`
	Channels []string              `json:"channels,omitempty"` // override default matrix
	Recipient string               `json:"recipient,omitempty"` // email / phone / chat id khi bypass user
}

type broadcastRequest struct {
	Type     string                 `json:"type" validate:"required"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message" validate:"required"`
	Data     map[string]interface{} `json:"data"`
	Audience map[string]interface{} `json:"audience"` // tenant_id, role, ...
}

type preferencesRequest struct {
	UserID     string            `json:"user_id" validate:"required"`
	Channels   map[string]bool   `json:"channels"`     // channel → enabled
	QuietHours []int             `json:"quiet_hours"`  // hour-of-day [start, end]
	DailyDigest bool             `json:"daily_digest"`
	Types      map[string]bool   `json:"types"`         // type-level enable/disable
}

type preferencesResponse struct {
	UserID      string          `json:"user_id"`
	Channels    map[string]bool `json:"channels"`
	QuietHours  []int           `json:"quiet_hours"`
	DailyDigest bool            `json:"daily_digest"`
	Types       map[string]bool `json:"types"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type subscriptionRequest struct {
	UserID string                 `json:"user_id" validate:"required"`
	Type   string                 `json:"type"` // web | fcm
	Keys   map[string]string      `json:"keys"` // p256dh, auth (web) or token (fcm)
}

type inAppItem struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Read      bool                   `json:"read"`
	CreatedAt time.Time              `json:"created_at"`
}

type statsResponse struct {
	Sent     int            `json:"sent"`
	ByChannel map[string]int `json:"by_channel"`
	Read     int            `json:"read"`
	Unread   int            `json:"unread"`
}

// =============================================================================
// State (in-memory; production: PostgreSQL + Redis)
// =============================================================================

var (
	muP, muN, muS sync.RWMutex
	prefs         = map[string]preferencesResponse{}
	inApp         = []inAppItem{}
	subs          = map[string][]subscriptionRequest{} // user_id → list
	stats         = struct {
		Sent       int
		ByChannel  map[string]int
		Read       int
	}{Sent: 0, ByChannel: map[string]int{}, Read: 0}
)

// =============================================================================
// main
// =============================================================================

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(rincowebmw.Recovery())
	e.Use(rincowebmw.Trace())
	e.Use(rincowebmw.Logger())
	e.Use(rincowebmw.Metrics(serviceName))
	e.Use(rincowebmw.CORS([]string{"*"}))
	e.Use(rincowebmw.SecurityHeaders())

	e.GET("/health", healthHandler)
	e.GET("/ready", readyHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	n := e.Group("/v1/notifications")
	n.POST("/send", sendHandler)
	n.POST("/broadcast", broadcastHandler)
	n.GET("", listInAppHandler)
	n.GET("/:id", getInAppHandler)
	n.POST("/:id/read", markReadHandler)
	n.POST("/preferences/:user_id", setPreferencesHandler)
	n.GET("/preferences/:user_id", getPreferencesHandler)
	n.POST("/subscriptions", addSubscriptionHandler)
	n.GET("/stats", statsHandler)

	port := ":" + getEnv("PORT", "8088")
	logger.Info(context.Background(), "starting notification service",
		zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

// =============================================================================
// Metrics
// =============================================================================

var (
	notifySent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rinco_notify_sent_total",
		Help: "Notifications sent per channel + priority",
	}, []string{"channel", "priority", "status"})
	notifyFanout = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rinco_notify_fanout_seconds",
		Help:    "Fan-out duration across channels",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"priority"})
)

// =============================================================================
// Send / Broadcast
// =============================================================================

func sendHandler(c echo.Context) error {
	var req notificationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id required"})
	}
	if req.Priority == "" {
		req.Priority = PriorityNormal
	}

	prefs := getOrDefaultPrefs(req.UserID)

	start := time.Now()
	channels := pickChannels(prefs, req)
	var sentChannels []string
	for _, ch := range channels {
		if !isChannelEnabled(prefs, ch, req.Type) {
			continue
		}
		if isInQuietHours(prefs) && ch != ChannelInApp {
			continue
		}
		if err := dispatchChannel(c.Request().Context(), ch, req); err != nil {
			logger.Error(c.Request().Context(), "notification dispatch failed", err,
				zap.String("channel", ch),
				zap.String("user_id", req.UserID))
			notifySent.WithLabelValues(ch, req.Priority, "failed").Inc()
			continue
		}
		notifySent.WithLabelValues(ch, req.Priority, "sent").Inc()
		sentChannels = append(sentChannels, ch)
		muN.Lock()
		stats.Sent++
		stats.ByChannel[ch]++
		muN.Unlock()
	}
	notifyFanout.WithLabelValues(req.Priority).Observe(time.Since(start).Seconds())

	// Persist in-app entry (always).
	entry := inAppItem{
		ID: uuid.NewV7().String(), UserID: req.UserID, Type: req.Type,
		Title: req.Title, Message: req.Message, Data: req.Data,
		Read: false, CreatedAt: time.Now(),
	}
	muN.Lock()
	inApp = append(inApp, entry)
	muN.Unlock()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"notification_id": entry.ID,
		"channels_used":   sentChannels,
		"sent_at":         time.Now(),
	})
}

func broadcastHandler(c echo.Context) error {
	var req broadcastRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	// In production: resolve audience to list of user_ids via CRM.
	// Here: echo back the audience keys.
	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"status":   "queued",
		"audience": req.Audience,
		"queued_at": time.Now(),
	})
}

func pickChannels(prefs preferencesResponse, req notificationRequest) []string {
	if len(req.Channels) > 0 {
		return req.Channels
	}
	switch req.Priority {
	case PriorityHigh:
		return []string{ChannelInApp, ChannelEmail, ChannelPush}
	case PriorityNormal:
		return []string{ChannelInApp, ChannelEmail}
	default:
		return []string{ChannelInApp}
	}
}

func isChannelEnabled(prefs preferencesResponse, channel, nType string) bool {
	if v, ok := prefs.Channels[channel]; ok {
		if !v {
			return false
		}
	}
	if v, ok := prefs.Types[nType]; ok {
		if !v {
			return false
		}
	}
	return true
}

func isInQuietHours(prefs preferencesResponse) bool {
	if len(prefs.QuietHours) != 2 {
		return false
	}
	h := time.Now().Hour()
	start, end := prefs.QuietHours[0], prefs.QuietHours[1]
	if start < end {
		return h >= start && h < end
	}
	// wraps midnight
	return h >= start || h < end
}

// =============================================================================
// Preferences
// =============================================================================

func setPreferencesHandler(c echo.Context) error {
	userID := c.Param("user_id")
	var req preferencesRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id required"})
	}
	p := preferencesResponse{
		UserID:      userID,
		Channels:    ensureChannels(req.Channels),
		QuietHours:  req.QuietHours,
		DailyDigest: req.DailyDigest,
		Types:       req.Types,
		UpdatedAt:   time.Now(),
	}
	muP.Lock()
	prefs[userID] = p
	muP.Unlock()
	return c.JSON(http.StatusOK, p)
}

func getPreferencesHandler(c echo.Context) error {
	userID := c.Param("user_id")
	return c.JSON(http.StatusOK, getOrDefaultPrefs(userID))
}

func getOrDefaultPrefs(userID string) preferencesResponse {
	muP.RLock()
	if p, ok := prefs[userID]; ok {
		muP.RUnlock()
		return p
	}
	muP.RUnlock()
	return preferencesResponse{
		UserID: userID,
		Channels: map[string]bool{
			ChannelInApp: true, ChannelEmail: true, ChannelSMS: false,
			ChannelPush: true, ChannelFCM: true,
		},
		Types: map[string]bool{},
	}
}

func ensureChannels(in map[string]bool) map[string]bool {
	out := map[string]bool{
		ChannelInApp: true, ChannelEmail: true, ChannelSMS: false,
		ChannelPush: true, ChannelFCM: true,
	}
	for k, v := range in {
		out[k] = v
	}
	return out
}

// =============================================================================
// Subscriptions (Web Push / FCM)
// =============================================================================

func addSubscriptionHandler(c echo.Context) error {
	var req subscriptionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	muS.Lock()
	subs[req.UserID] = append(subs[req.UserID], req)
	muS.Unlock()
	return c.JSON(http.StatusCreated, map[string]string{"status": "registered"})
}

// =============================================================================
// In-app notification list
// =============================================================================

func listInAppHandler(c echo.Context) error {
	userID := c.QueryParam("user_id")
	unreadOnly := c.QueryParam("unread") == "1"
	muN.RLock()
	out := []inAppItem{}
	for _, n := range inApp {
		if userID != "" && n.UserID != userID {
			continue
		}
		if unreadOnly && n.Read {
			continue
		}
		out = append(out, n)
	}
	muN.RUnlock()
	if out == nil {
		out = []inAppItem{}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"count":         len(out),
		"notifications": out,
	})
}

func getInAppHandler(c echo.Context) error {
	id := c.Param("id")
	muN.RLock()
	defer muN.RUnlock()
	for _, n := range inApp {
		if n.ID == id {
			return c.JSON(http.StatusOK, n)
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
}

func markReadHandler(c echo.Context) error {
	id := c.Param("id")
	muN.Lock()
	defer muN.Unlock()
	for i := range inApp {
		if inApp[i].ID == id {
			inApp[i].Read = true
			stats.Read++
			return c.JSON(http.StatusOK, inApp[i])
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
}

func statsHandler(c echo.Context) error {
	muN.RLock()
	defer muN.RUnlock()
	return c.JSON(http.StatusOK, statsResponse{
		Sent: stats.Sent, ByChannel: stats.ByChannel,
		Read: stats.Read, Unread: stats.Sent - stats.Read,
	})
}

// =============================================================================
// Channel dispatchers
// =============================================================================

func dispatchChannel(ctx context.Context, channel string, req notificationRequest) error {
	switch channel {
	case ChannelInApp:
		// Already persisted by sendHandler; no-op.
		return nil
	case ChannelEmail:
		return delegateEmail(ctx, req)
	case ChannelSMS:
		return sendTwilio(ctx, req)
	case ChannelPush:
		return sendWebPush(ctx, req)
	case ChannelFCM:
		return sendFCM(ctx, req)
	case ChannelSlack:
		return sendSlack(ctx, req)
	case ChannelDiscord:
		return sendDiscord(ctx, req)
	case ChannelTelegram:
		return sendTelegram(ctx, req)
	case ChannelWebhook:
		return sendWebhook(ctx, req)
	}
	return fmt.Errorf("unknown channel: %s", channel)
}

func delegateEmail(ctx context.Context, req notificationRequest) error {
	// Delegate to email-service over HTTP.  Tenant + recipient inferred.
	to := req.Recipient
	if to == "" {
		to = req.UserID + "@rinco.app"
	}
	body, _ := json.Marshal(map[string]interface{}{
		"to":        []string{to},
		"subject":   orDefault(req.Title, "Notification"),
		"body":      req.Message,
		"body_type": "text",
	})
	endpoint := getEnv("EMAIL_SERVICE_URL", "http://email-service:8087") + "/v1/email/send"
	r, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Tenant-ID", "")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func sendTwilio(ctx context.Context, req notificationRequest) error {
	if req.Recipient == "" {
		return fmt.Errorf("recipient (phone) required for SMS")
	}
	accountSID := getEnv("TWILIO_ACCOUNT_SID", "")
	authToken := getEnv("TWILIO_AUTH_TOKEN", "")
	from := getEnv("TWILIO_FROM", "")
	if accountSID == "" || authToken == "" || from == "" {
		// dev fallback: print to stdout
		fmt.Printf("[twilio] %s -> %s\n", req.Title, req.Recipient)
		return nil
	}
	body, _ := json.Marshal(map[string]string{
		"From": from, "To": req.Recipient, "Body": req.Message,
	})
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json",
		accountSID)
	r, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	r.SetBasicAuth(accountSID, authToken)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func sendWebPush(ctx context.Context, req notificationRequest) error {
	muS.RLock()
	list := subs[req.UserID]
	muS.RUnlock()
	for _, s := range list {
		if s.Type != "web" {
			continue
		}
		endpoint := getEnv("PUSH_SERVICE_URL", "http://push-worker:9000/send")
		body, _ := json.Marshal(map[string]interface{}{
			"keys":          s.Keys,
			"title":         req.Title,
			"body":          req.Message,
			"data":          req.Data,
		})
		r, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			logger.Warn(ctx, "web push failed", zap.Error(err))
			continue
		}
		resp.Body.Close()
	}
	return nil
}

func sendFCM(ctx context.Context, req notificationRequest) error {
	muS.RLock()
	list := subs[req.UserID]
	muS.RUnlock()
	token := ""
	for _, s := range list {
		if s.Type == "fcm" {
			token = s.Keys["token"]
		}
	}
	if token == "" {
		return nil // silently skip
	}
	endpoint := getEnv("FCM_URL",
		"https://fcm.googleapis.com/fcm/send")
	body, _ := json.Marshal(map[string]interface{}{
		"to":           token,
		"notification": map[string]string{"title": req.Title, "body": req.Message},
		"data":         req.Data,
	})
	r, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	r.Header.Set("Authorization", "key="+getEnv("FCM_SERVER_KEY", ""))
	r.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fcm %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func sendSlack(_ context.Context, req notificationRequest) error {
	webhook := req.Recipient
	if webhook == "" {
		webhook = getEnv("SLACK_DEFAULT_WEBHOOK", "")
	}
	if webhook == "" {
		return fmt.Errorf("slack webhook url required")
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"text": fmt.Sprintf("*%s*\n%s", req.Title, req.Message),
	})
	r, _ := http.NewRequest("POST", webhook, bytes.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func sendDiscord(_ context.Context, req notificationRequest) error {
	webhook := req.Recipient
	if webhook == "" {
		return fmt.Errorf("discord webhook url required")
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"content": fmt.Sprintf("**%s**\n%s", req.Title, req.Message),
	})
	r, _ := http.NewRequest("POST", webhook, bytes.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func sendTelegram(ctx context.Context, req notificationRequest) error {
	chatID := req.Recipient
	if chatID == "" {
		return fmt.Errorf("telegram chat_id required")
	}
	token := getEnv("TELEGRAM_BOT_TOKEN", "")
	if token == "" {
		return fmt.Errorf("telegram token not set")
	}
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload, _ := json.Marshal(map[string]interface{}{
		"chat_id": chatID, "text": fmt.Sprintf("*%s*\n%s", req.Title, req.Message),
		"parse_mode": "Markdown",
	})
	r, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func sendWebhook(_ context.Context, req notificationRequest) error {
	url := req.Recipient
	if url == "" {
		return fmt.Errorf("webhook url required")
	}
	body, _ := json.Marshal(map[string]interface{}{
		"user_id": req.UserID, "type": req.Type, "title": req.Title,
		"message": req.Message, "data": req.Data,
	})
	r, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func readyHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func strconvDefault(v string, def int) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
