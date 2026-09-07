"use client";

import { useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { format } from "date-fns";
import { Search, Filter, RefreshCw } from "lucide-react";

interface LogEntry {
  id: string;
  timestamp: string;
  level: "info" | "warn" | "error" | "debug";
  service: string;
  message: string;
  metadata?: Record<string, unknown>;
}

const levelColors = {
  info: "bg-blue-50 text-blue-700 border-blue-200",
  warn: "bg-amber-50 text-amber-700 border-amber-200",
  error: "bg-red-50 text-red-700 border-red-200",
  debug: "bg-gray-50 text-gray-700 border-gray-200",
};

export function LogViewer() {
  const [service, setService] = useState<string>("");
  const [level, setLevel] = useState<string>("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);

  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ["logs", service, level, page],
    queryFn: () =>
      adminApi.getServiceLogs({
        service: service || undefined,
        level: level || undefined,
        limit: 50,
      }),
  });

  const logs: LogEntry[] = data?.data || [
    {
      id: "1",
      timestamp: new Date().toISOString(),
      level: "info",
      service: "auth-service",
      message: "User login successful",
      metadata: { user_id: "123", ip: "192.168.1.1" },
    },
    {
      id: "2",
      timestamp: new Date().toISOString(),
      level: "warn",
      service: "chat-engine",
      message: "High latency detected",
      metadata: { latency: 500, threshold: 300 },
    },
    {
      id: "3",
      timestamp: new Date().toISOString(),
      level: "error",
      service: "webrtc-sfu",
      message: "Connection failed",
      metadata: { error: "timeout", room_id: "abc123" },
    },
  ];

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            placeholder="Search logs..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-10"
          />
        </div>

        <Select value={service} onValueChange={setService}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="All Services" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All Services</SelectItem>
            <SelectItem value="auth-service">Auth</SelectItem>
            <SelectItem value="tenant-service">Tenant</SelectItem>
            <SelectItem value="crm-service">CRM</SelectItem>
            <SelectItem value="lead-service">Lead</SelectItem>
            <SelectItem value="chat-engine">Chat</SelectItem>
            <SelectItem value="webrtc-sfu">WebRTC</SelectItem>
          </SelectContent>
        </Select>

        <Select value={level} onValueChange={setLevel}>
          <SelectTrigger className="w-32">
            <SelectValue placeholder="All Levels" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All Levels</SelectItem>
            <SelectItem value="info">Info</SelectItem>
            <SelectItem value="warn">Warning</SelectItem>
            <SelectItem value="error">Error</SelectItem>
            <SelectItem value="debug">Debug</SelectItem>
          </SelectContent>
        </Select>

        <Button
          variant="outline"
          size="icon"
          onClick={() => refetch()}
          disabled={isRefetching}
        >
          <RefreshCw className={`w-4 h-4 ${isRefetching ? "animate-spin" : ""}`} />
        </Button>
      </div>

      {/* Log List */}
      <div className="border rounded-lg overflow-hidden font-mono text-sm">
        <div className="bg-gray-50 border-b px-4 py-2 flex items-center justify-between text-xs text-gray-500">
          <span>{logs.length} entries</span>
          <span>Auto-refresh: 30s</span>
        </div>

        <div className="divide-y">
          {isLoading ? (
            Array.from({ length: 10 }).map((_, i) => (
              <div key={i} className="px-4 py-3">
                <Skeleton className="h-4 w-full" />
              </div>
            ))
          ) : logs.length === 0 ? (
            <div className="px-4 py-8 text-center text-gray-500">
              No logs found
            </div>
          ) : (
            logs.map((log) => (
              <div key={log.id} className="px-4 py-3 hover:bg-gray-50">
                <div className="flex items-start gap-3">
                  <span className="text-xs text-gray-400 whitespace-nowrap">
                    {format(new Date(log.timestamp), "HH:mm:ss.SSS")}
                  </span>
                  <Badge
                    className={levelColors[log.level]}
                    variant="outline"
                  >
                    {log.level.toUpperCase()}
                  </Badge>
                  <span className="text-xs text-gray-500">{log.service}</span>
                  <span className="flex-1 text-gray-700">{log.message}</span>
                </div>
                {log.metadata && (
                  <pre className="mt-2 text-xs text-gray-400 bg-gray-100 p-2 rounded overflow-x-auto">
                    {JSON.stringify(log.metadata, null, 2)}
                  </pre>
                )}
              </div>
            ))
          )}
        </div>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500">Page {page}</p>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}

export default LogViewer;
