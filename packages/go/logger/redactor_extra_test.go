// Additional tests for logger redactor.
package logger

import (
	"regexp"
	"testing"
)

func TestExtra_Redactor_Defaults(t *testing.T) {
	r := NewRedactor(nil)
	if r == nil {
		t.Fatal("nil redactor")
	}
	if r.config == nil {
		t.Error("nil config")
	}
	if r.config.HashPrefix != "[REDACTED:" {
		t.Errorf("HashPrefix: got %s", r.config.HashPrefix)
	}
}

func TestExtra_Redactor_DefaultPrefixApplied(t *testing.T) {
	r := NewRedactor(&RedactorConfig{})
	if r.config.HashPrefix != "[REDACTED:" {
		t.Errorf("HashPrefix should default: got %s", r.config.HashPrefix)
	}
	if r.config.HashSuffix != "]" {
		t.Errorf("HashSuffix should default: got %s", r.config.HashSuffix)
	}
}

func TestExtra_Redactor_DisableEmail(t *testing.T) {
	r := NewRedactor(&RedactorConfig{DisableEmail: true})
	if r.emailRegex != nil {
		t.Error("email regex should be nil when disabled")
	}
}

func TestExtra_Redactor_DisablePhone(t *testing.T) {
	r := NewRedactor(&RedactorConfig{DisablePhone: true})
	if r.phoneRegex != nil {
		t.Error("phone regex should be nil when disabled")
	}
}

func TestExtra_Redactor_DisableToken(t *testing.T) {
	r := NewRedactor(&RedactorConfig{DisableToken: true})
	if r.tokenRegex != nil || r.jwtRegex != nil {
		t.Error("token regexes should be nil when disabled")
	}
}

func TestExtra_Redactor_DisableCreditCard(t *testing.T) {
	r := NewRedactor(&RedactorConfig{DisableCreditCard: true})
	if r.ccRegex != nil {
		t.Error("cc regex should be nil when disabled")
	}
}

func TestExtra_Redactor_DisableIP(t *testing.T) {
	r := NewRedactor(&RedactorConfig{DisableIPAddress: true})
	if r.ipV4Regex != nil || r.ipV6Regex != nil {
		t.Error("ip regexes should be nil when disabled")
	}
}

func TestExtra_Redactor_DisableMAC(t *testing.T) {
	r := NewRedactor(&RedactorConfig{DisableMACAddress: true})
	if r.macRegex != nil {
		t.Error("mac regex should be nil when disabled")
	}
}

func TestExtra_Redactor_DisablePassword(t *testing.T) {
	r := NewRedactor(&RedactorConfig{DisablePassword: true})
	if r.passwordRegex != nil {
		t.Error("password regex should be nil when disabled")
	}
}

func TestExtra_RedactString_Email(t *testing.T) {
	r := NewRedactor(nil)
	result := r.RedactString("Contact: user@example.com")
	if result == "Contact: user@example.com" {
		t.Error("email should be redacted")
	}
	if !regexp.MustCompile(`REDACTED`).MatchString(result) {
		t.Errorf("expected REDACTED in result: %s", result)
	}
}

func TestExtra_RedactString_IPv4(t *testing.T) {
	r := NewRedactor(nil)
	result := r.RedactString("IP: 192.168.1.1")
	if result == "IP: 192.168.1.1" {
		t.Error("IPv4 should be redacted")
	}
}

func TestExtra_RedactString_IPv6(t *testing.T) {
	r := NewRedactor(nil)
	result := r.RedactString("IP: 2001:0db8:85a3:0000:0000:8a2e:0370:7334")
	if result == "IP: 2001:0db8:85a3:0000:0000:8a2e:0370:7334" {
		t.Error("IPv6 should be redacted")
	}
}

func TestExtra_RedactString_MAC(t *testing.T) {
	r := NewRedactor(nil)
	result := r.RedactString("MAC: AA:BB:CC:DD:EE:FF")
	if result == "MAC: AA:BB:CC:DD:EE:FF" {
		t.Error("MAC should be redacted")
	}
}

func TestExtra_RedactString_JWT(t *testing.T) {
	r := NewRedactor(nil)
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	result := r.RedactString("Token: " + jwt)
	if result == "Token: "+jwt {
		t.Error("JWT should be redacted")
	}
}

func TestExtra_RedactString_Empty(t *testing.T) {
	r := NewRedactor(nil)
	if got := r.RedactString(""); got != "" {
		t.Errorf("empty should stay empty: got %s", got)
	}
}

