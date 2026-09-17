"use client";

import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { LoadingSkeleton, EmptyState, ErrorBoundary } from "@rinco/ui";
import { Server, Zap, Clock, Cpu, MemoryStick, Activity } from "lucide-react";
import { adminApi, type ServiceHealth } from "@/lib/admin-api";

const statusMeta: Record<ServiceHealth["status"], {
  color: string;
  bg: string;
  label: string;
}> = {
  healthy: { color: "text-emerald-700", bg: "bg-emerald-50 border-emerald-200", label: "Healthy" },
  degraded: { color: "text-amber-700", bg: "bg-amber-50 border-amber-200", label: "Degraded" },
  down: { color: "text-red-700", bg: "bg-red-50 border-red-200", label: "Down" },
  unknown: { color: "text-gray-600", bg: "bg-gray-50 border-gray-200", label: "Unknown" },
};

function SystemHealthInner() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["services-health", "all"],
    queryFn: async () => {
      const r = await adminApi.getServicesHealth();
      return r.data ?? [];
    },
    staleTime: 10_000,
    refetchInterval: 15_000,
  });

  const services: ServiceHealth[] = data ?? [];
  const counts = {
    total: services.length,
    healthy: services.filter((s) => s.status === "healthy").length,
    degraded: services.filter((s) => s.status === "degraded").length,
    down: services.filter((s) => s.status === "down").length,
  };

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">System Health</h1>
        <p className="text-muted-foreground">
          Real-time status of all platform services (observability :8096)
        </p>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatTile label="Total Services" value={counts.total} icon={Server} tone="slate" loading={isLoading} />
        <StatTile label="Healthy" value={counts.healthy} icon={Zap} tone="emerald" loading={isLoading} />
        <StatTile label="Degraded" value={counts.degraded} icon={Activity} tone="amber" loading={isLoading} />
        <StatTile label="Down" value={counts.down} icon={Clock} tone="red" loading={isLoading} />
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 9 }).map((_, i) => (
            <LoadingSkeleton.Card key={i} lines={4} />
          ))}
        </div>
      ) : error ? (
        <EmptyState
          variant="error"
          title="Không tải được health"
          description={(error as Error).message}
        />
      ) : services.length === 0 ? (
        <EmptyState
          variant="default"
          title="No services registered"
          description="observability-service chưa có service nào trong registry."
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {services.map((s) => {
            const meta = statusMeta[s.status] ?? statusMeta.unknown;
            return (
              <Card key={s.id} className={`${meta.bg} border`}>
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-base font-mono">{s.name ?? s.id}</CardTitle>
                    <Badge variant="outline" className={meta.color}>
                      {meta.label}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  {s.version && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Version</span>
                      <span className="font-mono">{s.version}</span>
                    </div>
                  )}
                  {s.region && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Region</span>
                      <span className="font-mono">{s.region}</span>
                    </div>
                  )}
                  {s.latency !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground flex items-center gap-1">
                        <Clock className="w-3 h-3" /> Latency
                      </span>
                      <span className="font-mono tabular-nums">{s.latency}ms</span>
                    </div>
                  )}
                  {s.uptime !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Uptime</span>
                      <span className="font-mono tabular-nums">{s.uptime}%</span>
                    </div>
                  )}
                  {s.cpu !== undefined && (
                    <div>
                      <div className="flex justify-between text-xs">
                        <span className="text-muted-foreground flex items-center gap-1">
                          <Cpu className="w-3 h-3" /> CPU
                        </span>
                        <span className="font-mono tabular-nums">{s.cpu}%</span>
                      </div>
                      <div className="h-1.5 bg-slate-200 rounded-full mt-1 overflow-hidden">
                        <div
                          className={`h-full rounded-full ${s.cpu > 80 ? "bg-red-500" : s.cpu > 60 ? "bg-amber-500" : "bg-emerald-500"}`}
                          style={{ width: `${s.cpu}%` }}
                        />
                      </div>
                    </div>
                  )}
                  {s.memory !== undefined && (
                    <div>
                      <div className="flex justify-between text-xs">
                        <span className="text-muted-foreground flex items-center gap-1">
                          <MemoryStick className="w-3 h-3" /> Memory
                        </span>
                        <span className="font-mono tabular-nums">{s.memory}%</span>
                      </div>
                      <div className="h-1.5 bg-slate-200 rounded-full mt-1 overflow-hidden">
                        <div
                          className={`h-full rounded-full ${s.memory > 80 ? "bg-red-500" : s.memory > 60 ? "bg-amber-500" : "bg-emerald-500"}`}
                          style={{ width: `${s.memory}%` }}
                        />
                      </div>
                    </div>
                  )}
                  {s.errorRate !== undefined && (
                    <div className="flex justify-between text-xs">
                      <span className="text-muted-foreground">Error rate</span>
                      <span className="font-mono tabular-nums">{s.errorRate}%</span>
                    </div>
                  )}
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}

function StatTile({
  label,
  value,
  icon: Icon,
  tone,
  loading,
}: {
  label: string;
  value: number;
  icon: typeof Server;
  tone: "slate" | "emerald" | "amber" | "red";
  loading?: boolean;
}) {
  const tones = {
    slate: "bg-slate-100 text-slate-700",
    emerald: "bg-emerald-100 text-emerald-700",
    amber: "bg-amber-100 text-amber-700",
    red: "bg-red-100 text-red-700",
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

export default function SystemHealthPage() {
  return (
    <ErrorBoundary>
      <SystemHealthInner />
    </ErrorBoundary>
  );
}
