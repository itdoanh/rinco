// Package nats publishes chat-engine events to NATS JetStream.
//
// The chat-engine emits the following subjects:
//
//   - chat.message.sent
//   - chat.message.read
//   - chat.presence.changed
//   - chat.typing
//   - chat.group.created
//
// When no NATS URL is configured or the connection cannot be established
// the publisher falls back to an in-process bus so local builds still
// produce observable behaviour.
package nats

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// Subject constants used by other services to subscribe.
const (
	SubjectMessageSent      = "chat.message.sent"
	SubjectMessageRead      = "chat.message.read"
	SubjectPresenceChanged  = "chat.presence.changed"
	SubjectTyping           = "chat.typing"
	SubjectGroupCreated     = "chat.group.created"
)

// Publisher is the contract used by handlers.
type Publisher interface {
	PublishMessageSent(ctx context.Context, m types.Message) error
	PublishMessageRead(ctx context.Context, e types.MessageReadEvent) error
	PublishPresence(ctx context.Context, p types.Presence) error
	PublishTyping(ctx context.Context, e types.TypingEvent) error
	PublishGroupCreated(ctx context.Context, g types.Group) error
	Close()
}

// New returns a publisher that connects to NATS when url is non-empty.
// When the connection cannot be made a local in-memory bus is returned so
// subscribers within the same process still receive events.
func New(url string) Publisher {
	if url == "" {
		return newLocalBus()
	}
	conn, err := nats.Connect(url, nats.MaxReconnects(5), nats.ReconnectWait(2*time.Second))
	if err != nil || conn == nil {
		return newLocalBus()
	}
	return &natsPublisher{conn: conn}
}

// -----------------------------------------------------------------------------
// NATS implementation
// -----------------------------------------------------------------------------

type natsPublisher struct {
	conn *nats.Conn
}

func (n *natsPublisher) PublishMessageSent(ctx context.Context, m types.Message) error {
	return n.publish(ctx, SubjectMessageSent, m)
}
func (n *natsPublisher) PublishMessageRead(ctx context.Context, e types.MessageReadEvent) error {
	return n.publish(ctx, SubjectMessageRead, e)
}
func (n *natsPublisher) PublishPresence(ctx context.Context, p types.Presence) error {
	return n.publish(ctx, SubjectPresenceChanged, p)
}
func (n *natsPublisher) PublishTyping(ctx context.Context, e types.TypingEvent) error {
	return n.publish(ctx, SubjectTyping, e)
}
func (n *natsPublisher) PublishGroupCreated(ctx context.Context, g types.Group) error {
	return n.publish(ctx, SubjectGroupCreated, g)
}
func (n *natsPublisher) Close() {
	if n.conn != nil {
		n.conn.Close()
	}
}

func (n *natsPublisher) publish(_ context.Context, subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return n.conn.Publish(subject, data)
}

// -----------------------------------------------------------------------------
// In-process bus (used when NATS is not available)
// -----------------------------------------------------------------------------

type localBus struct {
	mu   sync.RWMutex
	subs map[string][]chan []byte
}

func newLocalBus() *localBus {
	return &localBus{subs: map[string][]chan []byte{}}
}

func (l *localBus) publish(_ context.Context, subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	l.mu.RLock()
	subs := append([]chan []byte{}, l.subs[subject]...)
	l.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- data:
		default:
		}
	}
	return nil
}

func (l *localBus) PublishMessageSent(ctx context.Context, m types.Message) error {
	return l.publish(ctx, SubjectMessageSent, m)
}
func (l *localBus) PublishMessageRead(ctx context.Context, e types.MessageReadEvent) error {
	return l.publish(ctx, SubjectMessageRead, e)
}
func (l *localBus) PublishPresence(ctx context.Context, p types.Presence) error {
	return l.publish(ctx, SubjectPresenceChanged, p)
}
func (l *localBus) PublishTyping(ctx context.Context, e types.TypingEvent) error {
	return l.publish(ctx, SubjectTyping, e)
}
func (l *localBus) PublishGroupCreated(ctx context.Context, g types.Group) error {
	return l.publish(ctx, SubjectGroupCreated, g)
}
func (l *localBus) Close() {}

// Subscribe registers a channel for the given subject. Used by tests and
// any other in-process consumer that does not have a NATS connection.
func (l *localBus) Subscribe(subject string, buffer int) <-chan []byte {
	ch := make(chan []byte, buffer)
	l.mu.Lock()
	l.subs[subject] = append(l.subs[subject], ch)
	l.mu.Unlock()
	return ch
}

// AsLocalBus returns the publisher as a *localBus when it actually is one,
// otherwise nil.
func AsLocalBus(p Publisher) *localBus {
	if lb, ok := p.(*localBus); ok {
		return lb
	}
	return nil
}
