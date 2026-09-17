// Package httpapi exposes the REST surface documented in the SFU
// contract.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/nats"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/peer"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/room"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/sfu"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/types"
)

// Server bundles the SFU dependencies.
type Server struct {
	Rooms     *room.Manager
	Peers     *peer.Manager
	Forwarder *sfu.Forwarder
	Publisher nats.Publisher
	Started   time.Time

	tokenSeq atomic.Uint64
}

// New returns a Server.
func New(r *room.Manager, p *peer.Manager, f *sfu.Forwarder, pub nats.Publisher) *Server {
	return &Server{Rooms: r, Peers: p, Forwarder: f, Publisher: pub, Started: time.Now()}
}

// Routes returns the http.ServeMux.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/rooms", s.handleRooms)
	mux.HandleFunc("/v1/rooms/", s.handleRoomByID)
	mux.HandleFunc("/v1/stats", s.handleStats)
	mux.HandleFunc("/ws/room/", s.handleWebSocket)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "healthy",
		"service":        "webrtc-sfu",
		"uptime_seconds": int64(time.Since(s.Started).Seconds()),
		"rooms":          s.Rooms.Count(),
		"participants":   s.Peers.Count(),
	})
}

type createRoomPayload struct {
	Name            string `json:"name,omitempty"`
	OwnerID         string `json:"owner_id"`
	MaxParticipants int    `json:"max_participants,omitempty"`
}

func (s *Server) handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var p createRoomPayload
		if err := decodeJSON(r, &p); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		rm, err := s.Rooms.Create(p.Name, p.OwnerID, p.MaxParticipants)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		_ = s.Publisher.PublishRoomCreated(r.Context(), rm.ID)
		writeJSON(w, http.StatusCreated, rm)
	case http.MethodGet:
		rooms := s.Rooms.List()
		if rooms == nil {
			rooms = []*types.Room{}
		}
		writeJSON(w, http.StatusOK, rooms)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRoomByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/rooms/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	roomID := parts[0]
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		rm, err := s.Rooms.Get(roomID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, rm)
	case len(parts) == 1 && r.Method == http.MethodDelete:
		if err := s.Rooms.Delete(roomID); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		_ = s.Publisher.PublishRoomDeleted(r.Context(), roomID)
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})

	case len(parts) >= 2 && parts[1] == "participants":
		rm, err := s.Rooms.Get(roomID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, rm.Participants)
		case http.MethodPost:
			var p struct {
				UserID   string            `json:"user_id"`
				Metadata map[string]string `json:"metadata,omitempty"`
			}
			if err := decodeJSON(r, &p); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			participant := types.Participant{
				ID:       uuid.NewString(),
				UserID:   p.UserID,
				JoinedAt: time.Now().UTC(),
				Metadata: p.Metadata,
			}
			if err := s.Rooms.AddParticipant(roomID, participant); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			s.Peers.Register(participant)
			s.Peers.AddToRoom(roomID, participant.ID)
			_ = s.Publisher.PublishParticipantJoined(r.Context(), roomID, p.UserID)
			writeJSON(w, http.StatusCreated, participant)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	case len(parts) >= 2 && parts[1] == "join" && r.Method == http.MethodPost:
		var p struct {
			UserID   string            `json:"user_id"`
			Metadata map[string]string `json:"metadata,omitempty"`
		}
		if err := decodeJSON(r, &p); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		participant := types.Participant{
			ID:       uuid.NewString(),
			UserID:   p.UserID,
			JoinedAt: time.Now().UTC(),
			Metadata: p.Metadata,
		}
		if err := s.Rooms.AddParticipant(roomID, participant); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		s.Peers.Register(participant)
		s.Peers.AddToRoom(roomID, participant.ID)
		s.tokenSeq.Add(1)
		writeJSON(w, http.StatusOK, map[string]any{
			"participant_id": participant.ID,
			"token":          uuid.NewString(),
			"room_id":        roomID,
		})

	case len(parts) >= 2 && parts[1] == "leave" && r.Method == http.MethodPost:
		var p struct {
			ParticipantID string `json:"participant_id"`
		}
		if err := decodeJSON(r, &p); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.Rooms.RemoveParticipant(roomID, p.ParticipantID); err != nil && !errors.Is(err, room.ErrParticipantNotFound) {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		s.Forwarder.Unsubscribe(roomID, p.ParticipantID)
		s.Peers.Remove(p.ParticipantID)
		_ = s.Publisher.PublishParticipantLeft(r.Context(), roomID, p.ParticipantID)
		writeJSON(w, http.StatusOK, map[string]bool{"left": true})

	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats := s.Forwarder.Stats()
	writeJSON(w, http.StatusOK, stats)
}

// handleWebSocket returns a stub 200 response for the /ws/room endpoint.
// A production deployment wires up Pion's WebSocket transport here.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/room/")
	if roomID == "" {
		writeError(w, http.StatusBadRequest, errors.New("room id required"))
		return
	}
	if _, err := s.Rooms.Get(roomID); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"room_id": roomID,
		"status":  "websocket endpoint ready",
		"hint":    "use the connect-rpc Publish/Subscribe streams for media transport",
	})
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("missing body")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
