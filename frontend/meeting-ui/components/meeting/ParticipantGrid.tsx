"use client";

import { ParticipantTile } from "./ParticipantTile";
import type { Participant } from "@/lib/types";

/**
 * Responsive participant grid.  Adapts column count to participant
 * count so a 1:1 call stays full-bleed while a 12-person call uses a
 * 4×3 grid.
 */
export interface ParticipantGridProps {
  participants: Participant[];
  localName: string;
  localStream?: MediaStream | null;
  isLocalMuted: boolean;
  isLocalVideoOn: boolean;
  isLocalScreenSharing: boolean;
  className?: string;
}

function gridCols(total: number): string {
  if (total <= 1) return "grid-cols-1";
  if (total <= 2) return "grid-cols-1 md:grid-cols-2";
  if (total <= 4) return "grid-cols-2";
  if (total <= 6) return "grid-cols-2 md:grid-cols-3";
  if (total <= 9) return "grid-cols-3";
  return "grid-cols-3 md:grid-cols-4";
}

export function ParticipantGrid({
  participants,
  localName,
  localStream,
  isLocalMuted,
  isLocalVideoOn,
  isLocalScreenSharing,
  className,
}: ParticipantGridProps) {
  const total = participants.length + (localStream ? 1 : 0);
  return (
    <div
      data-testid="participant-grid"
      className={
        "grid h-full gap-3 p-4 " + gridCols(total) + " " + (className ?? "")
      }
    >
      {localStream && (
        <ParticipantTile
          stream={localStream}
          name={`${localName} (Bạn)`}
          isLocal
          isMuted={isLocalMuted}
          isVideoOn={isLocalVideoOn}
          isScreenSharing={isLocalScreenSharing}
        />
      )}
      {participants.map((p) => (
        <ParticipantTile
          key={p.id}
          stream={p.stream}
          name={p.name}
          isMuted={p.isMuted}
          isVideoOn={p.isVideoOn}
          isSpeaking={p.isSpeaking}
          isScreenSharing={p.isScreenSharing}
        />
      ))}
    </div>
  );
}

export default ParticipantGrid;
