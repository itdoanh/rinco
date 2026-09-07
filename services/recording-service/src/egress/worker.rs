//! Per-room egress worker.
//!
//! The worker subscribes to the SFU's RTP stream (in production via a
//! server-streaming Connect-RPC), demuxes the streams, and feeds them to the
//! compositor + audio mixer. The composited frame plus the mixed audio track
//! is encoded into MP4 and uploaded to S3.

use chrono::Utc;
use parking_lot::Mutex;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tracing::info;
use uuid::Uuid;

use crate::config::Config;
use crate::egress::audio_mixer::{AudioFrame, AudioMixer};
use crate::egress::compositor::{Compositor, Layout, VideoFrame};
use crate::egress::uploader::Uploader;
use crate::error::RecordingResult;
use crate::storage::metadata::{InMemoryMetadata, RecordingRow, RecordingStatus};
use crate::storage::tiered::TieredStorage;

/// Per-room worker state.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkerState {
    pub recording_id: Uuid,
    pub room_id: Uuid,
    pub tenant_id: Uuid,
    pub title: String,
    pub started_at: chrono::DateTime<Utc>,
    pub layout: Layout,
    pub gpu: bool,
}

pub struct EgressWorker {
    state: WorkerState,
    compositor: Compositor,
    mixer: AudioMixer,
    uploader: Uploader,
    storage: TieredStorage,
    meta: InMemoryMetadata,
    /// Buffer of the last composited video frame; flushed to disk by uploader.
    last_frame: Mutex<Option<VideoFrame>>,
    /// Number of mixed audio frames produced.
    audio_frames: Mutex<u64>,
    config: Config,
}

impl EgressWorker {
    /// Spawn a new worker for a given room.
    pub fn spawn(
        config: Config,
        storage: TieredStorage,
        meta: InMemoryMetadata,
        room_id: Uuid,
        tenant_id: Uuid,
        title: String,
        gpu: bool,
    ) -> RecordingResult<Arc<Self>> {
        let recording_id = Uuid::new_v4();
        let layout = Layout::Gallery { cols: 2, rows: 2 };
        let s3_key = format!("recordings/{}/{}/{}.mp4", tenant_id, room_id, recording_id);
        let row = RecordingRow {
            id: recording_id,
            room_id,
            tenant_id,
            title: title.clone(),
            started_at: Utc::now(),
            ended_at: None,
            duration_seconds: 0,
            size_bytes: 0,
            s3_key: s3_key.clone(),
            s3_bucket: config.s3_hot_bucket.clone(),
            tier: crate::storage::tiered::Tier::Hot,
            format: "mp4".into(),
            status: RecordingStatus::Recording,
            transcript_url: None,
            thumbnail_url: None,
        };
        meta.insert(row);
        let worker = Arc::new(Self {
            state: WorkerState {
                recording_id,
                room_id,
                tenant_id,
                title,
                started_at: Utc::now(),
                layout,
                gpu,
            },
            compositor: Compositor::new(1280, 720, gpu),
            mixer: AudioMixer::new(48_000, 2),
            uploader: Uploader::new(s3_key),
            storage,
            meta,
            last_frame: Mutex::new(None),
            audio_frames: Mutex::new(0),
            config,
        });
        info!(recording_id = %recording_id, room_id = %room_id, "egress worker started");
        Ok(worker)
    }

    /// Submit a video frame from one participant.
    pub fn submit_video(&self, source: Uuid, frame: VideoFrame) -> RecordingResult<()> {
        self.compositor.add_tile(source, frame.clone());
        *self.last_frame.lock() = Some(frame);
        Ok(())
    }

    /// Submit an audio frame from one participant.
    pub fn submit_audio(&self, source: Uuid, frame: AudioFrame) -> RecordingResult<()> {
        self.mixer.push(source, frame);
        *self.audio_frames.lock() += 1;
        Ok(())
    }

    /// Stop the worker, flush the final upload, and persist metadata.
    pub async fn finalize(self: Arc<Self>) -> RecordingResult<Uuid> {
        let recording_id = self.state.recording_id;
        let composite = self.compositor.render();
        let mixed = self.mixer.mix_all();
        let bytes = self.uploader.encode_mp4(composite, mixed, self.config.gpu_available)?;
        let stream = futures::stream::once(async move { bytes });
        self.storage
            .upload_stream(&self.uploader.key(), Box::pin(stream))
            .await?;
        let size = self.meta.get(recording_id).map(|r| r.size_bytes).unwrap_or(0);
        let duration = (Utc::now() - self.state.started_at).num_seconds();
        self.meta.update(recording_id, |r| {
            r.status = RecordingStatus::Completed;
            r.ended_at = Some(Utc::now());
            r.duration_seconds = duration as i32;
            r.size_bytes = size;
        });
        info!(recording_id = %recording_id, "egress worker finalized");
        Ok(recording_id)
    }

    /// Idempotent cancel.
    pub fn cancel(self: Arc<Self>) {
        self.meta.update(self.state.recording_id, |r| {
            r.status = RecordingStatus::Failed;
            r.ended_at = Some(Utc::now());
        });
    }

    pub fn state(&self) -> &WorkerState {
        &self.state
    }
}
