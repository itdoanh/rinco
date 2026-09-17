// Package memory contains the in-memory implementation of the storage
// interfaces declared in the parent storage package.
//
// The implementation is safe for concurrent use and is selected when
// CHAT_IN_MEMORY_MODE is set or when no ScyllaDB / Valkey configuration is
// provided. It is also the basis for unit tests.
package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/itdoanh/rinco/services/chat-engine/internal/storage"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// Store is an in-memory implementation of storage.Aggregator.
type Store struct {
	mu       sync.RWMutex
	messages map[string]map[string]types.Message       // conversation -> message_id -> message
	read     map[string]map[string]map[string]struct{} // conversation -> message_id -> user_id -> {}

	conversations map[string]types.Conversation
	groups        map[string]types.Group

	presence map[string]types.Presence
	typing   map[string]map[string]time.Time
	unread   map[string]map[string]int
}

// New returns an empty in-memory store.
func New() *Store {
	return &Store{
		messages:      make(map[string]map[string]types.Message),
		read:          make(map[string]map[string]map[string]struct{}),
		conversations: make(map[string]types.Conversation),
		groups:        make(map[string]types.Group),
		presence:      make(map[string]types.Presence),
		typing:        make(map[string]map[string]time.Time),
		unread:        make(map[string]map[string]int),
	}
}

// NewAggregator wraps New() into the storage.Aggregator interface.
func NewAggregator() storage.Aggregator { return New() }

// Messages returns the message store view.
func (s *Store) Messages() storage.MessageStore { return &messageStore{s} }

// Conversations returns the conversation store view.
func (s *Store) Conversations() storage.ConversationStore { return &conversationStore{s} }

// Groups returns the group store view.
func (s *Store) Groups() storage.GroupStore { return &groupStore{s} }

// Presence returns the presence store view.
func (s *Store) Presence() storage.PresenceStore { return &presenceStore{s} }

// -----------------------------------------------------------------------------
// Messages
// -----------------------------------------------------------------------------

type messageStore struct{ *Store }

func (s *messageStore) Append(_ context.Context, msg types.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.messages[msg.ConversationID]; !ok {
		s.messages[msg.ConversationID] = make(map[string]types.Message)
	}
	s.messages[msg.ConversationID][msg.MessageID] = msg
	return nil
}

func (s *messageStore) GetByConversation(_ context.Context, conversationID string, limit int, beforeID string) ([]types.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.messages[conversationID]
	if !ok {
		return []types.Message{}, nil
	}

	all := make([]types.Message, 0, len(conv))
	for _, m := range conv {
		all = append(all, m)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	if beforeID != "" {
		idx := -1
		for i, m := range all {
			if m.MessageID == beforeID {
				idx = i
				break
			}
		}
		if idx >= 0 {
			all = all[idx+1:]
		}
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (s *messageStore) GetByID(_ context.Context, conversationID, messageID string) (types.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if conv, ok := s.messages[conversationID]; ok {
		if m, ok := conv[messageID]; ok {
			return m, nil
		}
	}
	return types.Message{}, ErrNotFound
}

func (s *messageStore) MarkRead(_ context.Context, conversationID, messageID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.read[conversationID]; !ok {
		s.read[conversationID] = make(map[string]map[string]struct{})
	}
	if _, ok := s.read[conversationID][messageID]; !ok {
		s.read[conversationID][messageID] = make(map[string]struct{})
	}
	s.read[conversationID][messageID][userID] = struct{}{}
	return nil
}

// -----------------------------------------------------------------------------
// Conversations
// -----------------------------------------------------------------------------

type conversationStore struct{ *Store }

func (s *conversationStore) Upsert(_ context.Context, c types.Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversations[c.ConversationID] = c
	return nil
}

func (s *conversationStore) Get(_ context.Context, id string) (types.Conversation, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.conversations[id]
	return c, ok, nil
}

func (s *conversationStore) ListForUser(_ context.Context, userID string) ([]types.Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]types.Conversation, 0, len(s.conversations))
	for _, c := range s.conversations {
		for _, p := range c.Participants {
			if p == userID {
				out = append(out, c)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastMessageAt.After(out[j].LastMessageAt)
	})
	return out, nil
}

func (s *conversationStore) SetArchived(_ context.Context, id string, archived bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.conversations[id]; ok {
		c.Archived = archived
		s.conversations[id] = c
	}
	return nil
}

func (s *conversationStore) SetMuted(_ context.Context, id string, muted bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.conversations[id]; ok {
		c.Muted = muted
		s.conversations[id] = c
	}
	return nil
}

func (s *conversationStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conversations, id)
	return nil
}

// -----------------------------------------------------------------------------
// Groups
// -----------------------------------------------------------------------------

type groupStore struct{ *Store }

func (s *groupStore) Create(_ context.Context, g types.Group) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups[g.GroupID] = g
	return nil
}

