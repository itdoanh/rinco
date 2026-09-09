// HTTP-driven tests for the channels that talk to external services
// (Telegram Bot API, Slack/Discord webhooks).  Each test stands up an
// httptest server, points the channel at it via the existing
// ``client`` field, and asserts the request shape + delivery result.
package channels

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// captureHandler is a tiny httptest handler that records the last
// received request so individual tests can assert on its shape.
type captureHandler struct {
	mu        sync.Mutex
	method    string
	path      string
	header    http.Header
	body      []byte
	httpError int // when non-zero, respond with this status code
}

func (h *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.method = r.Method
	h.path = r.URL.Path
	h.header = r.Header.Clone()
	b, _ := io.ReadAll(r.Body)
	h.body = b
	if h.httpError != 0 {
		http.Error(w, "boom", h.httpError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// newInjectedClient returns an *http.Client that always talks to the
// provided test server.
func newInjectedClient(srv *httptest.Server) *http.Client {
	return &http.Client{Transport: &rewriteTransport{target: srv.URL}}
}

type rewriteTransport struct {
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect the request to the test server.
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = strings.TrimPrefix(t.target, "http://")
	return http.DefaultTransport.RoundTrip(req2)
}

// ----- Telegram -----

func TestTelegramChannel_Success(t *testing.T) {
	h := &captureHandler{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	c := &TelegramChannel{botToken: "test-bot", client: newInjectedClient(srv)}
	n := Notification{
		Title: "Hello",
		Body:  "World",
		Data:  map[string]any{"chat_id": "42"},
	}
	res, err := c.Send(context.Background(), n)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.Status != "sent" {
		t.Errorf("Status: got %s", res.Status)
	}
	if res.Provider != "telegram" {
		t.Errorf("Provider: got %s", res.Provider)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.method != http.MethodPost {
		t.Errorf("method: got %s", h.method)
	}
	if !strings.HasSuffix(h.path, "/sendMessage") {
		t.Errorf("path: got %s", h.path)
	}
	if h.header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type: got %s", h.header.Get("Content-Type"))
	}
	var payload tgSendMessageReq
	if err := json.Unmarshal(h.body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if payload.ChatID != "42" {
		t.Errorf("ChatID: got %q", payload.ChatID)
	}
	if !strings.Contains(payload.Text, "Hello") || !strings.Contains(payload.Text, "World") {
		t.Errorf("text missing title/body: %q", payload.Text)
	}
	if payload.ParseMode != "Markdown" {
		t.Errorf("ParseMode: got %s", payload.ParseMode)
	}
}

func TestTelegramChannel_HTTPError(t *testing.T) {
	h := &captureHandler{httpError: http.StatusBadRequest}
	srv := httptest.NewServer(h)
	defer srv.Close()
	c := &TelegramChannel{botToken: "test-bot", client: newInjectedClient(srv)}
	_, err := c.Send(context.Background(), Notification{Data: map[string]any{"chat_id": "1"}})
	if err == nil {
		t.Fatal("expected error on 400")
	}
	if !strings.Contains(err.Error(), "telegram 400") {
		t.Errorf("error: %v", err)
	}
}

func TestTelegramChannel_Name(t *testing.T) {
	c := NewTelegramChannel("")
	if c.Name() != "telegram" {
		t.Errorf("Name: got %s", c.Name())
	}
}

// ----- Slack -----

func TestSlackChannel_Success(t *testing.T) {
	h := &captureHandler{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	c := &SlackChannel{client: newInjectedClient(srv)}
	n := Notification{
		Title: "Title",
		Body:  "Body",
		Data:  map[string]any{"webhook_url": srv.URL + "/webhook"},
	}
	res, err := c.Send(context.Background(), n)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.Status != "sent" {
		t.Errorf("Status: got %s", res.Status)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.method != http.MethodPost {
		t.Errorf("method: got %s", h.method)
	}
	var payload slackPayload
	if err := json.Unmarshal(h.body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if !strings.Contains(payload.Text, "Title") {
		t.Errorf("text: got %q", payload.Text)
	}
	if len(payload.Attachments) != 1 || payload.Attachments[0].Color == "" {
		t.Errorf("expected one colored attachment")
	}
}

func TestSlackChannel_HTTPError(t *testing.T) {
	h := &captureHandler{httpError: http.StatusInternalServerError}
	srv := httptest.NewServer(h)
	defer srv.Close()
	c := &SlackChannel{client: newInjectedClient(srv)}
	_, err := c.Send(context.Background(), Notification{Data: map[string]any{"webhook_url": srv.URL}})
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if !strings.Contains(err.Error(), "slack 500") {
		t.Errorf("error: %v", err)
	}
}

func TestSlackChannel_Name(t *testing.T) {
	if NewSlackChannel("").Name() != "slack" {
		t.Errorf("Name")
	}
}

// ----- Discord -----

func TestDiscordChannel_Success(t *testing.T) {
	h := &captureHandler{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	c := &DiscordChannel{client: newInjectedClient(srv)}
	n := Notification{
		Title: "T",
		Body:  "B",
		Data:  map[string]any{"webhook_url": srv.URL + "/dc"},
	}
	res, err := c.Send(context.Background(), n)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.Status != "sent" {
		t.Errorf("Status: got %s", res.Status)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var payload discordPayload
	if err := json.Unmarshal(h.body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if !strings.Contains(payload.Content, "T") {
		t.Errorf("content: got %q", payload.Content)
	}
	if len(payload.Embeds) != 1 {
		t.Errorf("expected 1 embed, got %d", len(payload.Embeds))
	}
}

func TestDiscordChannel_HTTPError(t *testing.T) {
	h := &captureHandler{httpError: http.StatusTooManyRequests}
	srv := httptest.NewServer(h)
	defer srv.Close()
	c := &DiscordChannel{client: newInjectedClient(srv)}
	_, err := c.Send(context.Background(), Notification{Data: map[string]any{"webhook_url": srv.URL}})
	if err == nil {
		t.Fatal("expected error on 429")
	}
	if !strings.Contains(err.Error(), "discord 429") {
		t.Errorf("error: %v", err)
	}
}

func TestDiscordChannel_Name(t *testing.T) {
	if NewDiscordChannel("").Name() != "discord" {
		t.Errorf("Name")
	}
}

// ----- InApp NATS happy path with stub -----

func TestInAppChannel_PrefixSubject(t *testing.T) {
	// Verify the constructed subject has the right shape.
	c := NewInAppChannel(nil)
	if c.prefix != "user." {
		t.Errorf("prefix: got %s", c.prefix)
	}
	// Prefix is package-private so we just assert construction; full
	// subject verification would need a fake NATS connection.
}
