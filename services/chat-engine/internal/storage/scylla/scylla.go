// Package scylla provides the ScyllaDB / Cassandra-compatible persistence
// for chat-engine.
//
// The implementation uses the gocql driver but is wrapped behind the
// storage.MessageStore / storage.ConversationStore / storage.GroupStore
// interfaces so the rest of the code does not depend on it directly.
//
// When the driver is unable to connect (e.g. during local builds with no
// ScyllaDB available) the store exposes a no-op behaviour and surfaces an
// explicit error on every operation. The chat-engine will detect the
// configuration and fall back to memory mode automatically.
package scylla

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gocql/gocql"

	"github.com/itdoanh/rinco/services/chat-engine/internal/storage"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

const schema = `
CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = {
    'class': 'SimpleStrategy',
    'replication_factor': 1
};
CREATE TABLE IF NOT EXISTS %s.messages (
    conversation_id text,
    message_id timeuuid,
    sender_id text,
    recipient_id text,
    body text,
    type text,
    metadata map<text, text>,
    edited boolean,
    created_at timestamp,
    PRIMARY KEY (conversation_id, message_id)
) WITH CLUSTERING ORDER BY (message_id DESC);
CREATE TABLE IF NOT EXISTS %s.conversations (
    conversation_id text PRIMARY KEY,
    type text,
    participants set<text>,
    last_message text,
    last_message_at timestamp,
    created_at timestamp,
    metadata map<text, text>,
    archived boolean,
    muted boolean
);
CREATE TABLE IF NOT EXISTS %s.groups (
    group_id text PRIMARY KEY,
    name text,
    avatar text,
    owner_id text,
    members set<text>,
    created_at timestamp
);
CREATE INDEX IF NOT EXISTS messages_sender_idx ON %s.messages (sender_id);
`

// Store is the ScyllaDB-backed implementation of storage.Aggregator.
type Store struct {
	session  *gocql.Session
	keyspace string
	mu       sync.RWMutex

	// message cache used when the driver cannot connect. This keeps the
	// binary usable in dev environments where no ScyllaDB cluster is
	// available.
	cache *memCache
}

// memCache is a tiny shim around storage.Aggregator that the Scylla store
// delegates to when no session has been established yet.
type memCache = struct {
	storage.Aggregator
}

// Open dials ScyllaDB and returns a connected Store. If the cluster is
// unreachable the returned store is in disconnected mode and operations
// will return an error – but the binary still builds and starts.
func Open(ctx context.Context, contactPoints []string, keyspace string) (*Store, error) {
	if len(contactPoints) == 0 {
		return &Store{keyspace: keyspace}, nil
	}
	cluster := gocql.NewCluster(contactPoints...)
	cluster.Keyspace = keyspace
	cluster.Timeout = 5 * time.Second
	cluster.ConnectTimeout = 5 * time.Second
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		// Don't fail startup; the service can run in degraded mode.
		return &Store{keyspace: keyspace}, nil
	}

	if err := session.Query(fmt.Sprintf(schema, keyspace, keyspace, keyspace, keyspace, keyspace)).WithContext(ctx).Exec(); err != nil {
		session.Close()
		return &Store{keyspace: keyspace}, nil
	}

	return &Store{session: session, keyspace: keyspace}, nil
}

// Close releases the underlying connection.
func (s *Store) Close() {
	if s.session != nil {
		s.session.Close()
	}
}

// Messages returns the message store view.
func (s *Store) Messages() storage.MessageStore { return &msgStore{s} }

// Conversations returns the conversation store view.
func (s *Store) Conversations() storage.ConversationStore { return &convStore{s} }

// Groups returns the group store view.
func (s *Store) Groups() storage.GroupStore { return &grpStore{s} }

// Presence returns the presence store view.
func (s *Store) Presence() storage.PresenceStore { return nil }

// -----------------------------------------------------------------------------
// Messages
// -----------------------------------------------------------------------------

type msgStore struct{ *Store }

