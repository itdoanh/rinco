"""Sample transcripts and audio-sample helpers for STT.

Used by admin tasks and integration tests to provide deterministic
test fixtures.  Provides a small catalog of Vietnamese + English
transcription examples that mirror the kind of content recorded by
RINCO meeting rooms (sales calls, standups, support calls).
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass(frozen=True)
class TranscriptSample:
    """A canned transcript used as a deterministic test fixture."""

    name: str
    language: str
    duration_seconds: float
    text: str
    segments: tuple[dict[str, Any], ...]

    def to_dict(self) -> dict[str, Any]:
        return {
            "name": self.name,
            "language": self.language,
            "duration": self.duration_seconds,
            "text": self.text,
            "segments": list(self.segments),
        }


SAMPLES: tuple[TranscriptSample, ...] = (
    TranscriptSample(
        name="sales_call_vi_01",
        language="vi",
        duration_seconds=42.5,
        text=(
            "Xin chào anh, em là Minh từ RINCO. Em muốn tư vấn cho anh gói "
            "CRM doanh nghiệp vừa và nhỏ. Hiện tại anh đang dùng phần mềm "
            "nào để quản lý khách hàng ạ?"
        ),
        segments=(
            {"start": 0.0, "end": 6.0, "text": "Xin chào anh, em là Minh từ RINCO.", "speaker": "SPEAKER_00"},
            {"start": 6.5, "end": 14.0, "text": "Em muốn tư vấn gói CRM doanh nghiệp vừa và nhỏ.", "speaker": "SPEAKER_00"},
            {"start": 14.5, "end": 28.0, "text": "Hiện tại anh đang dùng phần mềm nào để quản lý khách hàng ạ?", "speaker": "SPEAKER_00"},
        ),
    ),
    TranscriptSample(
        name="standup_en_01",
        language="en",
        duration_seconds=58.0,
        text=(
            "Yesterday I shipped the lead-scoring batch endpoint. Today I'm "
            "working on the SHAP explanations. No blockers."
        ),
        segments=(
            {"start": 0.0, "end": 9.0, "text": "Yesterday I shipped the lead-scoring batch endpoint.", "speaker": "SPEAKER_01"},
            {"start": 9.5, "end": 18.0, "text": "Today I'm working on the SHAP explanations.", "speaker": "SPEAKER_01"},
            {"start": 18.5, "end": 21.0, "text": "No blockers.", "speaker": "SPEAKER_01"},
        ),
    ),
    TranscriptSample(
        name="support_call_vi_02",
        language="vi",
        duration_seconds=75.0,
        text=(
            "Bên em đã nhận được yêu cầu hỗ trợ. Anh vui lòng cho em biết "
            "mã lỗi hiển thị trên màn hình ạ. Em sẽ hướng dẫn anh khắc phục "
            "trong vài phút."
        ),
        segments=(
            {"start": 0.0, "end": 10.0, "text": "Bên em đã nhận được yêu cầu hỗ trợ.", "speaker": "SPEAKER_02"},
            {"start": 10.5, "end": 26.0, "text": "Anh vui lòng cho em biết mã lỗi hiển thị trên màn hình ạ.", "speaker": "SPEAKER_02"},
            {"start": 26.5, "end": 40.0, "text": "Em sẽ hướng dẫn anh khắc phục trong vài phút.", "speaker": "SPEAKER_02"},
        ),
    ),
)


def list_samples() -> list[dict[str, Any]]:
    """Return all sample transcripts as plain dicts."""
    return [s.to_dict() for s in SAMPLES]


def get_sample(name: str) -> dict[str, Any] | None:
    """Return a single sample by name, or ``None`` if missing."""
    for s in SAMPLES:
        if s.name == name:
            return s.to_dict()
    return None


__all__ = ["TranscriptSample", "SAMPLES", "list_samples", "get_sample"]
