// Package connectrpc — wire types.
//
// These types are the JSON payload contract used by the Connect-RPC and
// REST surfaces. They are intentionally hand-written to avoid the protoc
// toolchain.
package connectrpc

import (
	"context"

	"connectrpc.com/connect"
)

// -----------------------------------------------------------------------------
// Chat
// -----------------------------------------------------------------------------

// SendMessageRequest is the streaming send payload.
type SendMessageRequest struct {
	SenderID       string            `json:"sender_id"`
	RecipientID    string            `json:"recipient_id"`
	Body           string            `json:"body"`
	Type           string            `json:"type,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
}

// Message is the canonical wire shape of a chat message.
type Message struct {
	ConversationID string            `json:"conversation_id"`
	MessageID      string            `json:"message_id"`
	SenderID       string            `json:"sender_id"`
	RecipientID    string            `json:"recipient_id,omitempty"`
	Body           string            `json:"body"`
	Type           string            `json:"type"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Edited         bool              `json:"edited"`
	CreatedAtUnix  int64             `json:"created_at_unix"`
}

// ReceiveMessageRequest asks the server to stream messages.
type ReceiveMessageRequest struct {
	ConversationID string `json:"conversation_id"`
}

// GetHistoryRequest asks the server for past messages.
type GetHistoryRequest struct {
	UserID         string `json:"user_id"`
	PeerID         string `json:"peer_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	Limit          int32  `json:"limit"`
	BeforeID       string `json:"before_id,omitempty"`
}

// GetHistoryResponse contains the requested messages.
type GetHistoryResponse struct {
	Messages []Message `json:"messages"`
}

// MarkReadRequest is the body of MarkRead.
type MarkReadRequest struct {
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	UserID         string `json:"user_id"`
}

// Empty is the standard "no payload" response.
type Empty struct{}

// TypingRequest toggles the typing indicator.
type TypingRequest struct {
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	IsTyping       bool   `json:"is_typing"`
}

// PresenceRequest updates presence.
type PresenceRequest struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
	Device string `json:"device,omitempty"`
}

// PresenceResponse returns the recorded presence.
type PresenceResponse struct {
	UserID   string `json:"user_id"`
	Status   string `json:"status"`
	LastSeen int64  `json:"last_seen_unix"`
	Device   string `json:"device,omitempty"`
}

// -----------------------------------------------------------------------------
// Group
// -----------------------------------------------------------------------------

// CreateGroupRequest creates a new group chat.
type CreateGroupRequest struct {
	Name    string   `json:"name"`
	OwnerID string   `json:"owner_id"`
	Members []string `json:"members"`
	Avatar  string   `json:"avatar,omitempty"`
}

// Group is the canonical wire representation of a group.
type Group struct {
	GroupID   string   `json:"group_id"`
	Name      string   `json:"name"`
	OwnerID   string   `json:"owner_id"`
	Members   []string `json:"members"`
	Avatar    string   `json:"avatar,omitempty"`
	CreatedAt int64    `json:"created_at_unix"`
}

// GroupMemberRequest is used by AddMember, RemoveMember and ListMembers.
type GroupMemberRequest struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id,omitempty"`
}

// ListGroupsRequest asks for the user's groups.
type ListGroupsRequest struct {
	UserID string `json:"user_id"`
}

// ListGroupsResponse returns the user's groups.
type ListGroupsResponse struct {
	Groups []Group `json:"groups"`
}

