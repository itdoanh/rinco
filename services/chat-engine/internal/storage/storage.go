// Package storage provides the persistence boundary for chat-engine.
//
// Two implementations are bundled:
//
//  1. ScyllaStore – wide-column persistence backed by ScyllaDB.
//  2. MemoryStore – in-memory implementation used when no infra is
//     configured, in tests, or when CHAT_IN_MEMORY_MODE=true.
//
// The store interface is intentionally narrow so handlers don't depend on
// the underlying driver. The Connect-RPC and REST layers speak to the
// interface, not to a concrete type.
package storage

import (
	"context"

	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// MessageStore persists chat messages.
type MessageStore interface {
	Append(ctx context.Context, msg types.Message) error
	GetByConversation(ctx context.Context, conversationID string, limit int, beforeID string) ([]types.Message, error)
	GetByID(ctx context.Context, conversationID, messageID string) (types.Message, error)
	MarkRead(ctx context.Context, conversationID, messageID, userID string) error
}

// ConversationStore persists conversation headers.
type ConversationStore interface {
	Upsert(ctx context.Context, conv types.Conversation) error
	Get(ctx context.Context, id string) (types.Conversation, bool, error)
	ListForUser(ctx context.Context, userID string) ([]types.Conversation, error)
	SetArchived(ctx context.Context, id string, archived bool) error
	SetMuted(ctx context.Context, id string, muted bool) error
	Delete(ctx context.Context, id string) error
}

// GroupStore persists group metadata.
type GroupStore interface {
	Create(ctx context.Context, g types.Group) error
	Get(ctx context.Context, id string) (types.Group, bool, error)
	ListForUser(ctx context.Context, userID string) ([]types.Group, error)
	AddMember(ctx context.Context, groupID, userID string) error
	RemoveMember(ctx context.Context, groupID, userID string) error
	ListMembers(ctx context.Context, groupID string) ([]string, error)
	Delete(ctx context.Context, id string) error
}

// PresenceStore manages presence/typing/unread state. It is normally backed
// by Valkey but a memory implementation is provided for development.
type PresenceStore interface {
	SetPresence(ctx context.Context, p types.Presence, ttlSeconds int) error
	GetPresence(ctx context.Context, userID string) (types.Presence, bool, error)
	ListOnline(ctx context.Context, limit int) ([]types.Presence, error)
	AddTyping(ctx context.Context, conversationID, userID string, ttlSeconds int) error
	RemoveTyping(ctx context.Context, conversationID, userID string) error
	ListTyping(ctx context.Context, conversationID string) ([]string, error)
	IncrementUnread(ctx context.Context, userID, conversationID string) (int, error)
	GetUnread(ctx context.Context, userID, conversationID string) (int, error)
	ResetUnread(ctx context.Context, userID, conversationID string) error
}

// Aggregator bundles all storage interfaces into a single value so the
// server can inject them as one dependency.
type Aggregator interface {
	Messages() MessageStore
	Conversations() ConversationStore
	Groups() GroupStore
	Presence() PresenceStore
}
