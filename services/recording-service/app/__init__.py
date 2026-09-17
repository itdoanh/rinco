"""Recording Service – RINCO.

FastAPI-based egress pipeline. Captures WebRTC meetings (either via
FFmpeg piped from a virtual device or via a direct RTMP ingest), stores
the resulting MP4 in MinIO, and exposes signed URLs for download / HLS
playback.

The module is intentionally split into small, testable units:

* ``models``     – Pydantic schemas for the API.
* ``storage``    – MinIO client wrapper.
* ``services``   – FFmpeg / STT orchestration.
* ``routers``    – FastAPI endpoints.
"""

__all__ = [
    "models",
    "routers",
    "services",
    "storage",
]
