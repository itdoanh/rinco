// Package httpapi exposes the REST fallback endpoints documented in the
// chat-engine contract.
//
// All endpoints share the same JSON shape as the Connect-RPC surface so
// that a client can target either protocol without translation.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/chat-engine/internal/chat"
	"github.com/itdoanh/rinco/services/chat-engine/internal/conversation"
	"github.com/itdoanh/rinco/services/chat-engine/internal/group"
	"github.com/itdoanh/rinco/services/chat-engine/internal/presence"
	"github.com/itdoanh/rinco/services/chat-engine/internal/types"
)

// Server bundles the services exposed by the REST surface.
type Server struct {
	Chat        *chat.Service
	Group       *group.Service
	Conv        *conversation.Service
	Presence    *presence.Service
	StartedAt   time.Time
}

// New returns a Server with the provided dependencies.
func New(c *chat.Service, g *group.Service, conv *conversation.Service, p *presence.Service) *Server {
	return &Server{Chat: c, Group: g, Conv: conv, Presence: p, StartedAt: time.Now()}
}

// Routes returns a fully-configured http.ServeMux.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleHealth)

	// REST fallback for the chat surface.
	mux.HandleFunc("/v1/messages", s.handleMessages)
	mux.HandleFunc("/v1/messages/", s.handleMessagesGet)
	mux.HandleFunc("/v1/conversations/", s.handleConversationByID)

	// Group surface.
	mux.HandleFunc("/v1/groups", s.handleGroups)
	mux.HandleFunc("/v1/groups/", s.handleGroupByID)

	// Presence surface.
	mux.HandleFunc("/v1/users/", s.handleUserPresence)
	mux.HandleFunc("/v1/typing/", s.handleTyping)

	return mux
}

// -----------------------------------------------------------------------------
// Health
// -----------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "healthy",
		"service":        "chat-engine",
		"uptime_seconds": int64(time.Since(s.StartedAt).Seconds()),
	})
}

// -----------------------------------------------------------------------------
// /v1/messages
// -----------------------------------------------------------------------------

