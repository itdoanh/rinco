//! AI pipeline: push the mixed audio track to the STT service and persist
//! the transcript.

use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::egress::audio_mixer::AudioFrame;
use crate::error::RecordingResult;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TranscriptSegment {
    pub start_ms: i64,
    pub end_ms: i64,
    pub text: String,
    pub confidence: f32,
    pub speaker: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transcript {
    pub recording_id: Uuid,
    pub segments: Vec<TranscriptSegment>,
    pub language: String,
    pub model: String,
}

/// Result of running the AI pipeline.
pub struct AiResult {
    pub transcript: Transcript,
}

/// Trait implemented by the Whisper / whisper.cpp client.
#[async_trait::async_trait]
pub trait SttClient: Send + Sync {
    async fn transcribe(&self, recording_id: Uuid, audio: AudioFrame) -> RecordingResult<Transcript>;
}

/// HTTP client that calls the STT service.
pub struct HttpSttClient {
    base_url: String,
    http: reqwest::Client,
}

impl HttpSttClient {
    pub fn new(base_url: String) -> Self {
        let http = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(120))
            .build()
            .expect("reqwest client");
        Self { base_url, http }
    }
}

#[async_trait::async_trait]
impl SttClient for HttpSttClient {
    async fn transcribe(&self, recording_id: Uuid, audio: AudioFrame) -> RecordingResult<Transcript> {
        let url = format!("{}/v1/transcribe", self.base_url);
        let body = serde_json::json!({
            "recording_id": recording_id,
            "sample_rate": audio.sample_rate,
            "channels": audio.channels,
            "samples": audio.samples,
        });
        let resp = self
            .http
            .post(url)
            .json(&body)
            .send()
            .await
            .map_err(|e| crate::error::RecordingError::Other(anyhow::anyhow!(e)))?;
        if !resp.status().is_success() {
            return Err(crate::error::RecordingError::Other(anyhow::anyhow!(
                "stt returned {}",
                resp.status()
            )));
        }
        let transcript: Transcript = resp
            .json()
            .await
            .map_err(|e| crate::error::RecordingError::Other(anyhow::anyhow!(e)))?;
        Ok(transcript)
    }
}