func (s *groupStore) Get(_ context.Context, id string) (types.Group, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.groups[id]
	return g, ok, nil
}

func (s *groupStore) ListForUser(_ context.Context, userID string) ([]types.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]types.Group, 0)
	for _, g := range s.groups {
		for _, m := range g.Members {
			if m == userID {
				out = append(out, g)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *groupStore) AddMember(_ context.Context, groupID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[groupID]
	if !ok {
		return ErrNotFound
	}
	for _, m := range g.Members {
		if m == userID {
			return nil
		}
	}
	g.Members = append(g.Members, userID)
	s.groups[groupID] = g
	return nil
}

func (s *groupStore) RemoveMember(_ context.Context, groupID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[groupID]
	if !ok {
		return ErrNotFound
	}
	out := g.Members[:0]
	for _, m := range g.Members {
		if m != userID {
			out = append(out, m)
		}
	}
	g.Members = out
	s.groups[groupID] = g
	return nil
}

func (s *groupStore) ListMembers(_ context.Context, groupID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if g, ok := s.groups[groupID]; ok {
		out := make([]string, len(g.Members))
		copy(out, g.Members)
		return out, nil
	}
	return []string{}, nil
}

func (s *groupStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.groups, id)
	return nil
}

// -----------------------------------------------------------------------------
// Presence
// -----------------------------------------------------------------------------

type presenceStore struct{ *Store }

func (s *presenceStore) SetPresence(_ context.Context, p types.Presence, ttlSeconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.LastSeen = time.Now().UTC()
	s.presence[p.UserID] = p
	if ttlSeconds > 0 {
		go func(userID string) {
			time.Sleep(time.Duration(ttlSeconds) * time.Second)
			s.mu.Lock()
			defer s.mu.Unlock()
			if cur, ok := s.presence[userID]; ok {
				if time.Since(cur.LastSeen) >= time.Duration(ttlSeconds)*time.Second {
					cur.Status = "offline"
					s.presence[userID] = cur
				}
			}
		}(p.UserID)
	}
	return nil
}

func (s *presenceStore) GetPresence(_ context.Context, userID string) (types.Presence, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.presence[userID]
	return p, ok, nil
}

func (s *presenceStore) ListOnline(_ context.Context, limit int) ([]types.Presence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]types.Presence, 0, len(s.presence))
	for _, p := range s.presence {
		if p.Status == "online" {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastSeen.After(out[j].LastSeen)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *presenceStore) AddTyping(_ context.Context, conversationID, userID string, ttlSeconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.typing[conversationID]; !ok {
		s.typing[conversationID] = make(map[string]time.Time)
	}
	s.typing[conversationID][userID] = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	return nil
}

func (s *presenceStore) RemoveTyping(_ context.Context, conversationID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if conv, ok := s.typing[conversationID]; ok {
		delete(conv, userID)
	}
	return nil
}

func (s *presenceStore) ListTyping(_ context.Context, conversationID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0)
	now := time.Now()
	for userID, exp := range s.typing[conversationID] {
		if exp.After(now) {
			out = append(out, userID)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *presenceStore) IncrementUnread(_ context.Context, userID, conversationID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.unread[userID]; !ok {
		s.unread[userID] = make(map[string]int)
	}
	s.unread[userID][conversationID]++
	return s.unread[userID][conversationID], nil
}

func (s *presenceStore) GetUnread(_ context.Context, userID, conversationID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unread[userID][conversationID], nil
}

func (s *presenceStore) ResetUnread(_ context.Context, userID, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.unread[userID]; ok {
		m[conversationID] = 0
	}
	return nil
}

// ErrNotFound is returned when the requested entity does not exist.
var ErrNotFound = notFoundError{}

type notFoundError struct{}

func (notFoundError) Error() string { return "not found" }

// IsNotFound returns whether the supplied error is the storage not-found
// sentinel.
func IsNotFound(err error) bool {
	_, ok := err.(notFoundError)
	return ok
}
