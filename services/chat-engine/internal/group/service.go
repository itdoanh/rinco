// Package group implements the multi-party chat use-case: create groups,
// add/remove members, send messages to a group, retrieve history.
package group

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/chat-engine/internal/nats"
	"github.com/itdoanh/rinco/services/chat-engine/internal/storage"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// Service coordinates group chats.
type Service struct {
	msgs      storage.MessageStore
	groups    storage.GroupStore
	convs     storage.ConversationStore
	presence  storage.PresenceStore
	publisher nats.Publisher
	maxHist   int
}

// New builds a Service.
func New(msgs storage.MessageStore, groups storage.GroupStore, convs storage.ConversationStore, presence storage.PresenceStore, pub nats.Publisher, maxHist int) *Service {
	return &Service{
		msgs:      msgs,
		groups:    groups,
		convs:     convs,
		presence:  presence,
		publisher: pub,
		maxHist:   maxHist,
	}
}

// CreateInput is the input of Service.Create.
type CreateInput struct {
	Name    string
	Avatar  string
	OwnerID string
	Members []string
}

// Create creates a new group chat.
func (s *Service) Create(ctx context.Context, in CreateInput) (types.Group, error) {
	if in.Name == "" {
		return types.Group{}, errors.New("group name is required")
	}
	if in.OwnerID == "" {
		return types.Group{}, errors.New("owner_id is required")
	}

	members := append([]string{in.OwnerID}, in.Members...)
	dedup := dedupeStrings(members)

	g := types.Group{
		GroupID:   uuid.NewString(),
		Name:      in.Name,
		Avatar:    in.Avatar,
		OwnerID:   in.OwnerID,
		Members:   dedup,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.groups.Create(ctx, g); err != nil {
		return types.Group{}, err
	}
	if err := s.convs.Upsert(ctx, types.Conversation{
		ConversationID: "group:" + g.GroupID,
		Type:           types.ConversationTypeGroup,
		Participants:   dedup,
		CreatedAt:      g.CreatedAt,
	}); err != nil {
		return types.Group{}, err
	}
	_ = s.publisher.PublishGroupCreated(ctx, g)
	return g, nil
}

// AddMember appends a member to the group.
func (s *Service) AddMember(ctx context.Context, groupID, userID string) error {
	return s.groups.AddMember(ctx, groupID, userID)
}

// RemoveMember removes a member from the group.
func (s *Service) RemoveMember(ctx context.Context, groupID, userID string) error {
	return s.groups.RemoveMember(ctx, groupID, userID)
}

// ListGroups returns every group the user is a member of.
func (s *Service) ListGroups(ctx context.Context, userID string) ([]types.Group, error) {
	return s.groups.ListForUser(ctx, userID)
}

// ListMembers returns the membership list of a group.
func (s *Service) ListMembers(ctx context.Context, groupID string) ([]string, error) {
	return s.groups.ListMembers(ctx, groupID)
}

// SendGroupMessage persists a message addressed to a group.
func (s *Service) SendGroupMessage(ctx context.Context, groupID, senderID, body string, msgType types.MessageType, metadata map[string]string) (types.Message, error) {
	if body == "" {
		return types.Message{}, errors.New("body is required")
	}
	if msgType == "" {
		msgType = types.MessageTypeText
	}
	convID := "group:" + groupID
	now := time.Now().UTC()
	msg := types.Message{
		ConversationID: convID,
		MessageID:      uuid.NewString(),
		SenderID:       senderID,
		Body:           body,
		Type:           msgType,
		Metadata:       metadata,
		CreatedAt:      now,
	}
	if err := msg.Validate(); err != nil {
		return types.Message{}, err
	}
	if err := s.msgs.Append(ctx, msg); err != nil {
		return types.Message{}, err
	}

	// bump unread counter for everyone except the sender.
	if s.presence != nil {
		members, _ := s.groups.ListMembers(ctx, groupID)
		for _, m := range members {
			if m != senderID {
				_, _ = s.presence.IncrementUnread(ctx, m, convID)
			}
		}
	}

	_ = s.convs.Upsert(ctx, types.Conversation{
		ConversationID: convID,
		LastMessage:    body,
		LastMessageAt:  now,
	})
	_ = s.publisher.PublishMessageSent(ctx, msg)
	return msg, nil
}

// GetGroupHistory returns the most recent messages of the group.
func (s *Service) GetGroupHistory(ctx context.Context, groupID string, limit int) ([]types.Message, error) {
	if limit <= 0 || limit > s.maxHist {
		limit = s.maxHist
	}
	return s.msgs.GetByConversation(ctx, "group:"+groupID, limit, "")
}

// Delete removes a group and its conversation header.
func (s *Service) Delete(ctx context.Context, groupID string) error {
	_ = s.convs.Delete(ctx, "group:"+groupID)
	return s.groups.Delete(ctx, groupID)
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
