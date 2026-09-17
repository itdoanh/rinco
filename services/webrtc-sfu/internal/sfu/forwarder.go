// Package sfu implements the WebRTC forwarding logic.
//
// The Forwarder is intentionally transport-agnostic: it doesn't know about
// Pion, NATS or HTTP, but it can be plugged into each of them. This makes
// the unit tests trivial (no real network) and the binary easy to swap to
// a Rust or C++ implementation later.
package sfu

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/types"
)

// RTPEvent represents a single RTP packet for the Connect-RPC signalling
// channel.
type RTPEvent struct {
	RoomID     string
	UserID     string
	Kind       string
	Payload    []byte
	SequenceNo uint16
	Timestamp  int64
}

// Forwarder routes RTP packets from publishers to subscribers inside the
// same room. The implementation is intentionally minimal – production
// deployments should add a real WebRTC pipeline through Pion.
type Forwarder struct {
	mu       sync.RWMutex
	rooms    map[string]map[string]*subscriber // room_id -> user_id -> subscriber
	bytesIn  int64
	bytesOut int64
}

// NewForwarder returns an empty Forwarder.
func NewForwarder() *Forwarder {
	return &Forwarder{rooms: make(map[string]map[string]*subscriber)}
}

// subscriber represents a single subscriber inside a room.
type subscriber struct {
	queue  chan RTPEvent
	closed atomic.Bool
}

func newSubscriber(buffer int) *subscriber {
	if buffer <= 0 {
		buffer = 256
	}
	return &subscriber{queue: make(chan RTPEvent, buffer)}
}

// Subscribe registers a subscriber for the room and returns the channel
// of RTP events. Closing the channel is the caller's responsibility.
func (f *Forwarder) Subscribe(roomID, userID string) <-chan RTPEvent {
	sub := newSubscriber(256)
	f.mu.Lock()
	if _, ok := f.rooms[roomID]; !ok {
		f.rooms[roomID] = make(map[string]*subscriber)
	}
	f.rooms[roomID][userID] = sub
	f.mu.Unlock()
	return sub.queue
}

// Publish broadcasts the RTP event to every subscriber in the room except
// the publisher.
func (f *Forwarder) Publish(ctx context.Context, ev RTPEvent) {
	f.mu.RLock()
	subs := make([]*subscriber, 0, len(f.rooms[ev.RoomID]))
	for uid, sub := range f.rooms[ev.RoomID] {
		if uid == ev.UserID {
			continue
		}
		if sub.closed.Load() {
			continue
		}
		subs = append(subs, sub)
	}
	f.mu.RUnlock()

	atomic.AddInt64(&f.bytesIn, int64(len(ev.Payload)))
	for _, sub := range subs {
		select {
		case sub.queue <- ev:
			atomic.AddInt64(&f.bytesOut, int64(len(ev.Payload)))
		case <-ctx.Done():
			return
		default:
			// Drop on backpressure to avoid blocking the publisher.
		}
	}
}

// Unsubscribe closes the subscriber channel and removes it from the room.
func (f *Forwarder) Unsubscribe(roomID, userID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if subs, ok := f.rooms[roomID]; ok {
		if sub, ok := subs[userID]; ok {
			sub.closed.Store(true)
			close(sub.queue)
			delete(subs, userID)
		}
		if len(subs) == 0 {
			delete(f.rooms, roomID)
		}
	}
}

// Stats returns forwarder statistics.
func (f *Forwarder) Stats() types.Stats {
	f.mu.RLock()
	defer f.mu.RUnlock()
	rooms := len(f.rooms)
	parts := 0
	tracks := 0
	for _, subs := range f.rooms {
		parts += len(subs)
	}
	return types.Stats{
		Rooms:        rooms,
		Participants: parts,
		Tracks:       tracks,
		BytesIn:      int(atomic.LoadInt64(&f.bytesIn)),
		BytesOut:     int(atomic.LoadInt64(&f.bytesOut)),
	}
}

// TickStartTime is exported so health endpoints can compute uptime.
var TickStartTime = time.Now()
