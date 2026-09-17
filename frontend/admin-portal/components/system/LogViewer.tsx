"use client";

import { useState } from "react";
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
import { Search, RefreshCw, Terminal } from "lucide-react";
import { mockSystemLogs, type MockSystemLog } from "@/lib/mock-data";

const levelColors: Record<MockSystemLog["level"], string> = {
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

  const filtered = mockSystemLogs.filter((log) => {
    if (service && log.service !== service) return false;
    if (level && log.level !== level) return false;
    if (search && !log.message.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const pageSize = 50;
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);
  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
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
          <SelectTrigger className="w-44">
            <SelectValue placeholder="All Services" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">All Services</SelectItem>
            <SelectItem value="auth-service">Auth Service</SelectItem>
            <SelectItem value="tenant-service">Tenant Service</SelectItem>
            <SelectItem value="crm-service">CRM Service</SelectItem>
            <SelectItem value="lead-service">Lead Service</SelectItem>
            <SelectItem value="chat-engine">Chat Engine</SelectItem>
            <SelectItem value="webrtc-sfu">WebRTC SFU</SelectItem>
            <SelectItem value="recording-service">Recording</SelectItem>
            <SelectItem value="lead-scoring">Lead Scoring</SelectItem>
            <SelectItem value="rag-chatbot">RAG Chatbot</SelectItem>
            <SelectItem value="stt-service">STT Service</SelectItem>
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

        <Button variant="outline" size="icon" aria-label="Refresh logs">
          <RefreshCw className="w-4 h-4" />
        </Button>
      </div>

      <div className="border rounded-lg overflow-hidden bg-slate-950 text-slate-200 font-mono text-sm shadow-inner">
        <div className="bg-slate-900 border-b border-slate-800 px-4 py-2 flex items-center justify-between text-xs">
          <div className="flex items-center gap-2 text-slate-400">
            <Terminal className="w-3.5 h-3.5" />
            <span>{filtered.length} entries</span>
          </div>
          <span className="text-slate-500">Live tail · auto-refresh 5s</span>
        </div>

        <div className="divide-y divide-slate-800 max-h-[600px] overflow-y-auto">
          {paged.length === 0 ? (
            <div className="px-4 py-12 text-center text-slate-500">No logs match the current filters</div>
          ) : (
            paged.map((log) => (
              <div key={log.id} className="px-4 py-2.5 hover:bg-slate-900/50 transition">
                <div className="flex items-start gap-3">
                  <span className="text-xs text-slate-500 whitespace-nowrap tabular-nums">
                    {format(new Date(log.timestamp), "HH:mm:ss.SSS")}
                  </span>
                  <Badge
                    className={levelColors[log.level]}
                    variant="outline"
                  >
                    {log.level.toUpperCase()}
                  </Badge>
                  <span className="text-xs text-slate-400 whitespace-nowrap">{log.service}</span>
                  <span className="flex-1 text-slate-200 break-all">{log.message}</span>
                  {log.traceId && (
                    <span className="text-xs text-slate-500 font-mono whitespace-nowrap">
                      trace:{log.traceId.slice(0, 12)}
                    </span>
                  )}
                </div>
                {log.metadata && (
                  <pre className="mt-1.5 text-xs text-slate-500 bg-slate-900 p-2 rounded overflow-x-auto">
                    {JSON.stringify(log.metadata, null, 2)}
                  </pre>
                )}
              </div>
            ))
          )}
        </div>
      </div>

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Page {page} of {totalPages}
        </p>
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
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}

export default LogViewer;
