/**
 * Thin WebSocket client for chat-engine (port 8101).
 *
 * Protocol: JSON envelopes over WS at `/v1/ws?room_id=…&user_id=…`.
 * On connect, the client auto-sends `{type: "join", …}`.
 */
import { wsUrl } from "@rinco/ui";

export type ChatServerMessage =
  | { type: "ready"; room_id: string }
  | { type: "message"; id: string; room_id: string; sender_id: string; sender_name: string; content: string; timestamp: number }
  | { type: "history"; messages: Array<{ type: "message"; id: string; room_id: string; sender_id: string; sender_name: string; content: string; timestamp: number }> }
  | { type: "user_joined"; user_id: string; user_name: string }
  | { type: "user_left"; user_id: string }
  | { type: "error"; message: string }
  | { type: "pong" };

export type ChatClientMessage =
  | { type: "join"; user_id: string; user_name: string }
  | { type: "leave" }
  | { type: "message"; content: string }
  | { type: "ping" };

export interface ChatClientOptions {
  roomId: string;
  userId: string;
  userName: string;
  onMessage: (msg: ChatServerMessage) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (err: Event) => void;
}

export class ChatClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private readonly maxReconnectAttempts = 5;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private explicitlyClosed = false;

  constructor(private readonly options: ChatClientOptions) {}

  connect(): void {
    this.explicitlyClosed = false;
    const base = wsUrl("chat-engine");
    const url = `${base.replace(/\/$/, "")}/v1/ws?room_id=${encodeURIComponent(this.options.roomId)}&user_id=${encodeURIComponent(this.options.userId)}`;

    try {
      this.ws = new WebSocket(url);
    } catch (err) {
      console.error("ChatClient connect failed", err);
      return;
    }

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      // Auto-join room on connect
      this.send({
        type: "join",
        user_id: this.options.userId,
        user_name: this.options.userName,
      });
      this.options.onOpen?.();
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as ChatServerMessage;
        this.options.onMessage(data);
      } catch (err) {
        console.error("ChatClient message parse failed", err);
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

  send(message: ChatClientMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  sendMessage(content: string): void {
    this.send({ type: "message", content });
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
