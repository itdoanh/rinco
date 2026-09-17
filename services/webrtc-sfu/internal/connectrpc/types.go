package connectrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// -----------------------------------------------------------------------------
// Wire types
// -----------------------------------------------------------------------------

// TrackEvent mirrors the Connect-RPC TrackEvent message.
type TrackEvent struct {
	RoomID    string `json:"room_id"`
	UserID    string `json:"user_id"`
	Kind      string `json:"kind"`
	RtpPacket []byte `json:"rtp_packet,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// JoinRoomRequest asks the server to register a participant.
type JoinRoomRequest struct {
	RoomID   string            `json:"room_id"`
	UserID   string            `json:"user_id"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// JoinRoomResponse returns the issued participant id.
type JoinRoomResponse struct {
	ParticipantID string `json:"participant_id"`
	RoomID        string `json:"room_id"`
	JoinedAt      int64  `json:"joined_at_unix"`
}

// LeaveRoomRequest asks the server to remove a participant.
type LeaveRoomRequest struct {
	RoomID        string `json:"room_id"`
	ParticipantID string `json:"participant_id"`
}

// Empty is the no-payload response.
type Empty struct{}

// StatsResponse returns SFU statistics.
type StatsResponse struct {
	Rooms        int32 `json:"rooms"`
	Participants int32 `json:"participants"`
	Tracks       int32 `json:"tracks"`
	BytesIn      int64 `json:"bytes_in"`
	BytesOut     int64 `json:"bytes_out"`
}

// -----------------------------------------------------------------------------
// JSON codec
// -----------------------------------------------------------------------------

type jsonCodec struct{}

var jsonCodecInstance = &jsonCodec{}

// NewJSONCodec returns the singleton json codec.
func NewJSONCodec() httpHandler { return jsonCodecInstance }

type httpHandler interface {
	http.Handler
	Name() string
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
}

func (jsonCodec) Name() string { return "json" }

func (jsonCodec) Marshal(message any) ([]byte, error) {
	if message == nil {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(message); err != nil {
		return nil, fmt.Errorf("encode %T: %w", message, err)
	}
	out := buf.Bytes()
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return out, nil
}

func (jsonCodec) Unmarshal(data []byte, message any) error {
	if len(data) == 0 {
		return errors.New("empty payload")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(message); err != nil {
		return fmt.Errorf("decode %T: %w", message, err)
	}
	return nil
}

// ServeHTTP is a stub that satisfies http.Handler; the codec is mounted via
// connect.WithCodec and never directly invoked through HTTP.
func (jsonCodec) ServeHTTP(http.ResponseWriter, *http.Request) {}

// timeNow is a swappable clock for testing.
var timeNow = func() time.Time { return time.Now().UTC() }

// Suppress unused import warnings.
var _ = context.Background
