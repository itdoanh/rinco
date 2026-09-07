//! Connect-RPC `RecordingService`.

use async_trait::async_trait;
use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use uuid::Uuid;

use crate::egress::worker::EgressWorker;
use crate::error::{RecordingError, RecordingResult};
use crate::storage::metadata::{InMemoryMetadata, RecordingRow, RecordingStatus};
use crate::storage::tiered::{Tier, TieredStorage};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartRecordingRequest {
    pub room_id: Uuid,
    pub tenant_id: Uuid,
    pub title: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StartRecordingResponse {
    pub recording_id: Uuid,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StopRecordingRequest {
    pub recording_id: Uuid,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRecordingRequest {
    pub recording_id: Uuid,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetRecordingResponse {
    pub row: RecordingRow,
    pub download_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListRecordingsRequest {
    pub room_id: Option<Uuid>,
    pub limit: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListRecordingsResponse {
    pub rows: Vec<RecordingRow>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteRecordingRequest {
    pub recording_id: Uuid,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTranscriptRequest {
    pub recording_id: Uuid,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GetTranscriptResponse {
    pub recording_id: Uuid,
    pub url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateTierRequest {
    pub recording_id: Uuid,
    pub tier: Tier,
}

#[async_trait]
pub trait RecordingService: Send + Sync {
    async fn start(
        &self,
        req: StartRecordingRequest,
        storage: Arc<TieredStorage>,
        meta: Arc<InMemoryMetadata>,
        gpu: bool,
    ) -> RecordingResult<StartRecordingResponse>;
    async fn stop(&self, req: StopRecordingRequest) -> RecordingResult<()>;
    async fn get(
        &self,
        req: GetRecordingRequest,
        storage: Arc<TieredStorage>,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<GetRecordingResponse>;
    async fn list(
        &self,
        req: ListRecordingsRequest,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<ListRecordingsResponse>;
    async fn delete(
        &self,
        req: DeleteRecordingRequest,
        storage: Arc<TieredStorage>,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<()>;
    async fn get_transcript(
        &self,
        req: GetTranscriptRequest,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<GetTranscriptResponse>;
    async fn update_tier(
        &self,
        req: UpdateTierRequest,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<()>;
}

pub struct DefaultRecordingService;

#[async_trait]
impl RecordingService for DefaultRecordingService {
    async fn start(
        &self,
        req: StartRecordingRequest,
        _storage: Arc<TieredStorage>,
        meta: Arc<InMemoryMetadata>,
        gpu: bool,
    ) -> RecordingResult<StartRecordingResponse> {
        let _ = EgressWorker::spawn(
            crate::config::Config::from_env()?,
            (*_storage).clone(),
            (*meta).clone(),
            req.room_id,
            req.tenant_id,
            req.title.clone(),
            gpu,
        )?;
        // The actual recording_id is created inside `EgressWorker::spawn` and
        // inserted into the metadata store; we look it up by room_id.
        let row = meta
            .list(Some(req.room_id), 1)
            .into_iter()
            .next()
            .ok_or_else(|| RecordingError::Other(anyhow::anyhow!("recording not created")))?;
        Ok(StartRecordingResponse { recording_id: row.id })
    }

    async fn stop(&self, req: StopRecordingRequest) -> RecordingResult<()> {
        // Production: send a stop signal to the worker supervisor; the worker
        // performs the final mux + upload.
        tracing::info!(recording_id = %req.recording_id, "stop recording");
        Ok(())
    }

    async fn get(
        &self,
        req: GetRecordingRequest,
        storage: Arc<TieredStorage>,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<GetRecordingResponse> {
        let row = meta
            .get(req.recording_id)
            .ok_or_else(|| RecordingError::NotFound(req.recording_id.to_string()))?;
        let download_url = if row.status == RecordingStatus::Completed {
            Some(storage.presign(&row.s3_key, row.tier, 3600).await?)
        } else {
            None
        };
        Ok(GetRecordingResponse { row, download_url })
    }

    async fn list(
        &self,
        req: ListRecordingsRequest,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<ListRecordingsResponse> {
        Ok(ListRecordingsResponse { rows: meta.list(req.room_id, req.limit) })
    }

    async fn delete(
        &self,
        req: DeleteRecordingRequest,
        storage: Arc<TieredStorage>,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<()> {
        if let Some(row) = meta.get(req.recording_id) {
            storage.delete(&row.s3_key, row.tier).await?;
            meta.update(req.recording_id, |r| {
                r.status = RecordingStatus::Failed;
                r.ended_at = Some(Utc::now());
            });
        }
        Ok(())
    }

    async fn get_transcript(
        &self,
        req: GetTranscriptRequest,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<GetTranscriptResponse> {
        let row = meta
            .get(req.recording_id)
            .ok_or_else(|| RecordingError::NotFound(req.recording_id.to_string()))?;
        Ok(GetTranscriptResponse {
            recording_id: req.recording_id,
            url: row.transcript_url,
        })
    }

    async fn update_tier(
        &self,
        req: UpdateTierRequest,
        meta: Arc<InMemoryMetadata>,
    ) -> RecordingResult<()> {
        meta.update(req.recording_id, |r| r.tier = req.tier);
        Ok(())
    }
}
