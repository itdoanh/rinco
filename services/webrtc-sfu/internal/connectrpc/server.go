// Package connectrpc hosts the Connect-RPC surface for the SFU.
package connectrpc

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/nats"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/peer"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/room"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/sfu"
	"github.com/itdoanh/rinco/services/webrtc-sfu/internal/types"
)

// SFUService is the Connect-RPC implementation of SFUService.
type SFUService struct {
	rooms    *room.Manager
	peers    *peer.Manager
	fw       *sfu.Forwarder
	publisher nats.Publisher
}

// NewSFUService builds an SFUService.
func NewSFUService(rooms *room.Manager, peers *peer.Manager, fw *sfu.Forwarder, pub nats.Publisher) *SFUService {
	return &SFUService{rooms: rooms, peers: peers, fw: fw, publisher: pub}
}

// JoinRoom registers a participant and starts streaming.
func (s *SFUService) JoinRoom(ctx context.Context, req *connect.Request[JoinRoomRequest]) (*connect.Response[JoinRoomResponse], error) {
	r := req.Msg
	if r.RoomID == "" || r.UserID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("room_id and user_id are required"))
	}
	if _, err := s.rooms.Get(r.RoomID); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	participant := types.Participant{
		ID:       uuid.NewString(),
		UserID:   r.UserID,
		JoinedAt: timeNow(),
		Metadata: r.Metadata,
	}
	if err := s.rooms.AddParticipant(r.RoomID, participant); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	s.peers.Register(participant)
	s.peers.AddToRoom(r.RoomID, participant.ID)
	_ = s.publisher.PublishParticipantJoined(ctx, r.RoomID, r.UserID)

	return connect.NewResponse(&JoinRoomResponse{
		ParticipantID: participant.ID,
		RoomID:        r.RoomID,
		JoinedAt:      participant.JoinedAt.Unix(),
	}), nil
}

// LeaveRoom removes the participant from the room.
func (s *SFUService) LeaveRoom(ctx context.Context, req *connect.Request[LeaveRoomRequest]) (*connect.Response[Empty], error) {
	r := req.Msg
	if r.RoomID == "" || r.ParticipantID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("room_id and participant_id are required"))
	}
	if err := s.rooms.RemoveParticipant(r.RoomID, r.ParticipantID); err != nil && !errors.Is(err, room.ErrParticipantNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.fw.Unsubscribe(r.RoomID, r.ParticipantID)
	s.peers.Remove(r.ParticipantID)
	_ = s.publisher.PublishParticipantLeft(ctx, r.RoomID, r.ParticipantID)
	return connect.NewResponse(&Empty{}), nil
}

// GetStats returns current SFU statistics.
func (s *SFUService) GetStats(_ context.Context, _ *connect.Request[Empty]) (*connect.Response[StatsResponse], error) {
	stats := s.fw.Stats()
	return connect.NewResponse(&StatsResponse{
		Rooms:        int32(stats.Rooms),
		Participants: int32(stats.Participants),
		Tracks:       int32(stats.Tracks),
		BytesIn:      int64(stats.BytesIn),
		BytesOut:     int64(stats.BytesOut),
	}), nil
}

// Publish receives RTP events from clients and forwards them to the
// other subscribers in the room.
func (s *SFUService) Publish(ctx context.Context, stream *connect.ClientStream[TrackEvent]) (*connect.Response[Empty], error) {
	for stream.Receive() {
		ev := stream.Msg()
		s.fw.Publish(ctx, sfu.RTPEvent{
			RoomID:    ev.RoomID,
			UserID:    ev.UserID,
			Kind:      ev.Kind,
			Payload:   ev.RtpPacket,
			Timestamp: ev.Timestamp,
		})
	}
	if err := stream.Err(); err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	return connect.NewResponse(&Empty{}), nil
}

// Subscribe streams RTP events from the room to the client.
func (s *SFUService) Subscribe(ctx context.Context, req *connect.Request[TrackEvent], stream *connect.ServerStream[TrackEvent]) error {
	sub := s.fw.Subscribe(req.Msg.RoomID, req.Msg.UserID)
	defer s.fw.Unsubscribe(req.Msg.RoomID, req.Msg.UserID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-sub:
			if !ok {
				return nil
			}
			if err := stream.Send(&TrackEvent{
				RoomID:    ev.RoomID,
				UserID:    ev.UserID,
				Kind:      ev.Kind,
				RtpPacket: ev.Payload,
				Timestamp: ev.Timestamp,
			}); err != nil {
				return err
			}
		}
	}
}

// -----------------------------------------------------------------------------
// HTTP + REST + WebSocket entry points
// -----------------------------------------------------------------------------

// CreateRoomRequest is the HTTP /v1/rooms payload.
type CreateRoomRequest struct {
	Name           string            `json:"name,omitempty"`
	OwnerID        string            `json:"owner_id"`
	MaxParticipants int              `json:"max_participants,omitempty"`
}

// JoinRoomTokenRequest asks for a join token. Currently the SFU returns the
// participant ID directly; a future revision will sign it.
type JoinRoomTokenRequest struct {
	RoomID   string            `json:"room_id"`
	UserID   string            `json:"user_id"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// JoinRoomTokenResponse contains the issued participant ID.
type JoinRoomTokenResponse struct {
	ParticipantID string `json:"participant_id"`
	RoomID        string `json:"room_id"`
	Token         string `json:"token"`
}

// MountRoutes registers the Connect-RPC handlers on the supplied mux.
func MountRoutes(mux *http.ServeMux, svc *SFUService) {
	codec := jsonCodecInstance

	register := func(path string, h *connect.Handler) {
		mux.Handle(path, h)
	}

	register("/rinco.sfu.v1.SFUService/JoinRoom", connect.NewUnaryHandlerSimple(
		"/rinco.sfu.v1.SFUService/JoinRoom",
		svc.JoinRoomSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.sfu.v1.SFUService/LeaveRoom", connect.NewUnaryHandlerSimple(
		"/rinco.sfu.v1.SFUService/LeaveRoom",
		svc.LeaveRoomSimple,
		connect.WithCodec(codec),
	))
	register("/rinco.sfu.v1.SFUService/GetStats", connect.NewUnaryHandlerSimple(
		"/rinco.sfu.v1.SFUService/GetStats",
		svc.GetStatsSimple,
		connect.WithCodec(codec),
	))

	register("/rinco.sfu.v1.SFUService/Publish", connect.NewClientStreamHandler(
		"/rinco.sfu.v1.SFUService/Publish",
		svc.Publish,
		connect.WithCodec(codec),
	))
	register("/rinco.sfu.v1.SFUService/Subscribe", connect.NewServerStreamHandler(
		"/rinco.sfu.v1.SFUService/Subscribe",
		svc.Subscribe,
		connect.WithCodec(codec),
	))
}

// JoinRoomSimple adapts SFUService.JoinRoom to NewUnaryHandlerSimple.
func (s *SFUService) JoinRoomSimple(ctx context.Context, req *JoinRoomRequest) (*JoinRoomResponse, error) {
	res, err := s.JoinRoom(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// LeaveRoomSimple adapts SFUService.LeaveRoom.
func (s *SFUService) LeaveRoomSimple(ctx context.Context, req *LeaveRoomRequest) (*Empty, error) {
	if _, err := s.LeaveRoom(ctx, connect.NewRequest(req)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// GetStatsSimple adapts SFUService.GetStats.
func (s *SFUService) GetStatsSimple(ctx context.Context, req *Empty) (*StatsResponse, error) {
	res, err := s.GetStats(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

// silence unused imports
var _ = sync.Mutex{}
