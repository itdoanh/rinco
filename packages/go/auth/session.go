// Package auth - Session store với sliding window TTL.
//
// Session được lưu ở Redis/Valkey (key: "session:<id>") với TTL = idle timeout.
// Mỗi lần Get/Set mới thì TTL được extend (sliding window).
// Touch() được gọi mỗi request để giữ session alive.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrSessionNotFound = errors.New("session: not found")
	ErrSessionExpired  = errors.New("session: expired")
)

// SessionData lưu thông tin user-level của session.
type SessionData struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	TenantID    string                 `json:"tenant_id"`
	Email       string                 `json:"email"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions,omitempty"`
	IPAddress   string                 `json:"ip,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	DeviceFP    string                 `json:"device_fp,omitempty"`
	MFAVerified bool                   `json:"mfa_verified"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	LastSeenAt  time.Time              `json:"last_seen_at"`
}

// SessionStore interface cho high-level swap (memory/redis/...).
type SessionStore interface {
	Create(ctx context.Context, data SessionData, ttl time.Duration) error
	Get(ctx context.Context, id string) (*SessionData, error)
	Touch(ctx context.Context, id string, ttl time.Duration) error
	Delete(ctx context.Context, id string) error
	DeleteByUser(ctx context.Context, userID string) error
	ListByUser(ctx context.Context, userID string) ([]SessionData, error)
}

// ===== Redis-backed store =====

// RedisSessionStore dùng go-redis (cũng tương thích Valkey).
type RedisSessionStore struct {
	rdb *redis.Client
	keyPrefix string
}

// NewRedisSessionStore tạo store với Redis/Valkey client.
func NewRedisSessionStore(rdb *redis.Client) *RedisSessionStore {
	return &RedisSessionStore{rdb: rdb, keyPrefix: "session:"}
}

func (s *RedisSessionStore) key(id string) string { return s.keyPrefix + id }

func (s *RedisSessionStore) Create(ctx context.Context, data SessionData, ttl time.Duration) error {
	if data.ID == "" {
		return errors.New("session: ID required")
	}
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}
	data.LastSeenAt = time.Now()
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, s.key(data.ID), payload, ttl)
	if data.UserID != "" {
		pipe.SAdd(ctx, s.userKey(data.UserID), data.ID)
		pipe.Expire(ctx, s.userKey(data.UserID), 30*24*time.Hour)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisSessionStore) Get(ctx context.Context, id string) (*SessionData, error) {
	payload, err := s.rdb.Get(ctx, s.key(id)).Bytes()
	if err == redis.Nil {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("session: get: %w", err)
	}
	var d SessionData
	if err := json.Unmarshal(payload, &d); err != nil {
		return nil, fmt.Errorf("session: unmarshal: %w", err)
	}
	return &d, nil
}

// Touch extend TTL và update LastSeenAt (sliding window).
func (s *RedisSessionStore) Touch(ctx context.Context, id string, ttl time.Duration) error {
	data, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	data.LastSeenAt = time.Now()
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, s.key(id), payload, ttl).Err()
}

func (s *RedisSessionStore) Delete(ctx context.Context, id string) error {
	data, _ := s.Get(ctx, id)
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, s.key(id))
	if data != nil && data.UserID != "" {
		pipe.SRem(ctx, s.userKey(data.UserID), id)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisSessionStore) DeleteByUser(ctx context.Context, userID string) error {
	members, err := s.rdb.SMembers(ctx, s.userKey(userID)).Result()
	if err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}
	pipe := s.rdb.Pipeline()
	for _, id := range members {
		pipe.Del(ctx, s.key(id))
	}
	pipe.Del(ctx, s.userKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisSessionStore) ListByUser(ctx context.Context, userID string) ([]SessionData, error) {
	members, err := s.rdb.SMembers(ctx, s.userKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]SessionData, 0, len(members))
	for _, id := range members {
		d, err := s.Get(ctx, id)
		if err == nil {
			out = append(out, *d)
		}
	}
	return out, nil
}

func (s *RedisSessionStore) userKey(userID string) string {
	return fmt.Sprintf("user-sessions:%s", userID)
}

// ===== In-memory store (test/dev) =====

// MemorySessionStore in-memory store dùng cho test/dev.
type MemorySessionStore struct {
	store map[string]SessionData
}

// NewMemorySessionStore tạo in-memory store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{store: make(map[string]SessionData)}
}

func (s *MemorySessionStore) Create(_ context.Context, data SessionData, _ time.Duration) error {
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}
	data.LastSeenAt = time.Now()
	s.store[data.ID] = data
	return nil
}

func (s *MemorySessionStore) Get(_ context.Context, id string) (*SessionData, error) {
	d, ok := s.store[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return &d, nil
}

func (s *MemorySessionStore) Touch(_ context.Context, id string, _ time.Duration) error {
	d, ok := s.store[id]
	if !ok {
		return ErrSessionNotFound
	}
	d.LastSeenAt = time.Now()
	s.store[id] = d
	return nil
}

func (s *MemorySessionStore) Delete(_ context.Context, id string) error {
	delete(s.store, id)
	return nil
}

func (s *MemorySessionStore) DeleteByUser(_ context.Context, userID string) error {
	for id, d := range s.store {
		if d.UserID == userID {
			delete(s.store, id)
		}
	}
	return nil
}

func (s *MemorySessionStore) ListByUser(_ context.Context, userID string) ([]SessionData, error) {
	out := make([]SessionData, 0)
	for _, d := range s.store {
		if d.UserID == userID {
			out = append(out, d)
		}
	}
	return out, nil
}