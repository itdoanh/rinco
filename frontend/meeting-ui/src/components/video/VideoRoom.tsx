'use client';

import { useEffect, useState, useRef } from 'react';
import { useWebRTC } from '@/hooks/useWebRTC';
import { VideoGrid } from '@/components/video/VideoGrid';
import { ControlBar } from '@/components/controls/ControlBar';
import { ParticipantList } from '@/components/controls/ParticipantList';
import { ChatPanel } from '@/components/chat/ChatPanel';
import { Video, VideoOff, Mic, MicOff, Monitor, MonitorOff, Phone } from 'lucide-react';
import Link from 'next/link';

interface VideoRoomProps {
  roomId: string;
}

export function VideoRoom({ roomId }: VideoRoomProps) {
  const [userName] = useState(`User_${Math.random().toString(36).substring(7)}`);
  const userId = useRef(`user_${Date.now()}`).current;
  const [isChatOpen, setIsChatOpen] = useState(false);
  const [isParticipantsOpen, setIsParticipantsOpen] = useState(false);

  const {
    localStream,
    isConnected,
    isConnecting,
    error,
    participants,
    toggleAudio,
    toggleVideo,
    startScreenShare,
    stopScreenShare,
    disconnect,
  } = useWebRTC({
    roomId,
    userId,
    userName,
    signalingUrl: process.env.NEXT_PUBLIC_SIGNALING_URL || 'ws://localhost:8080/ws',
  });

  const [isScreenSharing, setIsScreenSharing] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [isVideoOff, setIsVideoOff] = useState(false);

  const handleToggleAudio = () => {
    toggleAudio();
    setIsMuted(!isMuted);
  };

  const handleToggleVideo = () => {
    toggleVideo();
    setIsVideoOff(!isVideoOff);
  };

  const handleToggleScreenShare = () => {
    if (isScreenSharing) {
      stopScreenShare();
    } else {
      startScreenShare();
    }
    setIsScreenSharing(!isScreenSharing);
  };

  const handleLeave = () => {
    disconnect();
    window.location.href = '/';
  };

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-900">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-white mb-4">Connection Error</h1>
          <p className="text-red-400 mb-6">{error}</p>
          <button
            onClick={() => window.location.reload()}
            className="px-6 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-900 flex flex-col">
      {/* Header */}
      <header className="h-14 bg-gray-800 border-b border-gray-700 px-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/" className="text-xl font-bold text-blue-400">
            RINCO Meeting
          </Link>
          <span className="text-gray-400 text-sm">Room: {roomId}</span>
        </div>
        <div className="flex items-center gap-2">
          {isConnecting && (
            <span className="text-yellow-400 text-sm animate-pulse">Connecting...</span>
          )}
          {isConnected && (
            <span className="flex items-center gap-2 text-green-400 text-sm">
              <span className="w-2 h-2 bg-green-400 rounded-full animate-pulse" />
              Connected
            </span>
          )}
          <button
            onClick={() => setIsParticipantsOpen(!isParticipantsOpen)}
            className="p-2 rounded-lg hover:bg-gray-700 text-gray-400 hover:text-white"
          >
            {participants.length + 1} participants
          </button>
          <button
            onClick={() => setIsChatOpen(!isChatOpen)}
            className="p-2 rounded-lg hover:bg-gray-700 text-gray-400 hover:text-white"
          >
            Chat
          </button>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 flex overflow-hidden">
        {/* Video Grid */}
        <div className="flex-1 p-4">
          <VideoGrid
            localStream={localStream || undefined}
            localName={userName}
            participants={participants}
            isMuted={isMuted}
            isVideoOff={isVideoOff}
          />
        </div>

        {/* Chat Panel */}
        {isChatOpen && (
          <aside className="w-80 bg-gray-800 border-l border-gray-700">
            <ChatPanel roomId={roomId} userId={userId} userName={userName} />
          </aside>
        )}
      </main>

      {/* Control Bar */}
      <ControlBar
        isMuted={isMuted}
        isVideoOff={isVideoOff}
        isScreenSharing={isScreenSharing}
        onToggleAudio={handleToggleAudio}
        onToggleVideo={handleToggleVideo}
        onToggleScreenShare={handleToggleScreenShare}
        onLeave={handleLeave}
      />

      {/* Participant List Sidebar */}
      {isParticipantsOpen && (
        <aside className="absolute right-0 top-14 bottom-20 w-72 bg-gray-800 border-l border-gray-700 z-10">
          <ParticipantList
            participants={participants}
            localUser={{ id: userId, name: userName, isMuted, isVideoOff }}
          />
        </aside>
      )}
    </div>
  );
}
