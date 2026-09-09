package capi

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDefaultSignatureConfig(t *testing.T) {
	cfg := DefaultSignatureConfig("my-secret")
	if cfg.Secret != "my-secret" {
		t.Errorf("secret = %q", cfg.Secret)
	}
	if cfg.Algorithm != "HMAC-SHA256" {
		t.Errorf("algorithm = %q", cfg.Algorithm)
	}
	if cfg.TimestampWindow != 5*time.Minute {
		t.Errorf("window = %v", cfg.TimestampWindow)
	}
}

func TestNewSignerDefaultsWindow(t *testing.T) {
	s := NewSigner(SignatureConfig{Secret: "x"})
	if s.config.TimestampWindow != 5*time.Minute {
		t.Errorf("default window not applied: %v", s.config.TimestampWindow)
	}
}

func TestSignIncludesAllHeaders(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	body := []byte("test body")
	req, _ := http.NewRequest("POST", "http://x/y", bytes.NewReader(body))
	if err := s.SignRequest(req, body); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}
	if req.Header.Get("X-Signature") == "" {
		t.Error("missing signature header")
	}
	if req.Header.Get("X-Signature-Algorithm") == "" {
		t.Error("missing alg header")
	}
	if req.Header.Get("X-Signature-Timestamp") == "" {
		t.Error("missing timestamp header")
	}
	if req.Header.Get("X-Signature-Nonce") == "" {
		t.Error("missing nonce header")
	}
}

func TestSignSameInputSameOutput(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	ts := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	r1, err := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "hello", Timestamp: ts})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "hello", Timestamp: ts})
	if err != nil {
		t.Fatal(err)
	}
	if r1.Value != r2.Value {
		t.Error("same input should produce same signature")
	}
}

func TestSignDifferentBodyDifferentSig(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	r1, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "a"})
	r2, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "b"})
	if r1.Value == r2.Value {
		t.Error("different body should yield different sig")
	}
}

func TestVerifySuccess(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	ts := time.Now()
	sig, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "hello", Timestamp: ts})
	valid, err := s.Verify(SignatureRequest{Method: "POST", Path: "/x", Body: "hello", Timestamp: ts}, sig.Value)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !valid {
		t.Error("valid signature")
	}
}

func TestVerifyInvalid(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	// Within window
	valid, err := s.Verify(SignatureRequest{Method: "POST", Path: "/x", Timestamp: time.Now()}, "invalidsig")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if valid {
		t.Error("invalid signature should not verify")
	}
}

func TestVerifyExpired(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	old := time.Now().Add(-10 * time.Minute)
	sig, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Timestamp: old})
	_, err := s.Verify(SignatureRequest{Method: "POST", Path: "/x", Timestamp: old}, sig.Value)
	if err == nil {
		t.Error("expected expiry error")
	}
}

func TestVerifyRequestMissingHeaders(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	req, _ := http.NewRequest("POST", "http://x/y", nil)
	if _, err := s.VerifyRequest(req, nil); err == nil {
		t.Error("expected error for missing signature")
	}
}

func TestVerifyRequestInvalidTimestamp(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	req, _ := http.NewRequest("POST", "http://x/y", nil)
	req.Header.Set("X-Signature", "abc")
	req.Header.Set("X-Signature-Timestamp", "bad")
	if _, err := s.VerifyRequest(req, nil); err == nil {
		t.Error("expected error for bad timestamp")
	}
}

func TestGenerateAppSecretProofDeterministic(t *testing.T) {
	a := GenerateAppSecretProof("access", "secret")
	b := GenerateAppSecretProof("access", "secret")
	if a != b {
		t.Error("proof should be deterministic for same inputs")
	}
	if a == "" {
		t.Error("proof empty")
	}
}

func TestGenerateAppSecretProofLength(t *testing.T) {
	// hex(SHA-256) = 64 chars
	p := GenerateAppSecretProof("t", "s")
	if len(p) != 64 {
		t.Errorf("len = %d, want 64", len(p))
	}
}

func TestVerifyAppSecretProofRoundtrip(t *testing.T) {
	proof := GenerateAppSecretProof("access", "secret")
	if !VerifyAppSecretProof("access", "secret", proof) {
		t.Error("roundtrip failed")
	}
	if VerifyAppSecretProof("access", "wrong", proof) {
		t.Error("wrong secret should fail")
	}
	if VerifyAppSecretProof("wrong", "secret", proof) {
		t.Error("wrong token should fail")
	}
}

func TestGenerateRequestSignatureDeterministic(t *testing.T) {
	s1 := GenerateRequestSignature("GET", "/x", "body", "access", "secret")
	s2 := GenerateRequestSignature("GET", "/x", "body", "access", "secret")
	if s1 != s2 {
		t.Error("should be deterministic")
	}
	if s1 == "" {
		t.Error("sig empty")
	}
}

func TestParsePrivateKeyInvalidPEM(t *testing.T) {
	_, err := ParsePrivateKey("not pem data")
	if err == nil {
		t.Error("expected error")
	}
}

func TestParsePrivateKeyValid(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	parsed, err := ParsePrivateKey(string(pemBytes))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.N.Cmp(priv.N) != 0 {
		t.Error("parsed key mismatch")
	}
}

func TestSignWithRSA(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	data := []byte("hello world")
	sig, err := SignWithRSA(data, priv)
	if err != nil {
		t.Fatalf("SignWithRSA: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("base64: %v", err)
	}
	if len(decoded) == 0 {
		t.Error("empty decoded")
	}
}

func TestBuildStringToSignIncludesAll(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	ts := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	str := s.buildStringToSign(SignatureRequest{
		Method: "POST",
		Path:   "/v1/events",
		Body:   "hello",
		Timestamp: ts,
		QueryParams: map[string]string{"a": "1", "b": "2"},
	})
	for _, part := range []string{"POST", "/v1/events", "a=1&b=2"} {
		if !strings.Contains(str, part) {
			t.Errorf("missing %q in: %s", part, str)
		}
	}
	// Body is hashed (sha256), not stored in plaintext
	if strings.Contains(str, "hello") {
		t.Error("body should be hashed, not in plaintext")
	}
}

func TestGenerateNonceUnique(t *testing.T) {
	a := generateNonce()
	b := generateNonce()
	if a == b {
		t.Error("nonces should differ")
	}
	if a == "" {
		t.Error("empty nonce")
	}
}
