// Package nats publishes SFU events.
//
// The SFU emits the following subjects:
//
//   - sfu.room.created
//   - sfu.room.deleted
//   - sfu.participant.joined
//   - sfu.participant.left
//   - recording.start
//   - recording.stop
package nats

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Publisher is the contract used by the rest of the service.
type Publisher interface {
	PublishRoomCreated(ctx context.Context, roomID string) error
	PublishRoomDeleted(ctx context.Context, roomID string) error
	PublishParticipantJoined(ctx context.Context, roomID, userID string) error
	PublishParticipantLeft(ctx context.Context, roomID, userID string) error
	PublishRecordingStart(ctx context.Context, roomID string) error
	PublishRecordingStop(ctx context.Context, roomID string) error
	Close()
}

// New returns a publisher that connects to NATS when url is non-empty and
// falls back to an in-process bus otherwise.
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

func (n *natsPublisher) publish(_ context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return n.conn.Publish(subject, data)
}

func (n *natsPublisher) PublishRoomCreated(ctx context.Context, roomID string) error {
	return n.publish(ctx, "sfu.room.created", map[string]string{"room_id": roomID})
}
func (n *natsPublisher) PublishRoomDeleted(ctx context.Context, roomID string) error {
	return n.publish(ctx, "sfu.room.deleted", map[string]string{"room_id": roomID})
}
func (n *natsPublisher) PublishParticipantJoined(ctx context.Context, roomID, userID string) error {
	return n.publish(ctx, "sfu.participant.joined", map[string]string{"room_id": roomID, "user_id": userID})
}
func (n *natsPublisher) PublishParticipantLeft(ctx context.Context, roomID, userID string) error {
	return n.publish(ctx, "sfu.participant.left", map[string]string{"room_id": roomID, "user_id": userID})
}
func (n *natsPublisher) PublishRecordingStart(ctx context.Context, roomID string) error {
	return n.publish(ctx, "recording.start", map[string]string{"room_id": roomID})
}
func (n *natsPublisher) PublishRecordingStop(ctx context.Context, roomID string) error {
	return n.publish(ctx, "recording.stop", map[string]string{"room_id": roomID})
}
func (n *natsPublisher) Close() {
	if n.conn != nil {
		n.conn.Close()
	}
}

// -----------------------------------------------------------------------------
// In-process bus
// -----------------------------------------------------------------------------

type localBus struct {
	mu   sync.RWMutex
	subs map[string][]chan []byte
}

func newLocalBus() *localBus { return &localBus{subs: map[string][]chan []byte{}} }

func (l *localBus) publish(_ context.Context, subject string, payload any) error {
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

func (l *localBus) PublishRoomCreated(ctx context.Context, roomID string) error {
	return l.publish(ctx, "sfu.room.created", map[string]string{"room_id": roomID})
}
func (l *localBus) PublishRoomDeleted(ctx context.Context, roomID string) error {
	return l.publish(ctx, "sfu.room.deleted", map[string]string{"room_id": roomID})
}
func (l *localBus) PublishParticipantJoined(ctx context.Context, roomID, userID string) error {
	return l.publish(ctx, "sfu.participant.joined", map[string]string{"room_id": roomID, "user_id": userID})
}
func (l *localBus) PublishParticipantLeft(ctx context.Context, roomID, userID string) error {
	return l.publish(ctx, "sfu.participant.left", map[string]string{"room_id": roomID, "user_id": userID})
}
func (l *localBus) PublishRecordingStart(ctx context.Context, roomID string) error {
	return l.publish(ctx, "recording.start", map[string]string{"room_id": roomID})
}
func (l *localBus) PublishRecordingStop(ctx context.Context, roomID string) error {
	return l.publish(ctx, "recording.stop", map[string]string{"room_id": roomID})
}
func (l *localBus) Close() {}

// Subscribe registers a channel for the given subject. Returns nil if the
// publisher is backed by NATS (use the real NATS connection for that).
func (l *localBus) Subscribe(subject string, buffer int) <-chan []byte {
	ch := make(chan []byte, buffer)
	l.mu.Lock()
	l.subs[subject] = append(l.subs[subject], ch)
	l.mu.Unlock()
	return ch
}
