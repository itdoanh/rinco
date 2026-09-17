// Package types contains the shared domain types for the chat-engine.
//
// The chat-engine persists messages into ScyllaDB, presence/typing state
// into Valkey and emits cross-service events to NATS JetStream. Domain
// types are JSON-serialisable and shared between the Connect-RPC handlers,
// the REST fallback and the storage adapters.
package types

import (
	"errors"
	"strings"
	"time"
)

// MessageType describes the body kind of a chat message.
type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeFile  MessageType = "file"
	MessageTypeAudio MessageType = "audio"
)

// Valid reports whether the value is a recognised message type.
func (m MessageType) Valid() bool {
	switch m {
	case MessageTypeText, MessageTypeImage, MessageTypeFile, MessageTypeAudio:
		return true
	}
	return false
}

// Message is a single chat message persisted to ScyllaDB.
//
// The primary key in ScyllaDB is (conversation_id, message_id) where
// message_id is a time-ordered UUID.
type Message struct {
	ConversationID string            `json:"conversation_id"`
	MessageID      string            `json:"message_id"`
	SenderID       string            `json:"sender_id"`
	RecipientID    string            `json:"recipient_id,omitempty"`
	Body           string            `json:"body"`
	Type           MessageType       `json:"type"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Edited         bool              `json:"edited"`
	CreatedAt      time.Time         `json:"created_at"`
}

// Validate performs basic sanity checks.
func (m *Message) Validate() error {
	if strings.TrimSpace(m.ConversationID) == "" {
		return errors.New("conversation_id is required")
	}
	if strings.TrimSpace(m.SenderID) == "" {
		return errors.New("sender_id is required")
	}
	if !m.Type.Valid() {
		return errors.New("invalid message type")
	}
	return nil
}

// ConversationType is the kind of a conversation.
type ConversationType string

const (
	ConversationTypeDirect ConversationType = "direct"
	ConversationTypeGroup  ConversationType = "group"
)

// Conversation represents either a 1:1 chat or a group chat header.
type Conversation struct {
	ConversationID string            `json:"conversation_id"`
	Type           ConversationType  `json:"type"`
	Participants   []string          `json:"participants"`
	LastMessage    string            `json:"last_message,omitempty"`
	LastMessageAt  time.Time         `json:"last_message_at"`
	CreatedAt      time.Time         `json:"created_at"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Archived       bool              `json:"archived"`
	Muted          bool              `json:"muted"`
}

// Group is a chat group with members.
type Group struct {
	GroupID   string    `json:"group_id"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar,omitempty"`
	OwnerID   string    `json:"owner_id"`
	Members   []string  `json:"members"`
	CreatedAt time.Time `json:"created_at"`
}

// Presence is the user online status tracked in Valkey.
type Presence struct {
	UserID   string    `json:"user_id"`
	Status   string    `json:"status"` // online, away, offline
	LastSeen time.Time `json:"last_seen"`
	Device   string    `json:"device,omitempty"`
}

// TypingEvent is emitted to subscribers when a user starts typing.
type TypingEvent struct {
	ConversationID string    `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	IsTyping       bool      `json:"is_typing"`
	At             time.Time `json:"at"`
}

// MessageReadEvent is emitted when a user marks a message as read.
type MessageReadEvent struct {
	MessageID      string    `json:"message_id"`
	ConversationID string    `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	ReadAt         time.Time `json:"read_at"`
}

// ConversationListItem is the summary used by the conversation list view.
type ConversationListItem struct {
	Conversation
	UnreadCount int `json:"unread_count"`
}
