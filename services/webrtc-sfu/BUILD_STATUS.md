# webrtc-sfu Status
- Files: 16
- LOC: 1319
- Syntax: ✅ valid
- Notes: Brace balance verified; `mod` declarations match on-disk structure (api, config, error, observability, peer, room, safety, signaling, sfu). `use` statements resolve to either external crates (tokio, axum, tonic, opentelemetry, webrtc [optional], ...) or in-project modules. Cargo.toml deps: tonic 0.12, axum 0.7, webrtc 0.11 (optional behind feature flag), dashmap 6.0, parking_lot 0.12. Optional features: webrtc, str0m, svc.