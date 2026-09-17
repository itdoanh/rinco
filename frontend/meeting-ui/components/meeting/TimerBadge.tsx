"use client";

import { useEffect, useState } from "react";

/**
 * Live timer badge — displays the elapsed time since ``startedAt``.
 *
 * Updates once per second.  Stops counting after ``pausedAt`` so the
 * UI can freeze the timer when the host pauses the meeting.
 */
export interface TimerBadgeProps {
  startedAt: number;
  pausedAt?: number | null;
  className?: string;
}

function format(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  const pad = (n: number) => n.toString().padStart(2, "0");
  return hours > 0
    ? `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`
    : `${pad(minutes)}:${pad(seconds)}`;
}

export function TimerBadge({ startedAt, pausedAt, className }: TimerBadgeProps) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (pausedAt) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [pausedAt]);

  const elapsed = (pausedAt ?? now) - startedAt;
  return (
    <span
      data-testid="timer-badge"
      className={
        "inline-flex items-center gap-1 rounded-full bg-black/40 px-3 py-1 text-xs font-mono text-white " +
        (className ?? "")
      }
    >
      <span aria-hidden="true">⏱</span>
      <span>{format(elapsed)}</span>
    </span>
  );
}

export default TimerBadge;
