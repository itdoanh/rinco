// Extra tests for email-service driver helpers.
package driver

import (
	"strings"
	"testing"
	"time"

	"github.com/rinco/services/email-service/internal/platform"
)

// platformCfg returns a default *platform.Config.
func platformCfg() *platform.Config {
	return &platform.Config{}
}

func TestExtraSha256Hex_KnownVector(t *testing.T) {
	// SHA-256 of "" = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	got := sha256Hex([]byte(""))
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtraSha256Hex_Different(t *testing.T) {
	a := sha256Hex([]byte("a"))
	b := sha256Hex([]byte("b"))
	if a == b {
		t.Error("different inputs should hash differently")
	}
}

func TestExtraHmacSHA256_Deterministic(t *testing.T) {
	a := hmacSHA256([]byte("key"), "data")
	b := hmacSHA256([]byte("key"), "data")
	if string(a) != string(b) {
		t.Error("HMAC should be deterministic")
	}
}

func TestExtraHmacSHA256_DifferentKeys(t *testing.T) {
	a := hmacSHA256([]byte("key1"), "data")
	b := hmacSHA256([]byte("key2"), "data")
	if string(a) == string(b) {
		t.Error("different keys should produce different HMAC")
	}
}

func TestExtraFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "x"); got != "x" {
		t.Errorf("got %q, want x", got)
	}
}

func TestExtraFirstNonEmpty_Empty(t *testing.T) {
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestExtraFirstNonEmpty_NonEmpty(t *testing.T) {
	if got := firstNonEmpty("first", "second"); got != "first" {
		t.Errorf("got %q, want first", got)
	}
}

func TestExtraToJSONString_OK(t *testing.T) {
	if got := ToJSONString(map[string]int{"a": 1}); !strings.Contains(got, `"a":1`) {
		t.Errorf("unexpected: %s", got)
	}
}

func TestExtraToJSONString_BadValue(t *testing.T) {
	// channels can't be JSON marshaled
	ch := make(chan int)
	if got := ToJSONString(ch); got != "[]" {
		t.Errorf("got %q, want []", got)
	}
}

func TestExtraFromJSONString_Empty(t *testing.T) {
	type target struct{ A int }
	var x target
	FromJSONString("", &x) // should not panic
	if x.A != 0 {
		t.Error("should not modify on empty")
	}
}

func TestExtraFromJSONString_Valid(t *testing.T) {
	type target struct {
		A int `json:"a"`
	}
	var x target
	FromJSONString(`{"a": 5}`, &x)
	if x.A != 5 {
		t.Errorf("got %d, want 5", x.A)
	}
}

func TestExtraFromJSONString_InvalidJSON(t *testing.T) {
	type target struct {
		A int `json:"a"`
	}
	var x target
	FromJSONString(`not json`, &x) // should not panic
}

func TestExtraConsole_Name(t *testing.T) {
	if (Console{}).Name() != "console" {
		t.Error("Name mismatch")
	}
}

func TestExtraConsole_Close(t *testing.T) {
	if err := (Console{}).Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestExtraSES_Name(t *testing.T) {
	s := &SES{}
	if s.Name() != "ses" {
		t.Error("Name mismatch")
	}
}

func TestExtraSES_Close(t *testing.T) {
	s := &SES{}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestExtraSES_Sign_Format(t *testing.T) {
	s := &SES{
		region:    "us-east-1",
		accessKey: "AKIAIOSFODNN7EXAMPLE",
		secretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	body := []byte(`{"foo":"bar"}`)
	headers := map[string]string{
		"host":                 "email.us-east-1.amazonaws.com",
		"x-amz-date":           "20240101T000000Z",
		"content-type":         "application/x-amz-json-1.1",
		"x-amz-target":         "sesv2.SendEmail",
		"x-amz-content-sha256": sha256Hex(body),
	}
	sig := s.sign("POST", "email.us-east-1.amazonaws.com", "/", headers, body, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	if !strings.HasPrefix(sig, "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ses/aws4_request") {
		t.Errorf("SigV4 prefix mismatch")
	}
	if !strings.Contains(sig, "SignedHeaders=") {
		t.Errorf("SignedHeaders missing")
	}
	if !strings.Contains(sig, "Signature=") {
		t.Errorf("Signature missing")
	}
}

func TestExtraSES_Sign_Deterministic(t *testing.T) {
	s := &SES{region: "us-east-1", accessKey: "K1", secretKey: "secret"}
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	body := []byte("body")
	headers := map[string]string{
		"host":                 "h",
		"x-amz-date":           "20240101T000000Z",
		"content-type":         "ct",
		"x-amz-target":         "tgt",
		"x-amz-content-sha256": sha256Hex(body),
	}
	a := s.sign("POST", "h", "/", headers, body, t1)
	b := s.sign("POST", "h", "/", headers, body, t1)
	if a != b {
		t.Error("SigV4 should be deterministic for same inputs")
	}
}

func TestExtraNew_Console(t *testing.T) {
	d, err := New(platformCfg())
	if err != nil {
		t.Fatalf("New console: %v", err)
	}
	if d.Name() != "console" {
		t.Errorf("Name = %s", d.Name())
	}
}

func TestExtraNew_Unknown(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "unknown"
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for unknown driver")
	}
}

func TestExtraNew_SMTP_NoHost(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "smtp"
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for missing SMTP host")
	}
}

func TestExtraNew_Resend_NoKey(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "resend"
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for missing resend key")
	}
}

func TestExtraNew_SendGrid_NoKey(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "sendgrid"
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for missing sendgrid key")
	}
}

func TestExtraNew_SES_NoCreds(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "ses"
	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for missing SES creds")
	}
}

func TestExtraNew_SES_OK(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "ses"
	cfg.AWSRegion = "us-east-1"
	cfg.AWSAccessKey = "AKIA"
	cfg.AWSSecretKey = "secret"
	d, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "ses" {
		t.Errorf("Name = %s", d.Name())
	}
}

func TestExtraNew_SMTP_OK(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "smtp"
	cfg.SMTPHost = "smtp.example.com"
	cfg.SMTPPort = 587
	d, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "smtp" {
		t.Errorf("Name = %s", d.Name())
	}
}

func TestExtraNew_Resend_OK(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "resend"
	cfg.ResendAPIKey = "re_test"
	d, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "resend" {
		t.Errorf("Name = %s", d.Name())
	}
}

func TestExtraNew_SendGrid_OK(t *testing.T) {
	cfg := platformCfg()
	cfg.Driver = "sendgrid"
	cfg.SendGridAPIKey = "SG.test"
	d, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "sendgrid" {
		t.Errorf("Name = %s", d.Name())
	}
}
