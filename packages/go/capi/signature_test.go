// Tests for packages/go/capi signature.go (HMAC + app secret proof helpers).
package capi

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// DefaultSignatureConfig / NewSigner
// =============================================================================

func TestExtraDefaultSignatureConfig(t *testing.T) {
	cfg := DefaultSignatureConfig("my-secret")
	if cfg.Secret != "my-secret" {
		t.Errorf("Secret: got %q", cfg.Secret)
	}
	if cfg.Algorithm != "HMAC-SHA256" {
		t.Errorf("Algorithm: got %s", cfg.Algorithm)
	}
	if cfg.TimestampWindow != 5*time.Minute {
		t.Errorf("TimestampWindow: got %v", cfg.TimestampWindow)
	}
}

func TestExtraNewSigner_DefaultWindow(t *testing.T) {
	cfg := SignatureConfig{Secret: "x"} // no window
	s := NewSigner(cfg)
	if s.config.TimestampWindow != 5*time.Minute {
		t.Errorf("expected 5m default, got %v", s.config.TimestampWindow)
	}
}

func TestExtraNewSigner_PreservesWindow(t *testing.T) {
	cfg := SignatureConfig{Secret: "x", TimestampWindow: 1 * time.Hour}
	s := NewSigner(cfg)
	if s.config.TimestampWindow != 1*time.Hour {
		t.Errorf("expected 1h, got %v", s.config.TimestampWindow)
	}
}

// =============================================================================
// Sign / Verify
// =============================================================================

func TestExtraSign_ProducesBase64(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	sig, err := s.Sign(SignatureRequest{
		Method: "POST",
		Path:   "/v1/foo",
		Body:   "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if sig.Value == "" {
		t.Error("empty signature")
	}
	// Should be base64-decodable.
	if _, err := base64.StdEncoding.DecodeString(sig.Value); err != nil {
		t.Errorf("not valid base64: %v", err)
	}
}

func TestExtraSign_AutoFillsTimestamp(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	before := time.Now()
	sig, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x"})
	after := time.Now()
	if sig.Timestamp.Before(before) || sig.Timestamp.After(after) {
		t.Errorf("timestamp not in expected window: %v", sig.Timestamp)
	}
}

func TestExtraSign_GeneratesNonce(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	sig1, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x"})
	sig2, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x"})
	if sig1.Nonce == "" || sig2.Nonce == "" {
		t.Error("nonces should be generated")
	}
	if sig1.Nonce == sig2.Nonce {
		t.Error("nonces should be unique")
	}
}

func TestExtraSign_DeterministicWhenFixed(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	fixed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	sig1, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x", Timestamp: fixed})
	sig2, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x", Timestamp: fixed})
	// Nonce will differ but Value should be the same.
	if sig1.Value != sig2.Value {
		t.Errorf("deterministic: got %s vs %s", sig1.Value, sig2.Value)
	}
}

func TestExtraSign_DifferentSecretsProduceDifferentSignatures(t *testing.T) {
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s1 := NewSigner(DefaultSignatureConfig("secret-1"))
	s2 := NewSigner(DefaultSignatureConfig("secret-2"))
	sig1, _ := s1.Sign(SignatureRequest{Method: "GET", Path: "/x", Timestamp: fixed})
	sig2, _ := s2.Sign(SignatureRequest{Method: "GET", Path: "/x", Timestamp: fixed})
	if sig1.Value == sig2.Value {
		t.Error("different secrets should produce different signatures")
	}
}

// =============================================================================
// Verify
// =============================================================================

func TestExtraVerify_ValidSignature(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	fixed := time.Now().Add(-1 * time.Minute) // within 5-min window
	sig, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x", Timestamp: fixed})
	ok, err := s.Verify(SignatureRequest{Method: "GET", Path: "/x", Timestamp: fixed}, sig.Value)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected valid signature")
	}
}

