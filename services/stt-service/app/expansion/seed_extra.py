"""Extended mock data for the STT service (WS-B Loop 9).

20+ transcripts across vi/en/ja with realistic segments.
"""
from __future__ import annotations

import os
import time
import uuid
from typing import Any, Dict, List

__all__ = [
    "EXTENDED_LANGUAGES",
    "EXTENDED_TRANSCRIPTS",
    "EXTENDED_MODELS",
    "EXTENDED_SAMPLE_AUDIO",
]


_EXTENDED_LANG_CONFIG = [
    ("vi", "vi-VN", "Tiếng Việt", "Whisper-medium"),
    ("en", "en-US", "English (US)", "Whisper-medium"),
    ("en", "en-GB", "English (UK)", "Whisper-medium"),
    ("ja", "ja-JP", "日本語", "Whisper-medium"),
    ("ko", "ko-KR", "한국어", "Whisper-medium"),
    ("zh", "zh-CN", "简体中文", "Whisper-medium"),
    ("zh", "zh-TW", "繁體中文", "Whisper-medium"),
    ("fr", "fr-FR", "Français", "Whisper-medium"),
    ("de", "de-DE", "Deutsch", "Whisper-medium"),
    ("es", "es-ES", "Español", "Whisper-medium"),
]

EXTENDED_LANGUAGES: List[Dict[str, Any]] = [
    {
        "code": lang[0],
        "locale": lang[1],
        "display_name": lang[2],
        "model": lang[3],
        "wer_score": round(0.05 + (i * 0.01) % 0.15, 3),
        "supported_features": ["transcription", "translation", "diarization"] if i < 5 else ["transcription"],
    }
    for i, lang in enumerate(_EXTENDED_LANG_CONFIG)
]


# ---------------------------------------------------------------------------
# Transcripts (22 entries)
# ---------------------------------------------------------------------------
_VI_SAMPLES = [
    "Xin chào quý vị, hôm nay chúng ta sẽ thảo luận về chiến lược kinh doanh quý 3.",
    "Doanh thu tháng này đạt 120 tỷ đồng, tăng 18% so với cùng kỳ năm ngoái.",
    "Chúng ta cần tập trung vào việc mở rộng thị trường miền Bắc trong quý tới.",
    "Tôi đề xuất tăng ngân sách marketing cho kênh TikTok và Zalo OA.",
    "Hãy nhớ rằng khách hàng là trung tâm của mọi quyết định kinh doanh.",
    "Báo cáo tài chính chi tiết sẽ được gửi qua email vào cuối tuần này.",
    "Cảm ơn mọi người đã tham gia cuộc họp. Hẹn gặp lại tuần sau.",
    "Chúng tôi sẽ ra mắt sản phẩm mới vào ngày 15 tháng 12 tới đây.",
]

_EN_SAMPLES = [
    "Welcome everyone, today we'll discuss our Q3 strategy and roadmap.",
    "Revenue this month reached 5 million USD, up 18% year over year.",
    "We need to focus on expanding into the Northern market next quarter.",
    "I'm proposing to increase our marketing budget for TikTok and Zalo OA channels.",
    "Remember, the customer is at the center of every business decision we make.",
    "The detailed financial report will be sent via email by the end of this week.",
    "Thank you all for joining. See you next week for our next sync.",
    "We will launch the new product on December 15th of this year.",
]

_JA_SAMPLES = [
    "皆さん、こんにちは。本日は第3四半期の戦略について議論します。",
    "今月の収益は500万ドルに達し、前年比18%増となりました。",
    "次の四半期は北部市場への拡大に注力する必要があります。",
    "TikTokとZalo OAチャネルのマーケティング予算を増額することを提案します。",
    "顧客はすべてのビジネス決定の中心であることを忘れないでください。",
    "詳細な財務報告は今週末までにメールで送信されます。",
    "ご参加ありがとうございました。来週またお会いしましょう。",
]


