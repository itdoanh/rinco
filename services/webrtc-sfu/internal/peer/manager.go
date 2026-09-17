// Package peer manages per-participant state inside the SFU.
package peer

import (
	"sync"
	"time"

	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/types"
)

// Manager tracks the participants across rooms.
type Manager struct {
	mu    sync.RWMutex
	peers map[string]*types.Participant // peer_id -> participant
	byRoom map[string]map[string]struct{} // room_id -> peer_id set
}

// NewManager builds a Manager.
func NewManager() *Manager {
	return &Manager{
		peers:  make(map[string]*types.Participant),
		byRoom: make(map[string]map[string]struct{}),
	}
}

// Register stores the participant metadata.
func (m *Manager) Register(p types.Participant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.JoinedAt.IsZero() {
		p.JoinedAt = time.Now().UTC()
	}
	clone := p
	clone.Publishes = append([]string(nil), p.Publishes...)
	m.peers[p.ID] = &clone
	if _, ok := m.byRoom[p.ID]; !ok {
		// room id is not directly stored on participant; we use the
		// AddToRoom helper for that mapping.
	}
}

// AddToRoom binds the participant to a room.
func (m *Manager) AddToRoom(roomID, peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byRoom[roomID]; !ok {
		m.byRoom[roomID] = make(map[string]struct{})
	}
	m.byRoom[roomID][peerID] = struct{}{}
}

// Remove removes the participant from the manager entirely.
func (m *Manager) Remove(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.peers, peerID)
	for room, set := range m.byRoom {
		delete(set, peerID)
		m.byRoom[room] = set
	}
}

// Get returns the participant with the given id.
func (m *Manager) Get(peerID string) (types.Participant, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.peers[peerID]
	if !ok {
		return types.Participant{}, false
	}
	return *p, true
}

// ListByRoom returns every participant registered in the given room.
func (m *Manager) ListByRoom(roomID string) []types.Participant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []types.Participant{}
	for pid := range m.byRoom[roomID] {
		if p, ok := m.peers[pid]; ok {
			out = append(out, *p)
		}
	}
	return out
}

// Count returns the number of active participants.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.peers)
}
