"use client";

import { useState, useEffect, useRef } from "react";
import {
  Mic,
  MicOff,
  Video,
  VideoOff,
  Monitor,
  MonitorOff,
  PhoneOff,
  MessageSquare,
  Users,
  Settings,
  Maximize2,
  Minimize2,
  Record,
  MoreVertical,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { useMeetingStore } from "@/lib/store";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ChatPanel } from "./ChatPanel";
import { ParticipantList } from "./ParticipantList";
import { cn } from "@/lib/utils";

export function ControlBar() {
  const {
    isMuted,
    isVideoOn,
    isScreenSharing,
    isRecording,
    toggleMute,
    toggleVideo,
    toggleScreenShare,
    toggleRecording,
    participants,
  } = useMeetingStore();

  const [isFullscreen, setIsFullscreen] = useState(false);
  const [showChat, setShowChat] = useState(false);
  const [showParticipants, setShowParticipants] = useState(false);

  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen();
      setIsFullscreen(true);
    } else {
      document.exitFullscreen();
      setIsFullscreen(false);
    }
  };

  const leaveMeeting = () => {
    // Cleanup and redirect
    window.location.href = "/";
  };

  return (
    <div className="control-bar">
      <div className="max-w-5xl mx-auto px-4 py-4">
        <div className="flex items-center justify-between gap-4">
          {/* Left: Recording indicator */}
          <div className="flex items-center gap-4">
            {isRecording && (
              <div className="flex items-center gap-2 text-red-500">
                <div className="recording-dot" />
                <span className="text-sm font-medium">Recording</span>
              </div>
            )}
            <div className="text-sm text-gray-400">
              <Users className="w-4 h-4 inline mr-1" />
              {participants.length + 1} participants
            </div>
          </div>

          {/* Center: Main controls */}
          <div className="flex items-center gap-2">
            {/* Microphone */}
            <Button
              variant={isMuted ? "destructive" : "secondary"}
              size="icon"
              onClick={toggleMute}
              className="rounded-full w-12 h-12"
            >
              {isMuted ? (
                <MicOff className="w-5 h-5" />
              ) : (
                <Mic className="w-5 h-5" />
              )}
            </Button>

            {/* Camera */}
            <Button
              variant={!isVideoOn ? "destructive" : "secondary"}
              size="icon"
              onClick={toggleVideo}
              className="rounded-full w-12 h-12"
            >
              {isVideoOn ? (
                <Video className="w-5 h-5" />
              ) : (
                <VideoOff className="w-5 h-5" />
              )}
            </Button>

            {/* Screen share */}
            <Button
              variant={isScreenSharing ? "default" : "secondary"}
              size="icon"
              onClick={toggleScreenShare}
              className="rounded-full w-12 h-12"
            >
              {isScreenSharing ? (
                <MonitorOff className="w-5 h-5" />
              ) : (
                <Monitor className="w-5 h-5" />
              )}
            </Button>

            {/* Record */}
            <Button
              variant={isRecording ? "destructive" : "secondary"}
              size="icon"
              onClick={toggleRecording}
              className="rounded-full w-12 h-12"
            >
              <Record className={cn("w-5 h-5", isRecording && "animate-pulse")} />
            </Button>

            {/* Leave */}
            <Button
              variant="destructive"
              size="icon"
              onClick={leaveMeeting}
              className="rounded-full w-12 h-12"
            >
              <PhoneOff className="w-5 h-5" />
            </Button>
          </div>

          {/* Right: Additional controls */}
          <div className="flex items-center gap-2">
            {/* Chat */}
            <Dialog open={showChat} onOpenChange={setShowChat}>
              <DialogTrigger asChild>
                <Button variant="ghost" size="icon" className="rounded-full">
                  <MessageSquare className="w-5 h-5" />
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-md h-[80vh] p-0">
                <ChatPanel onClose={() => setShowChat(false)} />
              </DialogContent>
            </Dialog>

            {/* Participants */}
            <Dialog open={showParticipants} onOpenChange={setShowParticipants}>
              <DialogTrigger asChild>
                <Button variant="ghost" size="icon" className="rounded-full">
                  <Users className="w-5 h-5" />
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-sm">
                <ParticipantList onClose={() => setShowParticipants(false)} />
              </DialogContent>
            </Dialog>

            {/* Fullscreen */}
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleFullscreen}
              className="rounded-full"
            >
              {isFullscreen ? (
                <Minimize2 className="w-5 h-5" />
              ) : (
                <Maximize2 className="w-5 h-5" />
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ControlBar;
