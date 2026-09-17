/**
 * Thin client for the RINCO webrtc-sfu service.
 *
 * The SFU speaks a small JSON-over-WebSocket protocol on
 * ``ws://localhost:8095/v1/ws?room_id=…&user_id=…``.  This client wraps
 * the protocol and exposes typed helpers so the React hook
 * (``useWebRTC``) doesn't have to deal with raw JSON envelopes.
 */

export interface SfuIceServer {
  urls: string[];
  username?: string | null;
  credential?: string | null;
}

export interface SfuParticipant {
  id: string;
  user_id: string;
  display_name: string;
  is_audio_enabled: boolean;
  is_video_enabled: boolean;
  is_screen_sharing: boolean;
}

export interface SfuReady {
  room_id: string;
  participants: SfuParticipant[];
  ice_servers: SfuIceServer[];
}

export interface SfuJoinPayload {
  room_id: string;
  user_id: string;
  tenant_id: string;
  display_name: string;
}

export type SfuServerMessage =
  | { type: "ready"; room_id: string; participants: SfuParticipant[]; ice_servers: SfuIceServer[] }
  | { type: "offer"; sdp: string }
  | { type: "answer"; sdp: string }
  | {
      type: "ice";
      candidate: string;
      sdp_mid: string | null;
      sdp_mline_index: number;
    }
  | { type: "peer_joined"; participant: SfuParticipant }
  | { type: "peer_left"; user_id: string }
  | { type: "audio_mute"; user_id: string; muted: boolean }
  | { type: "video_toggle"; user_id: string; on: boolean }
  | { type: "screen_share"; user_id: string; on: boolean }
  | { type: "recording"; on: boolean }
  | { type: "room_closed" }
  | { type: "error"; code: number; message: string }
  | { type: "pong" };

export type SfuClientMessage =
  | { type: "join"; payload: SfuJoinPayload }
  | { type: "leave" }
  | { type: "offer"; sdp: string }
  | { type: "answer"; sdp: string }
  | {
      type: "ice";
      candidate: string;
      sdp_mid: string | null;
      sdp_mline_index: number;
    }
  | { type: "mute"; audio: boolean; video: boolean; screen: boolean }
  | { type: "ping" };

export interface SfuClientOptions {
  url: string;
  onMessage: (msg: SfuServerMessage) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (err: Event) => void;
}

export class SfuClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private readonly maxReconnectAttempts = 5;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private explicitlyClosed = false;

  constructor(private readonly options: SfuClientOptions) {}

  connect(): void {
    this.explicitlyClosed = false;
    try {
      this.ws = new WebSocket(this.options.url);
    } catch (err) {
      console.error("SfuClient connect failed", err);
      return;
    }

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.options.onOpen?.();
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as SfuServerMessage;
        this.options.onMessage(data);
      } catch (err) {
        console.error("SfuClient message parse failed", err);
      }
    };

    this.ws.onclose = () => {
      this.options.onClose?.();
      if (!this.explicitlyClosed) this.scheduleReconnect();
    };

    this.ws.onerror = (err) => {
      this.options.onError?.(err);
    };
  }

  send(message: SfuClientMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  join(payload: SfuJoinPayload): void {
    this.send({ type: "join", payload });
  }

  offer(sdp: string): void {
    this.send({ type: "offer", sdp });
  }

  answer(sdp: string): void {
    this.send({ type: "answer", sdp });
  }

  ice(
    candidate: RTCIceCandidateInit,
    sdpMid: string | null,
    sdpMlineIndex: number,
  ): void {
    this.send({
      type: "ice",
      candidate: candidate.candidate ?? "",
      sdp_mid: sdpMid,
      sdp_mline_index: sdpMlineIndex,
    });
  }

  setMuteState(audio: boolean, video: boolean, screen: boolean): void {
    this.send({ type: "mute", audio, video, screen });
  }

  leave(): void {
    this.explicitlyClosed = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    try {
      this.send({ type: "leave" });
    } catch {
      /* ignore */
    }
    this.disconnect();
  }

  disconnect(): void {
    try {
      this.ws?.close();
    } catch {
      /* ignore */
    }
    this.ws = null;
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return;
    this.reconnectAttempts += 1;
    const delay = 1000 * this.reconnectAttempts;
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }
}

/** Build the default SFU URL from env or services registry. */
export function defaultSfuUrl(roomId: string, userId: string): string {
  let base: string;
  try {
    // Dynamic require to avoid SSR issues with shared ui package
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { wsUrl } = require("@rinco/ui") as { wsUrl?: (svc: string) => string };
    base = wsUrl?.("webrtc-sfu") ?? process.env.NEXT_PUBLIC_SFU_URL ?? "ws://localhost:8102";
  } catch {
    base = process.env.NEXT_PUBLIC_SFU_URL ?? "ws://localhost:8102";
  }
  return `${base.replace(/\/$/, "")}/v1/ws?room_id=${encodeURIComponent(roomId)}&user_id=${encodeURIComponent(userId)}`;
}
