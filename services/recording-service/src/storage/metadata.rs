//! Recording metadata persisted in PostgreSQL.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::{RecordingError, RecordingResult};
use crate::storage::tiered::Tier;

/// Status of a recording object.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum RecordingStatus {
    Pending,
    Recording,
    Processing,
    Completed,
    Failed,
}

impl RecordingStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Pending => "pending",
            Self::Recording => "recording",
            Self::Processing => "processing",
            Self::Completed => "completed",
            Self::Failed => "failed",
        }
    }
    pub fn parse(s: &str) -> RecordingResult<Self> {
        match s {
            "pending" => Ok(Self::Pending),
            "recording" => Ok(Self::Recording),
            "processing" => Ok(Self::Processing),
            "completed" => Ok(Self::Completed),
            "failed" => Ok(Self::Failed),
            other => Err(RecordingError::InvalidRequest(format!("unknown status {other}"))),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecordingRow {
    pub id: Uuid,
    pub room_id: Uuid,
    pub tenant_id: Uuid,
    pub title: String,
    pub started_at: DateTime<Utc>,
    pub ended_at: Option<DateTime<Utc>>,
    pub duration_seconds: i32,
    pub size_bytes: i64,
    pub s3_key: String,
    pub s3_bucket: String,
    pub tier: Tier,
    pub format: String,
    pub status: RecordingStatus,
    pub transcript_url: Option<String>,
    pub thumbnail_url: Option<String>,
}

/// In-memory store used when the database is unreachable (dev mode).
#[derive(Default, Clone)]
pub struct InMemoryMetadata {
    rows: std::sync::Arc<parking_lot::RwLock<Vec<RecordingRow>>>,
}

impl InMemoryMetadata {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn insert(&self, row: RecordingRow) {
        self.rows.write().push(row);
    }

    pub fn get(&self, id: Uuid) -> Option<RecordingRow> {
        self.rows.read().iter().find(|r| r.id == id).cloned()
    }

    pub fn list(&self, room_id: Option<Uuid>, limit: i32) -> Vec<RecordingRow> {
        let mut rows: Vec<RecordingRow> = self
            .rows
            .read()
            .iter()
            .filter(|r| room_id.map_or(true, |id| r.room_id == id))
            .cloned()
            .collect();
        rows.sort_by_key(|r| r.started_at);
        rows.reverse();
        rows.truncate(limit as usize);
        rows
    }

    pub fn update(&self, id: Uuid, mut f: impl FnMut(&mut RecordingRow)) -> bool {
        let mut rows = self.rows.write();
        if let Some(r) = rows.iter_mut().find(|r| r.id == id) {
            f(r);
            true
        } else {
            false
        }
    }
}
