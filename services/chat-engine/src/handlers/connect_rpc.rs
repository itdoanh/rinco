//! Connect-RPC service definitions.
//!
//! Connect-RPC is generated from the same `.proto` files used by the Go
//! services – the wire format is plain gRPC. We hand-write the trait
//! implementations here to keep the dependency surface small and to allow the
//! handlers to be tested without a `tonic-build` step.

use std::sync::Arc;
use uuid::Uuid;

use crate::api;
use crate::error::{ChatError, ChatResult};
use crate::AppContext;

#[derive(Debug, Clone)]
pub struct SendMessageRequest {
    pub channel_id: Uuid,
    pub sender_id: Uuid,
    pub sender_device_id: u64,
    pub ciphertext: Vec<u8>,
    pub ratchet_pub: Vec<u8>,
    pub msg_number: u64,
    pub reply_to: Option<Uuid>,
    pub client_ts: Option<chrono::DateTime<chrono::Utc>>,
}

#[derive(Debug, Clone)]
pub struct ListMessagesRequest {
    pub channel_id: Uuid,
    pub cursor: Option<Uuid>,
    pub limit: i32,
}

#[derive(Debug, Clone)]
pub struct MarkReadRequest {
    pub user_id: Uuid,
    pub device_id: u64,
    pub channel_id: Uuid,
    pub msg_id: Uuid,
}

#[derive(Debug, Clone)]
pub struct RegisterDeviceRequest {
    pub user_id: Uuid,
    pub device_id: u64,
    pub prekey_count: usize,
}

#[derive(Debug, Clone)]
pub struct PreKeyBundleRequest {
    pub user_id: Uuid,
    pub device_id: u64,
}

#[derive(Debug, Clone)]
pub struct CreateChannelRequest {
    pub tenant_id: Uuid,
    pub kind: crate::api::types::ChannelKind,
    pub name: Option<String>,
    pub topic: Option<String>,
    pub members: Vec<Uuid>,
}

#[derive(Debug, Clone)]
pub struct AddGroupMemberRequest {
    pub channel_id: Uuid,
    pub user_id: Uuid,
}

#[derive(Debug, Clone)]
pub struct SetTypingRequest {
    pub channel_id: Uuid,
    pub user_id: Uuid,
    pub is_typing: bool,
}

#[derive(Debug, Clone)]
pub struct GetPresenceRequest {
    pub user_ids: Vec<Uuid>,
}

#[derive(Debug, Clone, Default)]
pub struct Empty;

#[async_trait::async_trait]
pub trait ChatRpc: Send + Sync {
    async fn send_message(&self, ctx: &AppContext, req: SendMessageRequest) -> ChatResult<api::types::Message>;
    async fn list_messages(&self, ctx: &AppContext, req: ListMessagesRequest) -> ChatResult<Vec<api::types::Message>>;
    async fn mark_read(&self, ctx: &AppContext, req: MarkReadRequest) -> ChatResult<Empty>;
    async fn register_device(&self, ctx: &AppContext, req: RegisterDeviceRequest) -> ChatResult<api::types::Device>;
    async fn upload_prekey_bundle(&self, ctx: &AppContext, _req: api::types::PreKeyBundle) -> ChatResult<Empty>;
    async fn get_prekey_bundle(&self, ctx: &AppContext, req: PreKeyBundleRequest) -> ChatResult<api::types::PreKeyBundle>;
    async fn create_channel(&self, ctx: &AppContext, req: CreateChannelRequest) -> ChatResult<api::types::Channel>;
    async fn add_group_member(&self, ctx: &AppContext, req: AddGroupMemberRequest) -> ChatResult<Empty>;
    async fn get_presence(&self, ctx: &AppContext, req: GetPresenceRequest) -> ChatResult<Vec<api::types::Presence>>;
    async fn set_typing(&self, ctx: &AppContext, req: SetTypingRequest) -> ChatResult<Empty>;
}

