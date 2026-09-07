//! Audio mixer: take N streams and produce 1 stereo track.

use parking_lot::Mutex;
use std::collections::HashMap;
use uuid::Uuid;

/// Audio frame in normalized float [-1, 1].
#[derive(Debug, Clone)]
pub struct AudioFrame {
    pub samples: Vec<f32>,
    pub sample_rate: u32,
    pub channels: u16,
}

pub struct AudioMixer {
    target_rate: u32,
    target_channels: u16,
    buffers: Mutex<HashMap<Uuid, Vec<f32>>>,
}

impl AudioMixer {
    pub fn new(target_rate: u32, target_channels: u16) -> Self {
        Self {
            target_rate,
            target_channels,
            buffers: Mutex::new(HashMap::new()),
        }
    }

    pub fn push(&self, source: Uuid, frame: AudioFrame) {
        // Trivial resampling – the production implementation would use a
        // polyphase filter.
        let mut buf = self.buffers.lock();
        let entry = buf.entry(source).or_default();
        if frame.sample_rate == self.target_rate {
            entry.extend(frame.samples);
        } else {
            // Linear interpolation.
            let ratio = frame.sample_rate as f32 / self.target_rate as f32;
            let mut t = 0.0f32;
            while (t as usize) < frame.samples.len() {
                let i = t as usize;
                let frac = t - i as f32;
                let a = frame.samples[i];
                let b = frame.samples.get(i + 1).copied().unwrap_or(a);
                entry.push(a + (b - a) * frac);
                t += ratio;
            }
        }
    }

    /// Mix all sources into one mono/stereo track.
    pub fn mix_all(&self) -> AudioFrame {
        let mut buffers = self.buffers.lock();
        let mut max_len = 0;
        for v in buffers.values() {
            if v.len() > max_len {
                max_len = v.len();
            }
        }
        let mut out = vec![0.0f32; max_len * self.target_channels as usize];
        let n = buffers.len().max(1) as f32;
        for samples in buffers.values() {
            for (i, s) in samples.iter().enumerate() {
                let v = s / n;
                let base = i * self.target_channels as usize;
                if base + 1 < out.len() {
                    out[base] += v;
                    if self.target_channels == 2 {
                        out[base + 1] += v;
                    }
                }
            }
        }
        AudioFrame {
            samples: out,
            sample_rate: self.target_rate,
            channels: self.target_channels,
        }
    }
}
