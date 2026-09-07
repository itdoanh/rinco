/**
 * WebSocket signaling for WebRTC
 */

export type SignalingMessage =
  | { type: "offer"; sdp: RTCSessionDescriptionInit; from: string; to: string }
  | { type: "answer"; sdp: RTCSessionDescriptionInit; from: string; to: string }
  | { type: "ice-candidate"; candidate: RTCIceCandidateInit; from: string; to: string }
  | { type: "join"; roomId: string; userId: string; userName: string }
  | { type: "leave"; userId: string }
  | { type: "chat"; content: string; from: string; fromName: string }
  | { type: "participants"; participants: Array<{ id: string; name: string }> };

export interface SignalingOptions {
  url: string;
  roomId: string;
  userId: string;
  userName: string;
  onMessage: (message: SignalingMessage) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
}

export class SignalingClient {
  private ws: WebSocket | null = null;
  private options: SignalingOptions;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor(options: SignalingOptions) {
    this.options = options;
  }

  connect(): void {
    const { url, roomId, userId, userName } = this.options;
    
    try {
      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.options.onOpen?.();
        
        // Join room
        this.send({
          type: "join",
          roomId,
          userId,
          userName,
        });
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as SignalingMessage;
          this.options.onMessage(message);
        } catch (e) {
          console.error("Failed to parse signaling message:", e);
        }
      };

      this.ws.onclose = () => {
        this.options.onClose?.();
        this.attemptReconnect();
      };

      this.ws.onerror = (error) => {
        this.options.onError?.(error);
      };
    } catch (error) {
      console.error("Failed to create WebSocket:", error);
    }
  }

  send(message: SignalingMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  sendOffer(to: string, sdp: RTCSessionDescriptionInit): void {
    this.send({
      type: "offer",
      sdp,
      from: this.options.userId,
      to,
    });
  }

  sendAnswer(to: string, sdp: RTCSessionDescriptionInit): void {
    this.send({
      type: "answer",
      sdp,
      from: this.options.userId,
      to,
    });
  }

  sendIceCandidate(to: string, candidate: RTCIceCandidateInit): void {
    this.send({
      type: "ice-candidate",
      candidate,
      from: this.options.userId,
      to,
    });
  }

  sendChatMessage(content: string): void {
    this.send({
      type: "chat",
      content,
      from: this.options.userId,
      fromName: this.options.userName,
    });
  }

  leave(): void {
    this.send({
      type: "leave",
      userId: this.options.userId,
    } as any);
    this.disconnect();
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      setTimeout(() => {
        console.log(`Reconnecting... attempt ${this.reconnectAttempts}`);
        this.connect();
      }, this.reconnectDelay * this.reconnectAttempts);
    }
  }
}
