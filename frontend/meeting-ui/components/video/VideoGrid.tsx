"use client";

import { useRef, useEffect } from "react";
import { useMeetingStore } from "@/lib/store";
import { VideoTile } from "../video/VideoTile";
import { cn } from "@/lib/utils";

export function VideoGrid() {
  const { participants, localStream, isSpeaking } = useMeetingStore();
  const gridRef = useRef<HTMLDivElement>(null);

  // Calculate grid layout based on participant count
  const totalParticipants = participants.length + (localStream ? 1 : 0);
  
  const getGridClass = () => {
    switch (totalParticipants) {
      case 1:
        return "grid-cols-1";
      case 2:
        return "grid-cols-1 md:grid-cols-2";
      case 3:
      case 4:
        return "grid-cols-2";
      case 5:
      case 6:
        return "grid-cols-2 md:grid-cols-3";
      default:
        return "grid-cols-2 md:grid-cols-3 lg:grid-cols-4";
    }
  };

  return (
    <div
      ref={gridRef}
      className={cn(
        "grid gap-4 p-4 h-full",
        getGridClass()
      )}
    >
      {/* Local video */}
      {localStream && (
        <VideoTile
          stream={localStream}
          name="You"
          isLocal
          isMuted={false}
          isVideoOn={true}
        />
      )}

      {/* Remote participants */}
      {participants.map((participant) => (
        <VideoTile
          key={participant.id}
          stream={participant.stream}
          name={participant.name}
          isMuted={participant.isMuted}
          isVideoOn={participant.isVideoOn}
          isSpeaking={participant.isSpeaking}
        />
      ))}

      {/* Empty state */}
      {totalParticipants === 0 && (
        <div className="col-span-full flex items-center justify-center h-full text-gray-500">
          <div className="text-center">
            <div className="text-6xl mb-4">🎥</div>
            <p>Waiting for participants to join...</p>
          </div>
        </div>
      )}
    </div>
  );
}

export default VideoGrid;
