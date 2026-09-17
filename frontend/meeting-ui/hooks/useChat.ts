"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import type { ChatMessage } from "@/lib/types";

/**
 * In-meeting chat hook.
 *
 * Connects to the chat-engine WebSocket (``ws://localhost:8094`` by
 * default — set ``NEXT_PUBLIC_CHAT_WS_URL`` to override) and exposes
 * the message list + a ``sendMessage`` helper.
 *
 * The connection is opened lazily on the first ``sendMessage`` call
 * and on the first effect run.  In a real deployment the chat engine
 * would be authenticated via a JWT — for the moment we accept any
 * ``userId``.
 */
export interface UseChatOptions {
  roomId: string;
  userId: string;
  userName: string;
  /** Override the WebSocket URL (defaults to NEXT_PUBLIC_CHAT_WS_URL or ws://localhost:8094). */
  url?: string;
}

export function useChat({ roomId, userId, userName, url }: UseChatOptions) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  const wsUrl =
    url ??
    (typeof process !== "undefined"
      ? process.env.NEXT_PUBLIC_CHAT_WS_URL
      : undefined) ??
    "ws://localhost:8094";

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (!roomId) return;

    let cancelled = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    const connect = (): void => {
      try {
        socket = new WebSocket(`${wsUrl}/ws/${roomId}?user_id=${userId}`);
      } catch {
        return;
      }
      wsRef.current = socket;

      socket.onopen = () => {
        if (cancelled) return;
        setIsConnected(true);
      };

      socket.onmessage = (event) => {
        if (cancelled) return;
        try {
          const data = JSON.parse(event.data) as
            | ChatMessage
            | { type?: string; message?: ChatMessage };
          const message: ChatMessage | undefined =
            "senderId" in data
              ? data
              : (data as { message?: ChatMessage }).message;
          if (message && message.id) {
            setMessages((prev) =>
              prev.some((m) => m.id === message.id) ? prev : [...prev, message],
            );
          }
        } catch {
          /* ignore malformed payloads */
        }
      };

      socket.onclose = () => {
        if (cancelled) return;
        setIsConnected(false);
        // Auto-reconnect after 2s — chat is best-effort during a meeting.
        reconnectTimer = setTimeout(connect, 2000);
      };

      socket.onerror = () => {
        try {
          socket?.close();
        } catch {
          /* ignore */
        }
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      try {
        socket?.close();
      } catch {
        /* ignore */
      }
      wsRef.current = null;
      setIsConnected(false);
    };
  }, [roomId, userId, wsUrl]);

  const sendMessage = useCallback(
    (text: string) => {
      const trimmed = text.trim();
      if (!trimmed) return;
      const message: ChatMessage = {
        id:
          typeof crypto !== "undefined" && "randomUUID" in crypto
            ? crypto.randomUUID()
            : `${Date.now()}-${Math.random().toString(36).slice(2)}`,
        senderId: userId,
        senderName: userName,
        content: trimmed,
        timestamp: Date.now(),
      };
      // Optimistic echo so the local sender sees their own message immediately.
      setMessages((prev) =>
        prev.some((m) => m.id === message.id) ? prev : [...prev, message],
      );
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify(message));
      }
    },
    [userId, userName],
  );

  return { messages, sendMessage, isConnected };
}
