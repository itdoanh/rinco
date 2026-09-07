package capi

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SignatureConfig holds signature configuration.
type SignatureConfig struct {
	// HMAC secret for request signing
	Secret string

	// API secret from Meta (for appsecret_proof)
	AppSecret string

	// App secret for generating appsecret_proof
	AppID     string
	AppSecretKey string

	// Algorithm (default: HMAC-SHA256)
	Algorithm string

	// Timestamp validity window
	TimestampWindow time.Duration
}

// DefaultSignatureConfig returns a config with HMAC-SHA256.
func DefaultSignatureConfig(secret string) SignatureConfig {
	return SignatureConfig{
		Secret:          secret,
		Algorithm:       "HMAC-SHA256",
		TimestampWindow: 5 * time.Minute,
	}
}

// Signer handles request signing.
type Signer struct {
	config SignatureConfig
}

// NewSigner creates a new request signer.
func NewSigner(cfg SignatureConfig) *Signer {
	if cfg.TimestampWindow == 0 {
		cfg.TimestampWindow = 5 * time.Minute
	}
	return &Signer{config: cfg}
}

// SignatureRequest contains data to be signed.
type SignatureRequest struct {
	// Method (GET, POST, etc.)
	Method string
	// URL path (e.g., /v18.0/act_123/events)
	Path string
	// Query parameters
	QueryParams map[string]string
	// Body content
	Body string
	// Timestamp
	Timestamp time.Time
	// App secret proof (Meta-specific)
	AppSecretProof string
}

// Signature represents a computed signature.
type Signature struct {
	// The signature value
	Value string
	// Algorithm used
	Algorithm string
	// Timestamp
	Timestamp time.Time
	// Nonce (if used)
	Nonce string
}

// Sign computes the signature for a request.
func (s *Signer) Sign(req SignatureRequest) (*Signature, error) {
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	// Build string to sign
	stringToSign := s.buildStringToSign(req)

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(s.config.Secret))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return &Signature{
		Value:      signature,
		Algorithm:  s.config.Algorithm,
		Timestamp:  req.Timestamp,
		Nonce:      generateNonce(),
	}, nil
}

// SignRequest adds signature headers to an HTTP request.
func (s *Signer) SignRequest(req *http.Request, body []byte) error {
	sigReq := SignatureRequest{
		Method: req.Method,
		Path:   req.URL.Path,
		Body:   string(body),
	}

	sig, err := s.Sign(sigReq)
	if err != nil {
		return err
	}

	// Add signature headers
	req.Header.Set("X-Signature", sig.Value)
	req.Header.Set("X-Signature-Algorithm", sig.Algorithm)
	req.Header.Set("X-Signature-Timestamp", fmt.Sprintf("%d", sig.Timestamp.Unix()))

	if sig.Nonce != "" {
		req.Header.Set("X-Signature-Nonce", sig.Nonce)
	}

	return nil
}

// Verify checks if a signature is valid.
func (s *Signer) Verify(req SignatureRequest, signature string) (bool, error) {
	computed, err := s.Sign(req)
	if err != nil {
		return false, err
	}

	// Constant-time comparison to prevent timing attacks
	valid := hmac.Equal([]byte(computed.Value), []byte(signature))

	// Check timestamp
	if time.Since(req.Timestamp) > s.config.TimestampWindow {
		return false, fmt.Errorf("signature timestamp expired")
	}

	return valid, nil
}

// VerifyRequest verifies a signature from HTTP request headers.
func (s *Signer) VerifyRequest(req *http.Request, body []byte) (bool, error) {
	// Get signature from header
	sigValue := req.Header.Get("X-Signature")
	if sigValue == "" {
		return false, fmt.Errorf("missing signature header")
	}

	// Get timestamp
	timestampStr := req.Header.Get("X-Signature-Timestamp")
	if timestampStr == "" {
		return false, fmt.Errorf("missing timestamp header")
	}

	timestamp, err := time.Parse("1136214245", timestampStr)
	if err != nil {
		return false, fmt.Errorf("invalid timestamp format")
	}

	sigReq := SignatureRequest{
		Method:    req.Method,
		Path:      req.URL.Path,
		QueryParams: nil,
		Body:      string(body),
		Timestamp: timestamp,
	}

	return s.Verify(sigReq, sigValue)
}

// buildStringToSign creates the canonical string to sign.
func (s *Signer) buildStringToSign(req SignatureRequest) string {
	var parts []string

	// Method
	parts = append(parts, req.Method)

	// Path
	parts = append(parts, req.Path)

	// Timestamp
	parts = append(parts, fmt.Sprintf("%d", req.Timestamp.Unix()))

	// Body hash
	hash := sha256.Sum256([]byte(req.Body))
	parts = append(parts, base64.StdEncoding.EncodeToString(hash[:]))

	// Query params (sorted)
	if len(req.QueryParams) > 0 {
		var params []string
		for k, v := range req.QueryParams {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		parts = append(parts, strings.Join(params, "&"))
	}

	return strings.Join(parts, "\n")
}

// GenerateAppSecretProof creates the appsecret_proof for Meta API.
// This is used to secure API calls by proving the app secret.
func GenerateAppSecretProof(accessToken, appSecret string) string {
	h := hmac.New(sha256.New, []byte(appSecret))
	h.Write([]byte(accessToken))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyAppSecretProof verifies an appsecret_proof.
func VerifyAppSecretProof(accessToken, appSecret, proof string) bool {
	expected := GenerateAppSecretProof(accessToken, appSecret)
	return hmac.Equal([]byte(expected), []byte(proof))
}

// GenerateRequestSignature generates a signature for Graph API requests.
// This is a Meta-specific implementation.
func GenerateRequestSignature(method, path, body, accessToken, appSecret string) string {
	// Generate appsecret_proof
	proof := GenerateAppSecretProof(accessToken, appSecret)

	// Build string to sign
	stringToSign := fmt.Sprintf("%s%s%s", method, path, body)
	
	// Sign with HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(proof))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// ParsePrivateKey parses a PEM-encoded RSA private key.
func ParsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		pkcs8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		var ok bool
		key, ok = pkcs8.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
	}

	return key, nil
}

// SignWithRSA signs data using RSA-SHA256.
func SignWithRSA(data []byte, key *rsa.PrivateKey) (string, error) {
	h := sha256.New()
	h.Write(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
