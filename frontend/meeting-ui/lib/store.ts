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
    set((state) => ({
      participants: [...state.participants, participant],
    })),
  
  removeParticipant: (id) =>
    set((state) => ({
      participants: state.participants.filter((p) => p.id !== id),
    })),
  
  updateParticipant: (id, updates) =>
    set((state) => ({
      participants: state.participants.map((p) =>
        p.id === id ? { ...p, ...updates } : p
      ),
    })),
  
  toggleMute: () => {
    const { localStream, isMuted } = get();
    if (localStream) {
      localStream.getAudioTracks().forEach((track) => {
        track.enabled = isMuted;
      });
    }
    set({ isMuted: !isMuted });
  },
  
  toggleVideo: () => {
    const { localStream, isVideoOn } = get();
    if (localStream) {
      localStream.getVideoTracks().forEach((track) => {
        track.enabled = !isVideoOn;
      });
    }
    set({ isVideoOn: !isVideoOn });
  },
  
  toggleScreenShare: () => set((state) => ({ isScreenSharing: !state.isScreenSharing })),
  toggleRecording: () => set((state) => ({ isRecording: !state.isRecording })),
  
  addChatMessage: (message) =>
    set((state) => ({
      chatMessages: [...state.chatMessages, message],
    })),
  
  setConnected: (connected) => set({ isConnected: connected }),
  
  reset: () =>
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
    }),
}));
