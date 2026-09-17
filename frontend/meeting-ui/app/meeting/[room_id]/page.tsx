"use client";

import { Suspense, use, useState, useEffect } from "react";
import { useSearchParams } from "next/navigation";
import { VideoGrid } from "@/components/video/VideoGrid";
import { ControlBar } from "@/components/controls/ControlBar";
import { RoomJoin } from "@/components/meeting/RoomJoin";
import { useWebRTC } from "@/hooks/useWebRTC";
import { useMeetingStore } from "@/lib/store";

interface PageProps {
  params: Promise<{ room_id: string }>;
}

function MeetingContent({ roomId }: { roomId: string }) {
  const searchParams = useSearchParams();
  const nameParam = searchParams.get("name") || "";
  const [userName, setUserName] = useState<string | null>(
    nameParam.trim() ? nameParam : null,
  );

  // Resolve initial userName from session storage if available (set by RoomJoin).
  useEffect(() => {
    if (userName) return;
    if (typeof window === "undefined") return;
    const stored = sessionStorage.getItem("meeting_name");
    if (stored && stored.trim()) setUserName(stored);
  }, [userName]);

  if (!userName) {
    return <RoomJoin roomId={roomId} onJoined={(n) => setUserName(n)} />;
  }

  const userId = `user-${Date.now()}`;

  const { isConnecting, error } = useWebRTC({ roomId, userId, userName });
  const { isConnected, isRecording } = useMeetingStore();

  if (isConnecting) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4" />
          <h2 className="text-xl font-bold text-white">Connecting to meeting...</h2>
          <p className="text-gray-400 mt-2">Room: {roomId}</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-900 flex items-center justify-center">
        <div className="text-center max-w-md">
          <div className="text-6xl mb-4">⚠️</div>
          <h2 className="text-xl font-bold text-white mb-2">Connection Error</h2>
          <p className="text-gray-400 mb-6">{error}</p>
          <a
            href="/"
            className="inline-block px-6 py-3 bg-primary text-white rounded-lg"
          >
            Return Home
          </a>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-900 flex flex-col">
      {/* Header */}
      <header className="h-14 bg-gray-900/50 backdrop-blur-sm border-b border-white/10 px-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center text-white font-bold">
            R
          </div>
          <span className="font-bold text-white">RINCO Meeting</span>
        </div>
        <div className="flex items-center gap-4">
          {isConnected ? (
            <div className="flex items-center gap-2 text-emerald-500">
              <div className="w-2 h-2 rounded-full bg-emerald-500" />
              <span className="text-sm">Connected</span>
            </div>
          ) : (
            <div className="flex items-center gap-2 text-red-500">
              <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
              <span className="text-sm">Reconnecting...</span>
            </div>
          )}
          {isRecording && (
            <div className="flex items-center gap-2 text-red-500">
              <div className="recording-dot" />
              <span className="text-sm">Recording</span>
            </div>
          )}
        </div>
      </header>

      {/* Video grid */}
      <div className="flex-1 overflow-hidden">
        <VideoGrid />
      </div>

      {/* Control bar */}
      <ControlBar />
    </div>
  );
}

export default function MeetingPage({ params }: PageProps) {
  const { room_id: roomId } = use(params);

  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-gray-900 flex items-center justify-center">
          <div className="text-white">Loading...</div>
        </div>
      }
    >
      <MeetingContent roomId={roomId} />
    </Suspense>
  );
}
