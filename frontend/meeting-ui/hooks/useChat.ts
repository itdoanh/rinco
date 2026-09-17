/**
 * React hook for the in-meeting chat transport.
 *
 * Mounts a `ChatClient` WebSocket to chat-engine:8101 and pipes
 * incoming messages into the global meeting store.
 */
"use client";

import { useEffect, useRef } from "react";
import { ChatClient, type ChatServerMessage } from "@/lib/chat-client";
import { useMeetingStore } from "@/lib/store";

export interface UseChatOptions {
  roomId: string;
  userId: string;
  userName: string;
  enabled?: boolean;
}

interface IncomingMessage {
  id: string;
  sender_id: string;
  sender_name: string;
  content: string;
  timestamp: number;
}

function asIncomingMessage(m: unknown): IncomingMessage | null {
  if (!m || typeof m !== "object") return null;
  const o = m as Record<string, unknown>;
  if (typeof o.id !== "string" || typeof o.sender_id !== "string") return null;
  if (typeof o.sender_name !== "string" || typeof o.content !== "string") return null;
  return {
    id: o.id,
    sender_id: o.sender_id,
    sender_name: o.sender_name,
    content: o.content,
    timestamp: typeof o.timestamp === "number" ? o.timestamp : Date.now(),
  };
}

export function useChat({ roomId, userId, userName, enabled = true }: UseChatOptions) {
  const clientRef = useRef<ChatClient | null>(null);
  const addChatMessage = useMeetingStore((s) => s.addChatMessage);

  useEffect(() => {
    if (!enabled || !roomId || !userId) return;

    const client = new ChatClient({
      roomId,
      userId,
      userName,
      onMessage: (msg: ChatServerMessage) => {
        if (msg.type === "message") {
          const m = asIncomingMessage(msg);
          if (m) {
            addChatMessage({
              id: m.id,
              senderId: m.sender_id,
              senderName: m.sender_name,
              content: m.content,
              timestamp: new Date(m.timestamp),
            });
          }
        } else if (msg.type === "history") {
          for (const raw of msg.messages) {
            const m = asIncomingMessage(raw);
            if (m) {
              addChatMessage({
                id: m.id,
                senderId: m.sender_id,
                senderName: m.sender_name,
                content: m.content,
                timestamp: new Date(m.timestamp),
              });
            }
          }
        }
      },
      onError: (err) => console.warn("[chat] ws error", err),
    });
    clientRef.current = client;
    client.connect();

    return () => {
      client.leave();
      clientRef.current = null;
    };
  }, [roomId, userId, userName, enabled, addChatMessage]);

  const send = (content: string): void => {
    clientRef.current?.sendMessage(content);
  };

  return { send };
}
