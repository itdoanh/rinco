// Package driver provides the email sending abstraction used by the email
// service.  The Driver interface is intentionally small (Send / Close) so
// that new providers can be plugged in without touching handlers.
//
// Supported backends:
//   - console -- prints to stdout (default for dev/test).
//   - smtp    -- uses net.SMTP for delivery (no third-party deps).
//   - resend  -- HTTPS POST to api.resend.com.
//   - sendgrid -- HTTPS POST to api.sendgrid.com/v3/mail/send.
//   - ses     -- minimal SigV4-signed POST to email.{region}.amazonaws.com.
package driver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/rinco/services/email-service/internal/platform"
)

// Message is the cross-driver input envelope.
type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	BodyType    string // text | html | markdown
	Headers     map[string]string
	Attachments []Attachment
	ReplyTo     string
}

// Attachment is a file attached to an outbound message.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Result describes what happened during a Send call.
type Result struct {
	ProviderID string
	Driver     string
	SentAt     time.Time
}

// Driver is the contract every backend must implement.
type Driver interface {
	Name() string
	Send(ctx context.Context, msg Message) (Result, error)
	Close() error
}

// New constructs a Driver based on cfg.Driver.
func New(cfg *platform.Config) (Driver, error) {
	switch strings.ToLower(cfg.Driver) {
	case "", "console":
		return NewConsole(), nil
	case "smtp":
		return NewSMTP(cfg)
	case "resend":
		return NewResend(cfg)
	case "sendgrid":
		return NewSendGrid(cfg)
	case "ses":
		return NewSES(cfg)
	default:
		return nil, fmt.Errorf("unknown driver %q", cfg.Driver)
	}
}

// =============================================================================
// Console
// =============================================================================

// Console driver logs the message to stdout. Useful for development.
type Console struct{}

func NewConsole() *Console { return &Console{} }
func (Console) Name() string { return "console" }
func (Console) Close() error { return nil }

func (Console) Send(_ context.Context, msg Message) (Result, error) {
	btype := msg.BodyType
	if btype == "" {
		btype = "text"
	}
	preview := msg.Body
	if len(preview) > 200 {
		preview = preview[:200] + "...[truncated]"
	}
	slog.Info("email (console)",
		slog.String("from", msg.From),
		slog.Any("to", msg.To),
		slog.String("subject", msg.Subject),
		slog.String("body_type", btype),
		slog.String("preview", preview),
	)
	return Result{ProviderID: fmt.Sprintf("console-%d", time.Now().UnixNano()), Driver: "console", SentAt: time.Now()}, nil
}

// =============================================================================
// SMTP
// =============================================================================

// SMTP driver uses Go's net/smtp to send mail.  Attachments are not
// supported here (the API only handles plain text + HTML bodies).
type SMTP struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewSMTP(cfg *platform.Config) (*SMTP, error) {
	if cfg.SMTPHost == "" {
		return nil, errors.New("smtp host not configured")
	}
	return &SMTP{
		host: cfg.SMTPHost, port: cfg.SMTPPort,
		user: cfg.SMTPUser, password: cfg.SMTPPass, from: cfg.FromDefault,
	}, nil
}

func (s *SMTP) Name() string { return "smtp" }
func (s *SMTP) Close() error { return nil }

func (s *SMTP) Send(_ context.Context, msg Message) (Result, error) {
	addr := net.JoinHostPort(s.host, fmt.Sprintf("%d", s.port))
	from := msg.From
	if from == "" {
		from = s.from
	}
	rcpts := append([]string{}, msg.To...)
	rcpts = append(rcpts, msg.Cc...)
	rcpts = append(rcpts, msg.Bcc...)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	b.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.Cc) > 0 {
		b.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.Cc, ", ")))
	}
	if msg.ReplyTo != "" {
		b.WriteString(fmt.Sprintf("Reply-To: %s\r\n", msg.ReplyTo))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	for k, v := range msg.Headers {
		b.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	switch msg.BodyType {
	case "html":
		b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n")
	default:
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}
	b.WriteString("\r\n")
	b.WriteString(msg.Body)

	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}
	if err := smtp.SendMail(addr, auth, from, rcpts, []byte(b.String())); err != nil {
		return Result{}, fmt.Errorf("smtp send: %w", err)
	}
	return Result{ProviderID: fmt.Sprintf("smtp-%d", time.Now().UnixNano()), Driver: "smtp", SentAt: time.Now()}, nil
}