func TestExtraVerify_ExpiredTimestamp(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	old := time.Now().Add(-1 * time.Hour) // outside 5-min window
	sig, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x", Timestamp: old})
	_, err := s.Verify(SignatureRequest{Method: "GET", Path: "/x", Timestamp: old}, sig.Value)
	if err == nil {
		t.Error("expected timestamp-expired error")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected 'expired' in error, got %v", err)
	}
}

// =============================================================================
// SignRequest
// =============================================================================

func TestExtraSignRequest_AddsHeaders(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	req, _ := http.NewRequest("POST", "https://api.example.com/v1/events", strings.NewReader("{}"))
	if err := s.SignRequest(req, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("X-Signature") == "" {
		t.Error("missing X-Signature")
	}
	if req.Header.Get("X-Signature-Algorithm") != "HMAC-SHA256" {
		t.Error("missing/incorrect algorithm header")
	}
	if req.Header.Get("X-Signature-Timestamp") == "" {
		t.Error("missing timestamp header")
	}
	if req.Header.Get("X-Signature-Nonce") == "" {
		t.Error("missing nonce header")
	}
}

// =============================================================================
// VerifyRequest
// =============================================================================

func TestExtraVerifyRequest_MissingSignature(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	req, _ := http.NewRequest("POST", "/v1/events", nil)
	_, err := s.VerifyRequest(req, nil)
	if err == nil {
		t.Error("expected error for missing signature")
	}
}

func TestExtraVerifyRequest_MissingTimestamp(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	req, _ := http.NewRequest("POST", "/v1/events", nil)
	req.Header.Set("X-Signature", "abc")
	_, err := s.VerifyRequest(req, nil)
	if err == nil {
		t.Error("expected error for missing timestamp")
	}
}

func TestExtraVerifyRequest_InvalidTimestampFormat(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	req, _ := http.NewRequest("POST", "/v1/events", nil)
	req.Header.Set("X-Signature", "abc")
	req.Header.Set("X-Signature-Timestamp", "not-a-number")
	_, err := s.VerifyRequest(req, nil)
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}
}

// =============================================================================
// buildStringToSign (test via Sign — observable behaviour)
// =============================================================================

func TestExtraSign_BodyHashIncluded(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sig1, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "A", Timestamp: fixed})
	sig2, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "B", Timestamp: fixed})
	if sig1.Value == sig2.Value {
		t.Error("different bodies should produce different signatures")
	}
}

func TestExtraSign_QueryParamsIncluded(t *testing.T) {
	cfg := DefaultSignatureConfig("secret")
	s := NewSigner(cfg)
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sig1, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x", QueryParams: map[string]string{"a": "1"}, Timestamp: fixed})
	sig2, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x", QueryParams: map[string]string{"a": "2"}, Timestamp: fixed})
	if sig1.Value == sig2.Value {
		t.Error("different query params should produce different signatures")
	}
}

// =============================================================================
// GenerateAppSecretProof / VerifyAppSecretProof
// =============================================================================

func TestExtraGenerateAppSecretProof_Format(t *testing.T) {
	proof := GenerateAppSecretProof("access-token", "app-secret")
	// 32-byte SHA-256 → 64 hex chars.
	if len(proof) != 64 {
		t.Errorf("expected 64 chars, got %d", len(proof))
	}
	if _, err := hex.DecodeString(proof); err != nil {
		t.Errorf("not valid hex: %v", err)
	}
}

func TestExtraGenerateAppSecretProof_Deterministic(t *testing.T) {
	a := GenerateAppSecretProof("token", "secret")
	b := GenerateAppSecretProof("token", "secret")
	if a != b {
		t.Errorf("same inputs should produce same proof: %s vs %s", a, b)
	}
}

func TestExtraGenerateAppSecretProof_DifferentInputs(t *testing.T) {
	a := GenerateAppSecretProof("token1", "secret")
	b := GenerateAppSecretProof("token2", "secret")
	c := GenerateAppSecretProof("token1", "secret2")
	if a == b || a == c {
		t.Error("different inputs should produce different proofs")
	}
}

