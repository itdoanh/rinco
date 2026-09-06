// Package auth cung cấp PASETO v4 token, FIDO2, Argon2, RBAC, OAuth2,
// session management, và API key helpers.
//
// Mọi helper trong package này stateless và thread-safe. Key state nên được
// wrap bởi KeyRing để hỗ trợ rotation, còn session thì dùng SessionStore.
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

// ===== Errors =====

var (
	ErrExpired      = errors.New("auth: token expired")
	ErrInvalid      = errors.New("auth: invalid token")
	ErrUnsupported  = errors.New("auth: unsupported token format")
	ErrKeyMismatch  = errors.New("auth: token signed with unknown key")
	ErrSessionExist = errors.New("auth: session already exists")
)

// ===== PASETO v4 (Local + Public) =====

// KeyPurpose phân biệt local/public key cho PASETO.
type KeyPurpose int

const (
	LocalKey KeyPurpose = iota // symmetric (local)
	PublicKey                  // asymmetric (public/private)
)

// Claims là JWT-style payload cho PASETO v4.
// Mọi field dùng omitempty để tránh leak thông tin không cần thiết.
type Claims struct {
	Issuer      string    `json:"iss,omitempty"`
	Subject     string    `json:"sub,omitempty"`
	Audience    string    `json:"aud,omitempty"`
	ExpiresAt   time.Time `json:"exp,omitempty"`
	IssuedAt    time.Time `json:"iat,omitempty"`
	NotBefore   time.Time `json:"nbf,omitempty"`
	JTI         string    `json:"jti,omitempty"`
	Kid         string    `json:"kid,omitempty"`
	TenantID    string    `json:"tenant_id,omitempty"`
	UserID      string    `json:"user_id,omitempty"`
	Email       string    `json:"email,omitempty"`
	Roles       []string  `json:"roles,omitempty"`
	Permissions []string  `json:"perms,omitempty"`
	Scope       string    `json:"scope,omitempty"`
	SessionID   string    `json:"sid,omitempty"`
}

// Paseto wraps PASETO v4 local (symmetric) operations.
// Key phải dài đúng chacha20poly1305.KeySize (32 bytes).
type Paseto struct {
	key       []byte
	kid       string
	clockSkew time.Duration
}

// NewPaseto tạo Paseto mới với key 32-byte (hex).
func NewPaseto(hexKey, kid string) (*Paseto, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: got %d, want %d", len(key), chacha20poly1305.KeySize)
	}
	return &Paseto{
		key:       key,
		kid:       kid,
		clockSkew: 60 * time.Second,
	}, nil
}

// SetClockSkew cho phép override khoảng dung sai giờ giữa client/server.
func (p *Paseto) SetClockSkew(d time.Duration) { p.clockSkew = d }

// Kid trả về key identifier hiện tại.
func (p *Paseto) Kid() string { return p.kid }

// Encrypt tạo token từ claims, tự sinh JTI + IssuedAt nếu thiếu.
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
	if claims.Kid == "" {
		claims.Kid = p.kid
	}

	pasetoObj := paseto.NewV4Local()
	return pasetoObj.Encrypt(p.key, claims, nil)
}

// Decrypt giải mã và validate token (exp + nbf trong clock-skew tolerance).
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

// KeyRing hỗ trợ key rotation với overlap period (current + previous).
// Verify sẽ thử current trước, fallback previous.
type KeyRing struct {
	current  *Paseto
	previous *Paseto
}

// NewKeyRing tạo key ring. previous có thể rỗng.
func NewKeyRing(currentHex, previousHex, currentKid, previousKid string) (*KeyRing, error) {
	current, err := NewPaseto(currentHex, currentKid)
	if err != nil {
		return nil, fmt.Errorf("current key: %w", err)
	}
	ring := &KeyRing{current: current}
	if previousHex != "" {
		previous, err := NewPaseto(previousHex, previousKid)
		if err != nil {
			return nil, fmt.Errorf("previous key: %w", err)
		}
		ring.previous = previous
	}
	return ring, nil
}

// Encrypt luôn dùng current key.
func (k *KeyRing) Encrypt(claims Claims) (string, error) {
	claims.Kid = k.current.Kid()
	return k.current.Encrypt(claims)
}

// Decrypt thử current key trước, fallback previous.
func (k *KeyRing) Decrypt(token string) (*Claims, error) {
	if claims, err := k.current.Decrypt(token); err == nil {
		return claims, nil
	}
	if k.previous != nil {
		if claims, err := k.previous.Decrypt(token); err == nil {
			return claims, nil
		}
	}
	return nil, ErrInvalid
}

// CurrentKid trả về kid hiện tại (dùng cho JWKS endpoint).
func (k *KeyRing) CurrentKid() string { return k.current.Kid() }

// GenerateSecureToken tạo random token hex với N bytes entropy.
// Dùng cho refresh token, session ID, idempotency key.
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

// ===== Argon2id password hashing =====

// Argon2Params cho password hashing. Tuned theo OWASP 2024.
type Argon2Params struct {
	Memory      uint32 // KB
	Iterations  uint32 // t
	Parallelism uint8  // p
	SaltLength  uint32 // bytes
	KeyLength   uint32 // bytes
}

// DefaultArgon2Params trả về params mạnh (OWASP baseline).
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// HashPassword hash password với Argon2id. Trả về encoded PHC-format string.
func HashPassword(password string) (string, error) {
	return HashPasswordWithParams(password, DefaultArgon2Params())
}

// HashPasswordWithParams cho phép custom params (vd test/dev).
func HashPasswordWithParams(password string, p *Argon2Params) (string, error) {
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		19, p.Memory, p.Iterations, p.Parallelism,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword so sánh password với hash, dùng constant-time comparison.
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