type sendMessagePayload struct {
	SenderID       string            `json:"sender_id"`
	RecipientID    string            `json:"recipient_id"`
	GroupID        string            `json:"group_id,omitempty"`
	Body           string            `json:"body"`
	Type           string            `json:"type,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var p sendMessagePayload
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if p.SenderID == "" {
		writeError(w, http.StatusBadRequest, errors.New("sender_id is required"))
		return
	}
	if p.GroupID != "" {
		msg, err := s.Group.SendGroupMessage(r.Context(), p.GroupID, p.SenderID, p.Body, typeFromString(p.Type), p.Metadata)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, msg)
		return
	}
	if p.RecipientID == "" {
		writeError(w, http.StatusBadRequest, errors.New("recipient_id is required for direct messages"))
		return
	}
	msg, err := s.Chat.Send(r.Context(), chat.SendRequest{
		SenderID:       p.SenderID,
		RecipientID:    p.RecipientID,
		Body:           p.Body,
		Type:           typeFromString(p.Type),
		Metadata:       p.Metadata,
		ConversationID: p.ConversationID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

// GET /v1/messages/:conversation_id?limit=&before=
func (s *Server) handleMessagesGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	convID := strings.TrimPrefix(r.URL.Path, "/v1/messages/")
	if convID == "" {
		writeError(w, http.StatusBadRequest, errors.New("conversation_id is required"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}
	before := r.URL.Query().Get("before")

	msgs, err := s.Chat.GetHistory(r.Context(), convID, limit, before)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if msgs == nil {
		msgs = []types.Message{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conversation_id": convID,
		"messages":        msgs,
	})
}

// GET /v1/messages/:conversation_id/mark-read
func (s *Server) handleConversationByID(w http.ResponseWriter, r *http.Request) {
	// path: /v1/conversations/{id}/...
	path := strings.TrimPrefix(r.URL.Path, "/v1/conversations/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	convID := parts[0]
	action := parts[1]

	switch {
	case action == "archive" && r.Method == http.MethodPost:
		var p struct {
			Value bool `json:"value"`
		}
		_ = decodeJSON(r, &p)
		if err := s.Conv.Archive(r.Context(), convID, p.Value); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"archived": p.Value})
	case action == "mute" && r.Method == http.MethodPost:
		var p struct {
			Value bool `json:"value"`
		}
		_ = decodeJSON(r, &p)
		if err := s.Conv.Mute(r.Context(), convID, p.Value); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"muted": p.Value})
	case action == "" && r.Method == http.MethodDelete:
		if err := s.Conv.Delete(r.Context(), convID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// -----------------------------------------------------------------------------
// /v1/groups
// -----------------------------------------------------------------------------

type createGroupPayload struct {
	Name    string   `json:"name"`
	OwnerID string   `json:"owner_id"`
	Members []string `json:"members"`
	Avatar  string   `json:"avatar,omitempty"`
}

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var p createGroupPayload
		if err := decodeJSON(r, &p); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		g, err := s.Group.Create(r.Context(), group.CreateInput{
			Name: p.Name, OwnerID: p.OwnerID, Members: p.Members, Avatar: p.Avatar,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, g)
	case http.MethodGet:
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			writeError(w, http.StatusBadRequest, errors.New("user_id is required"))
			return
		}
		gs, err := s.Group.ListGroups(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if gs == nil {
			gs = []types.Group{}
		}
		writeJSON(w, http.StatusOK, gs)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGroupByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/groups/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	groupID := parts[0]
	switch {
	case len(parts) >= 2 && parts[1] == "members":
		switch r.Method {
		case http.MethodGet:
			members, err := s.Group.ListMembers(r.Context(), groupID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"members": members})
		case http.MethodPost:
			var p struct {
				UserID string `json:"user_id"`
			}
			if err := decodeJSON(r, &p); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if err := s.Group.AddMember(r.Context(), groupID, p.UserID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"added": true})
		case http.MethodDelete:
			userID := r.URL.Query().Get("user_id")
			if err := s.Group.RemoveMember(r.Context(), groupID, userID); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"removed": true})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case len(parts) >= 2 && parts[1] == "messages" && r.Method == http.MethodPost:
		var p struct {
			SenderID string            `json:"sender_id"`
			Body     string            `json:"body"`
			Type     string            `json:"type,omitempty"`
			Metadata map[string]string `json:"metadata,omitempty"`
		}
		if err := decodeJSON(r, &p); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		m, err := s.Group.SendGroupMessage(r.Context(), groupID, p.SenderID, p.Body, typeFromString(p.Type), p.Metadata)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, m)
	case len(parts) >= 2 && parts[1] == "messages" && r.Method == http.MethodGet:
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit == 0 {
			limit = 50
		}
		msgs, err := s.Group.GetGroupHistory(r.Context(), groupID, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if msgs == nil {
			msgs = []types.Message{}
		}
		writeJSON(w, http.StatusOK, msgs)
	case r.Method == http.MethodDelete:
		if err := s.Group.Delete(r.Context(), groupID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// -----------------------------------------------------------------------------
// /v1/users/:id/presence
// -----------------------------------------------------------------------------

func (s *Server) handleUserPresence(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "presence" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	userID := parts[0]
	switch r.Method {
	case http.MethodGet:
		p, ok, err := s.Presence.Get(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{
				"user_id": userID,
				"status":  "offline",
			})
			return
		}
		writeJSON(w, http.StatusOK, p)
	case http.MethodPost:
		var p struct {
			Status string `json:"status"`
			Device string `json:"device,omitempty"`
		}
		if err := decodeJSON(r, &p); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.Presence.Set(r.Context(), types.Presence{UserID: userID, Status: p.Status, Device: p.Device}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"updated": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// POST /v1/typing/:conversation_id   body: { user_id, is_typing }
func (s *Server) handleTyping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	convID := strings.TrimPrefix(r.URL.Path, "/v1/typing/")
	if convID == "" {
		writeError(w, http.StatusBadRequest, errors.New("conversation_id is required"))
		return
	}
	var p struct {
		UserID   string `json:"user_id"`
		IsTyping bool   `json:"is_typing"`
	}
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if p.UserID == "" {
		writeError(w, http.StatusBadRequest, errors.New("user_id is required"))
		return
	}
	if err := s.Chat.Typing(r.Context(), convID, p.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
			return
	}
	typing, _ := s.Presence.ListTyping(r.Context(), convID)
	writeJSON(w, http.StatusOK, map[string]any{
		"conversation_id": convID,
		"typing":          typing,
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

func typeFromString(t string) types.MessageType {
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

// Ping is exported for tests.
func Ping(ctx context.Context) error {
	_ = ctx
	_ = uuid.NewString
	return nil
}