// SendGroupMessageRequest posts a message to a group.
type SendGroupMessageRequest struct {
	GroupID  string            `json:"group_id"`
	SenderID string            `json:"sender_id"`
	Body     string            `json:"body"`
	Type     string            `json:"type,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GetGroupHistoryRequest asks for group history.
type GetGroupHistoryRequest struct {
	GroupID string `json:"group_id"`
	Limit   int32  `json:"limit"`
}

// GroupHistoryResponse returns the history.
type GroupHistoryResponse struct {
	Messages []Message `json:"messages"`
}

// ListMembersResponse lists members.
type ListMembersResponse struct {
	Members []string `json:"members"`
}

// -----------------------------------------------------------------------------
// Conversation
// -----------------------------------------------------------------------------

// ListConversationsRequest asks for the user's conversations.
type ListConversationsRequest struct {
	UserID string `json:"user_id"`
}

// Conversation is the wire form of a conversation summary.
type Conversation struct {
	ConversationID string   `json:"conversation_id"`
	Type           string   `json:"type"`
	Participants   []string `json:"participants"`
	LastMessage    string   `json:"last_message,omitempty"`
	LastMessageAt  int64    `json:"last_message_at_unix"`
	CreatedAt      int64    `json:"created_at_unix"`
	UnreadCount    int32    `json:"unread_count"`
	Archived       bool     `json:"archived"`
	Muted          bool     `json:"muted"`
}

// ListConversationsResponse returns the user's conversations.
type ListConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
}

// ConversationToggleRequest flips archived / muted.
type ConversationToggleRequest struct {
	ConversationID string `json:"conversation_id"`
	Value          bool   `json:"value"`
}

// DeleteConversationRequest asks for deletion.
type DeleteConversationRequest struct {
	ConversationID string `json:"conversation_id"`
}

// -----------------------------------------------------------------------------
// Simple handlers
// -----------------------------------------------------------------------------
//
// connect-go's "simple" handlers take a strongly-typed req and return a
// strongly-typed res. The following wrappers bridge the connect-go RPC
// types to the request/response pointers used by the typed methods above.

// GetHistorySimple adapts ChatService.GetHistory to NewUnaryHandlerSimple.
func (s *ChatService) GetHistorySimple(ctx context.Context, req *GetHistoryRequest) (*GetHistoryResponse, error) {
	res, err := s.GetHistory(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// MarkReadSimple adapts ChatService.MarkRead.
func (s *ChatService) MarkReadSimple(ctx context.Context, req *MarkReadRequest) (*Empty, error) {
	if _, err := s.MarkRead(ctx, connect.NewRequest(req)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// TypingSimple adapts ChatService.Typing.
func (s *ChatService) TypingSimple(ctx context.Context, req *TypingRequest) (*Empty, error) {
	if _, err := s.Typing(ctx, connect.NewRequest(req)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// PresenceSimple adapts ChatService.Presence.
func (s *ChatService) PresenceSimple(ctx context.Context, req *PresenceRequest) (*PresenceResponse, error) {
	res, err := s.Presence(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// CreateGroupSimple adapts GroupService.CreateGroup.
func (s *GroupService) CreateGroupSimple(ctx context.Context, req *CreateGroupRequest) (*Group, error) {
	res, err := s.CreateGroup(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// AddMemberSimple adapts GroupService.AddMember.
func (s *GroupService) AddMemberSimple(ctx context.Context, req *GroupMemberRequest) (*Empty, error) {
	if _, err := s.AddMember(ctx, connect.NewRequest(req)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// RemoveMemberSimple adapts GroupService.RemoveMember.
func (s *GroupService) RemoveMemberSimple(ctx context.Context, req *GroupMemberRequest) (*Empty, error) {
	if _, err := s.RemoveMember(ctx, connect.NewRequest(req)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// ListGroupsSimple adapts GroupService.ListGroups.
func (s *GroupService) ListGroupsSimple(ctx context.Context, req *ListGroupsRequest) (*ListGroupsResponse, error) {
	res, err := s.ListGroups(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// SendGroupMessageSimple adapts GroupService.SendGroupMessage.
func (s *GroupService) SendGroupMessageSimple(ctx context.Context, req *SendGroupMessageRequest) (*Message, error) {
	res, err := s.SendGroupMessage(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// GetGroupHistorySimple adapts GroupService.GetGroupHistory.
func (s *GroupService) GetGroupHistorySimple(ctx context.Context, req *GetGroupHistoryRequest) (*GroupHistoryResponse, error) {
	res, err := s.GetGroupHistory(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// ListMembersSimple adapts GroupService.ListMembers.
func (s *GroupService) ListMembersSimple(ctx context.Context, req *GroupMemberRequest) (*ListMembersResponse, error) {
	res, err := s.ListMembers(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// ListConversationsSimple adapts ConversationService.ListConversations.
func (s *ConversationService) ListConversationsSimple(ctx context.Context, req *ListConversationsRequest) (*ListConversationsResponse, error) {
	res, err := s.ListConversations(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// ArchiveConversationSimple adapts ConversationService.ArchiveConversation.
func (s *ConversationService) ArchiveConversationSimple(ctx context.Context, req *ConversationToggleRequest) (*Empty, error) {
	if _, err := s.ArchiveConversation(ctx, connect.NewRequest(req)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// DeleteConversationSimple adapts ConversationService.DeleteConversation.
func (s *ConversationService) DeleteConversationSimple(ctx context.Context, req *DeleteConversationRequest) (*Empty, error) {
	if _, err := s.DeleteConversation(ctx, connect.NewRequest(req)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// MuteConversationSimple adapts ConversationService.MuteConversation.
func (s *ConversationService) MuteConversationSimple(ctx context.Context, req *ConversationToggleRequest) (*Empty, error) {
	if _, err := s.MuteConversation(ctx, connect.NewRequest(req)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}
