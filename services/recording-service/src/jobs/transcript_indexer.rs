//! Transcript indexer – pushes transcripts into Meilisearch / OpenSearch.

use serde::Serialize;
use std::sync::Arc;
use tracing::info;
use uuid::Uuid;

use crate::egress::ai_pipeline::Transcript;
use crate::storage::metadata::InMemoryMetadata;

#[derive(Debug, Clone, Serialize)]
struct IndexDoc {
    id: String,
    recording_id: String,
    room_id: String,
    text: String,
    speaker: Option<String>,
    start_ms: i64,
    end_ms: i64,
}

#[async_trait::async_trait]
pub trait SearchClient: Send + Sync {
    async fn index(&self, doc: IndexDoc) -> anyhow::Result<()>;
}

pub struct HttpSearchClient {
    base_url: String,
    http: reqwest::Client,
}

impl HttpSearchClient {
    pub fn new(base_url: String) -> Self {
        let http = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .expect("reqwest");
        Self { base_url, http }
    }
}

#[async_trait::async_trait]
impl SearchClient for HttpSearchClient {
    async fn index(&self, doc: IndexDoc) -> anyhow::Result<()> {
        let url = format!("{}/indexes/transcripts/documents", self.base_url);
        self.http
            .post(url)
            .json(&doc)
            .send()
            .await?
            .error_for_status()?;
        Ok(())
    }
}

pub async fn run_once(
    search: Arc<dyn SearchClient>,
    meta: Arc<InMemoryMetadata>,
) -> anyhow::Result<()> {
    let rows = meta.list(None, 10_000);
    let mut indexed = 0u32;
    for row in rows {
        if row.transcript_url.is_none() {
            continue;
        }
        // In a real deployment we'd fetch the transcript and push each segment.
        let transcript = Transcript {
            recording_id: row.id,
            segments: vec![],
            language: "en".into(),
            model: "whisper-1".into(),
        };
        for seg in &transcript.segments {
            let doc = IndexDoc {
                id: Uuid::new_v4().to_string(),
                recording_id: row.id.to_string(),
                room_id: row.room_id.to_string(),
                text: seg.text.clone(),
                speaker: seg.speaker.clone(),
                start_ms: seg.start_ms,
                end_ms: seg.end_ms,
            };
            if let Err(e) = search.index(doc).await {
                tracing::error!(error = %e, "index doc failed");
            } else {
                indexed += 1;
            }
        }
    }
    info!(indexed, "transcript indexer sweep done");
    Ok(())
}
