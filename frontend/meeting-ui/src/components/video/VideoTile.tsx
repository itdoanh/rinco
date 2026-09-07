'use client';

import { useRef, useEffect } from 'react';
import { Video, VideoOff, Mic, MicOff } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Participant {
  id: string;
  name: string;
  stream?: MediaStream;
  isMuted: boolean;
  isVideoOff: boolean;
  isScreenSharing: boolean;
  isSpeaking: boolean;
}

interface VideoTileProps {
  participant: Participant;
  isLocal?: boolean;
}

export function VideoTile({ participant, isLocal = false }: VideoTileProps) {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (videoRef.current && participant.stream) {
      videoRef.current.srcObject = participant.stream;
    }
  }, [participant.stream]);

  return (
    <div
      className={cn(
        'relative bg-gray-800 rounded-xl overflow-hidden border-2',
        participant.isSpeaking ? 'border-green-500' : 'border-gray-700',
        isLocal && 'ring-2 ring-blue-500 ring-offset-2 ring-offset-gray-900'
      )}
    >
      {/* Video or Avatar */}
      {participant.isVideoOff || !participant.stream ? (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-700">
          <div className="w-24 h-24 rounded-full bg-gray-600 flex items-center justify-center">
            <span className="text-3xl font-bold text-gray-400">
              {participant.name.charAt(0).toUpperCase()}
            </span>
          </div>
        </div>
      ) : (
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted={isLocal}
          className={cn(
            'w-full h-full object-cover',
            participant.isScreenSharing && 'transform scale-x-[-1]'
          )}
        />
      )}

      {/* Speaking indicator */}
      {participant.isSpeaking && (
        <div className="absolute inset-0 border-4 border-green-500 rounded-xl pointer-events-none animate-pulse" />
      )}

      {/* Name label */}
      <div className="absolute bottom-0 left-0 right-0 p-3 bg-gradient-to-t from-black/80 to-transparent">
        <div className="flex items-center justify-between">
          <span className="text-white font-medium text-sm">
            {participant.name}
            {isLocal && ' (You)'}
          </span>
          <div className="flex items-center gap-2">
            {participant.isMuted && (
              <div className="p-1 bg-red-500 rounded-full">
                <MicOff className="w-3 h-3 text-white" />
              </div>
            )}
            {participant.isVideoOff && (
              <div className="p-1 bg-gray-600 rounded-full">
                <VideoOff className="w-3 h-3 text-white" />
              </div>
            )}
            {participant.isScreenSharing && (
              <div className="px-2 py-1 bg-blue-500 rounded text-xs text-white">
                Sharing
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
