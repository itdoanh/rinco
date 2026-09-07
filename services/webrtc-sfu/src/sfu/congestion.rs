//! Bandwidth estimation / congestion control.
//!
//! A simplified delay-based estimator inspired by Google Congestion Control:
//! - track arrival deltas per SSRC
//! - when delays increase for `K` consecutive packets, halve the bitrate
//! - when delays drop for `K` consecutive packets, increase the bitrate by
//!   a small additive factor (1.05x)
//! - cap at the configured per-peer cap

use std::collections::{HashMap, VecDeque};
use std::time::Instant;

const WINDOW: usize = 16;
const K: usize = 4;

#[derive(Debug, Clone, Copy)]
pub struct EstimatorConfig {
    pub min_bitrate_bps: u32,
    pub max_bitrate_bps: u32,
    pub initial_bitrate_bps: u32,
}

impl Default for EstimatorConfig {
    fn default() -> Self {
        Self {
            min_bitrate_bps: 100_000,
            max_bitrate_bps: 4_000_000,
            initial_bitrate_bps: 1_000_000,
        }
    }
}

#[derive(Debug, Default)]
struct StreamState {
    samples: VecDeque<Instant>,
    delays_ms: VecDeque<f32>,
    current_bps: u32,
    increasing: u32,
    decreasing: u32,
}

impl StreamState {
    fn new(initial: u32) -> Self {
        Self {
            current_bps: initial,
            ..Default::default()
        }
    }
}

/// One estimator per SFU. Tracks every SSRC separately.
#[derive(Default)]
pub struct CongestionController {
    config: EstimatorConfig,
    streams: HashMap<u32, StreamState>,
}

impl CongestionController {
    pub fn new(config: EstimatorConfig) -> Self {
        Self {
            config,
            streams: HashMap::new(),
        }
    }

    /// Report a packet arrival for a given SSRC.
    pub fn on_packet(&mut self, ssrc: u32, size_bytes: usize, now: Instant) -> u32 {
        let state = self
            .streams
            .entry(ssrc)
            .or_insert_with(|| StreamState::new(self.config.initial_bitrate_bps));

        // Compute inter-arrival delay vs the previous packet.
        if let Some(prev) = state.samples.back() {
            let delta = now.duration_since(*prev).as_secs_f32() * 1000.0;
            state.delays_ms.push_back(delta);
            if state.delays_ms.len() > WINDOW {
                state.delays_ms.pop_front();
            }
        }
        state.samples.push_back(now);
        if state.samples.len() > WINDOW {
            state.samples.pop_front();
        }

        // Detect trend.
        if state.delays_ms.len() >= 2 {
            let len = state.delays_ms.len();
            let a = state.delays_ms[len - 2];
            let b = state.delays_ms[len - 1];
            if b > a * 1.10 {
                state.increasing += 1;
                state.decreasing = 0;
            } else if b < a * 0.90 {
                state.decreasing += 1;
                state.increasing = 0;
            } else {
                state.increasing = 0;
                state.decreasing = 0;
            }
        }

        if state.increasing >= K as u32 {
            state.current_bps = (state.current_bps as f32 * 0.7) as u32;
            state.current_bps = state.current_bps.max(self.config.min_bitrate_bps);
            state.increasing = 0;
        } else if state.decreasing >= K as u32 {
            state.current_bps = (state.current_bps as f32 * 1.05) as u32;
            state.current_bps = state.current_bps.min(self.config.max_bitrate_bps);
            state.decreasing = 0;
        }

        // Suppress 'unused' warning.
        let _ = size_bytes;
        state.current_bps
    }

    /// Current estimate for a SSRC.
    pub fn estimate(&self, ssrc: u32) -> u32 {
        self.streams
            .get(&ssrc)
            .map(|s| s.current_bps)
            .unwrap_or(self.config.initial_bitrate_bps)
    }
}
