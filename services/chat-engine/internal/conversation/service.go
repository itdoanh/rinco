// Package conversation implements the conversation-list use-case: list,
// archive, delete and mute.
package conversation

import (
	"context"

	"github.com/itdoanh/rinco/services/chat-engine/internal/storage"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// Service exposes high-level operations on conversation headers.
type Service struct {
	convs    storage.ConversationStore
	presence storage.PresenceStore
}

// New builds a Service.
func New(convs storage.ConversationStore, presence storage.PresenceStore) *Service {
	return &Service{convs: convs, presence: presence}
}

// ListConversations returns every conversation the user participates in,
// enriched with the unread counter when a presence store is configured.
func (s *Service) ListConversations(ctx context.Context, userID string) ([]types.ConversationListItem, error) {
	convs, err := s.convs.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]types.ConversationListItem, 0, len(convs))
	for _, c := range convs {
		item := types.ConversationListItem{Conversation: c}
		if s.presence != nil {
			item.UnreadCount, _ = s.presence.GetUnread(ctx, userID, c.ConversationID)
		}
		out = append(out, item)
	}
	return out, nil
}

// Archive marks the conversation as archived.
func (s *Service) Archive(ctx context.Context, conversationID string, archived bool) error {
	return s.convs.SetArchived(ctx, conversationID, archived)
}

// Mute toggles the muted flag on the conversation.
func (s *Service) Mute(ctx context.Context, conversationID string, muted bool) error {
	return s.convs.SetMuted(ctx, conversationID, muted)
}

// Delete removes the conversation header. Messages remain in ScyllaDB so
// that exports stay reproducible.
func (s *Service) Delete(ctx context.Context, conversationID string) error {
	return s.convs.Delete(ctx, conversationID)
}
