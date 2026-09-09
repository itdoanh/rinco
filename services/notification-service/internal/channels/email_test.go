// Tests for notification-service email channel.
package channels

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEmailChannel_Name checks the channel's name.
func TestEmailChannel_Name(t *testing.T) {
	c := NewEmailChannel("http://example.com")
	if c.Name() != "email" {
		t.Errorf("expected 'email', got %s", c.Name())
	}
}

// TestEmailChannel_MissingEmail verifies the channel rejects when email is empty.
func TestEmailChannel_MissingEmail(t *testing.T) {
	c := NewEmailChannel("http://example.com")
	_, err := c.Send(context.Background(), Notification{})
	if err == nil {
		t.Error("expected error for missing email")
	}
}

// TestEmailChannel_Success confirms a 2xx response yields DeliveryResult=sent.
func TestEmailChannel_Success(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"msg-123"}`))
	}))
	defer srv.Close()

	c := NewEmailChannel(srv.URL)
	res, err := c.Send(context.Background(), Notification{
		Email: "user@example.com",
		Title: "Hi",
		Body:  "Hello world",
		Type:  "welcome",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "sent" {
		t.Errorf("expected status sent, got %s", res.Status)
	}
	if res.Provider != "email" {
		t.Errorf("expected provider email, got %s", res.Provider)
	}
	// Verify the body contains the email and subject.
	if !strings.Contains(receivedBody, "user@example.com") {
		t.Errorf("body missing recipient: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, "Hi") {
		t.Errorf("body missing subject: %s", receivedBody)
	}
}

// TestEmailChannel_HTTPError verifies non-2xx produces an error.
func TestEmailChannel_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad address"}`))
	}))
	defer srv.Close()

	c := NewEmailChannel(srv.URL)
	_, err := c.Send(context.Background(), Notification{
		Email: "user@example.com",
		Title: "Hi",
		Body:  "Hello",
	})
	if err == nil {
		t.Error("expected error from 400 response")
	}
}

// TestEmailChannel_ServerError verifies 5xx also produces an error.
func TestEmailChannel_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`internal error`))
	}))
	defer srv.Close()

	c := NewEmailChannel(srv.URL)
	_, err := c.Send(context.Background(), Notification{
		Email: "user@example.com",
		Title: "Hi",
		Body:  "Hello",
	})
	if err == nil {
		t.Error("expected error from 500 response")
	}
}

// TestEmailChannel_RequestPath verifies the request is sent to /v1/email/send.
func TestEmailChannel_RequestPath(t *testing.T) {
	var hitPath string
	var hitMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitPath = r.URL.Path
		hitMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewEmailChannel(srv.URL)
	_, _ = c.Send(context.Background(), Notification{
		Email: "user@example.com",
		Title: "Hi",
		Body:  "Hello",
	})
	if hitPath != "/v1/email/send" {
		t.Errorf("expected /v1/email/send, got %s", hitPath)
	}
	if hitMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", hitMethod)
	}
}

// TestEmailChannel_PayloadShape verifies the JSON payload structure.
func TestEmailChannel_PayloadShape(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewEmailChannel(srv.URL)
	_, _ = c.Send(context.Background(), Notification{
		Email: "user@example.com",
		Title: "Welcome",
		Body:  "Welcome body",
		Type:  "signup",
	})
	// Verify required JSON keys are present.
	for _, key := range []string{`"to"`, `"subject"`, `"body"`, `"body_type"`, `"priority"`, `"headers"`} {
		if !strings.Contains(receivedBody, key) {
			t.Errorf("payload missing key %s in: %s", key, receivedBody)
		}
	}
	// Verify body_type is html and priority is normal.
	if !strings.Contains(receivedBody, `"html"`) {
		t.Errorf("body_type should be html: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, `"normal"`) {
		t.Errorf("priority should be normal: %s", receivedBody)
	}
}

// TestEmailChannel_NilData verifies the channel works with nil Data.
func TestEmailChannel_NilData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewEmailChannel(srv.URL)
	res, err := c.Send(context.Background(), Notification{
		Email: "u@example.com",
		Title: "x",
		Body:  "y",
		Data:  nil,
	})
	if err != nil {
		t.Errorf("unexpected error with nil data: %v", err)
	}
	if res.Status != "sent" {
		t.Errorf("expected sent, got %s", res.Status)
	}
}

// TestEmailChannel_ConnectionRefused verifies behavior when baseURL is unreachable.
func TestEmailChannel_ConnectionRefused(t *testing.T) {
	c := NewEmailChannel("http://127.0.0.1:1") // unreachable port
	_, err := c.Send(context.Background(), Notification{
		Email: "u@example.com",
		Title: "x",
		Body:  "y",
	})
	if err == nil {
		t.Error("expected error from unreachable host")
	}
}

// TestEmailChannel_ContextCancellation verifies context is propagated.
func TestEmailChannel_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewEmailChannel(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before sending
	_, err := c.Send(ctx, Notification{
		Email: "u@example.com",
		Title: "x",
		Body:  "y",
	})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}
