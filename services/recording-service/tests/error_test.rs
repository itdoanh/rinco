//! Tests for recording-service error types.

use recording_service::error::RecordingError;

#[test]
fn recording_error_storage() {
    let e = RecordingError::Storage("S3 unreachable".into());
    let s = e.to_string();
    assert!(s.contains("storage"));
    assert!(s.contains("S3 unreachable"));
}

#[test]
fn recording_error_database() {
    let e = RecordingError::Database("connection refused".into());
    assert!(e.to_string().contains("database"));
}

#[test]
fn recording_error_ffmpeg() {
    let e = RecordingError::Ffmpeg("missing codec".into());
    assert!(e.to_string().contains("ffmpeg"));
}

#[test]
fn recording_error_gpu() {
    let e = RecordingError::Gpu("device lost".into());
    assert!(e.to_string().contains("gpu"));
}

#[test]
fn recording_error_upload() {
    let e = RecordingError::Upload("timeout".into());
    assert!(e.to_string().contains("upload"));
}

#[test]
fn recording_error_not_found() {
    let e = RecordingError::NotFound("recording-abc".into());
    let s = e.to_string();
    assert!(s.contains("not found"));
    assert!(s.contains("recording-abc"));
}

#[test]
fn recording_error_invalid_request() {
    let e = RecordingError::InvalidRequest("missing fields".into());
    assert!(e.to_string().contains("invalid request"));
}

#[test]
fn recording_error_from_anyhow() {
    let any = anyhow::anyhow!("something bad happened");
    let e: RecordingError = any.into();
    assert!(e.to_string().contains("something bad happened"));
}

#[test]
fn recording_error_debug() {
    let e = RecordingError::Storage("x".into());
    let dbg = format!("{:?}", e);
    assert!(dbg.contains("Storage"));
}

#[test]
fn recording_error_display_variants() {
    let variants = vec![
        RecordingError::Storage("a".into()),
        RecordingError::Database("b".into()),
        RecordingError::Ffmpeg("c".into()),
        RecordingError::Gpu("d".into()),
        RecordingError::Upload("e".into()),
        RecordingError::NotFound("f".into()),
        RecordingError::InvalidRequest("g".into()),
    ];
    for v in variants {
        // Should not panic and should produce non-empty output
        assert!(!v.to_string().is_empty());
    }
}
