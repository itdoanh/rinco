# recording-service Status
- Files: 19
- LOC: 1377
- Syntax: ✅ valid
- Notes: Brace balance verified; `mod` declarations match on-disk structure (api, config, egress, error, jobs, observability, storage). `use` statements reference external crates (actix-web 4.9, sqlx 0.8, rust-s3 0.34, ffmpeg-next 7.0 optional, image 0.25, ...) or in-project modules. Optional features: gpu, ffmpeg, opencv.