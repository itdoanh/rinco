//! Integration tests for the WebRTC SFU.

use bytes::Bytes;
use uuid::Uuid;
use webrtc_sfu::peer::Peer;
use webrtc_sfu::room::RoomManager;
use webrtc_sfu::sfu::congestion::{CongestionController, EstimatorConfig};
use webrtc_sfu::sfu::forwarder::{Forwarder, OutgoingRtp, RtpHeader, Subscription};

fn packet(spatial: u8, temporal: u8) -> OutgoingRtp {
    OutgoingRtp {
        ssrc: 1,
        payload: Bytes::from_static(b"\x00\x00"),
        header: RtpHeader {
            ssrc: 1,
            sequence_number: 0,
            timestamp: 0,
            marker: false,
            spatial_layer: spatial,
            temporal_layer: temporal,
        },
        received_at: std::time::Instant::now(),
    }
}

#[test]
fn svc_layer_filtering_drops_high_layers() {
    let f = Forwarder::new();
    let room = Uuid::new_v4();
    let src = Uuid::new_v4();
    let sub1 = Uuid::new_v4();
    let sub2 = Uuid::new_v4();

    f.add_subscription(Subscription {
        room_id: room,
        subscriber: sub1,
        source: src,
        max_spatial_layer: 0,
        max_temporal_layer: 0,
    });
    f.add_subscription(Subscription {
        room_id: room,
        subscriber: sub2,
        source: src,
        max_spatial_layer: 2,
        max_temporal_layer: 2,
    });

    let pkt = packet(2, 2);
    let out = f.forward(room, src, pkt);
    // sub1 should be dropped; sub2 should be forwarded.
    let targets: Vec<Uuid> = out.into_iter().map(|(u, _)| u).collect();
    assert_eq!(targets, vec![sub2]);

    let stats = f.stats();
    assert_eq!(stats.packets_in, 1);
    assert_eq!(stats.packets_dropped_svc, 1);
    assert_eq!(stats.packets_forwarded, 1);
}

#[tokio::test]
async fn room_lifecycle() {
    let rooms = RoomManager::new();
    let tenant = Uuid::new_v4();
    let user = Uuid::new_v4();
    let r = rooms.create(tenant, "test".into(), 10);
    let id = r.id;
    r.add_participant(webrtc_sfu::room::Participant {
        id: Uuid::new_v4(),
        user_id: user,
        tenant_id: tenant,
        display_name: "alice".into(),
        is_audio_enabled: true,
        is_video_enabled: true,
        is_screen_sharing: false,
        joined_at: chrono::Utc::now(),
        connection_quality: 1.0,
    })
    .unwrap();
    assert_eq!(r.list_participants().len(), 1);
    r.remove_participant(user).unwrap();
    assert!(r.list_participants().is_empty());
    assert!(rooms.delete(id));
}

#[tokio::test]
async fn peer_sdp_validation() {
    let peer = Peer::new(Uuid::new_v4(), Uuid::new_v4());
    peer.validate_sdp("v=0\r\no=- 1 1 IN IP4 127.0.0.1\r\nm=audio 9 UDP/TLS/RTP/SAVPF 111\r\n")
        .unwrap();
    assert!(peer.validate_sdp("not sdp").is_err());
}

#[test]
fn congestion_estimator_adapts() {
    let mut cc = CongestionController::new(EstimatorConfig {
        min_bitrate_bps: 100_000,
        max_bitrate_bps: 10_000_000,
        initial_bitrate_bps: 1_000_000,
    });
    let now = std::time::Instant::now();
    // Even spacing – should not drop the estimate.
    for i in 0..5u32 {
        cc.on_packet(42, 1000, now + std::time::Duration::from_millis(i * 20));
    }
    let mid = cc.estimate(42);
    assert!(mid >= 1_000_000);
    // Spaced too far – triggers decrease.
    for i in 0..10u32 {
        cc.on_packet(42, 1000, now + std::time::Duration::from_millis(200 * i));
    }
    let after = cc.estimate(42);
    assert!(after <= mid);
}
