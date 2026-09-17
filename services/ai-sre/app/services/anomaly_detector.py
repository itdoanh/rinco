"""Anomaly detection for time-series metrics.

Implements simple statistical detectors (z-score, EWMA, IQR) that work
on Prometheus query results.  Heavier ML-based detectors (Prophet, LSTM)
are intentionally out of scope for this in-process implementation;
they live behind an HTTP endpoint that delegates to an external
inference service.

This module is deliberately framework-free — only :mod:`app.core` and
:mod:`prometheus_client` are used so it can be unit-tested without a
running observability stack.
"""
from __future__ import annotations

import math
import statistics
import time
from collections import deque
from dataclasses import dataclass, field
from typing import Any, Deque, Iterable, Sequence


@dataclass
class MetricPoint:
    """A single sample of a time-series metric."""

    timestamp: float
    value: float


@dataclass
class Anomaly:
    """Result of an anomaly check."""

    metric: str
    timestamp: float
    value: float
    expected: float
    deviation: float
    severity: str  # low | medium | high | critical
    detector: str
    description: str

    def to_dict(self) -> dict[str, Any]:
        return {
            "metric": self.metric,
            "timestamp": self.timestamp,
            "value": round(self.value, 4),
            "expected": round(self.expected, 4),
            "deviation": round(self.deviation, 4),
            "severity": self.severity,
            "detector": self.detector,
            "description": self.description,
        }


@dataclass
class _StreamStats:
    """Online mean/std/variance using Welford's algorithm.

    Maintains running statistics so callers can feed samples one at a
    time without retaining the full history.
    """

    n: int = 0
    mean: float = 0.0
    m2: float = 0.0
    min: float = math.inf
    max: float = -math.inf

    def update(self, value: float) -> None:
        self.n += 1
        delta = value - self.mean
        self.mean += delta / self.n
        delta2 = value - self.mean
        self.m2 += delta * delta2
        if value < self.min:
            self.min = value
        if value > self.max:
            self.max = value

    @property
    def variance(self) -> float:
        return self.m2 / self.n if self.n > 1 else 0.0

    @property
    def stddev(self) -> float:
        return math.sqrt(self.variance)


def zscore_detect(
    metric: str,
    samples: Sequence[float],
    *,
    threshold: float = 3.0,
) -> list[Anomaly]:
    """Detect anomalies using a z-score test on the *most recent* sample.

    The detector compares the last sample to the mean/std of the
    preceding samples.  Returns an empty list when there are fewer than
    5 samples.
    """
    if len(samples) < 5:
        return []
    history = samples[:-1]
    current = samples[-1]
    mean = statistics.fmean(history)
    std = statistics.pstdev(history)
    if std == 0:
        return []
    deviation = (current - mean) / std
    if abs(deviation) < threshold:
        return []
    severity = _severity_for(deviation, threshold)
    return [
        Anomaly(
            metric=metric,
            timestamp=time.time(),
            value=current,
            expected=mean,
            deviation=deviation,
            severity=severity,
            detector="zscore",
            description=(
                f"z-score {deviation:+.2f} exceeds ±{threshold:.1f} "
                f"(mean={mean:.3f}, std={std:.3f})"
            ),
        )
    ]


def iqr_detect(
    metric: str,
    samples: Sequence[float],
    *,
    k: float = 1.5,
) -> list[Anomaly]:
    """Tukey fence / IQR based detector on the latest sample."""
    if len(samples) < 5:
        return []
    sorted_hist = sorted(samples[:-1])
    q1 = _percentile(sorted_hist, 0.25)
    q3 = _percentile(sorted_hist, 0.75)
    iqr = q3 - q1
    if iqr == 0:
        return []
    lo = q1 - k * iqr
    hi = q3 + k * iqr
    current = samples[-1]
    if lo <= current <= hi:
        return []
    deviation = (current - (q1 + q3) / 2) / iqr
    severity = _severity_for(deviation, k)
    side = "above" if current > hi else "below"
    return [
        Anomaly(
            metric=metric,
            timestamp=time.time(),
            value=current,
            expected=(q1 + q3) / 2,
            deviation=deviation,
            severity=severity,
            detector="iqr",
            description=(
                f"IQR fence violation {side} (q1={q1:.3f}, q3={q3:.3f}, "
                f"iqr={iqr:.3f}); fences=[{lo:.3f}, {hi:.3f}]"
            ),
        )
    ]


