"use client";

import { useState, useEffect } from "react";
import { useAutoRefresh } from "./useRealtime";
import { adminApi, type ServiceHealth } from "@/lib/api";

/**
 * Hook for real-time service health monitoring
 */
export function useServiceHealth() {
  const fetcher = async (): Promise<ServiceHealth[]> => {
    try {
      const response = await adminApi.getServicesHealth();
      return response.data || [];
    } catch {
      // Return empty array on error - components will use mock data
      return [];
    }
  };

  const { data: services, lastUpdated, isRefreshing, refresh } = useAutoRefresh(fetcher, {
    interval: 10000, // Refresh every 10 seconds
    pauseOnBlur: false, // Keep monitoring even when tab is blurred
  });

  const healthyCount = services?.filter((s) => s.status === "healthy").length ?? 0;
  const degradedCount = services?.filter((s) => s.status === "degraded").length ?? 0;
  const downCount = services?.filter((s) => s.status === "down").length ?? 0;
  const totalCount = services?.length ?? 0;

  return {
    services: services ?? [],
    healthyCount,
    degradedCount,
    downCount,
    totalCount,
    lastUpdated,
    isRefreshing,
    refresh,
    isHealthy: downCount === 0 && degradedCount === 0,
    hasIssues: downCount > 0 || degradedCount > 0,
  };
}

/**
 * Hook for real-time dashboard stats
 */
export function useDashboardStats() {
  const fetcher = async () => {
    try {
      const response = await adminApi.getStats();
      return response.data;
    } catch {
      return null;
    }
  };

  const { data: stats, lastUpdated, isRefreshing, refresh } = useAutoRefresh(fetcher, {
    interval: 30000, // Refresh every 30 seconds
    pauseOnBlur: true, // Pause when tab is blurred
  });

  return {
    stats,
    lastUpdated,
    isRefreshing,
    refresh,
  };
}

/**
 * Hook for real-time metrics
 */
export function useMetrics(from?: string, to?: string) {
  const fetcher = async () => {
    try {
      const response = await adminApi.getMetrics({ from, to });
      return response.data || [];
    } catch {
      return [];
    }
  };

  const { data: metrics, lastUpdated, isRefreshing, refresh } = useAutoRefresh(fetcher, {
    interval: 15000, // Refresh every 15 seconds
    pauseOnBlur: false,
  });

  return {
    metrics,
    lastUpdated,
    isRefreshing,
    refresh,
  };
}

/**
 * Hook for real-time system logs
 */
export function useSystemLogs(params: {
  service?: string;
  level?: string;
  search?: string;
}) {
  const [logs, setLogs] = useState<Record<string, unknown>[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isLive, setIsLive] = useState(true);

  const fetchLogs = async () => {
    setIsLoading(true);
    try {
      const response = await adminApi.getServiceLogs({
        ...params,
        limit: 100,
      });
      setLogs(response.data || []);
    } catch {
      // Silently fail - logs are not critical
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (isLive) {
      fetchLogs();
      const interval = setInterval(fetchLogs, 5000); // Refresh every 5 seconds in live mode
      return () => clearInterval(interval);
    }
  }, [isLive, params.service, params.level, params.search]);

  return {
    logs,
    isLoading,
    isLive,
    setIsLive,
    refresh: fetchLogs,
  };
}

/**
 * Hook for notifications count
 */
export function useNotificationCount() {
  const [count, setCount] = useState(0);
  const [unreadCount, setUnreadCount] = useState(0);

  useEffect(() => {
    // Poll for notification updates every minute
    const fetchCount = async () => {
      try {
        // This would be an actual API call in production
        // For now, we'll simulate it
        // const response = await adminApi.getNotificationCount();
        // setCount(response.total || 0);
        // setUnreadCount(response.unread || 0);
      } catch {
        // Silently fail
      }
    };

    fetchCount();
    const interval = setInterval(fetchCount, 60000);
    return () => clearInterval(interval);
  }, []);

  return { count, unreadCount };
}
