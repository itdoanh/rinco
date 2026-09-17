// Package connectrpc hosts the Connect-RPC handlers for the chat-engine.
//
// The handlers speak the Connect protocol (compatible with gRPC, gRPC-Web
// and Connect clients). To avoid pulling in a protoc toolchain, the wire
// types are plain Go structs serialised through a custom JSON codec that
// implements connect.Codec. The shape of the JSON payload is documented in
// the corresponding REST handler so browser clients can target either
// surface.
package connectrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/itdoanh/rinco/services/chat-engine/internal/chat"
	"github.com/itdoanh/rinco/services/chat-engine/internal/conversation"
	"github.com/itdoanh/rinco/services/chat-engine/internal/group"
	"github.com/itdoanh/rinco/services/chat-engine/internal/presence"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// ChatService exposes the 1:1 chat RPCs.
type ChatService struct {
	chat      *chat.Service
	presence  *presence.Service
	typingTTL int
}

// NewChatService builds a ChatService.
func NewChatService(c *chat.Service, p *presence.Service, typingTTL int) *ChatService {
	return &ChatService{chat: c, presence: p, typingTTL: typingTTL}
}

// SendMessage implements the client-streaming send RPC. Each request is
// persisted immediately; the function returns the most recently persisted
// message as the response.
func (s *ChatService) SendMessage(ctx context.Context, stream *connect.ClientStream[SendMessageRequest]) (*connect.Response[Message], error) {
	var last Message
	for stream.Receive() {
		req := *stream.Msg()
		if err := validateSend(&req); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		out, err := s.chat.Send(ctx, chat.SendRequest{
			SenderID:       req.SenderID,
			RecipientID:    req.RecipientID,
			Body:           req.Body,
			Type:           messageTypeFromString(req.Type),
			Metadata:       req.Metadata,
			ConversationID: req.ConversationID,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		last = messageFromDomain(out)
	}
	if err := stream.Err(); err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	return connect.NewResponse(&last), nil
}

// ReceiveMessage streams incoming messages to the client. The simple
// implementation polls the message store for the conversation every second
// and pushes new entries. A production deployment would subscribe to the
// NATS subject instead.
func (s *ChatService) ReceiveMessage(ctx context.Context, req *connect.Request[ReceiveMessageRequest], stream *connect.ServerStream[Message]) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	lastSeen := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		msgs, err := s.chat.GetHistory(ctx, req.Msg.ConversationID, 50, "")
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		for i := len(msgs) - 1; i >= 0; i-- {
			m := msgs[i]
			if !m.CreatedAt.After(lastSeen) {
				continue
			}
			lastSeen = m.CreatedAt
			if err := stream.Send(messagePtr(messageFromDomain(m))); err != nil {
				return err
			}
		}
	}
}

// GetHistory returns recent messages for the conversation.
func (s *ChatService) GetHistory(ctx context.Context, req *connect.Request[GetHistoryRequest]) (*connect.Response[GetHistoryResponse], error) {
	r := req.Msg
	convID := r.ConversationID
	if convID == "" && r.UserID != "" && r.PeerID != "" {
		convID = deriveDirect(r.UserID, r.PeerID)
	}
	if convID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("conversation_id or (user_id, peer_id) required"))
	}
	msgs, err := s.chat.GetHistory(ctx, convID, int(r.Limit), r.BeforeID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	res := &GetHistoryResponse{Messages: make([]Message, 0, len(msgs))}
	for _, m := range msgs {
		res.Messages = append(res.Messages, messageFromDomain(m))
	}
	return connect.NewResponse(res), nil
}

// MarkRead marks a message as read.
func (s *ChatService) MarkRead(ctx context.Context, req *connect.Request[MarkReadRequest]) (*connect.Response[Empty], error) {
	r := req.Msg
	if r.ConversationID == "" || r.MessageID == "" || r.UserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("conversation_id, message_id and user_id are required"))
	}
	if err := s.chat.MarkRead(ctx, r.ConversationID, r.MessageID, r.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&Empty{}), nil
}