// =============================================================================
// Resend
// =============================================================================

// Resend driver posts to https://api.resend.com/emails.
type Resend struct {
	apiKey string
	from   string
	client *http.Client
}

func NewResend(cfg *platform.Config) (*Resend, error) {
	if cfg.ResendAPIKey == "" {
		return nil, errors.New("resend api key not configured")
	}
	return &Resend{
		apiKey: cfg.ResendAPIKey,
		from:   cfg.FromDefault,
		client: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (r *Resend) Name() string { return "resend" }
func (r *Resend) Close() error { return nil }

func (r *Resend) Send(ctx context.Context, msg Message) (Result, error) {
	from := msg.From
	if from == "" {
		from = r.from
	}
	payload := map[string]any{
		"from":    from,
		"to":      msg.To,
		"subject": msg.Subject,
	}
	if len(msg.Cc) > 0 {
		payload["cc"] = msg.Cc
	}
	if len(msg.Bcc) > 0 {
		payload["bcc"] = msg.Bcc
	}
	if msg.ReplyTo != "" {
		payload["reply_to"] = []string{msg.ReplyTo}
	}
	switch msg.BodyType {
	case "html":
		payload["html"] = msg.Body
	default:
		payload["text"] = msg.Body
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("resend http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("resend %d: %s", resp.StatusCode, string(buf))
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return Result{ProviderID: out.ID, Driver: "resend", SentAt: time.Now()}, nil
}

// =============================================================================
// SendGrid
// =============================================================================

// SendGrid driver posts to https://api.sendgrid.com/v3/mail/send.
type SendGrid struct {
	apiKey string
	from   string
	client *http.Client
}

func NewSendGrid(cfg *platform.Config) (*SendGrid, error) {
	if cfg.SendGridAPIKey == "" {
		return nil, errors.New("sendgrid api key not configured")
	}
	return &SendGrid{
		apiKey: cfg.SendGridAPIKey,
		from:   cfg.FromDefault,
		client: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func (s *SendGrid) Name() string { return "sendgrid" }
func (s *SendGrid) Close() error { return nil }
func (s *SendGrid) Send(ctx context.Context, msg Message) (Result, error) {
	from := msg.From
	if from == "" {
		from = s.from
	}
	content := []map[string]string{}
	switch msg.BodyType {
	case "html":
		content = append(content, map[string]string{"type": "text/html", "value": msg.Body})
	default:
		content = append(content, map[string]string{"type": "text/plain", "value": msg.Body})
	}
	payload := map[string]any{
		"personalizations": []map[string]any{
			{"to": toPersonalizations(msg.To), "subject": msg.Subject},
		},
		"from":     map[string]string{"email": from},
		"content":  content,
	}
	if len(msg.Cc) > 0 {
		payload["personalizations"].([]map[string]any)[0]["cc"] = toPersonalizations(msg.Cc)
	}
	if len(msg.Bcc) > 0 {
		payload["personalizations"].([]map[string]any)[0]["bcc"] = toPersonalizations(msg.Bcc)
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("sendgrid http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("sendgrid %d: %s", resp.StatusCode, string(buf))
	}
	return Result{ProviderID: resp.Header.Get("X-Message-Id"), Driver: "sendgrid", SentAt: time.Now()}, nil
}

func toPersonalizations(addrs []string) []map[string]string {
	out := make([]map[string]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, map[string]string{"email": a})
	}
	return out
}

// =============================================================================
// AWS SES (minimal SigV4)
// =============================================================================

// SES driver signs a POST against AWS SES using SigV4.  It avoids the
// large aws-sdk-go-v2 dependency.
type SES struct {
	region      string
	accessKey   string
	secretKey   string
	from        string
	client      *http.Client
	serviceHost string
}

func NewSES(cfg *platform.Config) (*SES, error) {
	if cfg.AWSRegion == "" || cfg.AWSAccessKey == "" || cfg.AWSSecretKey == "" {
		return nil, errors.New("ses requires AWS_REGION + AWS_ACCESS_KEY + AWS_SECRET")
	}
	return &SES{
		region:      cfg.AWSRegion,
		accessKey:   cfg.AWSAccessKey,
		secretKey:   cfg.AWSSecretKey,
		from:        cfg.FromDefault,
		client:      &http.Client{Timeout: 15 * time.Second},
		serviceHost: fmt.Sprintf("email.%s.amazonaws.com", cfg.AWSRegion),
	}, nil
}

func (s *SES) Name() string { return "ses" }
func (s *SES) Close() error { return nil }

func (s *SES) Send(ctx context.Context, msg Message) (Result, error) {
	bodyType := "Text"
	content := msg.Body
	if msg.BodyType == "html" {
		bodyType = "Html"
	}
	payload := map[string]any{
		"Source":      firstNonEmpty(msg.From, s.from),
		"Destination": map[string]any{"ToAddresses": msg.To, "CcAddresses": msg.Cc, "BccAddresses": msg.Bcc},
		"Message": map[string]any{
			"Subject": map[string]any{"Data": msg.Subject, "Charset": "UTF-8"},
			"Body":    map[string]any{bodyType: map[string]any{"Data": content, "Charset": "UTF-8"}},
		},
	}
	body, _ := json.Marshal(payload)
	now := time.Now().UTC()
	host := s.serviceHost
	url := "https://" + host + "/"
	headers := map[string]string{
		"host":                 host,
		"x-amz-date":           now.Format("20060102T150405Z"),
		"content-type":         "application/x-amz-json-1.1",
		"x-amz-target":         "sesv2.SendEmail",
		"x-amz-content-sha256": sha256Hex(body),
	}
	auth := s.sign("POST", host, "/", headers, body, now)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", auth)
	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("ses http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return Result{}, fmt.Errorf("ses %d: %s", resp.StatusCode, string(buf))
	}
	var out struct {
		MessageID string `json:"MessageId"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return Result{ProviderID: out.MessageID, Driver: "ses", SentAt: time.Now()}, nil
}

// sign returns the Authorization header value for a SigV4 signed request.
func (s *SES) sign(method, host, path string, headers map[string]string, body []byte, t time.Time) string {
	const service = "ses"
	region := s.region
	amzdate := t.Format("20060102T150405Z")
	datestamp := t.Format("20060102")

	canonicalHeaders := ""
	signedHeaders := ""
	headerKeys := []string{"content-type", "host", "x-amz-date", "x-amz-target", "x-amz-content-sha256"}
	for _, k := range headerKeys {
		canonicalHeaders += k + ":" + strings.TrimSpace(headers[k]) + "\n"
		signedHeaders += k + ";"
	}
	signedHeaders = strings.TrimSuffix(signedHeaders, ";")
	canonicalRequest := method + "\n" + path + "\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + sha256Hex(body)

	credentialScope := datestamp + "/" + region + "/" + service + "/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzdate + "\n" + credentialScope + "\n" + sha256Hex([]byte(canonicalRequest))

	kDate := hmacSHA256([]byte("AWS4"+s.secretKey), datestamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	return "AWS4-HMAC-SHA256 Credential=" + s.accessKey + "/" + credentialScope +
		", SignedHeaders=" + signedHeaders + ", Signature=" + signature
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// =============================================================================
// Helpers shared with handlers
// =============================================================================

// ToJSONString marshals a value as compact JSON, returning "[]" on error.
func ToJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// FromJSONString unmarshals into the supplied pointer, silently ignoring
// errors (handlers can decide whether to react to the result).
func FromJSONString(s string, out any) {
	if s == "" {
		return
	}
	_ = json.Unmarshal([]byte(s), out)
}

// Helper used by tests / debugging.
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}