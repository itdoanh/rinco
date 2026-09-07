import { create } from 'zustand';

export interface Participant {
  id: string;
  name: string;
  stream?: MediaStream;
  isMuted: boolean;
  isVideoOff: boolean;
  isScreenSharing: boolean;
  isSpeaking: boolean;
  joinedAt: Date;
}

export interface RoomState {
  roomId: string | null;
  localStream: MediaStream | null;
  participants: Map<string, Participant>;
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;

  // Actions
  setRoomId: (roomId: string) => void;
  setLocalStream: (stream: MediaStream | null) => void;
  addParticipant: (participant: Participant) => void;
  removeParticipant: (participantId: string) => void;
  updateParticipant: (participantId: string, updates: Partial<Participant>) => void;
  setConnected: (connected: boolean) => void;
  setConnecting: (connecting: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

export const useRoomStore = create<RoomState>((set) => ({
  roomId: null,
  localStream: null,
  participants: new Map(),
  isConnected: false,
  isConnecting: false,
  error: null,

  setRoomId: (roomId) => set({ roomId }),

  setLocalStream: (stream) => set({ localStream: stream }),

  addParticipant: (participant) =>
    set((state) => {
      const newParticipants = new Map(state.participants);
      newParticipants.set(participant.id, participant);
      return { participants: newParticipants };
    }),

  removeParticipant: (participantId) =>
    set((state) => {
      const newParticipants = new Map(state.participants);
      const participant = newParticipants.get(participantId);
      if (participant?.stream) {
        participant.stream.getTracks().forEach((track) => track.stop());
      }
      newParticipants.delete(participantId);
      return { participants: newParticipants };
    }),

  updateParticipant: (participantId, updates) =>
    set((state) => {
      const newParticipants = new Map(state.participants);
      const participant = newParticipants.get(participantId);
      if (participant) {
        newParticipants.set(participantId, { ...participant, ...updates });
      }
      return { participants: newParticipants };
    }),

  setConnected: (connected) => set({ isConnected: connected }),

  setConnecting: (connecting) => set({ isConnecting: connecting }),

  setError: (error) => set({ error }),

  reset: () => {
    set((state) => {
      // Stop all streams
      state.localStream?.getTracks().forEach((track) => track.stop());
      state.participants.forEach((participant) => {
        participant.stream?.getTracks().forEach((track) => track.stop());
      });
    });
    set({
      roomId: null,
      localStream: null,
      participants: new Map(),
      isConnected: false,
      isConnecting: false,
      error: null,
    });
  },
}));