// Typing records the typing indicator.
func (s *ChatService) Typing(ctx context.Context, req *connect.Request[TypingRequest]) (*connect.Response[Empty], error) {
	r := req.Msg
	if r.ConversationID == "" || r.UserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("conversation_id and user_id are required"))
	}
	if err := s.chat.Typing(ctx, r.ConversationID, r.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&Empty{}), nil
}

// Presence updates the user presence.
func (s *ChatService) Presence(ctx context.Context, req *connect.Request[PresenceRequest]) (*connect.Response[PresenceResponse], error) {
	r := req.Msg
	if r.UserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("user_id is required"))
	}
	status := r.Status
	if status == "" {
		status = "online"
	}
	_ = s.presence.Set(ctx, types.Presence{
		UserID: r.UserID,
		Status: status,
		Device: r.Device,
	})
	return connect.NewResponse(&PresenceResponse{
		UserID:   r.UserID,
		Status:   status,
		LastSeen: time.Now().Unix(),
		Device:   r.Device,
	}), nil
}

// -----------------------------------------------------------------------------
// GroupService
// -----------------------------------------------------------------------------

// GroupService exposes the group chat RPCs.
type GroupService struct {
	grp *group.Service
}

// NewGroupService builds a GroupService.
func NewGroupService(g *group.Service) *GroupService { return &GroupService{grp: g} }

