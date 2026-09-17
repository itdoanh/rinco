"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LoadingSkeleton, EmptyState, ErrorBoundary } from "@rinco/ui";
import { Search, Shield, AlertTriangle, Info, CheckCircle, Clock, User } from "lucide-react";
import { format, formatDistanceToNow } from "date-fns";
import { adminApi, type AuditLogEntry } from "@/lib/admin-api";

type Level = "info" | "warning" | "error" | "success";

const levelMeta: Record<Level, {
  icon: typeof Info;
  color: string;
  bg: string;
  label: string;
}> = {
  info: { icon: Info, color: "text-blue-700", bg: "bg-blue-50", label: "INFO" },
  warning: { icon: AlertTriangle, color: "text-amber-700", bg: "bg-amber-50", label: "WARN" },
  error: { icon: AlertTriangle, color: "text-red-700", bg: "bg-red-50", label: "ERROR" },
  success: { icon: CheckCircle, color: "text-emerald-700", bg: "bg-emerald-50", label: "OK" },
};

function classifyLevel(action: string): Level {
  const a = action.toLowerCase();
  if (a.includes("delete") || a.includes("failed") || a.includes("deny")) return "error";
  if (a.includes("suspend") || a.includes("reset")) return "warning";
  if (a.includes("create") || a.includes("sign") || a.includes("verify")) return "success";
  return "info";
}

const ROLE_COLORS: Record<string, string> = {
  OWNER: "bg-purple-100 text-purple-700",
  SRE_ADMIN: "bg-blue-100 text-blue-700",
  SECURITY_ADMIN: "bg-red-100 text-red-700",
  SUPPORT_ADMIN: "bg-emerald-100 text-emerald-700",
  FINANCE_ADMIN: "bg-amber-100 text-amber-700",
  READONLY_VIEWER: "bg-gray-100 text-gray-700",
};

