// Package auth cung cấp PASETO v4 token, FIDO2, Argon2 helpers.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/o1egl/paseto"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	ErrExpired = errors.New("token expired")
	ErrInvalid = errors.New("invalid token")
)

// Claims represents PASETO v4 claims.
type Claims struct {
	Issuer      string    `json:"iss"`
	Subject     string    `json:"sub"`
	Audience    string    `json:"aud"`
	ExpiresAt   time.Time `json:"exp"`
	IssuedAt    time.Time `json:"iat"`
	NotBefore   time.Time `json:"nbf"`
	JTI         string    `json:"jti"`
	TenantID    string    `json:"tenant_id,omitempty"`
	UserID      string    `json:"user_id,omitempty"`
	Roles       []string  `json:"roles,omitempty"`
	Permissions []string  `json:"perms,omitempty"`
	Scope       string    `json:"scope,omitempty"`
}

// Paseto wraps PASETO v4 operations.
type Paseto struct {
	key       []byte
	clockSkew time.Duration
}

// NewPaseto tạo Paseto mới với key 32-byte (hex).
func NewPaseto(hexKey string) (*Paseto, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: got %d, want %d", len(key), chacha20poly1305.KeySize)
	}
	return &Paseto{
		key:       key,
		clockSkew: 60 * time.Second,
	}, nil
}

// Encrypt tạo token từ claims.
func (p *Paseto) Encrypt(claims Claims) (string, error) {
	if claims.JTI == "" {
		claims.JTI = uuid.NewV7().String()
	}
	if claims.IssuedAt.IsZero() {
		claims.IssuedAt = time.Now()
	}
	if claims.Issuer == "" {
		claims.Issuer = "rinco"
	}
	if claims.Audience == "" {
		claims.Audience = "rinco-app"
	}

	pasetoObj := paseto.NewV4Local()
	return pasetoObj.Encrypt(p.key, claims, nil)
}

// Decrypt giải mã và validate token.
func (p *Paseto) Decrypt(token string) (*Claims, error) {
	pasetoObj := paseto.NewV4Local()
	var claims Claims
	if err := pasetoObj.Decrypt(token, p.key, &claims, nil); err != nil {
		return nil, ErrInvalid
	}

	now := time.Now()
	if !claims.ExpiresAt.IsZero() && now.After(claims.ExpiresAt.Add(p.clockSkew)) {
		return nil, ErrExpired
	}
	if !claims.NotBefore.IsZero() && now.Before(claims.NotBefore.Add(-p.clockSkew)) {
		return nil, ErrInvalid
	}

	return &claims, nil
}

// KeyRing hỗ trợ key rotation (overlap period).
type KeyRing struct {
	current  []byte
	previous []byte
}

// NewKeyRing tạo key ring với 2 keys.
func NewKeyRing(currentHex, previousHex string) (*KeyRing, error) {
	current, err := hex.DecodeString(currentHex)
	if err != nil {
		return nil, fmt.Errorf("decode current key: %w", err)
	}
	if len(current) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid current key size")
	}

	var previous []byte
	if previousHex != "" {
		previous, err = hex.DecodeString(previousHex)
		if err != nil {
			return nil, fmt.Errorf("decode previous key: %w", err)
		}
		if len(previous) != chacha20poly1305.KeySize {
			return nil, errors.New("invalid previous key size")
		}
	}

	return &KeyRing{current: current, previous: previous}, nil
}

// Encrypt luôn dùng current key.
func (k *KeyRing) Encrypt(claims Claims) (string, error) {
	p := paseto.NewV4Local()
	if claims.JTI == "" {
		claims.JTI = uuid.NewV7().String()
	}
	return p.Encrypt(k.current, claims, nil)
}

// Decrypt thử current key trước, sau đó previous.
func (k *KeyRing) Decrypt(token string) (*Claims, error) {
	p := paseto.NewV4Local()
	var claims Claims

	if err := p.Decrypt(token, k.current, &claims, nil); err == nil {
		return &claims, nil
	}

	if k.previous != nil {
		var prevClaims Claims
		if err := p.Decrypt(token, k.previous, &prevClaims, nil); err == nil {
			return &prevClaims, nil
		}
	}

	return nil, ErrInvalid
}

// GenerateSecureToken tạo random token hex (cho refresh tokens).
func GenerateSecureToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GeneratePASETOKey tạo key ngẫu nhiên cho PASETO.
func GeneratePASETOKey() (string, error) {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// ============ Argon2 helpers ============

// Argon2Params cho password hashing.
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params trả về params mạnh.
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// HashPassword hash password với Argon2id.
func HashPassword(password string) (string, error) {
	p := DefaultArgon2Params()
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

	// Format: $argon2id$v=19$m=memory,t=iter,p=parallelism$salt$hash
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		19, p.Memory, p.Iterations, p.Parallelism,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword so sánh password với hash.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := splitEncoded(encoded)
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	if version != 19 {
		return false, fmt.Errorf("unsupported version: %d", version)
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, err
	}

	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expected, err := hex.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	keyLen := uint32(len(expected))
	computed := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLen)

	return constantTimeEqual(expected, computed), nil
}

func splitEncoded(s string) []string {
	parts := make([]string, 0, 6)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '$' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
