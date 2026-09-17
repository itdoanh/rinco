// Package room manages the lifecycle of meeting rooms.
package room

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/types"
)

// Manager is a thread-safe in-memory store of meeting rooms.
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*types.Room
	max   int
}

// NewManager builds a Manager with the given maximum room count.
func NewManager(maxRooms int) *Manager {
	if maxRooms <= 0 {
		maxRooms = 1000
	}
	return &Manager{rooms: make(map[string]*types.Room), max: maxRooms}
}

// Create creates a new room. The supplied name is optional; if empty a
// random identifier is used. Returns the created room.
func (m *Manager) Create(name, ownerID string, maxParticipants int) (*types.Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.rooms) >= m.max {
		return nil, errors.New("max rooms reached")
	}
	if maxParticipants <= 0 {
		maxParticipants = 50
	}
	r := &types.Room{
		ID:              uuid.NewString(),
		Name:            name,
		OwnerID:         ownerID,
		CreatedAt:       time.Now().UTC(),
		Participants:    []types.Participant{},
		MaxParticipants: maxParticipants,
	}
	m.rooms[r.ID] = r
	return r, nil
}

// Get returns the room with the supplied id, or an error if absent.
func (m *Manager) Get(id string) (*types.Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[id]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// List returns every active room.
func (m *Manager) List() []*types.Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*types.Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		out = append(out, r)
	}
	return out
}

// Delete removes the room.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rooms[id]; !ok {
		return ErrNotFound
	}
	delete(m.rooms, id)
	return nil
}

// AddParticipant registers a participant in the room.
func (m *Manager) AddParticipant(roomID string, p types.Participant) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	for _, cur := range r.Participants {
		if cur.ID == p.ID {
			return nil // idempotent
		}
	}
	if len(r.Participants) >= r.MaxParticipants {
		return errors.New("max participants reached")
	}
	if p.JoinedAt.IsZero() {
		p.JoinedAt = time.Now().UTC()
	}
	r.Participants = append(r.Participants, p)
	return nil
}

// RemoveParticipant removes a participant from the room.
func (m *Manager) RemoveParticipant(roomID, participantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	out := r.Participants[:0]
	removed := false
	for _, p := range r.Participants {
		if p.ID == participantID {
			removed = true
			continue
		}
		out = append(out, p)
	}
	r.Participants = out
	if !removed {
		return ErrParticipantNotFound
	}
	return nil
}

// CleanupEmpty removes rooms whose last participant left more than ttl ago.
func (m *Manager) CleanupEmpty() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	removed := 0
	now := time.Now()
	for id, r := range m.rooms {
		if len(r.Participants) == 0 && now.Sub(r.CreatedAt) > 5*time.Minute {
			delete(m.rooms, id)
			removed++
		}
	}
	return removed
}

// Count returns the number of active rooms.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}

// ErrNotFound is returned when the requested room does not exist.
var ErrNotFound = errors.New("room not found")

// ErrParticipantNotFound is returned when removing a participant that is
// not part of the room.
var ErrParticipantNotFound = errors.New("participant not found")