function AuditPageInner() {
  const [search, setSearch] = useState("");
  const [level, setLevel] = useState<string>("all");
  const [actor, setActor] = useState<string>("all");
  const [page, setPage] = useState(1);
  const limit = 25;

  const { data, isLoading, error } = useQuery({
    queryKey: ["audit-logs", { level, actor }],
    queryFn: () => adminApi.getAuditLogs({ limit: 200 }),
    staleTime: 30_000,
  });

  const entries: AuditLogEntry[] = data?.data ?? [];

  const filtered = useMemo(
    () =>
      entries.filter((e) => {
        const matchesSearch =
          (e.actorEmail ?? "").toLowerCase().includes(search.toLowerCase()) ||
          (e.action ?? "").toLowerCase().includes(search.toLowerCase()) ||
          (e.tenantSlug ?? "").toLowerCase().includes(search.toLowerCase()) ||
          (e.targetId ?? "").toLowerCase().includes(search.toLowerCase());
        const matchesLevel =
          level === "all" ||
          classifyLevel(e.action) === level ||
          (level === "success" && e.status === "success") ||
          (level === "error" && e.status === "failure");
        const matchesActor = actor === "all" || e.actorEmail === actor;
        return matchesSearch && matchesLevel && matchesActor;
      }),
    [entries, search, level, actor],
  );

  const actors = useMemo(
    () => Array.from(new Set(entries.map((e) => e.actorEmail).filter(Boolean))).sort(),
    [entries],
  );

  const totalPages = Math.max(1, Math.ceil(filtered.length / limit));
  const paged = filtered.slice((page - 1) * limit, page * limit);

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-start justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Shield className="w-7 h-7 text-primary" />
            Audit Log
          </h1>
          <p className="text-muted-foreground mt-1">
            Immutable record of admin actions from CRM :8083 + observability :8096 — {entries.length} events
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="gap-1.5">
            <Clock className="w-3 h-3" />
            Last 30 days
          </Badge>
          <Button variant="outline" size="sm">Export CSV</Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <SummaryTile label="Total events" value={entries.length} icon={Shield} tone="slate" loading={isLoading} />
        <SummaryTile
          label="Success"
          value={entries.filter((e) => e.status === "success").length}
          icon={CheckCircle}
          tone="emerald"
          loading={isLoading}
        />
        <SummaryTile
          label="Failures"
          value={entries.filter((e) => e.status === "failure").length}
          icon={AlertTriangle}
          tone="red"
          loading={isLoading}
        />
        <SummaryTile
          label="Unique actors"
          value={actors.length}
          icon={User}
          tone="purple"
          loading={isLoading}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Events</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder="Search by actor, action, resource..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={level} onValueChange={setLevel}>
              <SelectTrigger className="w-36">
                <SelectValue placeholder="Level" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Levels</SelectItem>
                <SelectItem value="info">Info</SelectItem>
                <SelectItem value="warning">Warning</SelectItem>
                <SelectItem value="error">Error</SelectItem>
                <SelectItem value="success">Success</SelectItem>
              </SelectContent>
            </Select>
            <Select value={actor} onValueChange={setActor}>
              <SelectTrigger className="w-52">
                <SelectValue placeholder="Actor" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Actors</SelectItem>
                {actors.filter((a): a is string => !!a).map((a) => (
                  <SelectItem key={a} value={a}>
                    {a}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {isLoading ? (
            <LoadingSkeleton.Table rows={10} columns={7} />
          ) : error ? (
            <EmptyState
              variant="error"
              title="Không tải được audit logs"
              description={(error as Error).message}
            />
          ) : paged.length === 0 ? (
            <EmptyState
              variant="search"
              title="Không có event nào"
              description="Audit logs sẽ xuất hiện khi có admin actions."
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
                    <th className="py-3 px-3">Time</th>
                    <th className="py-3 px-3">Actor</th>
                    <th className="py-3 px-3">Action</th>
                    <th className="py-3 px-3">Target</th>
                    <th className="py-3 px-3">Tenant</th>
                    <th className="py-3 px-3">IP</th>
                    <th className="py-3 px-3">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {paged.map((entry) => {
                    const lvl = classifyLevel(entry.action);
                    const meta = levelMeta[lvl];
                    const Icon = meta.icon;
                    return (
                      <tr key={entry.id} className="border-b hover:bg-slate-50/60">
                        <td className="py-2.5 px-3">
                          <div className="text-xs font-mono">
                            {format(new Date(entry.timestamp), "yyyy-MM-dd HH:mm:ss")}
                          </div>
                          <div className="text-[10px] text-muted-foreground">
                            {formatDistanceToNow(new Date(entry.timestamp), { addSuffix: true })}
                          </div>
                        </td>
                        <td className="py-2.5 px-3">
                          <div className="font-medium text-xs">{entry.actorEmail ?? "—"}</div>
                          <Badge
                            variant="secondary"
                            className={`text-[10px] mt-0.5 ${ROLE_COLORS[entry.actorRole ?? ""] ?? ""}`}
                          >
                            {entry.actorRole ?? "system"}
                          </Badge>
                        </td>
                        <td className="py-2.5 px-3 font-mono text-xs">{entry.action}</td>
                        <td className="py-2.5 px-3">
                          <div className="text-xs font-mono">{entry.targetType ?? "—"}</div>
                          <div className="text-[10px] text-muted-foreground font-mono">
                            {entry.targetId ?? "—"}
                          </div>
                        </td>
                        <td className="py-2.5 px-3 font-mono text-xs">
                          {entry.tenantSlug ?? "—"}
                        </td>
                        <td className="py-2.5 px-3 font-mono text-xs">
                          {entry.ipAddress ?? "—"}
                        </td>
                        <td className="py-2.5 px-3">
                          <Badge variant="outline" className={`${meta.color} ${meta.bg}`}>
                            <Icon className="w-3 h-3 mr-1" />
                            {entry.status === "failure" ? "FAIL" : meta.label}
                          </Badge>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}

          {filtered.length > 0 && (
            <div className="flex items-center justify-between pt-2 text-sm">
              <div className="text-muted-foreground">
                Showing {(page - 1) * limit + 1}–{Math.min(page * limit, filtered.length)} of {filtered.length}
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Prev
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function SummaryTile({
  label,
  value,
  icon: Icon,
  tone,
  loading,
}: {
  label: string;
  value: number;
  icon: typeof Shield;
  tone: "slate" | "emerald" | "red" | "purple";
  loading?: boolean;
}) {
  const tones = {
    slate: "bg-slate-100 text-slate-700",
    emerald: "bg-emerald-100 text-emerald-700",
    red: "bg-red-100 text-red-700",
    purple: "bg-purple-100 text-purple-700",
  };
  return (
    <Card>
      <CardContent className="p-4 flex items-center gap-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${tones[tone]}`}>
          <Icon className="w-5 h-5" />
        </div>
        <div className="flex-1 min-w-0">
          {loading ? (
            <LoadingSkeleton className="h-7 w-12" />
          ) : (
            <div className="text-2xl font-bold tabular-nums">{value}</div>
          )}
          <div className="text-xs text-muted-foreground">{label}</div>
        </div>
      </CardContent>
    </Card>
  );
}

export default function AuditPage() {
  return (
    <ErrorBoundary>
      <AuditPageInner />
    </ErrorBoundary>
  );
}
