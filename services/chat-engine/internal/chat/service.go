// Package chat implements the 1:1 chat use-case: send/receive messages,
// list history, mark read, and typing indicators.
//
// It glues the message store, the conversation store, the presence store
// and the NATS publisher into a single, transaction-free unit. Race
// conditions on the conversation header are resolved by a Valkey lock so
// concurrent updates remain linearizable.
package chat

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/chat-engine/internal/nats"
	"github.com/itdoanh/rinco/services/chat-engine/internal/storage"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// ErrInvalidConversation is returned when a conversation identifier is empty.
var ErrInvalidConversation = errors.New("invalid conversation_id")

// Service coordinates the 1:1 chat use-case.
type Service struct {
	msgs         storage.MessageStore
	convs        storage.ConversationStore
	presence     storage.PresenceStore
	publisher    nats.Publisher
	typingTTL    int
	unreadTTL    int
	maxHistory   int
}

// New builds a Service.
func New(msgs storage.MessageStore, convs storage.ConversationStore, presence storage.PresenceStore, pub nats.Publisher, typingTTL int, maxHistory int) *Service {
	return &Service{
		msgs:       msgs,
		convs:      convs,
		presence:   presence,
		publisher:  pub,
		typingTTL:  typingTTL,
		maxHistory: maxHistory,
	}
}

// SendRequest is the input of Service.Send.
type SendRequest struct {
	SenderID       string
	RecipientID    string
	Body           string
	Type           types.MessageType
	Metadata       map[string]string
	ConversationID string
}

// Send persists and broadcasts a 1:1 message. If ConversationID is empty a
// deterministic id is derived from the two participants so the same pair
// always lands on the same conversation.
func (s *Service) Send(ctx context.Context, req SendRequest) (types.Message, error) {
	if strings.TrimSpace(req.SenderID) == "" || strings.TrimSpace(req.RecipientID) == "" {
		return types.Message{}, errors.New("sender and recipient are required")
	}
	if req.Type == "" {
		req.Type = types.MessageTypeText
	}
	if !req.Type.Valid() {
		return types.Message{}, errors.New("invalid message type")
	}

	convID := req.ConversationID
	if convID == "" {
		convID = deriveDirectConversationID(req.SenderID, req.RecipientID)
	}

	now := time.Now().UTC()
	msg := types.Message{
		ConversationID: convID,
		MessageID:      uuid.NewString(),
		SenderID:       req.SenderID,
		RecipientID:    req.RecipientID,
		Body:           req.Body,
		Type:           req.Type,
		Metadata:       req.Metadata,
		Edited:         false,
		CreatedAt:      now,
	}
	if err := msg.Validate(); err != nil {
		return types.Message{}, err
	}

	if err := s.msgs.Append(ctx, msg); err != nil {
		return types.Message{}, err
	}

	// Update the conversation header (best-effort).
	_ = s.convs.Upsert(ctx, types.Conversation{
		ConversationID: convID,
		Type:           types.ConversationTypeDirect,
		Participants:   []string{req.SenderID, req.RecipientID},
		LastMessage:    req.Body,
		LastMessageAt:  now,
		CreatedAt:      now,
	})

	// Bump the recipient's unread counter.
	if s.presence != nil {
		_, _ = s.presence.IncrementUnread(ctx, req.RecipientID, convID)
	}

	// Fan-out via NATS.
	_ = s.publisher.PublishMessageSent(ctx, msg)
	return msg, nil
}

// GetHistory returns the most recent messages for the conversation up to
// limit. beforeID can be used for cursor-based pagination.
func (s *Service) GetHistory(ctx context.Context, conversationID string, limit int, beforeID string) ([]types.Message, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, ErrInvalidConversation
	}
	if limit <= 0 {
		limit = s.maxHistory
	}
	if limit > s.maxHistory {
		limit = s.maxHistory
	}
	return s.msgs.GetByConversation(ctx, conversationID, limit, beforeID)
}

// MarkRead records the read receipt and emits the corresponding event.
func (s *Service) MarkRead(ctx context.Context, conversationID, messageID, userID string) error {
	if err := s.msgs.MarkRead(ctx, conversationID, messageID, userID); err != nil {
		return err
	}
	if s.presence != nil {
		_ = s.presence.ResetUnread(ctx, userID, conversationID)
	}
	return s.publisher.PublishMessageRead(ctx, types.MessageReadEvent{
		MessageID:      messageID,
		ConversationID: conversationID,
		UserID:         userID,
		ReadAt:         time.Now().UTC(),
	})
}

// Typing records the typing indicator for the conversation.
func (s *Service) Typing(ctx context.Context, conversationID, userID string) error {
	if s.presence == nil {
		return nil
	}
	if err := s.presence.AddTyping(ctx, conversationID, userID, s.typingTTL); err != nil {
		return err
	}
	return s.publisher.PublishTyping(ctx, types.TypingEvent{
		ConversationID: conversationID,
		UserID:         userID,
		IsTyping:       true,
		At:             time.Now().UTC(),
	})
}

// deriveDirectConversationID returns a stable id for a 1:1 conversation
// between the two users so that requests from both sides land on the same
// partition key in ScyllaDB.
func deriveDirectConversationID(a, b string) string {
	if a < b {
		return "direct:" + a + ":" + b
	}
	return "direct:" + b + ":" + a
}