def _build_segments(text: str, lang: str) -> List[Dict[str, Any]]:
    """Naively split text into 4-8 word segments."""
    words = text.replace(",", "").replace(".", "").split()
    segments = []
    cur_start = 0.0
    cur_seg: List[str] = []
    target = max(4, len(words) // 6)
    for i, w in enumerate(words):
        cur_seg.append(w)
        if len(cur_seg) >= target or i == len(words) - 1:
            dur = 1.0 + len(cur_seg) * 0.3
            segments.append({
                "start": round(cur_start, 2),
                "end": round(cur_start + dur, 2),
                "text": " ".join(cur_seg),
                "confidence": round(0.85 + (i % 3) * 0.04, 3),
                "speaker_id": f"speaker-{(i // 5) % 2}",
                "language": lang,
            })
            cur_start += dur
            cur_seg = []
    return segments


def _build_extended_transcripts() -> List[Dict[str, Any]]:
    out: List[Dict[str, Any]] = []
    # Vietnamese: 8 samples
    for i, text in enumerate(_VI_SAMPLES):
        out.append({
            "transcript_id": f"trans-vi-{i:03d}",
            "language": "vi",
            "locale": "vi-VN",
            "audio_url": f"https://s3.example.com/audio/vi/{i:03d}.mp3",
            "duration_sec": 45 + i * 6,
            "speakers": 1 + (i % 2),
            "model": "whisper-medium",
            "text": text,
            "segments": _build_segments(text, "vi"),
            "created_at": "2026-09-18T00:00:00Z",
            "tenant_id": (["demo-tenant-apexfintech","demo-tenant-hct-consulting","demo-tenant"])[i % 3],
            "tags": ["meeting", "sales", "strategy"] if i < 4 else ["training", "webinar"],
        })

    # English: 8 samples
    for i, text in enumerate(_EN_SAMPLES):
        out.append({
            "transcript_id": f"trans-en-{i:03d}",
            "language": "en",
            "locale": "en-US",
            "audio_url": f"https://s3.example.com/audio/en/{i:03d}.mp3",
            "duration_sec": 50 + i * 5,
            "speakers": 1 + (i % 2),
            "model": "whisper-medium",
            "text": text,
            "segments": _build_segments(text, "en"),
            "created_at": "2026-09-17T00:00:00Z",
            "tenant_id": (["demo-tenant-apexfintech","demo-tenant-hct-consulting","demo-tenant"])[i % 3],
            "tags": ["customer-call", "support"] if i < 4 else ["demo", "onboarding"],
        })

    # Japanese: 6 samples
    for i, text in enumerate(_JA_SAMPLES):
        out.append({
            "transcript_id": f"trans-ja-{i:03d}",
            "language": "ja",
            "locale": "ja-JP",
            "audio_url": f"https://s3.example.com/audio/ja/{i:03d}.mp3",
            "duration_sec": 40 + i * 7,
            "speakers": 1 + (i % 2),
            "model": "whisper-medium",
            "text": text,
            "segments": _build_segments(text, "ja"),
            "created_at": "2026-09-16T00:00:00Z",
            "tenant_id": (["demo-tenant-apexfintech","demo-tenant-hct-consulting"])[i % 2],
            "tags": ["meeting", "japanese", "training"],
        })

    return out


EXTENDED_TRANSCRIPTS: List[Dict[str, Any]] = _build_extended_transcripts()


# ---------------------------------------------------------------------------
# Models (5)
# ---------------------------------------------------------------------------
EXTENDED_MODELS: List[Dict[str, Any]] = [
    {
        "model_id": "whisper-tiny",
        "size_mb": 75,
        "languages_supported": 99,
        "speed_x_realtime": 32,
        "accuracy_wer": 0.12,
    },
    {
        "model_id": "whisper-base",
        "size_mb": 145,
        "languages_supported": 99,
        "speed_x_realtime": 16,
        "accuracy_wer": 0.08,
    },
    {
        "model_id": "whisper-small",
        "size_mb": 466,
        "languages_supported": 99,
        "speed_x_realtime": 6,
        "accuracy_wer": 0.06,
    },
    {
        "model_id": "whisper-medium",
        "size_mb": 1500,
        "languages_supported": 99,
        "speed_x_realtime": 2,
        "accuracy_wer": 0.045,
    },
    {
        "model_id": "whisper-large-v3",
        "size_mb": 3100,
        "languages_supported": 99,
        "speed_x_realtime": 1,
        "accuracy_wer": 0.035,
    },
]


# ---------------------------------------------------------------------------
# Sample audio metadata
# ---------------------------------------------------------------------------
EXTENDED_SAMPLE_AUDIO: List[Dict[str, Any]] = [
    {
        "audio_id": f"sample-{lang[0]}-{i:03d}",
        "language": lang[0],
        "url": f"https://s3.example.com/samples/{lang[0]}/{i:03d}.wav",
        "duration_sec": 30 + i * 5,
        "sample_rate_hz": 16000,
        "channels": 1,
        "size_bytes": (30 + i * 5) * 16000 * 2,
        "noise_level_db": -45 + (i % 5) * 2,
        "snr_db": 25 + (i % 4) * 3,
    }
    for i, lang in enumerate(_EXTENDED_LANG_CONFIG)
]
