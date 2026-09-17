// Package presence implements the presence use-case: heartbeat, listing,
// and lookups.
package presence

import (
	"context"

	"github.com/itdoanh/rinco/services/chat-engine/internal/nats"
	"github.com/itdoanh/rinco/services/chat-engine/internal/storage"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// Service tracks user presence via the storage.PresenceStore abstraction.
type Service struct {
	store     storage.PresenceStore
	publisher nats.Publisher
	ttl       int
}

// New builds a Service.
func New(store storage.PresenceStore, pub nats.Publisher, ttlSeconds int) *Service {
	return &Service{store: store, publisher: pub, ttl: ttlSeconds}
}

// Set marks the user as online and emits the corresponding event.
func (s *Service) Set(ctx context.Context, p types.Presence) error {
	if p.Status == "" {
		p.Status = "online"
	}
	if err := s.store.SetPresence(ctx, p, s.ttl); err != nil {
		return err
	}
	return s.publisher.PublishPresence(ctx, p)
}

// Get returns the presence for the user.
func (s *Service) Get(ctx context.Context, userID string) (types.Presence, bool, error) {
	return s.store.GetPresence(ctx, userID)
}

// ListOnline returns the most recently seen online users.
func (s *Service) ListOnline(ctx context.Context, limit int) ([]types.Presence, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.store.ListOnline(ctx, limit)
}

// ListTyping returns the users currently typing in a conversation.
func (s *Service) ListTyping(ctx context.Context, conversationID string) ([]string, error) {
	return s.store.ListTyping(ctx, conversationID)
}