func TestExtra_RedactString_UUIDPreserved(t *testing.T) {
	r := NewRedactor(nil)
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	result := r.RedactString("ID: " + uuid)
	// UUIDs should not be redacted (only generic tokens 32+ chars)
	if result == "ID: "+uuid {
		t.Error("UUID should remain")
	}
}

func TestExtra_IsValidCreditCard_Valid(t *testing.T) {
	// Valid test card number (Luhn-valid)
	if !isValidCreditCard("4532 0151 1283 0366") {
		t.Error("valid CC should pass")
	}
}

func TestExtra_IsValidCreditCard_Invalid(t *testing.T) {
	if isValidCreditCard("1234 5678 9012 3456") {
		t.Error("invalid CC should fail")
	}
	if isValidCreditCard("not a number") {
		t.Error("non-numeric should fail")
	}
	if isValidCreditCard("123") {
		t.Error("too short should fail")
	}
}

func TestExtra_IsUUIDFormat_Valid(t *testing.T) {
	if !isUUIDFormat("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("valid UUID should pass")
	}
}

func TestExtra_IsUUIDFormat_Invalid(t *testing.T) {
	if isUUIDFormat("not-a-uuid") {
		t.Error("invalid UUID should fail")
	}
	if isUUIDFormat("550e8400e29b41d4a716446655440000") {
		t.Error("missing dashes should fail")
	}
}

func TestExtra_IsLikelyPhone(t *testing.T) {
	if !isLikelyPhone("0901234567") {
		t.Error("phone should pass")
	}
	if !isLikelyPhone("+84901234567") {
		t.Error("intl phone should pass")
	}
	if isLikelyPhone("123") {
		t.Error("too short should fail")
	}
	if isLikelyPhone("12345678901234567890") {
		t.Error("too long should fail")
	}
}

func TestExtra_HashValue(t *testing.T) {
	r := NewRedactor(nil)
	h := r.hashValue("email", "test@example.com")
	if h == "" {
		t.Error("hash should not be empty")
	}
	// Same input = same hash
	h2 := r.hashValue("email", "test@example.com")
	if h != h2 {
		t.Error("hash should be deterministic")
	}
	// Different input = different hash
	h3 := r.hashValue("email", "other@example.com")
	if h == h3 {
		t.Error("different inputs should hash differently")
	}
}

func TestExtra_AddCustomPattern(t *testing.T) {
	r := NewRedactor(nil)
	pattern := &RedactionPattern{
		Name:    "ssn",
		Pattern: regexp.MustCompile(`\d{3}-\d{2}-\d{4}`),
	}
	if err := r.AddCustomPattern(pattern); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(r.config.CustomPatterns) != 1 {
		t.Errorf("expected 1 custom pattern, got %d", len(r.config.CustomPatterns))
	}
}

func TestExtra_AddCustomPattern_NilPattern(t *testing.T) {
	r := NewRedactor(nil)
	err := r.AddCustomPattern(&RedactionPattern{Pattern: nil})
	if err == nil {
		t.Error("expected error for nil pattern")
	}
}

func TestExtra_AddFieldName(t *testing.T) {
	r := NewRedactor(nil)
	r.AddFieldName("custom_secret")
	found := false
	for _, f := range r.config.FieldNames {
		if f == "custom_secret" {
			found = true
			break
		}
	}
	if !found {
		t.Error("custom_secret should be in field names")
	}
}

func TestExtra_RedactString_CustomPattern(t *testing.T) {
	r := NewRedactor(nil)
	r.AddCustomPattern(&RedactionPattern{
		Name:        "ssn",
		Pattern:     regexp.MustCompile(`\d{3}-\d{2}-\d{4}`),
		Replacement: "[REDACTED:SSN]",
	})
	result := r.RedactString("SSN: 123-45-6789")
	if result == "SSN: 123-45-6789" {
		t.Error("SSN should be redacted by custom pattern")
	}
	if !regexp.MustCompile(`REDACTED:SSN`).MatchString(result) {
		t.Errorf("expected custom replacement: %s", result)
	}
}

func TestExtra_DefaultRedactorConfig(t *testing.T) {
	cfg := DefaultRedactorConfig()
	if cfg == nil {
		t.Fatal("nil config")
	}
	if cfg.HashPrefix != "[REDACTED:" {
		t.Errorf("HashPrefix: got %s", cfg.HashPrefix)
	}
	if len(cfg.FieldNames) < 5 {
		t.Error("should have several default field names")
	}
	if !cfg.MaskAllValuesOfMaskFields {
		t.Error("should mask all values of mask fields by default")
	}
}
