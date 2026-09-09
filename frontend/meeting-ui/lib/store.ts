import { create } from "zustand";

export interface Participant {
  id: string;
  name: string;
  avatar?: string;
  isMuted: boolean;
  isVideoOn: boolean;
  isScreenSharing: boolean;
  isSpeaking: boolean;
  stream?: MediaStream;
}

export interface ChatMessage {
  id: string;
  senderId: string;
  senderName: string;
  content: string;
  timestamp: Date;
}

interface MeetingState {
  roomId: string | null;
  localStream: MediaStream | null;
  participants: Participant[];
  chatMessages: ChatMessage[];
  isMuted: boolean;
  isVideoOn: boolean;
  isScreenSharing: boolean;
  isRecording: boolean;
  isConnected: boolean;

  // Actions
  setRoomId: (id: string) => void;
  setLocalStream: (stream: MediaStream | null) => void;
  addParticipant: (participant: Participant) => void;
  removeParticipant: (id: string) => void;
  updateParticipant: (id: string, updates: Partial<Participant>) => void;
  toggleMute: () => void;
  toggleVideo: () => void;
  toggleScreenShare: () => void;
  toggleRecording: () => void;
  addChatMessage: (message: ChatMessage) => void;
  setConnected: (connected: boolean) => void;
  reset: () => void;
}

/**
 * Stop every track on ``stream`` so the browser releases the
 * camera/microphone hardware.  Safe to call on null.
 *
 * Exported so that React effects can clean up streams directly without
 * going through the store mutation helpers.
 */
export function stopMediaStream(stream: MediaStream | null): void {
  if (!stream) return;
  for (const track of stream.getTracks()) {
    try {
      track.stop();
    } catch {
      /* track may already be ended — ignore */
    }
  }
}

export const useMeetingStore = create<MeetingState>((set, get) => ({
  roomId: null,
  localStream: null,
  participants: [],
  chatMessages: [],
  isMuted: false,
  isVideoOn: true,
  isScreenSharing: false,
  isRecording: false,
  isConnected: false,

  setRoomId: (id) => set({ roomId: id }),

  setLocalStream: (stream) => set({ localStream: stream }),

  addParticipant: (participant) =>
    set((state) => {
      // Avoid duplicate participants (race between join and participants list).
      if (state.participants.some((p) => p.id === participant.id)) {
        return state;
      }
      return { participants: [...state.participants, participant] };
    }),

  removeParticipant: (id) =>
    set((state) => ({
      participants: state.participants.filter((p) => p.id !== id),
    })),

  updateParticipant: (id, updates) =>
    set((state) => ({
      participants: state.participants.map((p) =>
        p.id === id ? { ...p, ...updates } : p,
      ),
    })),

  /**
   * Toggle the local microphone.  Mutates ``track.enabled`` so the
   * browser-level capture is paused without renegotiating the WebRTC
   * peer connection.  No-op when ``localStream`` isn't set yet.
   */
  toggleMute: () => {
    const { localStream, isMuted } = get();
    // ``track.enabled = false`` is the WebRTC signal for "muted"; flip
    // it to the OPPOSITE of the next state we'll set, so callers see a
    // consistent picture.
    const nextMuted = !isMuted;
    if (localStream) {
      for (const track of localStream.getAudioTracks()) {
        track.enabled = !nextMuted;
      }
    }
    set({ isMuted: nextMuted });
  },

  /**
   * Toggle the local camera.  Same pattern as ``toggleMute``.
   */
  toggleVideo: () => {
    const { localStream, isVideoOn } = get();
    const nextOn = !isVideoOn;
    if (localStream) {
      for (const track of localStream.getVideoTracks()) {
        track.enabled = nextOn;
      }
    }
    set({ isVideoOn: nextOn });
  },

  toggleScreenShare: () =>
    set((state) => ({ isScreenSharing: !state.isScreenSharing })),
  toggleRecording: () =>
    set((state) => ({ isRecording: !state.isRecording })),

  /**
   * Append a chat message, deduping on ``message.id``.  Returns true
   * when the message was actually added so callers can log/ignore.
   */
  addChatMessage: (message) =>
    set((state) => {
      if (state.chatMessages.some((m) => m.id === message.id)) return state;
      return { chatMessages: [...state.chatMessages, message] };
    }),

  setConnected: (connected) => set({ isConnected: connected }),

  /**
   * Reset meeting state.  Stops the local stream's tracks before
   * clearing it so the OS releases the camera/microphone.
   */
  reset: () => {
    const { localStream } = get();
    stopMediaStream(localStream);
    set({
      roomId: null,
      localStream: null,
      participants: [],
      chatMessages: [],
      isMuted: false,
      isVideoOn: true,
      isScreenSharing: false,
      isRecording: false,
      isConnected: false,
    });
  },
}));
