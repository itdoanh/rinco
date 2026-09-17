"use client";

import { Button } from "@/components/ui/button";
import {
  Mic,
  MicOff,
  Video,
  VideoOff,
  MonitorUp,
  PhoneOff,
  MessageSquare,
  Users,
} from "lucide-react";

/**
 * Bottom control bar — mute/camera/share/chat/participants/leave.
 *
 * Receives the current state and toggle callbacks from the parent
 * (typically ``app/meeting/[room_id]/page.tsx``) so the bar stays a
 * presentational component.
 */
export interface ControlsProps {
  isMuted: boolean;
  isVideoOn: boolean;
  isScreenSharing: boolean;
  onToggleMute: () => void;
  onToggleVideo: () => void;
  onToggleScreenShare: () => void;
  onToggleChat: () => void;
  onToggleParticipants: () => void;
  onLeave: () => void;
}

export function Controls(props: ControlsProps) {
  return (
    <footer
      data-testid="meeting-controls"
      className="flex items-center justify-center gap-2 border-t border-white/10 bg-slate-900/95 px-4 py-3 backdrop-blur"
    >
      <Button
        data-testid="toggle-mute"
        variant={props.isMuted ? "destructive" : "secondary"}
        size="icon"
        className="h-12 w-12 rounded-full"
        onClick={props.onToggleMute}
        aria-label={props.isMuted ? "Bật mic" : "Tắt mic"}
      >
        {props.isMuted ? <MicOff className="h-5 w-5" /> : <Mic className="h-5 w-5" />}
      </Button>

      <Button
        data-testid="toggle-video"
        variant={props.isVideoOn ? "secondary" : "destructive"}
        size="icon"
        className="h-12 w-12 rounded-full"
        onClick={props.onToggleVideo}
        aria-label={props.isVideoOn ? "Tắt camera" : "Bật camera"}
      >
        {props.isVideoOn ? <Video className="h-5 w-5" /> : <VideoOff className="h-5 w-5" />}
      </Button>

      <Button
        data-testid="toggle-screen-share"
        variant={props.isScreenSharing ? "default" : "secondary"}
        size="icon"
        className="h-12 w-12 rounded-full"
        onClick={props.onToggleScreenShare}
        aria-label="Chia sẻ màn hình"
      >
        <MonitorUp className="h-5 w-5" />
      </Button>

      <Button
        data-testid="toggle-chat"
        variant="ghost"
        size="icon"
        className="h-12 w-12 rounded-full"
        onClick={props.onToggleChat}
        aria-label="Mở chat"
      >
        <MessageSquare className="h-5 w-5" />
      </Button>

      <Button
        data-testid="toggle-participants"
        variant="ghost"
        size="icon"
        className="h-12 w-12 rounded-full"
        onClick={props.onToggleParticipants}
        aria-label="Danh sách người tham gia"
      >
        <Users className="h-5 w-5" />
      </Button>

      <Button
        data-testid="leave-meeting"
        variant="destructive"
        size="lg"
        className="ml-2 rounded-full px-6"
        onClick={props.onLeave}
      >
        <PhoneOff className="mr-2 h-4 w-4" />
        Rời phòng
      </Button>
    </footer>
  );
}

export default Controls;
