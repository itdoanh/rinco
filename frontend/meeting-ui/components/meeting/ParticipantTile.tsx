"use client";

import { useEffect, useRef } from "react";

/**
 * Renders a single participant's video tile.
 *
 * Kept as a presentation-only component — the screen share stream is
 * rendered full-bleed inside the tile, replacing the camera feed.
 */
export interface ParticipantTileProps {
  stream?: MediaStream | null;
  name: string;
  isLocal?: boolean;
  isMuted?: boolean;
  isVideoOn?: boolean;
  isSpeaking?: boolean;
  isScreenSharing?: boolean;
  className?: string;
}

export function ParticipantTile({
  stream,
  name,
  isLocal = false,
  isMuted = false,
  isVideoOn = true,
  isSpeaking = false,
  isScreenSharing = false,
  className,
}: ParticipantTileProps) {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  const initials = name
    .split(/\s+/)
    .map((p) => p[0] ?? "")
    .join("")
    .toUpperCase()
    .slice(0, 2) || "?";

  const showVideo = isVideoOn && stream && !isScreenSharing;
  const showScreenShare = isScreenSharing && stream;

  return (
    <div
      data-testid={isLocal ? "local-tile" : `remote-tile-${name}`}
      className={
        "relative aspect-video overflow-hidden rounded-xl bg-slate-800 ring-offset-2 ring-offset-slate-900 " +
        (isSpeaking ? "ring-4 ring-emerald-500 " : "ring-0 ") +
        (className ?? "")
      }
    >
      {showScreenShare ? (
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted
          className="h-full w-full bg-black object-contain"
        />
      ) : showVideo ? (
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted={isLocal}
          className={
            "h-full w-full object-cover " + (isLocal ? "-scale-x-100" : "")
          }
        />
      ) : (
        <div className="flex h-full w-full items-center justify-center bg-slate-800">
          <div className="flex h-20 w-20 items-center justify-center rounded-full bg-blue-600 text-2xl font-bold text-white">
            {initials}
          </div>
        </div>
      )}

      {/* Name + mute badge */}
      <div className="absolute bottom-2 left-2 flex items-center gap-2">
        <div className="flex items-center gap-1 rounded-full bg-black/60 px-3 py-1 text-xs text-white backdrop-blur">
          <span>{name}</span>
          {isMuted && <span aria-label="muted">🔇</span>}
        </div>
      </div>

      {/* Top-right indicator */}
      <div className="absolute right-2 top-2 flex items-center gap-1">
        {showScreenShare && (
          <span className="rounded bg-blue-600 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-white">
            Sharing
          </span>
        )}
        {isLocal && (
          <span className="rounded bg-emerald-500 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-white">
            You
          </span>
        )}
      </div>
    </div>
  );
}

export default ParticipantTile;
