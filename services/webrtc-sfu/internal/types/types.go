// Package types contains the shared domain types for the SFU.
package types

import "time"

// TrackKind is the kind of media track.
type TrackKind string

const (
	TrackKindAudio   TrackKind = "audio"
	TrackKindVideo   TrackKind = "video"
	TrackKindScreen  TrackKind = "screen"
)

// Participant is a user connected to a room.
type Participant struct {
	ID         string            `json:"id"`
	UserID     string            `json:"user_id"`
	JoinedAt   time.Time         `json:"joined_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Publishes  []string          `json:"publishes"`
	Recording  bool              `json:"recording"`
}

// Room is a meeting room.
type Room struct {
	ID            string        `json:"id"`
	Name          string        `json:"name,omitempty"`
	OwnerID       string        `json:"owner_id"`
	CreatedAt     time.Time     `json:"created_at"`
	Participants  []Participant `json:"participants"`
	Recording     bool          `json:"recording"`
	MaxParticipants int         `json:"max_participants"`
}

// Stats are SFU health statistics.
type Stats struct {
	Rooms        int `json:"rooms"`
	Participants int `json:"participants"`
	Tracks       int `json:"tracks"`
	BytesIn      int `json:"bytes_in"`
	BytesOut     int `json:"bytes_out"`
}