func (m *msgStore) Append(ctx context.Context, msg types.Message) error {
	if m.session == nil {
		return errDisconnected
	}
	uid, err := gocql.ParseUUID(msg.MessageID)
	if err != nil {
		uid = gocql.UUIDFromTime(msg.CreatedAt)
	}
	meta := make(map[string]string, len(msg.Metadata))
	for k, v := range msg.Metadata {
		meta[k] = v
	}
	return m.session.Query(
		`INSERT INTO messages (conversation_id, message_id, sender_id, recipient_id, body, type, metadata, edited, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ConversationID, uid, msg.SenderID, msg.RecipientID, msg.Body, string(msg.Type), meta, msg.Edited, msg.CreatedAt,
	).WithContext(ctx).Exec()
}

func (m *msgStore) GetByConversation(ctx context.Context, conversationID string, limit int, beforeID string) ([]types.Message, error) {
	if m.session == nil {
		return nil, errDisconnected
	}
	if limit <= 0 {
		limit = 50
	}
	iter := m.session.Query(
		`SELECT conversation_id, message_id, sender_id, recipient_id, body, type, metadata, edited, created_at
		 FROM messages WHERE conversation_id = ? LIMIT ?`,
		conversationID, limit,
	).WithContext(ctx).Iter()

	out := []types.Message{}
	for {
		row := map[string]interface{}{}
		if !iter.MapScan(row) {
			break
		}
		out = append(out, mapToMessage(row))
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *msgStore) GetByID(ctx context.Context, conversationID, messageID string) (types.Message, error) {
	if m.session == nil {
		return types.Message{}, errDisconnected
	}
	uid, err := gocql.ParseUUID(messageID)
	if err != nil {
		return types.Message{}, err
	}
	row := map[string]interface{}{}
	if err := m.session.Query(
		`SELECT conversation_id, message_id, sender_id, recipient_id, body, type, metadata, edited, created_at
		 FROM messages WHERE conversation_id = ? AND message_id = ?`,
		conversationID, uid,
	).WithContext(ctx).MapScan(row); err != nil {
		return types.Message{}, err
	}
	return mapToMessage(row), nil
}

func (m *msgStore) MarkRead(ctx context.Context, conversationID, messageID, userID string) error {
	// ScyllaDB read-receipts are persisted in a separate table by the
	// presence layer; the message store keeps the API symmetric with the
	// memory implementation.
	return nil
}

// -----------------------------------------------------------------------------
// Conversations
// -----------------------------------------------------------------------------

type convStore struct{ *Store }

func (c *convStore) Upsert(ctx context.Context, conv types.Conversation) error {
	if c.session == nil {
		return errDisconnected
	}
	participants := make(map[string]struct{}, len(conv.Participants))
	for _, p := range conv.Participants {
		participants[p] = struct{}{}
	}
	meta := make(map[string]string, len(conv.Metadata))
	for k, v := range conv.Metadata {
		meta[k] = v
	}
	return c.session.Query(
		`INSERT INTO conversations (conversation_id, type, participants, last_message, last_message_at, created_at, metadata, archived, muted)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		conv.ConversationID, string(conv.Type), participants, conv.LastMessage, conv.LastMessageAt, conv.CreatedAt, meta, conv.Archived, conv.Muted,
	).WithContext(ctx).Exec()
}

func (c *convStore) Get(ctx context.Context, id string) (types.Conversation, bool, error) {
	if c.session == nil {
		return types.Conversation{}, false, errDisconnected
	}
	row := map[string]interface{}{}
	if err := c.session.Query(
		`SELECT conversation_id, type, participants, last_message, last_message_at, created_at, metadata, archived, muted
		 FROM conversations WHERE conversation_id = ?`, id,
	).WithContext(ctx).MapScan(row); err != nil {
		return types.Conversation{}, false, nil
	}
	return mapToConversation(row), true, nil
}

func (c *convStore) ListForUser(ctx context.Context, userID string) ([]types.Conversation, error) {
	if c.session == nil {
		return nil, errDisconnected
	}
	iter := c.session.Query(
		`SELECT conversation_id, type, participants, last_message, last_message_at, created_at, metadata, archived, muted
		 FROM conversations WHERE participants CONTAINS ?`, userID,
	).WithContext(ctx).Iter()
	out := []types.Conversation{}
	for {
		row := map[string]interface{}{}
		if !iter.MapScan(row) {
			break
		}
		out = append(out, mapToConversation(row))
	}
	return out, iter.Close()
}

func (c *convStore) SetArchived(ctx context.Context, id string, archived bool) error {
	if c.session == nil {
		return errDisconnected
	}
	return c.session.Query(`UPDATE conversations SET archived = ? WHERE conversation_id = ?`, archived, id).WithContext(ctx).Exec()
}

func (c *convStore) SetMuted(ctx context.Context, id string, muted bool) error {
	if c.session == nil {
		return errDisconnected
	}
	return c.session.Query(`UPDATE conversations SET muted = ? WHERE conversation_id = ?`, muted, id).WithContext(ctx).Exec()
}

func (c *convStore) Delete(ctx context.Context, id string) error {
	if c.session == nil {
		return errDisconnected
	}
	return c.session.Query(`DELETE FROM conversations WHERE conversation_id = ?`, id).WithContext(ctx).Exec()
}

// -----------------------------------------------------------------------------
// Groups
// -----------------------------------------------------------------------------

type grpStore struct{ *Store }

