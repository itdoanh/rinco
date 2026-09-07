"use client";

import { useRef, useEffect } from "react";
import { cn } from "@/lib/utils";
import { MicOff, VideoOff } from "lucide-react";

interface VideoTileProps {
  stream?: MediaStream | null;
  name: string;
  isLocal?: boolean;
  isMuted: boolean;
  isVideoOn: boolean;
  isSpeaking?: boolean;
  isScreenSharing?: boolean;
}

export function VideoTile({
  stream,
  name,
  isLocal = false,
  isMuted,
  isVideoOn,
  isSpeaking = false,
  isScreenSharing = false,
}: VideoTileProps) {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream;
    }
  }, [stream]);

  const initials = name
    .split(" ")
    .map((n) => n[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);

  return (
    <div
      className={cn(
        "video-tile relative",
        isSpeaking && "speaking"
      )}
    >
      {/* Video element */}
      {isVideoOn && stream ? (
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted={isLocal}
          className={cn(
            "w-full h-full object-cover",
            isLocal && "scale-x-[-1]"
          )}
        />
      ) : (
        /* Avatar placeholder */
        <div className="w-full h-full flex items-center justify-center bg-gray-800">
          <div className="w-20 h-20 rounded-full bg-primary/20 flex items-center justify-center">
            <span className="text-3xl font-bold text-white">{initials}</span>
          </div>
        </div>
      )}

      {/* Name badge */}
      <div className="absolute bottom-2 left-2 flex items-center gap-2">
        <div className="bg-black/50 backdrop-blur-sm px-3 py-1 rounded-full flex items-center gap-2">
          <span className="text-sm font-medium text-white">{name}</span>
          {isMuted && <MicOff className="w-4 h-4 text-red-400" />}
        </div>
        {isScreenSharing && (
          <div className="bg-blue-500 px-2 py-1 rounded text-xs font-medium text-white">
            Sharing
          </div>
        )}
      </div>

      {/* Video off indicator */}
      {!isVideoOn && (
        <div className="absolute top-2 right-2 bg-red-500/80 p-1.5 rounded-full">
          <VideoOff className="w-4 h-4 text-white" />
        </div>
      )}

      {/* Local indicator */}
      {isLocal && (
        <div className="absolute top-2 left-2 bg-emerald-500 px-2 py-0.5 rounded text-xs font-medium text-white">
          You
        </div>
      )}
    </div>
  );
}

export default VideoTile;
