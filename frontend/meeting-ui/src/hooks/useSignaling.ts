import { useEffect, useRef, useCallback, useState } from 'react';

export interface SignalingMessage {
  type: string;
  from?: string;
  to?: string;
  data?: unknown;
}

export interface UseSignalingOptions {
  url: string;
  roomId: string;
  userId: string;
  userName: string;
  onMessage?: (message: SignalingMessage) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
}

export interface UseSignalingReturn {
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
  send: (message: SignalingMessage) => void;
  connect: () => void;
  disconnect: () => void;
}

export function useSignaling({
  url,
  roomId,
  userId,
  userName,
  onMessage,
  onConnect,
  onDisconnect,
  onError,
}: UseSignalingOptions): UseSignalingReturn {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout>>();
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Connect to signaling server
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    setIsConnecting(true);
    setError(null);

    try {
      const wsUrl = `${url}?room_id=${roomId}&user_id=${userId}&user_name=${encodeURIComponent(userName)}`;
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('[Signaling] Connected');
        setIsConnected(true);
        setIsConnecting(false);
        onConnect?.();
      };

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as SignalingMessage;
          onMessage?.(message);
        } catch (err) {
          console.error('[Signaling] Failed to parse message:', err);
        }
      };

      ws.onerror = (err) => {
        console.error('[Signaling] Error:', err);
        setError('Connection error');
        onError?.(err);
      };

      ws.onclose = (event) => {
        console.log('[Signaling] Disconnected:', event.code, event.reason);
        setIsConnected(false);
        setIsConnecting(false);

        if (!event.wasClean) {
          // Auto reconnect after 3 seconds
          reconnectTimeoutRef.current = setTimeout(() => {
            console.log('[Signaling] Attempting to reconnect...');
            connect();
          }, 3000);
        }

        onDisconnect?.();
      };

      wsRef.current = ws;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect');
      setIsConnecting(false);
    }
  }, [url, roomId, userId, userName, onMessage, onConnect, onDisconnect, onError]);

  // Disconnect from signaling server
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = undefined;
    }

    if (wsRef.current) {
      wsRef.current.close(1000, 'User disconnected');
      wsRef.current = null;
    }

    setIsConnected(false);
    setIsConnecting(false);
  }, []);

  // Send message
  const send = useCallback((message: SignalingMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    } else {
      console.warn('[Signaling] Cannot send message, WebSocket not connected');
    }
  }, []);

  // Connect on mount
  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return {
    isConnected,
    isConnecting,
    error,
    send,
    connect,
    disconnect,
  };
}
