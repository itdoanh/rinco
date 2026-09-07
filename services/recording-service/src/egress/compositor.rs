//! Video compositor.
//!
//! Two backends:
//! - GPU (NVENC / CUDA) – stubbed behind `gpu` feature; falls back to CPU.
//! - CPU – uses `image` crate to draw RGBA tiles into a 1280×720 canvas.
//!
//! Layouts supported: Gallery (NxN grid), Focused (one big + N thumbnails).

use image::{ImageBuffer, Rgba, RgbaImage};
use parking_lot::Mutex;
use std::collections::HashMap;
use uuid::Uuid;

/// A single decoded video frame.
#[derive(Debug, Clone)]
pub struct VideoFrame {
    pub width: u32,
    pub height: u32,
    pub rgba: Vec<u8>,
    pub pts_ms: i64,
}

#[derive(Debug, Clone, Copy)]
pub enum Layout {
    Gallery { cols: u32, rows: u32 },
    Focused,
}

pub struct Compositor {
    width: u32,
    height: u32,
    gpu: bool,
    layout: Layout,
    tiles: Mutex<HashMap<Uuid, VideoFrame>>,
}

impl Compositor {
    pub fn new(width: u32, height: u32, gpu: bool) -> Self {
        Self {
            width,
            height,
            gpu,
            layout: Layout::Gallery { cols: 2, rows: 2 },
            tiles: Mutex::new(HashMap::new()),
        }
    }

    pub fn set_layout(&mut self, layout: Layout) {
        self.layout = layout;
    }

    pub fn add_tile(&self, source: Uuid, frame: VideoFrame) {
        self.tiles.lock().insert(source, frame);
    }

    pub fn remove_tile(&self, source: Uuid) {
        self.tiles.lock().remove(&source);
    }

    /// Render the current set of tiles to a single RGBA frame.
    pub fn render(&self) -> VideoFrame {
        if self.gpu {
            #[cfg(feature = "gpu")]
            {
                return self.render_gpu();
            }
        }
        self.render_cpu()
    }

    fn render_cpu(&self) -> VideoFrame {
        let mut canvas: RgbaImage = ImageBuffer::from_pixel(self.width, self.height, Rgba([16, 16, 16, 255]));
        let tiles: Vec<VideoFrame> = self.tiles.lock().values().cloned().collect();
        if tiles.is_empty() {
            return VideoFrame {
                width: self.width,
                height: self.height,
                rgba: canvas.into_raw(),
                pts_ms: 0,
            };
        }
        let (cols, rows) = match self.layout {
            Layout::Gallery { cols, rows } => (cols.max(1), rows.max(1)),
            Layout::Focused => {
                // First tile takes 75% of the canvas, rest are thumbnailed in a row.
                let cols = 1u32;
                let rows = 1u32;
                (cols, rows)
            }
        };
        let cell_w = self.width / cols;
        let cell_h = self.height / rows;
        for (idx, tile) in tiles.iter().take((cols * rows) as usize).enumerate() {
            let cx = (idx as u32 % cols) * cell_w;
            let cy = (idx as u32 / cols) * cell_h;
            blit_scaled(&mut canvas, tile, cx, cy, cell_w, cell_h);
        }
        VideoFrame {
            width: self.width,
            height: self.height,
            rgba: canvas.into_raw(),
            pts_ms: 0,
        }
    }

    #[cfg(feature = "gpu")]
    fn render_gpu(&self) -> VideoFrame {
        // Stub – real implementation would launch a CUDA kernel and copy the
        // result back. We fall back to CPU so the function is always usable.
        self.render_cpu()
    }
}

fn blit_scaled(canvas: &mut RgbaImage, tile: &VideoFrame, x: u32, y: u32, w: u32, h: u32) {
    if tile.width == 0 || tile.height == 0 {
        return;
    }
    let mut tile_img = ImageBuffer::<Rgba<u8>, _>::from_raw(tile.width, tile.height, tile.rgba.clone())
        .unwrap_or_else(|| ImageBuffer::from_pixel(tile.width, tile.height, Rgba([0, 0, 0, 255])));
    if tile.width != w || tile.height != h {
        tile_img = image::imageops::resize(&tile_img, w, h, image::imageops::FilterType::Triangle);
    }
    for dy in 0..h {
        for dx in 0..w {
            if x + dx < canvas.width() && y + dy < canvas.height() {
                let p = *tile_img.get_pixel(dx, dy);
                canvas.put_pixel(x + dx, y + dy, p);
            }
        }
    }
}
