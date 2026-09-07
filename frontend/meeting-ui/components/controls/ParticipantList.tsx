"use client";

import { useMeetingStore } from "@/lib/store";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { X, Mic, MicOff, Video, VideoOff } from "lucide-react";

interface ParticipantListProps {
  onClose: () => void;
}

export function ParticipantList({ onClose }: ParticipantListProps) {
  const { participants, localStream, toggleMute, toggleVideo, isMuted, isVideoOn } = useMeetingStore();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-bold">Participants ({participants.length + 1})</h3>
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="w-4 h-4" />
        </Button>
      </div>

      <div className="space-y-2">
        {/* Self */}
        <div className="flex items-center justify-between p-3 rounded-lg bg-gray-50">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-primary flex items-center justify-center text-white font-bold">
              Y
            </div>
            <div>
              <p className="font-medium">You</p>
              <p className="text-sm text-gray-500">Host</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleMute}
              className="rounded-full"
            >
              {isMuted ? (
                <MicOff className="w-4 h-4 text-red-500" />
              ) : (
                <Mic className="w-4 h-4" />
              )}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleVideo}
              className="rounded-full"
            >
              {isVideoOn ? (
                <Video className="w-4 h-4" />
              ) : (
                <VideoOff className="w-4 h-4 text-red-500" />
              )}
            </Button>
          </div>
        </div>

        {/* Other participants */}
        {participants.map((participant) => (
          <div
            key={participant.id}
            className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50"
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full bg-emerald-500 flex items-center justify-center text-white font-bold">
                {participant.name[0]?.toUpperCase()}
              </div>
              <div>
                <p className="font-medium">{participant.name}</p>
                <p className="text-sm text-gray-500">Participant</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {participant.isMuted ? (
                <MicOff className="w-4 h-4 text-red-500" />
              ) : (
                <Mic className="w-4 h-4 text-gray-400" />
              )}
              {!participant.isVideoOn && (
                <VideoOff className="w-4 h-4 text-red-500" />
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default ParticipantList;