/// Default implementation backed by the in-process store layer.
#[derive(Default, Clone)]
pub struct DefaultChatRpc;

#[async_trait::async_trait]
impl ChatRpc for DefaultChatRpc {
    async fn send_message(&self, ctx: &AppContext, req: SendMessageRequest) -> ChatResult<api::types::Message> {
        let msg = api::types::Message {
            id: Uuid::new_v4(),
            tenant_id: ctx.tenant_id,
            channel_id: req.channel_id,
            sender_id: req.sender_id,
            sender_device_id: req.sender_device_id,
            recipient_device_id: None,
            ciphertext: req.ciphertext,
            ratchet_pub: req.ratchet_pub,
            msg_number: req.msg_number,
            attachments: vec![],
            reply_to: req.reply_to,
            server_ts: chrono::Utc::now(),
            client_ts: req.client_ts,
            edited: false,
            deleted: false,
        };
        api::messages::send_message(&ctx.db, &ctx.redis, msg).await
    }

    async fn list_messages(&self, ctx: &AppContext, req: ListMessagesRequest) -> ChatResult<Vec<api::types::Message>> {
        api::messages::list_messages(&ctx.db, req.channel_id, req.cursor, req.limit).await
    }

    async fn mark_read(&self, ctx: &AppContext, req: MarkReadRequest) -> ChatResult<Empty> {
        api::messages::mark_read(&ctx.db, &ctx.redis, req.user_id, req.device_id, req.channel_id, req.msg_id).await?;
        Ok(Empty)
    }

    async fn register_device(&self, ctx: &AppContext, req: RegisterDeviceRequest) -> ChatResult<api::types::Device> {
        let identity = ctx
            .identity_for(&req.user_id, req.device_id)
            .await
            .ok_or_else(|| ChatError::NotFound("identity not initialised".into()))?;
        api::devices::register_device(&ctx.db, req.user_id, req.device_id, &*identity, req.prekey_count).await
    }

    async fn upload_prekey_bundle(&self, ctx: &AppContext, _req: api::types::PreKeyBundle) -> ChatResult<Empty> {
        let _ = ctx;
        Ok(Empty)
    }

    async fn get_prekey_bundle(&self, ctx: &AppContext, req: PreKeyBundleRequest) -> ChatResult<api::types::PreKeyBundle> {
        api::devices::get_prekey_bundle(&ctx.db, req.user_id, req.device_id).await
    }

    async fn create_channel(&self, ctx: &AppContext, req: CreateChannelRequest) -> ChatResult<api::types::Channel> {
        api::channels::create_channel(&ctx.db, req.tenant_id, req.kind, req.name, req.topic, req.members).await
    }

    async fn add_group_member(&self, ctx: &AppContext, req: AddGroupMemberRequest) -> ChatResult<Empty> {
        api::channels::add_group_member(&ctx.db, req.channel_id, req.user_id).await?;
        Ok(Empty)
    }

    async fn get_presence(&self, ctx: &AppContext, req: GetPresenceRequest) -> ChatResult<Vec<api::types::Presence>> {
        crate::presence::presence_for(&ctx.redis, &req.user_ids).await
    }

    async fn set_typing(&self, ctx: &AppContext, req: SetTypingRequest) -> ChatResult<Empty> {
        ctx.redis.mark_typing(req.channel_id, req.user_id, req.is_typing).await?;
        Ok(Empty)
    }
}

/// Mount the Connect-RPC services on the given tonic server builder.
///
/// The generated `ChatServiceServer` would be added here in the production
/// binary; the trait + handlers above are the source of truth and are wired
/// in via `connect_rpc::DefaultChatRpc`.
pub fn mount(_ctx: Arc<AppContext>) {
    // Kept as a stub so the binary can call `mount` without pulling the
    // generated proto stubs. Replace with the generated server registration
    // in `chat-engine-bin`.
}
