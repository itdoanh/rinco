import axios from 'axios';

const SIGNALING_URL = process.env.NEXT_PUBLIC_SIGNALING_URL || 'ws://localhost:8080/ws';

export type SignalingMessageType =
  | 'offer'
  | 'answer'
  | 'ice_candidate'
  | 'join'
  | 'leave'
  | 'user_joined'
  | 'user_left'
  | 'toggle_audio'
  | 'toggle_video'
  | 'screen_share_started'
  | 'screen_share_stopped';

export interface SignalingMessage {
  type: SignalingMessageType;
  from?: string;
  to?: string;
  data?: unknown;
}

export class SignalingClient {
  private ws: WebSocket | null = null;
  private url: string;
  private roomId: string;
  private userId: string;
  private userName: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor(roomId: string, userId: string, userName: string) {
    this.roomId = roomId;
    this.userId = userId;
    this.userName = userName;
    this.url = SIGNALING_URL;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        const wsUrl = `${this.url}?room_id=${this.roomId}&user_id=${this.userId}&user_name=${encodeURIComponent(this.userName)}`;
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          console.log('[Signaling] Connected');
          this.reconnectAttempts = 0;
          resolve();
        };

        this.ws.onerror = (error) => {
          console.error('[Signaling] Error:', error);
          reject(error);
        };

        this.ws.onclose = (event) => {
          console.log('[Signaling] Closed:', event.code, event.reason);
          if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
            console.log(`[Signaling] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
            setTimeout(() => this.connect(), delay);
          }
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close(1000, 'User disconnected');
      this.ws = null;
    }
  }

  send(message: SignalingMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.warn('[Signaling] Cannot send, not connected');
    }
  }

  sendOffer(to: string, offer: RTCSessionDescriptionInit): void {
    this.send({ type: 'offer', to, data: offer });
  }

  sendAnswer(to: string, answer: RTCSessionDescriptionInit): void {
    this.send({ type: 'answer', to, data: answer });
  }

  sendIceCandidate(to: string, candidate: RTCIceCandidateInit): void {
    this.send({ type: 'ice_candidate', to, data: candidate });
  }

  leave(): void {
    this.send({ type: 'leave' });
    this.disconnect();
  }

  onMessage(handler: (message: SignalingMessage) => void): void {
    if (this.ws) {
      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as SignalingMessage;
          handler(message);
        } catch (error) {
          console.error('[Signaling] Failed to parse message:', error);
        }
      };
    }
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

export { SIGNALING_URL };
