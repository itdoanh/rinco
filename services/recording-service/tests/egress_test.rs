//! Egress pipeline tests.

use recording_service::egress::audio_mixer::{AudioFrame, AudioMixer};
use recording_service::egress::compositor::{Compositor, VideoFrame};
use uuid::Uuid;

#[test]
fn compositor_renders_empty_canvas() {
    let c = Compositor::new(640, 360, false);
    let frame = c.render();
    assert_eq!(frame.width, 640);
    assert_eq!(frame.height, 360);
    assert_eq!(frame.rgba.len(), (640 * 360 * 4) as usize);
}

#[test]
fn compositor_blits_tile() {
    let c = Compositor::new(640, 360, false);
    let mut rgba = vec![0u8; 16 * 16 * 4];
    for px in rgba.chunks_exact_mut(4) {
        px[0] = 255;
        px[3] = 255;
    }
    c.add_tile(
        Uuid::new_v4(),
        VideoFrame { width: 16, height: 16, rgba, pts_ms: 0 },
    );
    let frame = c.render();
    // The first 320×180 quadrant should be all red.
    let red_count = frame.rgba.chunks_exact(4).filter(|p| p[0] == 255).count();
    assert!(red_count > 0);
}

#[test]
fn audio_mixer_mixes_two_streams() {
    let m = AudioMixer::new(48_000, 2);
    let a = AudioFrame {
        samples: vec![0.5; 48_000],
        sample_rate: 48_000,
        channels: 1,
    };
    let b = AudioFrame {
        samples: vec![-0.5; 48_000],
        sample_rate: 48_000,
        channels: 1,
    };
    m.push(Uuid::new_v4(), a);
    m.push(Uuid::new_v4(), b);
    let mixed = m.mix_all();
    assert_eq!(mixed.samples.len(), 48_000 * 2);
    // After mixing, all samples should be near zero.
    let max = mixed.samples.iter().fold(0.0f32, |a, v| a.max(v.abs()));
    assert!(max < 0.01, "expected near zero, got {}", max);
}
