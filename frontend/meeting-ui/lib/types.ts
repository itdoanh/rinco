/**
 * Shared TypeScript types for the meeting UI.
 *
 * Keep this file framework-agnostic (no React, no Next.js) so it can
 * be imported from both client components and tests.
 */

/** A participant visible in the room. */
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

/** A single chat message exchanged during the meeting. */
export interface ChatMessage {
  id: string;
  senderId: string;
  senderName: string;
  content: string;
  /** Unix epoch milliseconds.  Using a number keeps the JSON wire
   *  format simple and survives SSR re-hydration without surprises. */
  timestamp: number;
}

/** Screen-share state returned by ``useScreenShare``. */
export interface ScreenShareState {
  stream: MediaStream | null;
  isSharing: boolean;
  error: string | null;
}

/** Connection state reported by the WebRTC hook. */
export type ConnectionState =
  | "idle"
  | "connecting"
  | "connected"
  | "disconnected"
  | "reconnecting"
  | "failed";
