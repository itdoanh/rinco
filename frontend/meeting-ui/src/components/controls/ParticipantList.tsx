'use client';

import { MicOff, VideoOff } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Participant {
  id: string;
  name: string;
  isMuted: boolean;
  isVideoOff: boolean;
}

interface ParticipantListProps {
  participants: Participant[];
  localUser: Participant;
}

export function ParticipantList({ participants, localUser }: ParticipantListProps) {
  const allParticipants = [localUser, ...participants];

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b border-gray-700">
        <h3 className="text-white font-semibold">
          Participants ({allParticipants.length})
        </h3>
      </div>

      <div className="flex-1 overflow-y-auto p-2">
        {allParticipants.map((participant) => (
          <div
            key={participant.id}
            className="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-700/50"
          >
            <div className="w-10 h-10 rounded-full bg-gray-600 flex items-center justify-center">
              <span className="text-white font-medium">
                {participant.name.charAt(0).toUpperCase()}
              </span>
            </div>

            <div className="flex-1 min-w-0">
              <div className="text-white text-sm font-medium truncate">
                {participant.name}
                {participant.id === localUser.id && ' (You)'}
              </div>
            </div>

            <div className="flex items-center gap-1">
              {participant.isMuted && (
                <div className="p-1 bg-gray-600 rounded-full">
                  <MicOff className="w-3 h-3 text-gray-400" />
                </div>
              )}
              {participant.isVideoOff && (
                <div className="p-1 bg-gray-600 rounded-full">
                  <VideoOff className="w-3 h-3 text-gray-400" />
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