def ewma_detect(
    metric: str,
    samples: Sequence[float],
    *,
    alpha: float = 0.3,
    threshold: float = 3.5,
) -> list[Anomaly]:
    """EWMA control chart detector.

    Compares each new sample against an exponentially weighted moving
    average and a control limit of ``threshold * stddev``.
    """
    if len(samples) < 5:
        return []
    ewma = samples[0]
    ewma_var = 0.0
    for x in samples[:-1]:
        ewma = alpha * x + (1 - alpha) * ewma
        ewma_var = alpha * (x - ewma) ** 2 + (1 - alpha) * ewma_var
    ewma_std = math.sqrt(max(ewma_var, 1e-9))
    current = samples[-1]
    deviation = abs(current - ewma) / ewma_std
    if deviation < threshold:
        return []
    severity = _severity_for(deviation, threshold)
    return [
        Anomaly(
            metric=metric,
            timestamp=time.time(),
            value=current,
            expected=ewma,
            deviation=deviation,
            severity=severity,
            detector="ewma",
            description=(
                f"EWMA control limit violation "
                f"(ewma={ewma:.3f}, std={ewma_std:.3f}, deviation={deviation:.2f})"
            ),
        )
    ]


@dataclass
class _SlidingWindow:
    """Tiny sliding window over recent values for online detectors."""

    capacity: int
    buf: Deque[float] = field(default_factory=deque)

    def __post_init__(self) -> None:
        if self.buf.maxlen is None:
            self.buf = deque(maxlen=self.capacity)

    def push(self, value: float) -> None:
        self.buf.append(value)

    def values(self) -> list[float]:
        return list(self.buf)


class StreamingAnomalyDetector:
    """Online detector maintaining a sliding window per metric.

    Each :meth:`observe` call updates the window and triggers the three
    detectors; :meth:`detect_all` runs them on demand against the
    current snapshot.  Designed to be cheap enough to call on every
    metric scrape (15s).
    """

    def __init__(
        self,
        *,
        window_size: int = 60,
        zscore_threshold: float = 3.0,
        iqr_k: float = 1.5,
        ewma_alpha: float = 0.3,
        ewma_threshold: float = 3.5,
    ) -> None:
        self.window_size = window_size
        self.zscore_threshold = zscore_threshold
        self.iqr_k = iqr_k
        self.ewma_alpha = ewma_alpha
        self.ewma_threshold = ewma_threshold
        self._windows: dict[str, Deque[float]] = {}

    def _window_for(self, metric: str) -> Deque[float]:
        if metric not in self._windows:
            self._windows[metric] = deque(maxlen=self.window_size)
        return self._windows[metric]

    def observe(self, metric: str, value: float) -> list[Anomaly]:
        win = self._window_for(metric)
        win.append(value)
        return self.detect(metric)

    def detect(self, metric: str) -> list[Anomaly]:
        samples = list(self._window_for(metric))
        if len(samples) < 5:
            return []
        anomalies: list[Anomaly] = []
        anomalies.extend(zscore_detect(metric, samples, threshold=self.zscore_threshold))
        anomalies.extend(iqr_detect(metric, samples, k=self.iqr_k))
        anomalies.extend(ewma_detect(metric, samples, alpha=self.ewma_alpha, threshold=self.ewma_threshold))
        return anomalies

    def detect_all(self) -> dict[str, list[Anomaly]]:
        return {m: self.detect(m) for m in list(self._windows.keys())}


def _severity_for(deviation: float, threshold: float) -> str:
    """Map an absolute deviation to a severity bucket."""
    ratio = abs(deviation) / max(threshold, 1e-9)
    if ratio >= 4:
        return "critical"
    if ratio >= 2:
        return "high"
    if ratio >= 1.5:
        return "medium"
    return "low"


def _percentile(sorted_values: Sequence[float], q: float) -> float:
    """Linear interpolation percentile for already sorted values."""
    if not sorted_values:
        return 0.0
    if len(sorted_values) == 1:
        return sorted_values[0]
    pos = q * (len(sorted_values) - 1)
    lo = math.floor(pos)
    hi = math.ceil(pos)
    if lo == hi:
        return sorted_values[lo]
    frac = pos - lo
    return sorted_values[lo] + (sorted_values[hi] - sorted_values[lo]) * frac


def summarise(anomalies: Iterable[Anomaly]) -> dict[str, Any]:
    """Build a small summary dictionary used by the /anomalies endpoint."""
    items = list(anomalies)
    by_severity: dict[str, int] = {}
    by_metric: dict[str, int] = {}
    for a in items:
        by_severity[a.severity] = by_severity.get(a.severity, 0) + 1
        by_metric[a.metric] = by_metric.get(a.metric, 0) + 1
    return {
        "total": len(items),
        "by_severity": by_severity,
        "by_metric": by_metric,
    }


__all__ = [
    "MetricPoint",
    "Anomaly",
    "StreamingAnomalyDetector",
    "zscore_detect",
    "iqr_detect",
    "ewma_detect",
    "summarise",
]
