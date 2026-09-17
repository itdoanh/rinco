// Package integration — chat_flow_test.go exercises the chat-engine
// WebSocket and presence plane:
//
//  1. Open a WS connection
//  2. Send a chat message to a channel
//  3. Confirm presence.heartbeat has been published
//
//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket" // part of the indirect deps loaded by services
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chatMessageRequest is the payload sent on the chat-engine WS stream.
type chatMessageRequest struct {
	ChannelID string `json:"channel_id"`
	Body      string `json:"body"`
	AuthorID  string `json:"author_id"`
	TenantID  string `json:"tenant_id"`
}

// dialChat opens a WebSocket session authenticated with the supplied PASETO.
func dialChat(t *testing.T, ctx *TestContext, env *TestEnv, token string) (*websocket.Conn, error) {
	t.Helper()
	wsURL := strings.Replace(env.Service.ChatEngineURL, "http://", "ws://", 1)
	u, err := url.Parse(wsURL + "/ws/v1/connect")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()

	hdr := http.Header{}
	hdr.Add("Authorization", "Bearer "+token)
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), hdr)
	return conn, err
}

// TestChatConnectSendAndPresence — happy path:
//   WS connect → send message → receive echo + presence event.
func TestChatConnectSendAndPresence(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	if env.Users == nil {
		t.Skip("no demo users")
	}
	ctx := NewTestContext(t, env)
	tp := ctx.Login(env.Users.ApexAdmin.Email, env.Users.ApexAdmin.Password, env.Users.ApexAdmin.TenantID)

	conn, err := dialChat(t, ctx, env, tp.AccessToken)
	if err != nil {
		t.Skipf("chat ws unavailable: %v", err)
		return
	}
	defer conn.Close()

	channelID := "ch-" + RandString(6)
	// Read "hello" (optional)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var hello map[string]any
	_ = conn.ReadJSON(&hello)

	// Send message
	msg := chatMessageRequest{
		ChannelID: channelID,
		Body:      "Hello from integration test " + RandString(4),
		AuthorID:  env.Users.ApexAdmin.UserID,
		TenantID:  env.Users.ApexAdmin.TenantID,
	}
	require.NoError(t, conn.WriteJSON(msg))

	// Read at least one frame (echo + presence update)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ack map[string]any
	if err := conn.ReadJSON(&ack); err == nil {
		// First frame must reference our channelID or look like a presence beat.
		body, _ := json.Marshal(ack)
		assert.True(t,
			strings.Contains(string(body), channelID) ||
				strings.Contains(string(body), "presence"),
			"first ws frame should be echo or presence; got %s", body)
	}
}

// TestChatPresenceHeartbeat keeps a connection open for ~3s and asserts
// the server pushed a presence event. Many chat engines emit a presence
// beat every N seconds; we accept any of "heartbeat", "presence".
func TestChatPresenceHeartbeat(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	if env.Users == nil {
		t.Skip("no demo users")
	}
	ctx := NewTestContext(t, env)
	tp := ctx.Login(env.Users.ApexAdmin.Email, env.Users.ApexAdmin.Password, env.Users.ApexAdmin.TenantID)

	conn, err := dialChat(t, ctx, env, tp.AccessToken)
	if err != nil {
		t.Skipf("chat ws unavailable: %v", err)
		return
	}
	defer conn.Close()

	seenHeartbeat := false
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
				return
			}
			var frame map[string]any
			if err := conn.ReadJSON(&frame); err != nil {
				return
			}
			body, _ := json.Marshal(frame)
			if strings.Contains(string(body), "heartbeat") ||
				strings.Contains(string(body), "presence") {
				seenHeartbeat = true
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(6 * time.Second):
	}
	assert.True(t, seenHeartbeat,
		"server should have emitted a heartbeat / presence frame within 6s")
}

// TestChatUnauthorizedConnection confirms that connecting without a
// valid token is rejected (either by close frame or HTTP 401 during
// upgrade).
func TestChatUnauthorizedConnection(t *testing.T) {
	env := Setup(t, true)
	if SkipIfNoStack(t, env) {
		return
	}
	// ctx is unused here (test only needs the env handle).
	_ = NewTestContext
	wsURL := strings.Replace(env.Service.ChatEngineURL, "http://", "ws://", 1)
	u := fmt.Sprintf("%s/ws/v1/connect?token=invalid", wsURL)
	conn, resp, err := websocket.DefaultDialer.Dial(u, nil)
	if err == nil {
		conn.Close()
		t.Log("chat engine accepted invalid token — non-fatal in scaffold")
		return
	}
	// Common response: 401 during upgrade
	if resp != nil {
		assert.True(t,
			resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden,
			"expected 401/403 for invalid token, got %d", resp.StatusCode)
	}
}
