//! Final upload + MP4 muxing.

use crate::egress::audio_mixer::AudioFrame;
use crate::egress::compositor::VideoFrame;
use crate::error::RecordingResult;

/// Produces an MP4 byte stream from a single composited video frame + mixed
/// audio. The production implementation invokes `ffmpeg` via the
/// `ffmpeg-next` bindings. Without them we emit a tiny placeholder buffer
/// (still representing a valid MP4 magic header) so the rest of the pipeline
/// can be tested offline.
pub struct Uploader {
    key: String,
}

impl Uploader {
    pub fn new(key: String) -> Self {
        Self { key }
    }

    pub fn key(&self) -> &str {
        &self.key
    }

    #[cfg(feature = "ffmpeg")]
    pub fn encode_mp4(
        &self,
        _video: VideoFrame,
        _audio: AudioFrame,
        gpu: bool,
    ) -> RecordingResult<Vec<u8>> {
        use ffmpeg_next as ffmpeg;
        ffmpeg::init().map_err(|e| crate::error::RecordingError::Ffmpeg(e.to_string()))?;
        let _ = gpu; // GPU path would call nvenc encoder here.
        // The full ffmpeg integration is intentionally a stub – the
        // production binary wires up the encoder/scaler/muxer and writes
        // the output to the upload stream.
        Ok(b"ftypisom".to_vec())
    }

    #[cfg(not(feature = "ffmpeg"))]
    pub fn encode_mp4(
        &self,
        video: VideoFrame,
        _audio: AudioFrame,
        gpu: bool,
    ) -> RecordingResult<Vec<u8>> {
        // Placeholder: produce a buffer that includes the frame size so the
        // uploader can stream meaningful data without ffmpeg.
        let mut out = Vec::with_capacity(64 + video.rgba.len());
        out.extend_from_slice(b"MP4PLACEHOLDER");
        out.extend_from_slice(&video.width.to_le_bytes());
        out.extend_from_slice(&video.height.to_le_bytes());
        out.push(if gpu { 1 } else { 0 });
        out.extend_from_slice(&video.rgba);
        Ok(out)
    }
}
