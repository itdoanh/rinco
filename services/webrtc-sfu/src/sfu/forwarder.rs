//! Zero-transcoding SRTP forwarder.
//!
//! The forwarder keeps a routing table from `(room_id, source_user_id)` to
//! every subscriber, and a per-stream bandwidth budget. It does NOT decode
//! media – it only inspects RTP headers to perform SVC layer switching and
//! forwards the original packet to the next hop.
//!
//! SVC-aware switching rules:
//! 1. If the subscriber's `max_spatial_layer < packet.spatial_layer`, drop.
//! 2. If the subscriber's `max_temporal_layer < packet.temporal_layer`, drop.
//! 3. Otherwise forward.

use bytes::Bytes;
use dashmap::DashMap;
use parking_lot::Mutex;
use std::sync::Arc;
use std::time::Instant;
use uuid::Uuid;

use crate::error::SfuResult;
use crate::peer::Peer;

#[derive(Debug, Clone, Copy)]
pub struct RtpHeader {
    pub ssrc: u32,
    pub sequence_number: u16,
    pub timestamp: u32,
    pub marker: bool,
    pub spatial_layer: u8,
    pub temporal_layer: u8,
}

#[derive(Debug, Clone)]
pub struct OutgoingRtp {
    pub ssrc: u32,
    pub payload: Bytes,
    pub header: RtpHeader,
    /// Receive instant for downstream rate calculation.
    pub received_at: Instant,
}

/// Subscription from one subscriber to one source.
#[derive(Debug, Clone)]
pub struct Subscription {
    pub room_id: Uuid,
    pub subscriber: Uuid,
    pub source: Uuid,
    /// SVC layer ceiling for this subscription.
    pub max_spatial_layer: u8,
    pub max_temporal_layer: u8,
}

/// The forwarder. Holds the routing table and per-source statistics.
#[derive(Default, Clone)]
pub struct Forwarder {
    routes: Arc<DashMap<Uuid, Vec<Subscription>>>,
    stats: Arc<Mutex<ForwarderStats>>,
}

#[derive(Debug, Default, Clone, Copy)]
pub struct ForwarderStats {
    pub packets_in: u64,
    pub packets_dropped_svc: u64,
    pub packets_forwarded: u64,
    pub bytes_in: u64,
    pub bytes_out: u64,
}

impl Forwarder {
    pub fn new() -> Self {
        Self::default()
    }

    /// Add a subscription entry.
    pub fn add_subscription(&self, sub: Subscription) {
        self.routes.entry(sub.room_id).or_default().push(sub);
    }

    /// Remove all subscriptions for a user in a room (called on leave).
    pub fn remove_user(&self, room_id: Uuid, user_id: Uuid) {
        if let Some(mut entry) = self.routes.get_mut(&room_id) {
            entry.retain(|s| s.subscriber != user_id);
        }
    }

    /// Compute the list of subscribers for an incoming RTP packet.
    pub fn subscribers_for(&self, room_id: Uuid, source: Uuid) -> Vec<Uuid> {
        self.routes
            .get(&room_id)
            .map(|subs| {
                subs.iter()
                    .filter(|s| s.source == source)
                    .map(|s| s.subscriber)
                    .collect()
            })
            .unwrap_or_default()
    }

    /// Decide whether a packet should be forwarded to a specific subscriber
    /// based on SVC layer rules.
    pub fn should_forward(&self, sub: &Subscription, pkt: &OutgoingRtp) -> bool {
        pkt.header.spatial_layer <= sub.max_spatial_layer
            && pkt.header.temporal_layer <= sub.max_temporal_layer
    }

    /// Process one incoming packet and emit one OutgoingRtp per subscriber.
    pub fn forward(&self, room_id: Uuid, source: Uuid, pkt: OutgoingRtp) -> Vec<(Uuid, OutgoingRtp)> {
        let subs = self.routes.get(&room_id).cloned().unwrap_or_default();
        let mut out = Vec::new();
        for sub in subs.iter().filter(|s| s.source == source) {
            if self.should_forward(sub, &pkt) {
                out.push((sub.subscriber, pkt.clone()));
                self.stats.lock().packets_forwarded += 1;
                self.stats.lock().bytes_out += pkt.payload.len() as u64;
            } else {
                self.stats.lock().packets_dropped_svc += 1;
            }
        }
        self.stats.lock().packets_in += 1;
        self.stats.lock().bytes_in += pkt.payload.len() as u64;
        out
    }

    /// Forward to a specific peer (used by egress).
    pub fn forward_to_peer(
        &self,
        peer: &Peer,
        pkt: &OutgoingRtp,
    ) -> SfuResult<Option<OutgoingRtp>> {
        if pkt.header.spatial_layer <= peer.max_spatial_layer
            && pkt.header.temporal_layer <= peer.max_temporal_layer
        {
            Ok(Some(pkt.clone()))
        } else {
            Ok(None)
        }
    }

    /// Current stats snapshot.
    pub fn stats(&self) -> ForwarderStats {
        *self.stats.lock()
    }
}
