//! Tests for webrtc-sfu error types.

use webrtc_sfu::error::SfuError;

#[test]
fn sfu_error_room_not_found() {
    let e = SfuError::RoomNotFound("room-123".into());
    let s = e.to_string();
    assert!(s.contains("room not found"));
    assert!(s.contains("room-123"));
}

#[test]
fn sfu_error_participant_not_found() {
    let e = SfuError::ParticipantNotFound("user-abc".into());
    assert!(e.to_string().contains("participant not found"));
}

#[test]
fn sfu_error_signaling() {
    let e = SfuError::Signaling("bad JSON".into());
    assert!(e.to_string().contains("signaling"));
}

#[test]
fn sfu_error_sdp() {
    let e = SfuError::Sdp("invalid codec".into());
    assert!(e.to_string().contains("sdp"));
}

#[test]
fn sfu_error_ice() {
    let e = SfuError::Ice("connection lost".into());
    assert!(e.to_string().contains("ice"));
}

#[test]
fn sfu_error_forwarder() {
    let e = SfuError::Forwarder("buffer overflow".into());
    assert!(e.to_string().contains("forwarder"));
}

#[test]
fn sfu_error_egress() {
    let e = SfuError::Egress("upload failed".into());
    assert!(e.to_string().contains("egress"));
}

#[test]
fn sfu_error_invalid_request() {
    let e = SfuError::InvalidRequest("missing room id".into());
    assert!(e.to_string().contains("invalid request"));
}

#[test]
fn sfu_error_forbidden() {
    let e = SfuError::Forbidden("not moderator".into());
    let s = e.to_string();
    assert!(s.contains("forbidden"));
    assert!(s.contains("moderator"));
}

#[test]
fn sfu_error_from_anyhow() {
    let any = anyhow::anyhow!("underlying error");
    let e: SfuError = any.into();
    assert!(e.to_string().contains("underlying error"));
}

#[test]
fn sfu_error_debug() {
    let e = SfuError::RoomNotFound("r1".into());
    let dbg = format!("{:?}", e);
    assert!(dbg.contains("RoomNotFound"));
}

#[test]
fn sfu_error_all_variants_displayable() {
    let variants = vec![
        SfuError::RoomNotFound("a".into()),
        SfuError::ParticipantNotFound("b".into()),
        SfuError::Signaling("c".into()),
        SfuError::Sdp("d".into()),
        SfuError::Ice("e".into()),
        SfuError::Forwarder("f".into()),
        SfuError::Egress("g".into()),
        SfuError::InvalidRequest("h".into()),
        SfuError::Forbidden("i".into()),
    ];
    for v in variants {
        assert!(!v.to_string().is_empty());
    }
}