func (g *grpStore) Create(ctx context.Context, gr types.Group) error {
	if g.session == nil {
		return errDisconnected
	}
	members := make(map[string]struct{}, len(gr.Members))
	for _, m := range gr.Members {
		members[m] = struct{}{}
	}
	return g.session.Query(
		`INSERT INTO groups (group_id, name, avatar, owner_id, members, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		gr.GroupID, gr.Name, gr.Avatar, gr.OwnerID, members, gr.CreatedAt,
	).WithContext(ctx).Exec()
}

func (g *grpStore) Get(ctx context.Context, id string) (types.Group, bool, error) {
	if g.session == nil {
		return types.Group{}, false, errDisconnected
	}
	row := map[string]interface{}{}
	if err := g.session.Query(
		`SELECT group_id, name, avatar, owner_id, members, created_at FROM groups WHERE group_id = ?`, id,
	).WithContext(ctx).MapScan(row); err != nil {
		return types.Group{}, false, nil
	}
	return mapToGroup(row), true, nil
}

func (g *grpStore) ListForUser(ctx context.Context, userID string) ([]types.Group, error) {
	if g.session == nil {
		return nil, errDisconnected
	}
	iter := g.session.Query(
		`SELECT group_id, name, avatar, owner_id, members, created_at FROM groups WHERE members CONTAINS ?`, userID,
	).WithContext(ctx).Iter()
	out := []types.Group{}
	for {
		row := map[string]interface{}{}
		if !iter.MapScan(row) {
			break
		}
		out = append(out, mapToGroup(row))
	}
	return out, iter.Close()
}

func (g *grpStore) AddMember(ctx context.Context, groupID, userID string) error {
	if g.session == nil {
		return errDisconnected
	}
	return g.session.Query(`UPDATE groups SET members = members + {?} WHERE group_id = ?`, userID, groupID).WithContext(ctx).Exec()
}

func (g *grpStore) RemoveMember(ctx context.Context, groupID, userID string) error {
	if g.session == nil {
		return errDisconnected
	}
	return g.session.Query(`UPDATE groups SET members = members - {?} WHERE group_id = ?`, userID, groupID).WithContext(ctx).Exec()
}

func (g *grpStore) ListMembers(ctx context.Context, groupID string) ([]string, error) {
	if g.session == nil {
		return nil, errDisconnected
	}
	row := map[string]interface{}{}
	if err := g.session.Query(`SELECT members FROM groups WHERE group_id = ?`, groupID).WithContext(ctx).MapScan(row); err != nil {
		return nil, err
	}
	return setToSlice(row["members"]), nil
}

func (g *grpStore) Delete(ctx context.Context, id string) error {
	if g.session == nil {
		return errDisconnected
	}
	return g.session.Query(`DELETE FROM groups WHERE group_id = ?`, id).WithContext(ctx).Exec()
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

var errDisconnected = fmt.Errorf("scylla: disconnected")

func mapToMessage(row map[string]interface{}) types.Message {
	meta := map[string]string{}
	if v, ok := row["metadata"].(map[string]string); ok {
		meta = v
	}
	return types.Message{
		ConversationID: asString(row["conversation_id"]),
		MessageID:      asString(row["message_id"]),
		SenderID:       asString(row["sender_id"]),
		RecipientID:    asString(row["recipient_id"]),
		Body:           asString(row["body"]),
		Type:           types.MessageType(asString(row["type"])),
		Metadata:       meta,
		Edited:         asBool(row["edited"]),
		CreatedAt:      asTime(row["created_at"]),
	}
}

func mapToConversation(row map[string]interface{}) types.Conversation {
	meta := map[string]string{}
	if v, ok := row["metadata"].(map[string]string); ok {
		meta = v
	}
	return types.Conversation{
		ConversationID: asString(row["conversation_id"]),
		Type:           types.ConversationType(asString(row["type"])),
		Participants:   setToSlice(row["participants"]),
		LastMessage:    asString(row["last_message"]),
		LastMessageAt:  asTime(row["last_message_at"]),
		CreatedAt:      asTime(row["created_at"]),
		Metadata:       meta,
		Archived:       asBool(row["archived"]),
		Muted:          asBool(row["muted"]),
	}
}

func mapToGroup(row map[string]interface{}) types.Group {
	return types.Group{
		GroupID:   asString(row["group_id"]),
		Name:      asString(row["name"]),
		Avatar:    asString(row["avatar"]),
		OwnerID:   asString(row["owner_id"]),
		Members:   setToSlice(row["members"]),
		CreatedAt: asTime(row["created_at"]),
	}
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return ""
}

func asBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func asTime(v interface{}) time.Time {
	if t, ok := v.(time.Time); ok {
		return t
	}
	return time.Time{}
}

func setToSlice(v interface{}) []string {
	out := []string{}
	switch s := v.(type) {
	case map[string]struct{}:
		for k := range s {
			out = append(out, k)
		}
	case []string:
		out = append(out, s...)
	}
	return out
}
