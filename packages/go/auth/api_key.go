// Package auth - API key generation, hashing, validation.
//
// API key format:
//   <prefix>_<env>_<random>
// ví dụ: rinco_live_a8s9d7f6g5h4j3k2
//
//   prefix: "rinco" (configurable)
//   env:    "live" hoặc "test"
//   random: 32 bytes base62 (~43 chars)
//
// DB lưu:
//   id              uuid
//   tenant_id       uuid
//   user_id         uuid (creator)
//   name            text
//   prefix          text (rinco_live_abc12)
//   hash            text (sha256 hex của full key)
//   last_4          text (4 char cuối để hiển thị)
//   scopes          text[]
//   rate_limit      int
//   expires_at      timestamptz
//   last_used_at    timestamptz
//   created_at      timestamptz
//   revoked_at      timestamptz
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// APIKeyEnv phân biệt môi trường.
type APIKeyEnv string

const (
	APIKeyLive APIKeyEnv = "live"
	APIKeyTest APIKeyEnv = "test"
)

// DefaultPrefix mặc định của RINCO.
const DefaultPrefix = "rinco"

// APIKey là struct trả về cho client khi create.
type APIKey struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	Scopes    []string  `json:"scopes"`
	RateLimit int       `json:"rate_limit"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKeyWithSecret bao gồm cả plain key (chỉ trả về 1 lần khi tạo).
type APIKeyWithSecret struct {
	APIKey
	Key      string    `json:"key"` // plain text - chỉ hiển thị 1 lần
	Last4    string    `json:"last_4"`
}

// APIKeyRecord là representation ở DB layer.
type APIKeyRecord struct {
	ID         string
	TenantID   string
	UserID     string
	Name       string
	Prefix     string
	Hash       string // sha256 hex
	Last4      string
	Scopes     []string
	RateLimit  int
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
	RevokedAt  *time.Time
}

// HashKey trả về sha256 hex digest của API key (để so sánh trong DB).
func HashKey(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}

// GenerateAPIKey sinh API key mới với prefix = rinco, env = live|test, secret = base64 (24 bytes).
//
// Output format:
//   rinco_live_a8s9d7f6g5h4j3k2m9n8b7v6c5x4z3q
func GenerateAPIKey(env APIKeyEnv, prefix string) (string, error) {
	if prefix == "" {
		prefix = DefaultPrefix
	}
	if env != APIKeyLive && env != APIKeyTest {
		return "", errors.New("api_key: invalid env")
	}
	// 24 bytes ngẫu nhiên → base64 url-safe (32 chars).
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("api_key: rand: %w", err)
	}
	secret := strings.TrimRight(base64.URLEncoding.EncodeToString(buf), "=")
	// Base64 URL alphabet includes '_' which would break the parsing split.
	// Replace '_' with a different character that's also URL-safe.
	secret = strings.ReplaceAll(secret, "_", "-")
	return fmt.Sprintf("%s_%s_%s", prefix, env, secret), nil
}

// ParseAPIKey tách API key thành prefix, env, secret.
// Trả về error nếu format không hợp lệ.
func ParseAPIKey(key string) (prefix string, env APIKeyEnv, secret string, err error) {
	parts := strings.Split(key, "_")
	if len(parts) != 3 {
		err = errors.New("api_key: invalid format")
		return
	}
	prefix = parts[0]
	switch parts[1] {
	case "live":
		env = APIKeyLive
	case "test":
		env = APIKeyTest
	default:
		err = fmt.Errorf("api_key: invalid env %q", parts[1])
		return
	}
	secret = parts[2]
	if len(secret) < 16 {
		err = errors.New("api_key: secret too short")
	}
	return
}

// Last4 trả về 4 char cuối của secret (để hiển thị UI).
func Last4(key string) string {
	if len(key) < 4 {
		return key
	}
	return key[len(key)-4:]
}

// IsExpired kiểm tra key còn hạn không.
func IsExpired(record *APIKeyRecord, now time.Time) bool {
	if record == nil || record.ExpiresAt == nil {
		return false
	}
	return now.After(*record.ExpiresAt)
}

// IsRevoked kiểm tra key đã bị thu hồi chưa.
func IsRevoked(record *APIKeyRecord) bool {
	return record != nil && record.RevokedAt != nil
}

// ScopesContain kiểm tra scope có nằm trong key không (hỗ trợ wildcard).
func ScopesContain(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == "*" || s == want {
			return true
		}
		if strings.HasSuffix(s, ".*") {
			prefix := strings.TrimSuffix(s, ".*")
			if strings.HasPrefix(want, prefix+".") {
				return true
			}
		}
	}
	return false
}

// NewRecord tạo record mới với đầy đủ fields (chưa persist).
func NewRecord(tenantID, userID, name string, env APIKeyEnv, scopes []string, rateLimit int, ttl time.Duration) (*APIKeyWithSecret, error) {
	if rateLimit == 0 {
		rateLimit = 1000 // per hour default
	}
	key, err := GenerateAPIKey(env, DefaultPrefix)
	if err != nil {
		return nil, err
	}
	_ = HashKey(key) // keep helper visible from this file's perspective
	id := newUUID()
	now := time.Now()
	var expiresAt *time.Time
	if ttl > 0 {
		t := now.Add(ttl)
		expiresAt = &t
	}
	prefix := fmt.Sprintf("%s_%s_%s", DefaultPrefix, env, key[len(key)-8:][:5])

	return &APIKeyWithSecret{
		APIKey: APIKey{
			ID:        id,
			TenantID:  tenantID,
			UserID:    userID,
			Name:      name,
			Prefix:    prefix,
			Scopes:    scopes,
			RateLimit: rateLimit,
			ExpiresAt: expiresAt,
			CreatedAt: now,
		},
		Key:   key,
		Last4: Last4(key),
	}, nil
}

func newUUID() string {
	// Tránh import uuid trực tiếp vào file này, dùng helper.
	return uuidGen()
}