// CreateGroup creates a new group.
func (s *GroupService) CreateGroup(ctx context.Context, req *connect.Request[CreateGroupRequest]) (*connect.Response[Group], error) {
	r := req.Msg
	g, err := s.grp.Create(ctx, group.CreateInput{
		Name:    r.Name,
		Avatar:  r.Avatar,
		OwnerID: r.OwnerID,
		Members: r.Members,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(groupToWire(g)), nil
}

// AddMember adds a member to the group.
func (s *GroupService) AddMember(ctx context.Context, req *connect.Request[GroupMemberRequest]) (*connect.Response[Empty], error) {
	if err := s.grp.AddMember(ctx, req.Msg.GroupID, req.Msg.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&Empty{}), nil
}

// RemoveMember removes a member from the group.
func (s *GroupService) RemoveMember(ctx context.Context, req *connect.Request[GroupMemberRequest]) (*connect.Response[Empty], error) {
	if err := s.grp.RemoveMember(ctx, req.Msg.GroupID, req.Msg.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&Empty{}), nil
}

// ListGroups lists the user's groups.
func (s *GroupService) ListGroups(ctx context.Context, req *connect.Request[ListGroupsRequest]) (*connect.Response[ListGroupsResponse], error) {
	groups, err := s.grp.ListGroups(ctx, req.Msg.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	res := &ListGroupsResponse{Groups: make([]Group, 0, len(groups))}
	for _, g := range groups {
		res.Groups = append(res.Groups, *groupToWire(g))
	}
	return connect.NewResponse(res), nil
}

// SendGroupMessage sends a message to the group.
func (s *GroupService) SendGroupMessage(ctx context.Context, req *connect.Request[SendGroupMessageRequest]) (*connect.Response[Message], error) {
	r := req.Msg
	m, err := s.grp.SendGroupMessage(ctx, r.GroupID, r.SenderID, r.Body, messageTypeFromString(r.Type), r.Metadata)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(messagePtr(messageFromDomain(m))), nil
}

// GetGroupHistory returns the group history.
func (s *GroupService) GetGroupHistory(ctx context.Context, req *connect.Request[GetGroupHistoryRequest]) (*connect.Response[GroupHistoryResponse], error) {
	msgs, err := s.grp.GetGroupHistory(ctx, req.Msg.GroupID, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	res := &GroupHistoryResponse{Messages: make([]Message, 0, len(msgs))}
	for _, m := range msgs {
		res.Messages = append(res.Messages, messageFromDomain(m))
	}
	return connect.NewResponse(res), nil
}

// ListMembers returns the members of the group.
func (s *GroupService) ListMembers(ctx context.Context, req *connect.Request[GroupMemberRequest]) (*connect.Response[ListMembersResponse], error) {
	members, err := s.grp.ListMembers(ctx, req.Msg.GroupID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ListMembersResponse{Members: members}), nil
}

// -----------------------------------------------------------------------------
// ConversationService
// -----------------------------------------------------------------------------

// ConversationService exposes the conversation-list RPCs.
type ConversationService struct {
	conv *conversation.Service
}

// NewConversationService builds a ConversationService.
func NewConversationService(c *conversation.Service) *ConversationService {
	return &ConversationService{conv: c}
}

// ListConversations lists the conversations for a user.
func (s *ConversationService) ListConversations(ctx context.Context, req *connect.Request[ListConversationsRequest]) (*connect.Response[ListConversationsResponse], error) {
	items, err := s.conv.ListConversations(ctx, req.Msg.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	res := &ListConversationsResponse{Conversations: make([]Conversation, 0, len(items))}
	for _, item := range items {
		res.Conversations = append(res.Conversations, Conversation{
			ConversationID: item.ConversationID,
			Type:           string(item.Type),
			Participants:   item.Participants,
			LastMessage:    item.LastMessage,
			LastMessageAt:  item.LastMessageAt.Unix(),
			CreatedAt:      item.CreatedAt.Unix(),
			UnreadCount:    int32(item.UnreadCount),
			Archived:       item.Archived,
			Muted:          item.Muted,
		})
	}
	return connect.NewResponse(res), nil
}

// ArchiveConversation toggles the archived flag.
func (s *ConversationService) ArchiveConversation(ctx context.Context, req *connect.Request[ConversationToggleRequest]) (*connect.Response[Empty], error) {
	if err := s.conv.Archive(ctx, req.Msg.ConversationID, req.Msg.Value); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&Empty{}), nil
}

// DeleteConversation removes the conversation.
func (s *ConversationService) DeleteConversation(ctx context.Context, req *connect.Request[DeleteConversationRequest]) (*connect.Response[Empty], error) {
	if err := s.conv.Delete(ctx, req.Msg.ConversationID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&Empty{}), nil
}

// MuteConversation toggles the muted flag.
func (s *ConversationService) MuteConversation(ctx context.Context, req *connect.Request[ConversationToggleRequest]) (*connect.Response[Empty], error) {
	if err := s.conv.Mute(ctx, req.Msg.ConversationID, req.Msg.Value); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&Empty{}), nil
}

// -----------------------------------------------------------------------------
// Mount
// -----------------------------------------------------------------------------

// MountRoutes registers every Connect-RPC handler on the supplied mux. The
// path scheme follows the Connect convention:
//
//   /rinco.chat.v1.ChatService/SendMessage
//   /rinco.chat.v1.GroupService/CreateGroup
//   /rinco.chat.v1.ConversationService/ListConversations
func MountRoutes(mux *http.ServeMux, chatSvc *ChatService, groupSvc *GroupService, convSvc *ConversationService) {
	codec := jsonCodecInstance

	register := func(path string, h *connect.Handler) {
		mux.Handle(path, h)
	}

	register("/rinco.chat.v1.ChatService/GetHistory", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.ChatService/GetHistory",
		chatSvc.GetHistorySimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.ChatService/MarkRead", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.ChatService/MarkRead",
		chatSvc.MarkReadSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.ChatService/Typing", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.ChatService/Typing",
		chatSvc.TypingSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.ChatService/Presence", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.ChatService/Presence",
		chatSvc.PresenceSimple,
		connect.WithCodec(codec),
	))

	register("/rinco.chat.v1.ChatService/SendMessage", connect.NewClientStreamHandler(
		"/rinco.chat.v1.ChatService/SendMessage",
		chatSvc.SendMessage,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.ChatService/ReceiveMessage", connect.NewServerStreamHandler(
		"/rinco.chat.v1.ChatService/ReceiveMessage",
		chatSvc.ReceiveMessage,
		connect.WithCodec(codec),
	))

	register("/rinco.chat.v1.GroupService/CreateGroup", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.GroupService/CreateGroup",
		groupSvc.CreateGroupSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.GroupService/AddMember", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.GroupService/AddMember",
		groupSvc.AddMemberSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.GroupService/RemoveMember", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.GroupService/RemoveMember",
		groupSvc.RemoveMemberSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.GroupService/ListGroups", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.GroupService/ListGroups",
		groupSvc.ListGroupsSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.GroupService/SendGroupMessage", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.GroupService/SendGroupMessage",
		groupSvc.SendGroupMessageSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.GroupService/GetGroupHistory", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.GroupService/GetGroupHistory",
		groupSvc.GetGroupHistorySimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.GroupService/ListMembers", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.GroupService/ListMembers",
		groupSvc.ListMembersSimple,
		connect.WithCodec(codec),
	))

	register("/rinco.chat.v1.ConversationService/ListConversations", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.ConversationService/ListConversations",
		convSvc.ListConversationsSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.ConversationService/ArchiveConversation", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.ConversationService/ArchiveConversation",
		convSvc.ArchiveConversationSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.ConversationService/DeleteConversation", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.ConversationService/DeleteConversation",
		convSvc.DeleteConversationSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.chat.v1.ConversationService/MuteConversation", connect.NewUnaryHandlerSimple(
		"/rinco.chat.v1.ConversationService/MuteConversation",
		convSvc.MuteConversationSimple,
		connect.WithCodec(codec),
	))
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func validateSend(r *SendMessageRequest) error {
	if strings.TrimSpace(r.SenderID) == "" {
		return errors.New("sender_id is required")
	}
	if strings.TrimSpace(r.RecipientID) == "" {
		return errors.New("recipient_id is required")
	}
	return nil
}

func messageTypeFromString(t string) types.MessageType {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "image":
		return types.MessageTypeImage
	case "file":
		return types.MessageTypeFile
	case "audio":
		return types.MessageTypeAudio
	default:
		return types.MessageTypeText
	}
}

