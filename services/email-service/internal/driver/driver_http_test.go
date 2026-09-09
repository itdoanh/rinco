// HTTP-driven tests for the email-service drivers that talk to
// external APIs (Resend, SendGrid, AWS SES).  Each test points the
// driver's ``client`` at a local httptest server via a RoundTripper
// rewrite so we can validate request shape, headers, and error paths
// without leaving the test machine.
package driver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// captureRecorder records the last request received by the test server.
type captureRecorder struct {
	mu     sync.Mutex
	method string
	path   string
	header http.Header
	body   []byte
	status int
}

func (r *captureRecorder) record(req *http.Request, body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.method = req.Method
	r.path = req.URL.Path
	r.header = req.Header.Clone()
	r.body = body
}

func (r *captureRecorder) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		r.record(req, b)
		status := r.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		switch status {
		case http.StatusOK:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"provider-msg-id","MessageId":"ses-msg"}`))
		default:
			_, _ = w.Write([]byte(`{"error":"provider rejected"}`))
		}
	})
}

// rewriteTransport rewrites every request URL to point at ``target``.
// Used to point external API calls at a local httptest server.
type rewriteTransport struct{ target string }

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = strings.TrimPrefix(t.target, "http://")
	return http.DefaultTransport.RoundTrip(req2)
}

// =============================================================================
// Resend
// =============================================================================

func TestResend_Send_TextBody(t *testing.T) {
	rec := &captureRecorder{}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()

	r := &Resend{
		apiKey: "test-key",
		from:   "noreply@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
	}
	res, err := r.Send(context.Background(), Message{
		To:      []string{"alice@example.com"},
		Subject: "Hi",
		Body:    "Plain text body",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.Driver != "resend" {
		t.Errorf("Driver: %s", res.Driver)
	}
	if res.ProviderID != "provider-msg-id" {
		t.Errorf("ProviderID: %s", res.ProviderID)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.method != http.MethodPost {
		t.Errorf("method: %s", rec.method)
	}
	if rec.path != "/emails" {
		t.Errorf("path: %s", rec.path)
	}
	if got := rec.header.Get("Authorization"); got != "Bearer test-key" {
		t.Errorf("Authorization: %s", got)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.body, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["from"] != "noreply@example.com" {
		t.Errorf("from: %v", payload["from"])
	}
	if payload["text"] != "Plain text body" {
		t.Errorf("text body: %v", payload["text"])
	}
	if _, ok := payload["html"]; ok {
		t.Errorf("html should be absent for text body")
	}
}

func TestResend_Send_HtmlBody(t *testing.T) {
	rec := &captureRecorder{}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()

	r := &Resend{
		apiKey: "k",
		from:   "f@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
	}
	_, err := r.Send(context.Background(), Message{
		To:      []string{"x@y.com"},
		Body:    "<p>HTML</p>",
		BodyType: "html",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	var payload map[string]any
	_ = json.Unmarshal(rec.body, &payload)
	if _, ok := payload["html"]; !ok {
		t.Errorf("expected html key in payload: %v", payload)
	}
}

func TestResend_Send_CcBccReplyTo(t *testing.T) {
	rec := &captureRecorder{}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()

	r := &Resend{
		apiKey: "k",
		from:   "f@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
	}
	_, err := r.Send(context.Background(), Message{
		To:      []string{"to@example.com"},
		Cc:      []string{"cc@example.com"},
		Bcc:     []string{"bcc@example.com"},
		ReplyTo: "reply@example.com",
		Body:    "x",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	var payload map[string]any
	_ = json.Unmarshal(rec.body, &payload)
	if _, ok := payload["cc"]; !ok {
		t.Errorf("missing cc: %v", payload)
	}
	if _, ok := payload["bcc"]; !ok {
		t.Errorf("missing bcc: %v", payload)
	}
	if rt, ok := payload["reply_to"].([]any); !ok || len(rt) == 0 {
		t.Errorf("missing reply_to: %v", payload["reply_to"])
	}
}

func TestResend_Send_HTTPError(t *testing.T) {
	rec := &captureRecorder{status: http.StatusUnprocessableEntity}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()
	r := &Resend{
		apiKey: "k",
		from:   "f@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
	}
	_, err := r.Send(context.Background(), Message{To: []string{"x@y.com"}, Body: "x"})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
	if !strings.Contains(err.Error(), "resend 422") {
		t.Errorf("error: %v", err)
	}
}

// =============================================================================
// SendGrid
// =============================================================================

func TestSendGrid_Send_TextBody(t *testing.T) {
	rec := &captureRecorder{}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()

	s := &SendGrid{
		apiKey: "sg-key",
		from:   "from@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
	}
	_, err := s.Send(context.Background(), Message{
		To:      []string{"to@example.com"},
		Subject: "Hi",
		Body:    "Hello",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.path != "/v3/mail/send" {
		t.Errorf("path: %s", rec.path)
	}
	if rec.header.Get("Authorization") != "Bearer sg-key" {
		t.Errorf("Authorization: %s", rec.header.Get("Authorization"))
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.body, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// personalizations[0].to[0].email
	pers, ok := payload["personalizations"].([]any)
	if !ok || len(pers) == 0 {
		t.Fatalf("personalizations missing: %v", payload)
	}
	first := pers[0].(map[string]any)
	tos, _ := first["to"].([]any)
	if len(tos) == 0 || tos[0].(map[string]any)["email"] != "to@example.com" {
		t.Errorf("to: %v", first["to"])
	}
	content, _ := payload["content"].([]any)
	if len(content) == 0 || content[0].(map[string]any)["type"] != "text/plain" {
		t.Errorf("content: %v", payload["content"])
	}
}

func TestSendGrid_Send_HtmlBody(t *testing.T) {
	rec := &captureRecorder{}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()
	s := &SendGrid{
		apiKey: "k",
		from:   "f@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
	}
	_, err := s.Send(context.Background(), Message{To: []string{"x@y.com"}, Body: "<h1>x</h1>", BodyType: "html"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	var payload map[string]any
	_ = json.Unmarshal(rec.body, &payload)
	content, _ := payload["content"].([]any)
	if content[0].(map[string]any)["type"] != "text/html" {
		t.Errorf("content type: %v", content)
	}
}

func TestSendGrid_Send_HTTPError(t *testing.T) {
	rec := &captureRecorder{status: http.StatusInternalServerError}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()
	s := &SendGrid{
		apiKey: "k",
		from:   "f@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
	}
	_, err := s.Send(context.Background(), Message{To: []string{"x@y.com"}, Body: "x"})
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "sendgrid 500") {
		t.Errorf("error: %v", err)
	}
}

// =============================================================================
// SES
// =============================================================================

func TestSES_Send_TextBody(t *testing.T) {
	rec := &captureRecorder{}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()

	s := &SES{
		region:      "us-east-1",
		accessKey:   "AKIA0000",
		secretKey:   "secret",
		from:        "f@example.com",
		client:      &http.Client{Transport: &rewriteTransport{target: srv.URL}},
		serviceHost: "email.us-east-1.amazonaws.com",
	}
	res, err := s.Send(context.Background(), Message{
		To:      []string{"alice@example.com"},
		Subject: "Hi",
		Body:    "Hello",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.ProviderID != "ses-msg" {
		t.Errorf("ProviderID: %s", res.ProviderID)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	// Verify SigV4 headers are set.
	if !strings.HasPrefix(rec.header.Get("Authorization"), "AWS4-HMAC-SHA256") {
		t.Errorf("Authorization: %s", rec.header.Get("Authorization"))
	}
	if rec.header.Get("X-Amz-Target") != "sesv2.SendEmail" {
		t.Errorf("X-Amz-Target: %s", rec.header.Get("X-Amz-Target"))
	}
	if rec.header.Get("Content-Type") != "application/x-amz-json-1.1" {
		t.Errorf("Content-Type: %s", rec.header.Get("Content-Type"))
	}
	if rec.header.Get("X-Amz-Content-Sha256") == "" {
		t.Error("missing X-Amz-Content-Sha256")
	}
	// Body should be the SES JSON envelope.
	var payload map[string]any
	if err := json.Unmarshal(rec.body, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["Source"] != "f@example.com" {
		t.Errorf("Source: %v", payload["Source"])
	}
}

func TestSES_Send_HtmlBody(t *testing.T) {
	rec := &captureRecorder{}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()
	s := &SES{
		region: "us-east-1",
		from:   "f@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
		serviceHost: "email.us-east-1.amazonaws.com",
	}
	_, err := s.Send(context.Background(), Message{To: []string{"x@y.com"}, Body: "<h1>x</h1>", BodyType: "html"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	var payload map[string]any
	_ = json.Unmarshal(rec.body, &payload)
	msg, _ := payload["Message"].(map[string]any)
	if _, ok := msg["Body"].(map[string]any)["Html"]; !ok {
		t.Errorf("expected Html body key: %v", msg)
	}
}

func TestSES_Send_HTTPError(t *testing.T) {
	rec := &captureRecorder{status: http.StatusForbidden}
	srv := httptest.NewServer(rec.Handler())
	defer srv.Close()
	s := &SES{
		region: "us-east-1",
		from:   "f@example.com",
		client: &http.Client{Transport: &rewriteTransport{target: srv.URL}},
		serviceHost: "email.us-east-1.amazonaws.com",
	}
	_, err := s.Send(context.Background(), Message{To: []string{"x@y.com"}, Body: "x"})
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !strings.Contains(err.Error(), "ses 403") {
		t.Errorf("error: %v", err)
	}
}

func TestSES_Sign_Deterministic(t *testing.T) {
	// Two signatures with the same inputs should produce identical output
	// (modulo timestamp).  This protects against accidental entropy in
	// the canonicalization step.
	s := &SES{
		region:    "us-east-1",
		accessKey: "AKIA0000",
		secretKey: "secret",
	}
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	headers := map[string]string{
		"host":                 "email.us-east-1.amazonaws.com",
		"x-amz-date":           t0.Format("20060102T150405Z"),
		"content-type":         "application/x-amz-json-1.1",
		"x-amz-target":         "sesv2.SendEmail",
		"x-amz-content-sha256": sha256Hex([]byte(`{"foo":1}`)),
	}
	sig1 := s.sign("POST", headers["host"], "/", headers, []byte(`{"foo":1}`), t0)
	sig2 := s.sign("POST", headers["host"], "/", headers, []byte(`{"foo":1}`), t0)
	if sig1 != sig2 {
		t.Errorf("signing is not deterministic:\n%s\n%s", sig1, sig2)
	}
	if !strings.Contains(sig1, "Credential=AKIA0000/20260101/us-east-1/ses/aws4_request") {
		t.Errorf("credential scope wrong: %s", sig1)
	}
}

// =============================================================================
// Helpers
// =============================================================================

func TestToJSONString(t *testing.T) {
	if ToJSONString([]int{1, 2, 3}) != "[1,2,3]" {
		t.Errorf("compact JSON: %s", ToJSONString([]int{1, 2, 3}))
	}
	// Error path returns "[]".
	if ToJSONString(make(chan int)) != "[]" {
		t.Errorf("expected [] on error, got %s", ToJSONString(make(chan int)))
	}
}

func TestFromJSONString_SilentOnInvalid(t *testing.T) {
	var out []string
	// Should not panic on invalid JSON.
	FromJSONString("{not-json", &out)
}

func TestFirstNonEmpty(t *testing.T) {
	if firstNonEmpty("", "x", "y") != "x" {
		t.Error("first non-empty should be x")
	}
	if firstNonEmpty("", "") != "" {
		t.Error("all empty should return empty")
	}
}

func TestSHA256Hex(t *testing.T) {
	// Known SHA-256 of empty string.
	if sha256Hex(nil) != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("sha256Hex(nil) returned unexpected value")
	}
}
