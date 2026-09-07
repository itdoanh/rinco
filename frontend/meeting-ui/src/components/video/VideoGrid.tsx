'use client';

import { VideoTile } from './VideoTile';

interface Participant {
  id: string;
  name: string;
  stream?: MediaStream;
  isMuted: boolean;
  isVideoOff: boolean;
  isScreenSharing: boolean;
  isSpeaking: boolean;
}

interface VideoGridProps {
  localStream?: MediaStream;
  localName: string;
  participants: Participant[];
  isMuted: boolean;
  isVideoOff: boolean;
}

export function VideoGrid({
  localStream,
  localName,
  participants,
  isMuted,
  isVideoOff,
}: VideoGridProps) {
  const allParticipants = [
    {
      id: 'local',
      name: localName,
      stream: localStream,
      isMuted,
      isVideoOff,
      isScreenSharing: false,
      isSpeaking: false,
    },
    ...participants,
  ];

  const gridClass = getGridClass(allParticipants.length);

  return (
    <div className={`grid ${gridClass} gap-4 h-full`}>
      {allParticipants.map((participant) => (
        <VideoTile
          key={participant.id}
          participant={participant}
          isLocal={participant.id === 'local'}
        />
      ))}

      {/* Empty slots */}
      {Array.from({ length: Math.max(0, 4 - allParticipants.length) }).map((_, i) => (
        <div
          key={`empty-${i}`}
          className="bg-gray-800 rounded-xl flex items-center justify-center border border-gray-700"
        >
          <span className="text-gray-600 text-sm">Waiting for participants...</span>
        </div>
      ))}
    </div>
  );
}

function getGridClass(count: number): string {
  switch (count) {
    case 1:
      return 'grid-cols-1';
    case 2:
      return 'grid-cols-2';
    case 3:
      return 'grid-cols-2 grid-rows-2';
    case 4:
      return 'grid-cols-2 grid-rows-2';
    case 5:
    case 6:
      return 'grid-cols-3 grid-rows-2';
    case 7:
    case 8:
    case 9:
      return 'grid-cols-3 grid-rows-3';
    default:
      return 'grid-cols-4 grid-rows-3';
  }
}