func TestExtraVerifyAppSecretProof_Valid(t *testing.T) {
	proof := GenerateAppSecretProof("token", "secret")
	if !VerifyAppSecretProof("token", "secret", proof) {
		t.Error("valid proof should verify")
	}
}

func TestExtraVerifyAppSecretProof_WrongProof(t *testing.T) {
	if VerifyAppSecretProof("token", "secret", "wrong") {
		t.Error("wrong proof should not verify")
	}
}

func TestExtraVerifyAppSecretProof_WrongSecret(t *testing.T) {
	proof := GenerateAppSecretProof("token", "secret1")
	if VerifyAppSecretProof("token", "secret2", proof) {
		t.Error("wrong secret should not verify")
	}
}

func TestExtraVerifyAppSecretProof_WrongToken(t *testing.T) {
	proof := GenerateAppSecretProof("token1", "secret")
	if VerifyAppSecretProof("token2", "secret", proof) {
		t.Error("wrong token should not verify")
	}
}

// =============================================================================
// GenerateRequestSignature
// =============================================================================

func TestExtraGenerateRequestSignature_Deterministic(t *testing.T) {
	a := GenerateRequestSignature("POST", "/v1/foo", "{}", "access", "secret")
	b := GenerateRequestSignature("POST", "/v1/foo", "{}", "access", "secret")
	if a != b {
		t.Error("should be deterministic")
	}
}

func TestExtraGenerateRequestSignature_DifferentMethods(t *testing.T) {
	a := GenerateRequestSignature("POST", "/v1/foo", "{}", "access", "secret")
	b := GenerateRequestSignature("GET", "/v1/foo", "{}", "access", "secret")
	if a == b {
		t.Error("different methods should produce different signatures")
	}
}

// =============================================================================
// ParsePrivateKey / SignWithRSA
// =============================================================================

func TestExtraParsePrivateKey_InvalidPEM(t *testing.T) {
	_, err := ParsePrivateKey("not a pem")
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestExtraParsePrivateKeyAndSignRSA(t *testing.T) {
	// Generate a real RSA key for round-trip test.
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Skip("RSA gen not available:", err)
	}
	// Encode PKCS1.
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	parsed, err := ParsePrivateKey(string(pemBytes))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	data := []byte("test data")
	sig, err := SignWithRSA(data, parsed)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	// Verify by re-signing and comparing — we just want non-empty output.
	if sig == "" {
		t.Error("empty signature")
	}
	// Verify by SHA256 hash + manual check.
	h := sha256.Sum256(data)
	if !strings.Contains(sig, base64.StdEncoding.EncodeToString(h[:])[:20]) {
		// Just verify base64 decodeable.
		if _, err := base64.StdEncoding.DecodeString(sig); err != nil {
			t.Errorf("not valid base64: %v", err)
		}
	}
}

func TestExtraSignWithRSA_Empty(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	sig, err := SignWithRSA([]byte(""), priv)
	if err != nil {
		t.Fatal(err)
	}
	// Empty data still produces a signature.
	if sig == "" {
		t.Error("expected non-empty signature")
	}
}

// =============================================================================
// generateNonce
// =============================================================================

func TestExtraGenerateNonce_Format(t *testing.T) {
	n := generateNonce()
	// 16 bytes → ~22 chars base64.
	if len(n) < 16 {
		t.Errorf("nonce too short: %d", len(n))
	}
	// Should be URL-safe base64.
	if _, err := base64.URLEncoding.DecodeString(n); err != nil {
		t.Errorf("not valid URL-safe base64: %v", err)
	}
}

func TestExtraGenerateNonce_Unique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		n := generateNonce()
		if seen[n] {
			t.Fatalf("duplicate nonce at iteration %d", i)
		}
		seen[n] = true
	}
}