func messageFromDomain(m types.Message) Message {
	return Message{
		ConversationID: m.ConversationID,
		MessageID:      m.MessageID,
		SenderID:       m.SenderID,
		RecipientID:    m.RecipientID,
		Body:           m.Body,
		Type:           string(m.Type),
		Metadata:       m.Metadata,
		Edited:         m.Edited,
		CreatedAtUnix:  m.CreatedAt.Unix(),
	}
}

func messagePtr(m Message) *Message { return &m }

func groupToWire(g types.Group) *Group {
	return &Group{
		GroupID:   g.GroupID,
		Name:      g.Name,
		OwnerID:   g.OwnerID,
		Members:   g.Members,
		Avatar:    g.Avatar,
		CreatedAt: g.CreatedAt.Unix(),
	}
}

func deriveDirect(a, b string) string {
	if a < b {
		return "direct:" + a + ":" + b
	}
	return "direct:" + b + ":" + a
}

// -----------------------------------------------------------------------------
// JSON codec
// -----------------------------------------------------------------------------

// jsonCodec implements connect.Codec using encoding/json. It lets us serve
// Connect-RPC requests without running protoc.
type jsonCodec struct{}

var jsonCodecInstance = &jsonCodec{}

// NewJSONCodec returns a singleton jsonCodec.
func NewJSONCodec() connect.Codec { return jsonCodecInstance }

func (jsonCodec) Name() string { return "json" }

func (jsonCodec) Marshal(message any) ([]byte, error) {
	if message == nil {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(message); err != nil {
		return nil, fmt.Errorf("encode %T: %w", message, err)
	}
	out := buf.Bytes()
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return out, nil
}

func (jsonCodec) Unmarshal(data []byte, message any) error {
	if len(data) == 0 {
		return errors.New("empty payload")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(message); err != nil {
		return fmt.Errorf("decode %T: %w", message, err)
	}
	return nil
}
