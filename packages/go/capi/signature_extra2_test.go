// Extra tests for capi signature helpers.
package capi

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSign_DefaultTimestamp(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	req := SignatureRequest{Method: "POST", Path: "/x", Body: "hello"}
	sig, err := s.Sign(req)
	if err != nil {
		t.Fatal(err)
	}
	if sig.Timestamp.IsZero() {
		t.Error("timestamp should be auto-set")
	}
}

func TestSign_Deterministic(t *testing.T) {
	// Same input with same timestamp should produce same signature
	s := NewSigner(DefaultSignatureConfig("sec"))
	ts := time.Now()
	sig1, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "hello", Timestamp: ts})
	sig2, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "hello", Timestamp: ts})
	if sig1.Value != sig2.Value {
		t.Error("same input should produce same signature")
	}
}

func TestSign_DifferentBodies(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	ts := time.Now()
	sig1, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "hello", Timestamp: ts})
	sig2, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x", Body: "world", Timestamp: ts})
	if sig1.Value == sig2.Value {
		t.Error("different bodies should produce different signatures")
	}
}

func TestSign_NoncePresent(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	sig, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x"})
	if sig.Nonce == "" {
		t.Error("nonce should be generated")
	}
}

func TestSign_NonceUnique(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	sig1, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x"})
	sig2, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x"})
	if sig1.Nonce == sig2.Nonce {
		t.Error("nonces should be unique")
	}
}

func TestSign_AlgorithmSet(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	sig, _ := s.Sign(SignatureRequest{Method: "POST", Path: "/x"})
	if sig.Algorithm != "HMAC-SHA256" {
		t.Errorf("algorithm: got %s", sig.Algorithm)
	}
}

func TestSignRequest_AddsHeaders(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	req, _ := http.NewRequest("POST", "/x", bytes.NewReader([]byte("body")))
	if err := s.SignRequest(req, []byte("body")); err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("X-Signature") == "" {
		t.Error("missing X-Signature")
	}
	if req.Header.Get("X-Signature-Algorithm") != "HMAC-SHA256" {
		t.Error("missing algorithm header")
	}
	if req.Header.Get("X-Signature-Timestamp") == "" {
		t.Error("missing timestamp header")
	}
}

func TestVerifyRequest_MissingSignature(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	req, _ := http.NewRequest("POST", "/x", nil)
	_, err := s.VerifyRequest(req, nil)
	if err == nil || !strings.Contains(err.Error(), "missing signature") {
		t.Errorf("expected missing signature error, got %v", err)
	}
}

func TestVerifyRequest_MissingTimestamp(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	req, _ := http.NewRequest("POST", "/x", nil)
	req.Header.Set("X-Signature", "abc")
	_, err := s.VerifyRequest(req, nil)
	if err == nil || !strings.Contains(err.Error(), "missing timestamp") {
		t.Errorf("expected missing timestamp error, got %v", err)
	}
}

func TestVerifyRequest_InvalidTimestamp(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	req, _ := http.NewRequest("POST", "/x", nil)
	req.Header.Set("X-Signature", "abc")
	req.Header.Set("X-Signature-Timestamp", "not-a-number")
	_, err := s.VerifyRequest(req, nil)
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}
}

func TestGenerateAppSecretProof(t *testing.T) {
	proof := GenerateAppSecretProof("token", "secret")
	if proof == "" {
		t.Error("proof should not be empty")
	}
	if len(proof) != 64 { // hex of sha256 = 64 chars
		t.Errorf("proof length: %d", len(proof))
	}
}

func TestVerifyAppSecretProof_Valid(t *testing.T) {
	proof := GenerateAppSecretProof("token", "secret")
	if !VerifyAppSecretProof("token", "secret", proof) {
		t.Error("valid proof should verify")
	}
}

func TestVerifyAppSecretProof_InvalidToken(t *testing.T) {
	proof := GenerateAppSecretProof("token", "secret")
	if VerifyAppSecretProof("other", "secret", proof) {
		t.Error("invalid token should not verify")
	}
}

func TestVerifyAppSecretProof_InvalidSecret(t *testing.T) {
	proof := GenerateAppSecretProof("token", "secret")
	if VerifyAppSecretProof("token", "other", proof) {
		t.Error("invalid secret should not verify")
	}
}

func TestGenerateAppSecretProof_Deterministic(t *testing.T) {
	p1 := GenerateAppSecretProof("t", "s")
	p2 := GenerateAppSecretProof("t", "s")
	if p1 != p2 {
		t.Error("should be deterministic")
	}
}

func TestGenerateRequestSignature(t *testing.T) {
	sig := GenerateRequestSignature("POST", "/path", "body", "token", "secret")
	if sig == "" {
		t.Error("signature should not be empty")
	}
}

func TestGenerateRequestSignature_Different(t *testing.T) {
	a := GenerateRequestSignature("POST", "/p", "b", "t", "s")
	b := GenerateRequestSignature("GET", "/p", "b", "t", "s")
	if a == b {
		t.Error("different methods should give different signatures")
	}
}

func TestParsePrivateKey_Invalid(t *testing.T) {
	_, err := ParsePrivateKey("not a pem")
	if err == nil {
		t.Error("expected error")
	}
}

func TestParsePrivateKey_InvalidPEMBlock(t *testing.T) {
	_, err := ParsePrivateKey("-----BEGIN PRIVATE KEY-----\nINVALID\n-----END PRIVATE KEY-----")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGenerateNonce_Length(t *testing.T) {
	n := generateNonce()
	if n == "" {
		t.Error("empty")
	}
}

func TestGenerateNonce_Unique(t *testing.T) {
	a := generateNonce()
	b := generateNonce()
	if a == b {
		t.Error("nonces should differ")
	}
}

func TestSign_QueryParams(t *testing.T) {
	s := NewSigner(DefaultSignatureConfig("sec"))
	ts := time.Now()
	sig1, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x", QueryParams: map[string]string{"a": "1"}, Timestamp: ts})
	sig2, _ := s.Sign(SignatureRequest{Method: "GET", Path: "/x", QueryParams: map[string]string{"a": "2"}, Timestamp: ts})
	if sig1.Value == sig2.Value {
		t.Error("different query params should produce different signatures")
	}
}
