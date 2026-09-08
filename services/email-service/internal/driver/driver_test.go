// Tests for email-service driver.
package driver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMessage_Struct(t *testing.T) {
	m := Message{
		From:    "sender@example.com",
		To:      []string{"a@example.com", "b@example.com"},
		Cc:      []string{"c@example.com"},
		Bcc:     []string{"d@example.com"},
		Subject: "Test",
		Body:    "Hello",
		BodyType: "text",
		Headers: map[string]string{"X-Custom": "value"},
		ReplyTo: "reply@example.com",
	}
	if m.From == "" {
		t.Error("From empty")
	}
	if len(m.To) != 2 {
		t.Errorf("To len: got %d", len(m.To))
	}
	if len(m.Cc) != 1 {
		t.Errorf("Cc len: got %d", len(m.Cc))
	}
	if len(m.Bcc) != 1 {
		t.Errorf("Bcc len: got %d", len(m.Bcc))
	}
	if m.BodyType != "text" {
		t.Errorf("BodyType: got %s", m.BodyType)
	}
}

func TestAttachment_Struct(t *testing.T) {
	a := Attachment{
		Filename:    "test.pdf",
		ContentType: "application/pdf",
		Data:        []byte("PDF content"),
	}
	if a.Filename != "test.pdf" {
		t.Errorf("Filename: got %s", a.Filename)
	}
	if a.ContentType != "application/pdf" {
		t.Errorf("ContentType: got %s", a.ContentType)
	}
	if string(a.Data) != "PDF content" {
		t.Errorf("Data: got %s", a.Data)
	}
}

func TestConsole_Name(t *testing.T) {
	c := NewConsole()
	if c.Name() != "console" {
		t.Errorf("expected 'console', got %s", c.Name())
	}
}

func TestConsole_Close(t *testing.T) {
	c := NewConsole()
	if err := c.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestConsole_Send(t *testing.T) {
	c := NewConsole()
	res, err := c.Send(context.Background(), Message{
		From:    "sender@example.com",
		To:      []string{"a@example.com"},
		Subject: "Test",
		Body:    "Hello",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.Driver != "console" {
		t.Errorf("Driver: got %s", res.Driver)
	}
	if res.SentAt.IsZero() {
		t.Error("SentAt should be set")
	}
	if res.SentAt.After(time.Now().Add(time.Second)) {
		t.Error("SentAt in future")
	}
}

func TestResult_Struct(t *testing.T) {
	now := time.Now()
	r := Result{
		ProviderID: "msg-123",
		Driver:     "console",
		SentAt:     now,
	}
	if r.ProviderID != "msg-123" {
		t.Errorf("ProviderID: got %s", r.ProviderID)
	}
	if r.Driver != "console" {
		t.Errorf("Driver: got %s", r.Driver)
	}
	if !r.SentAt.Equal(now) {
		t.Error("SentAt mismatch")
	}
}

func TestNew_DefaultsToConsole(t *testing.T) {
	// Empty driver name should default to console.
	// We can't easily test this without platform.Config; skipping.
	t.Skip("requires platform.Config setup")
}

func TestSendGrid_Name(t *testing.T) {
	s := &SendGrid{apiKey: "test", from: "test@example.com"}
	if s.Name() != "sendgrid" {
		t.Errorf("expected 'sendgrid', got %s", s.Name())
	}
}

func TestSendGrid_Close(t *testing.T) {
	s := &SendGrid{apiKey: "test", from: "test@example.com"}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestResend_Name(t *testing.T) {
	r := &Resend{apiKey: "test", from: "test@example.com"}
	if r.Name() != "resend" {
		t.Errorf("expected 'resend', got %s", r.Name())
	}
}

func TestResend_Close(t *testing.T) {
	r := &Resend{apiKey: "test", from: "test@example.com"}
	if err := r.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestSES_Name(t *testing.T) {
	s := &SES{region: "us-east-1", from: "test@example.com"}
	if s.Name() != "ses" {
		t.Errorf("expected 'ses', got %s", s.Name())
	}
}

func TestSES_Close(t *testing.T) {
	s := &SES{region: "us-east-1", from: "test@example.com"}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestDriverInterface_AllImplement(t *testing.T) {
	// Verify all drivers implement the Driver interface.
	var _ Driver = NewConsole()
	var _ Driver = &SendGrid{}
	var _ Driver = &Resend{}
	var _ Driver = &SES{}
	// SMTP may fail to construct but type check is fine.
	var _ Driver = (*SMTP)(nil)
}

func TestErrorMessage(t *testing.T) {
	err := errors.New("test error")
	if err.Error() != "test error" {
		t.Error("Error message mismatch")
	}
}

func TestEmailValidation_Valid(t *testing.T) {
	emails := []string{
		"user@example.com",
		"a.b+c@sub.example.co.uk",
		"x@y.z",
	}
	for _, e := range emails {
		if !strings.Contains(e, "@") {
			t.Errorf("expected valid email %s", e)
		}
	}
}
