"use client";

import { useEffect, useRef, useState, useCallback } from "react";

export type SSEEventType = 
  | "metric" 
  | "alert" 
  | "audit" 
  | "tenant_status"
  | "heartbeat";

export interface SSEEvent {
  type: SSEEventType;
  payload: Record<string, unknown>;
  ts: number;
}

export interface MetricPayload {
  service: string;
  name: string;
  value: number;
}

export interface AlertPayload {
  severity: "P0" | "P1" | "P2";
  message: string;
  link: string;
}

export interface AuditPayload {
  actor: string;
  action: string;
  target: string;
}

export interface TenantStatusPayload {
  tenant_id: string;
  status: string;
}

export function useSSE<T = unknown>(
  url: string,
  options: {
    onMessage?: (event: SSEEvent) => void;
    onError?: (error: Error) => void;
    enabled?: boolean;
  } = {}
) {
  const [data, setData] = useState<T | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const { onMessage, onError, enabled = true } = options;

  const connect = useCallback(() => {
    if (typeof window === "undefined") return;

    try {
      const eventSource = new EventSource(url);
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        setConnected(true);
        setError(null);
      };

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as SSEEvent;
          setData(data.payload as T);
          onMessage?.(data);
        } catch {
          // Ignore parse errors for non-JSON messages (like heartbeat comments)
        }
      };

      eventSource.onerror = () => {
        setConnected(false);
        const err = new Error("SSE connection error");
        setError(err);
        onError?.(err);

        // Auto reconnect after 3 seconds
        eventSource.close();
        reconnectTimeoutRef.current = setTimeout(() => {
          if (enabled) {
            connect();
          }
        }, 3000);
      };
    } catch (err) {
      setError(err instanceof Error ? err : new Error("Failed to connect"));
    }
  }, [url, onMessage, onError, enabled]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setConnected(false);
  }, []);

  useEffect(() => {
    if (enabled) {
      connect();
    }
    return () => {
      disconnect();
    };
  }, [connect, disconnect, enabled]);

  return { data, connected, error, disconnect, reconnect: connect };
}

/**
 * Hook for polling data at regular intervals
 */
export function usePolling<T = unknown>(
  fetcher: () => Promise<T>,
  options: {
    interval?: number;
    enabled?: boolean;
    onSuccess?: (data: T) => void;
    onError?: (error: Error) => void;
  } = {}
) {
  const { interval = 30000, enabled = true, onSuccess, onError } = options;
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const isMountedRef = useRef(true);

  const fetch = useCallback(async () => {
    try {
      setIsLoading(true);
      const result = await fetcher();
      if (isMountedRef.current) {
        setData(result);
        setError(null);
        onSuccess?.(result);
      }
    } catch (err) {
      if (isMountedRef.current) {
        setError(err instanceof Error ? err : new Error("Fetch failed"));
        onError?.(err instanceof Error ? err : new Error("Fetch failed"));
      }
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }, [fetcher, onSuccess, onError]);

  useEffect(() => {
    isMountedRef.current = true;
    if (enabled) {
      // Initial fetch
      fetch();

      // Set up polling
      intervalRef.current = setInterval(fetch, interval);
    }

    return () => {
      isMountedRef.current = false;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [fetch, interval, enabled]);

  return { data, error, isLoading, refetch: fetch };
}

/**
 * Hook for auto-refreshing data with pause on blur
 */
export function useAutoRefresh<T = unknown>(
  fetcher: () => Promise<T>,
  options: {
    interval?: number;
    pauseOnBlur?: boolean;
  } = {}
) {
  const { interval = 30000, pauseOnBlur = true } = options;
  const [data, setData] = useState<T | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const isPausedRef = useRef(false);

  const fetch = useCallback(async () => {
    if (isPausedRef.current) return;
    
    setIsRefreshing(true);
    try {
      const result = await fetcher();
      setData(result);
      setLastUpdated(new Date());
    } finally {
      setIsRefreshing(false);
    }
  }, [fetcher]);

  useEffect(() => {
    // Initial fetch
    fetch();

    // Set up polling
    intervalRef.current = setInterval(fetch, interval);

    // Handle visibility change for pause on blur
    if (pauseOnBlur) {
      const handleVisibilityChange = () => {
        if (document.hidden) {
          isPausedRef.current = true;
        } else {
          isPausedRef.current = false;
          // Refresh immediately when tab becomes visible
          fetch();
        }
      };

      document.addEventListener("visibilitychange", handleVisibilityChange);

      return () => {
        document.removeEventListener("visibilitychange", handleVisibilityChange);
      };
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [fetch, interval, pauseOnBlur]);

  return { data, lastUpdated, isRefreshing, refresh: fetch };
}
