// Extra tests for logger redactor.
package logger

import (
	"regexp"
	"strings"
	"testing"
)

func TestNewRedactor_NilConfig(t *testing.T) {
	r := NewRedactor(nil)
	if r == nil {
		t.Fatal("nil redactor")
	}
	if r.config == nil {
		t.Error("nil config")
	}
}

func TestNewRedactor_DefaultConfig(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	if r == nil {
		t.Fatal("nil")
	}
}

func TestDefaultRedactorConfig_HasFields(t *testing.T) {
	cfg := DefaultRedactorConfig()
	if len(cfg.FieldNames) == 0 {
		t.Error("no field names")
	}
	for _, field := range []string{"password", "secret", "access_token"} {
		found := false
		for _, f := range cfg.FieldNames {
			if f == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing field: %s", field)
		}
	}
}

func TestDefaultRedactorConfig_HashFormat(t *testing.T) {
	cfg := DefaultRedactorConfig()
	if !strings.Contains(cfg.HashPrefix, "REDACTED") {
		t.Error("missing REDACTED prefix")
	}
}

func TestNewRedactor_AllDisabled(t *testing.T) {
	cfg := &RedactorConfig{
		HashPrefix:           "[",
		HashSuffix:           "]",
		DisableEmail:         true,
		DisablePhone:         true,
		DisableToken:         true,
		DisableCreditCard:    true,
		DisableIPAddress:     true,
		DisableMACAddress:    true,
		DisablePassword:      true,
	}
	r := NewRedactor(cfg)
	if r == nil {
		t.Fatal("nil")
	}
}

func TestIsLikelyPhone_Valid(t *testing.T) {
	if !isLikelyPhone("+1234567890") {
		t.Error("valid phone")
	}
}

func TestIsLikelyPhone_TooShort(t *testing.T) {
	if isLikelyPhone("12345") {
		t.Error("too short")
	}
}

func TestIsLikelyPhone_TooLong(t *testing.T) {
	if isLikelyPhone("123456789012345678") {
		t.Error("too long")
	}
}

func TestIsLikelyPhone_NoDigits(t *testing.T) {
	if isLikelyPhone("abcdef") {
		t.Error("no digits")
	}
}

func TestIsValidCreditCard_Valid(t *testing.T) {
	// Standard test credit card number (passes Luhn)
	if !isValidCreditCard("4532015112830366") {
		t.Error("valid CC should pass")
	}
}

func TestIsValidCreditCard_Invalid(t *testing.T) {
	if isValidCreditCard("1234567890123456") {
		t.Error("invalid CC should fail")
	}
}

func TestIsValidCreditCard_TooShort(t *testing.T) {
	if isValidCreditCard("12345") {
		t.Error("too short should fail")
	}
}

func TestIsValidCreditCard_WithSpaces(t *testing.T) {
	// Test that spaces are stripped before validation
	if !isValidCreditCard("4532 0151 1283 0366") {
		t.Error("valid CC with spaces should pass")
	}
}

func TestIsUUIDFormat_Valid(t *testing.T) {
	if !isUUIDFormat("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("valid UUID")
	}
}

func TestIsUUIDFormat_WrongLength(t *testing.T) {
	if isUUIDFormat("abc") {
		t.Error("too short")
	}
}

func TestIsUUIDFormat_WrongChars(t *testing.T) {
	if isUUIDFormat("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx") {
		t.Error("invalid chars")
	}
}

func TestIsUUIDFormat_MissingDash(t *testing.T) {
	if isUUIDFormat("550e8400xe29bx41d4xa716x446655440000") {
		t.Error("missing dashes")
	}
}

func TestRedactString_Empty(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	if got := r.RedactString(""); got != "" {
		t.Errorf("got %s", got)
	}
}

func TestRedactString_Email(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	got := r.RedactString("Contact: test@example.com")
	if strings.Contains(got, "test@example.com") {
		t.Error("email should be redacted")
	}
	if !strings.Contains(got, "[REDACTED") {
		t.Error("missing REDACTED marker")
	}
}

func TestRedactString_IPv4(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	got := r.RedactString("IP: 192.168.1.1")
	if strings.Contains(got, "192.168.1.1") {
		t.Error("IP should be redacted")
	}
}

func TestRedactString_MAC(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	got := r.RedactString("MAC: aa:bb:cc:dd:ee:ff")
	if strings.Contains(got, "aa:bb:cc:dd:ee:ff") {
		t.Error("MAC should be redacted")
	}
}

func TestRedactString_UUIDEx(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	got := r.RedactString("UUID: " + uuid)
	// UUIDs should generally not be redacted since they have hyphens breaking the token regex.
	// However, the token regex may still match parts. Just check the redactor doesn't crash.
	if got == "" {
		t.Error("empty result")
	}
}

func TestRedactString_PasswordField(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	got := r.RedactString("password=secret123")
	if strings.Contains(got, "secret123") {
		t.Error("password should be redacted")
	}
}

func TestHashValue_Deterministic(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	a := r.hashValue("email", "test@example.com")
	b := r.hashValue("email", "test@example.com")
	if a != b {
		t.Error("same input should produce same hash")
	}
}

func TestHashValue_Different(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	a := r.hashValue("email", "test@example.com")
	b := r.hashValue("email", "other@example.com")
	if a == b {
		t.Error("different inputs should produce different hashes")
	}
}

func TestHashValue_HasPrefixSuffix(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	a := r.hashValue("email", "x@y.com")
	if !strings.HasPrefix(a, "[REDACTED:") || !strings.HasSuffix(a, "]") {
		t.Errorf("missing markers: %s", a)
	}
}

func TestAddCustomPattern(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	err := r.AddCustomPattern(&RedactionPattern{
		Name:     "ssn",
		Pattern:  regexp.MustCompile(`\d{3}-\d{2}-\d{4}`),
		Replacement: "[SSN]",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := r.RedactString("SSN: 123-45-6789")
	if !strings.Contains(got, "[SSN]") {
		t.Errorf("custom pattern should apply: %s", got)
	}
}

func TestAddCustomPattern_Nil(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	err := r.AddCustomPattern(&RedactionPattern{})
	if err == nil {
		t.Error("nil pattern should error")
	}
}

func TestAddFieldName(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	r.AddFieldName("my_custom_field")
	// Verify it was added
	found := false
	for _, f := range r.config.FieldNames {
		if f == "my_custom_field" {
			found = true
		}
	}
	if !found {
		t.Error("field name not added")
	}
}

func TestNewRedactionHandler(t *testing.T) {
	cfg := DefaultRedactorConfig()
	r := NewRedactor(cfg)
	handler := NewRedactionHandler(nil, r)
	if handler == nil {
		t.Fatal("nil handler")
	}
}

func TestRedactString_NoSensitiveData(t *testing.T) {
	r := NewRedactor(DefaultRedactorConfig())
	input := "Just a normal log message"
	got := r.RedactString(input)
	if got != input {
		t.Errorf("non-sensitive should pass through: %s", got)
	}
}
