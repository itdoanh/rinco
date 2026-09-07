//! Egress interface to the recording service.
//!
//! In the production build the egress is a Connect-RPC client that opens a
//! server-streaming RPC to `recording-service`. The test build can replace the
//! implementation with an in-memory pipe.

use async_trait::async_trait;
use bytes::Bytes;
use parking_lot::Mutex;
use std::sync::Arc;
use uuid::Uuid;

use crate::error::SfuResult;
use crate::sfu::forwarder::OutgoingRtp;

#[async_trait]
pub trait EgressSink: Send + Sync {
    /// Called when a new room is being recorded.
    async fn on_recording_start(&self, room_id: Uuid, tenant_id: Uuid) -> SfuResult<()>;
    /// Called for every RTP packet that should be persisted.
    async fn on_rtp(&self, room_id: Uuid, source: Uuid, pkt: OutgoingRtp) -> SfuResult<()>;
    /// Called when recording stops.
    async fn on_recording_stop(&self, room_id: Uuid) -> SfuResult<()>;
}

/// No-op sink used in unit tests.
#[derive(Default, Clone)]
pub struct NullEgress {
    pub started: Arc<Mutex<Vec<(Uuid, Uuid)>>>,
    pub stopped: Arc<Mutex<Vec<Uuid>>>,
    pub packets: Arc<Mutex<u64>>,
}

#[async_trait]
impl EgressSink for NullEgress {
    async fn on_recording_start(&self, room_id: Uuid, tenant_id: Uuid) -> SfuResult<()> {
        self.started.lock().push((room_id, tenant_id));
        Ok(())
    }

    async fn on_rtp(&self, _room_id: Uuid, _source: Uuid, _pkt: OutgoingRtp) -> SfuResult<()> {
        *self.packets.lock() += 1;
        Ok(())
    }

    async fn on_recording_stop(&self, room_id: Uuid) -> SfuResult<()> {
        self.stopped.lock().push(room_id);
        Ok(())
    }
}

/// Buffered sink used for tests; records the raw bytes per stream.
#[derive(Default, Clone)]
pub struct BufferEgress {
    pub buffers: Arc<Mutex<Vec<(Uuid, Uuid, Bytes)>>>,
}

#[async_trait]
impl EgressSink for BufferEgress {
    async fn on_recording_start(&self, _room_id: Uuid, _tenant_id: Uuid) -> SfuResult<()> {
        Ok(())
    }
    async fn on_rtp(&self, room_id: Uuid, source: Uuid, pkt: OutgoingRtp) -> SfuResult<()> {
        self.buffers.lock().push((room_id, source, pkt.payload));
        Ok(())
    }
    async fn on_recording_stop(&self, _room_id: Uuid) -> SfuResult<()> {
        Ok(())
    }
}
