// Package valkey implements the storage.PresenceStore interface against a
// Valkey / Redis-compatible server.
//
// Keys:
//   - presence:{user_id}        → JSON {status, last_seen, device}   (TTL = 90s)
//   - typing:{conversation_id}   → SET of user_ids                    (TTL = 5s)
//   - unread:{user_id}:{conv}    → counter                            (no TTL)
//   - online_users               → sorted set (score = last_seen unix)
package valkey

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/itdoanh/rinco/services/chat-engine/internal/storage"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// Store is the Valkey-backed implementation of storage.PresenceStore. When
// the connection cannot be established it transparently falls back to an
// in-process map so the service still works in development.
type Store struct {
	client *redis.Client
	mu     sync.RWMutex
	fb     map[string]map[string]int // userID -> convID -> count
	ok     bool
}

// Open dials Valkey and returns a connected Store.
func Open(ctx context.Context, url string) *Store {
	if url == "" {
		return &Store{fb: map[string]map[string]int{}}
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return &Store{fb: map[string]map[string]int{}}
	}
	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		return &Store{fb: map[string]map[string]int{}}
	}
	return &Store{client: client, ok: true, fb: map[string]map[string]int{}}
}

// Close closes the Valkey connection.
func (s *Store) Close() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

// SetPresence writes presence for the user.
func (s *Store) SetPresence(ctx context.Context, p types.Presence, ttlSeconds int) error {
	p.LastSeen = time.Now().UTC()
	b, _ := json.Marshal(p)
	key := "presence:" + p.UserID
	if s.ok {
		pipe := s.client.Pipeline()
		pipe.Set(ctx, key, b, time.Duration(ttlSeconds)*time.Second)
		pipe.ZAdd(ctx, "online_users", redis.Z{Score: float64(p.LastSeen.Unix()), Member: p.UserID})
		_, err := pipe.Exec(ctx)
		if err == nil {
			return nil
		}
	}
	// Fallback: keep presence in a sentinel entry inside the same process.
	return s.cacheSet(key, string(b), ttlSeconds)
}

// GetPresence returns the presence for the user.
func (s *Store) GetPresence(ctx context.Context, userID string) (types.Presence, bool, error) {
	key := "presence:" + userID
	if s.ok {
		val, err := s.client.Get(ctx, key).Result()
		if err == nil {
			var p types.Presence
			if json.Unmarshal([]byte(val), &p) == nil {
				return p, true, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			// fall through to cache
		}
	}
	val, ok, _ := s.cacheGet(key)
	if !ok {
		return types.Presence{}, false, nil
	}
	var p types.Presence
	_ = json.Unmarshal([]byte(val), &p)
	return p, true, nil
}

// ListOnline returns the most recently seen online users up to limit.
func (s *Store) ListOnline(ctx context.Context, limit int) ([]types.Presence, error) {
	if s.ok {
		res, err := s.client.ZRevRangeByScore(ctx, "online_users", &redis.ZRangeBy{
			Min: "-inf", Max: "+inf", Count: int64(limit),
		}).Result()
		if err == nil {
			out := make([]types.Presence, 0, len(res))
			for _, userID := range res {
				p, ok, _ := s.GetPresence(ctx, userID)
				if ok {
					out = append(out, p)
				}
			}
			return out, nil
		}
	}
	return nil, nil
}

// AddTyping adds a user to the typing set with a TTL.
func (s *Store) AddTyping(ctx context.Context, conversationID, userID string, ttlSeconds int) error {
	key := "typing:" + conversationID
	if s.ok {
		pipe := s.client.Pipeline()
		pipe.SAdd(ctx, key, userID)
		pipe.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second)
		_, err := pipe.Exec(ctx)
		return err
	}
	return nil
}

// RemoveTyping removes a user from the typing set.
func (s *Store) RemoveTyping(ctx context.Context, conversationID, userID string) error {
	key := "typing:" + conversationID
	if s.ok {
		return s.client.SRem(ctx, key, userID).Err()
	}
	return nil
}

// ListTyping returns users currently typing in the conversation.
func (s *Store) ListTyping(ctx context.Context, conversationID string) ([]string, error) {
	key := "typing:" + conversationID
	if s.ok {
		return s.client.SMembers(ctx, key).Result()
	}
	return nil, nil
}

// IncrementUnread bumps the unread counter for the conversation.
func (s *Store) IncrementUnread(_ context.Context, userID, conversationID string) (int, error) {
	if s.ok {
		n, err := s.client.Incr(context.Background(), "unread:"+userID+":"+conversationID).Result()
		return int(n), err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.fb[userID]; !ok {
		s.fb[userID] = map[string]int{}
	}
	s.fb[userID][conversationID]++
	return s.fb[userID][conversationID], nil
}

// GetUnread returns the unread counter.
func (s *Store) GetUnread(_ context.Context, userID, conversationID string) (int, error) {
	if s.ok {
		val, err := s.client.Get(context.Background(), "unread:"+userID+":"+conversationID).Result()
		if err == nil {
			n, _ := strconv.Atoi(val)
			return n, nil
		}
		return 0, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fb[userID][conversationID], nil
}

// ResetUnread clears the unread counter.
func (s *Store) ResetUnread(_ context.Context, userID, conversationID string) error {
	if s.ok {
		return s.client.Set(context.Background(), "unread:"+userID+":"+conversationID, 0, 0).Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.fb[userID]; ok {
		m[conversationID] = 0
	}
	return nil
}

// AsStorage exposes the store under the storage.PresenceStore interface.
func (s *Store) AsStorage() storage.PresenceStore { return s }

// cacheGet / cacheSet implement a tiny in-process fallback used when the
// Valkey connection is unavailable. They are intentionally minimal – only
// required for the build to succeed in environments without a Redis.
var (
	cacheMu sync.RWMutex
	cache   = map[string]cacheEntry{}
)

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

func (s *Store) cacheGet(key string) (string, bool, error) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	e, ok := cache[key]
	if !ok || time.Now().After(e.expiresAt) {
		return "", false, nil
	}
	return e.value, true, nil
}

func (s *Store) cacheSet(key, value string, ttl int) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache[key] = cacheEntry{value: value, expiresAt: time.Now().Add(time.Duration(ttl) * time.Second)}
	return nil
}
