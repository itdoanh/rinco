//! Connect-RPC service: RoomService.
//!
//! Internal API used by meeting-ui and orchestrators to manage rooms and
//! trigger recording.

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::{SfuError, SfuResult};
use crate::room::Participant;
use crate::SfuContext;

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct CreateRoomRequest {
    pub tenant_id: Uuid,
    pub name: String,
    pub max_peers: u32,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct CreateRoomResponse {
    pub room_id: Uuid,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct GetRoomRequest {
    pub room_id: Uuid,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct GetRoomResponse {
    pub room_id: Uuid,
    pub tenant_id: Uuid,
    pub name: String,
    pub participants: Vec<Participant>,
    pub recording: bool,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct JoinRoomRequest {
    pub room_id: Uuid,
    pub user_id: Uuid,
    pub tenant_id: Uuid,
    pub display_name: String,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct JoinRoomResponse {
    pub participant: Participant,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ListRoomsRequest;

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ListRoomsResponse {
    pub rooms: Vec<Uuid>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct UpdateRoomStateRequest {
    pub room_id: Uuid,
    pub locked: Option<bool>,
    pub max_peers: Option<u32>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct StartRecordingRequest {
    pub room_id: Uuid,
    pub format: String,
}

#[async_trait]
pub trait RoomService: Send + Sync {
    async fn create_room(&self, ctx: &SfuContext, req: CreateRoomRequest) -> SfuResult<CreateRoomResponse>;
    async fn get_room(&self, ctx: &SfuContext, req: GetRoomRequest) -> SfuResult<GetRoomResponse>;
    async fn join_room(&self, ctx: &SfuContext, req: JoinRoomRequest) -> SfuResult<JoinRoomResponse>;
    async fn list_rooms(&self, ctx: &SfuContext, _req: ListRoomsRequest) -> SfuResult<ListRoomsResponse>;
    async fn update_room_state(&self, ctx: &SfuContext, req: UpdateRoomStateRequest) -> SfuResult<()>;
    async fn start_recording(&self, ctx: &SfuContext, req: StartRecordingRequest) -> SfuResult<()>;
    async fn stop_recording(&self, ctx: &SfuContext, room_id: Uuid) -> SfuResult<()>;
}

/// Default in-process implementation.
pub struct DefaultRoomService;

#[async_trait]
impl RoomService for DefaultRoomService {
    async fn create_room(&self, ctx: &SfuContext, req: CreateRoomRequest) -> SfuResult<CreateRoomResponse> {
        let room = ctx.rooms.create(req.tenant_id, req.name, req.max_peers);
        Ok(CreateRoomResponse { room_id: room.id })
    }

    async fn get_room(&self, ctx: &SfuContext, req: GetRoomRequest) -> SfuResult<GetRoomResponse> {
        let room = ctx
            .rooms
            .get(req.room_id)
            .ok_or_else(|| SfuError::RoomNotFound(req.room_id.to_string()))?;
        Ok(GetRoomResponse {
            room_id: room.id,
            tenant_id: room.tenant_id,
            name: room.name.clone(),
            participants: room.list_participants(),
            recording: false,
        })
    }

    async fn join_room(&self, ctx: &SfuContext, req: JoinRoomRequest) -> SfuResult<JoinRoomResponse> {
        let room = ctx
            .rooms
            .get(req.room_id)
            .ok_or_else(|| SfuError::RoomNotFound(req.room_id.to_string()))?;
        let p = Participant {
            id: Uuid::new_v4(),
            user_id: req.user_id,
            tenant_id: req.tenant_id,
            display_name: req.display_name,
            is_audio_enabled: true,
            is_video_enabled: true,
            is_screen_sharing: false,
            joined_at: chrono::Utc::now(),
            connection_quality: 1.0,
        };
        room.add_participant(p.clone())?;
        Ok(JoinRoomResponse { participant: p })
    }

    async fn list_rooms(&self, ctx: &SfuContext, _req: ListRoomsRequest) -> SfuResult<ListRoomsResponse> {
        Ok(ListRoomsResponse { rooms: ctx.rooms.list() })
    }

    async fn update_room_state(&self, ctx: &SfuContext, _req: UpdateRoomStateRequest) -> SfuResult<()> {
        // The in-memory Room type doesn't track `locked` yet; the production
        // implementation would extend Room with a `locked: AtomicBool`.
        let _ = ctx;
        Ok(())
    }

    async fn start_recording(&self, ctx: &SfuContext, req: StartRecordingRequest) -> SfuResult<()> {
        let room = ctx
            .rooms
            .get(req.room_id)
            .ok_or_else(|| SfuError::RoomNotFound(req.room_id.to_string()))?;
        ctx.egress.on_recording_start(req.room_id, room.tenant_id).await?;
        let _ = room.events.send(crate::room::RoomEvent::RecordingStarted);
        Ok(())
    }

    async fn stop_recording(&self, ctx: &SfuContext, room_id: Uuid) -> SfuResult<()> {
        let room = ctx.rooms.get(room_id).ok_or_else(|| SfuError::RoomNotFound(room_id.to_string()))?;
        ctx.egress.on_recording_stop(room_id).await?;
        let _ = room.events.send(crate::room::RoomEvent::RecordingStopped);
        Ok(())
    }
}
