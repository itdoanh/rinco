'use client';

import { Mic, MicOff, Video, VideoOff, Monitor, MonitorOff, Phone, MessageSquare } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ControlBarProps {
  isMuted: boolean;
  isVideoOff: boolean;
  isScreenSharing: boolean;
  onToggleAudio: () => void;
  onToggleVideo: () => void;
  onToggleScreenShare: () => void;
  onLeave: () => void;
}

export function ControlBar({
  isMuted,
  isVideoOff,
  isScreenSharing,
  onToggleAudio,
  onToggleVideo,
  onToggleScreenShare,
  onLeave,
}: ControlBarProps) {
  return (
    <div className="h-20 bg-gray-800 border-t border-gray-700 px-4 flex items-center justify-center gap-4">
      {/* Audio toggle */}
      <button
        onClick={onToggleAudio}
        className={cn(
          'p-4 rounded-full transition-colors',
          isMuted
            ? 'bg-red-500 hover:bg-red-600'
            : 'bg-gray-700 hover:bg-gray-600'
        )}
        title={isMuted ? 'Unmute' : 'Mute'}
      >
        {isMuted ? (
          <MicOff className="w-6 h-6 text-white" />
        ) : (
          <Mic className="w-6 h-6 text-white" />
        )}
      </button>

      {/* Video toggle */}
      <button
        onClick={onToggleVideo}
        className={cn(
          'p-4 rounded-full transition-colors',
          isVideoOff
            ? 'bg-red-500 hover:bg-red-600'
            : 'bg-gray-700 hover:bg-gray-600'
        )}
        title={isVideoOff ? 'Turn on camera' : 'Turn off camera'}
      >
        {isVideoOff ? (
          <VideoOff className="w-6 h-6 text-white" />
        ) : (
          <Video className="w-6 h-6 text-white" />
        )}
      </button>

      {/* Screen share */}
      <button
        onClick={onToggleScreenShare}
        className={cn(
          'p-4 rounded-full transition-colors',
          isScreenSharing
            ? 'bg-blue-500 hover:bg-blue-600'
            : 'bg-gray-700 hover:bg-gray-600'
        )}
        title={isScreenSharing ? 'Stop sharing' : 'Share screen'}
      >
        {isScreenSharing ? (
          <MonitorOff className="w-6 h-6 text-white" />
        ) : (
          <Monitor className="w-6 h-6 text-white" />
        )}
      </button>

      {/* Leave call */}
      <button
        onClick={onLeave}
        className="p-4 rounded-full bg-red-500 hover:bg-red-600 transition-colors"
        title="Leave meeting"
      >
        <Phone className="w-6 h-6 text-white transform rotate-135" />
      </button>
    </div>
  );
}
