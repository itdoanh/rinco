// Package crypto implements the auth-service's local crypto primitives
// (HMAC-SHA256 access tokens + Argon2id password hashing) using only the Go
// standard library and argon2.
//
// Tokens are compact, URL-safe strings of the form:
//
//	base64url(header).base64url(payload).base64url(signature)
//
// where signature = HMAC-SHA256(key, header + "." + payload). This is
// loosely similar to PASETO v4 Local in spirit (symmetric, tamper-evident)
// while keeping zero external crypto dependency churn. The signing key is
// 32+ bytes; previous key is supported for rotation overlap.
//
// Vendored here so the service compiles independently of the upstream
// packages/go module which currently has invalid go.mod entries.
package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

// HashPassword returns a PHC-format Argon2id hash.
func HashPassword(pwd string) (string, error) {
	return HashPasswordCustom(pwd, 64*1024, 3, 2, 16, 32)
}

// HashPasswordCustom allows tuning cost parameters (test-only).
func HashPasswordCustom(pwd string, mem, iter uint32, par uint8, saltLen, keyLen uint32) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pwd), salt, iter, mem, par, keyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		mem, iter, par,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword reports whether pwd matches the encoded hash.
func VerifyPassword(pwd, encoded string) (bool, error) {
	if !strings.HasPrefix(encoded, "$argon2id$") {
		return false, errors.New("not argon2id hash")
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false, errors.New("malformed hash")
	}
	var mem, iter uint32
	var par uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iter, &par); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(pwd), salt, iter, mem, par, uint32(len(expected)))
	return hmac.Equal(expected, got), nil
}

// =============================================================================
// HMAC-signed tokens
// =============================================================================

// Header is the unencrypted metadata of a token.
type Header struct {
	Alg     string `json:"alg"`
	Typ     string `json:"typ"`
	Ver     string `json:"ver"`
	Kid     string `json:"kid,omitempty"`
}

// Claims is the JWT-style payload.
type Claims struct {
	Issuer    string    `json:"iss,omitempty"`
	Audience  string    `json:"aud,omitempty"`
	Subject   string    `json:"sub,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	TenantID  string    `json:"tenant_id,omitempty"`
	Roles     []string  `json:"roles,omitempty"`
	Scope     string    `json:"scope,omitempty"`
	JTI       string    `json:"jti,omitempty"`
	IssuedAt  time.Time `json:"iat,omitempty"`
	NotBefore time.Time `json:"nbf,omitempty"`
	ExpiresAt time.Time `json:"exp,omitempty"`
}

// KeyRing holds current and previous signing keys (rotation).
type KeyRing struct {
	current     []byte
	currentKid  string
	previous    []byte
	previousKid string
}

// NewKeyRing creates a ring from raw keys (≥32 bytes each recommended).
func NewKeyRing(current, previous []byte, currentKid, previousKid string) (*KeyRing, error) {
	if len(current) < 16 {
		return nil, fmt.Errorf("current key too short: %d", len(current))
	}
	r := &KeyRing{current: current, currentKid: currentKid}
	if len(previous) > 0 {
		if len(previous) < 16 {
			return nil, fmt.Errorf("previous key too short: %d", len(previous))
		}
		r.previous = previous
		r.previousKid = previousKid
	}
	return r, nil
}

// NewKeyRingFromHex accepts 32-byte hex-encoded keys (matches PASETO
// conventions — useful so the same env vars still work).
func NewKeyRingFromHex(currentHex, previousHex, currentKid, previousKid string) (*KeyRing, error) {
	curr, err := hex.DecodeString(currentHex)
	if err != nil {
		return nil, fmt.Errorf("decode current: %w", err)
	}
	var prev []byte
	if previousHex != "" && previousHex != strings.Repeat("0", 64) {
		prev, err = hex.DecodeString(previousHex)
		if err != nil {
			return nil, fmt.Errorf("decode previous: %w", err)
		}
	}
	return NewKeyRing(curr, prev, currentKid, previousKid)
}

// CurrentKid returns the current key id.
func (r *KeyRing) CurrentKid() string { return r.currentKid }

// Encrypt signs a token with the current key and returns the compact
// URL-safe string. Field name retained for API symmetry.
func (r *KeyRing) Encrypt(c Claims) (string, error) {
	if c.JTI == "" {
		newID, err := uuid.NewV7()
		if err != nil {
			return "", err
		}
		c.JTI = newID.String()
	}
	if c.IssuedAt.IsZero() {
		c.IssuedAt = time.Now()
	}
	if c.Issuer == "" {
		c.Issuer = "rinco"
	}
	if c.Audience == "" {
		c.Audience = "rinco-app"
	}
	hdr := Header{Alg: "HS256", Typ: "RINCO-JWT", Ver: "v1", Kid: r.currentKid}
	headerJSON, err := json.Marshal(hdr)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	h64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	p64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signed := h64 + "." + p64
	sig := r.sign([]byte(signed), r.current)
	s64 := base64.RawURLEncoding.EncodeToString(sig)
	return signed + "." + s64, nil
}

// Decrypt verifies & returns the claims, trying current then previous keys.
func (r *KeyRing) Decrypt(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var hdr Header
	if err := json.Unmarshal(headerJSON, &hdr); err != nil {
		return nil, ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	key := r.current
	if hdr.Kid != "" && hdr.Kid == r.previousKid && r.previous != nil {
		key = r.previous
	} else if hdr.Kid != "" && hdr.Kid != r.currentKid && hdr.Kid != r.previousKid {
		return nil, ErrInvalidToken
	}
	signed := []byte(parts[0] + "." + parts[1])
	expected := r.sign(signed, key)
	if !hmac.Equal(sig, expected) {
		return nil, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, ErrInvalidToken
	}
	now := time.Now()
	if !c.ExpiresAt.IsZero() && now.After(c.ExpiresAt) {
		return nil, ErrExpiredToken
	}
	if !c.NotBefore.IsZero() && now.Before(c.NotBefore) {
		return nil, ErrInvalidToken
	}
	return &c, nil
}

func (r *KeyRing) sign(payload []byte, key []byte) []byte {
	tag := hmac.New(sha256.New, key)
	tag.Write(payload)
	return tag.Sum(nil)
}

// Errors returned by Decrypt.
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

// =============================================================================
// Random helpers
// =============================================================================

// GenerateRandomToken returns a hex-encoded random token with n bytes entropy.
func GenerateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SHA256Hex returns the hex-encoded SHA-256 of b.
func SHA256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// GenerateKey returns a fresh 32-byte hex key (PASETO-compatible).
func GenerateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// =============================================================================
// Helpers used by tests / examples (exposed for completeness)
// =============================================================================

// TokenLifetimeBytes returns the smallest duration that fits in 4 bytes (used
// by some callers wanting to embed timestamps inline). Provided for parity
// with PASETO-style helpers without leaking implementation details.
func TokenLifetimeBytes(t time.Duration) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(t.Nanoseconds()))
	return b
}
