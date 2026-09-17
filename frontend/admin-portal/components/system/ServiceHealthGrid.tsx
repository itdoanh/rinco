"use client";

import { useState, useEffect } from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { adminApi, type ServiceHealth as ApiServiceHealth } from "@/lib/admin-api";
import { mockServiceHealth } from "@/lib/mock-data";
import {
  Activity,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Server,
  Cpu,
  MemoryStick,
  RefreshCw,
} from "lucide-react";
import { format } from "date-fns";

type Status = "healthy" | "degraded" | "down" | "unknown";

const statusConfig: Record<Status, { icon: typeof CheckCircle2; color: string; bg: string; label: string; badge: "default" | "secondary" | "destructive" | "outline" }> = {
  healthy: { icon: CheckCircle2, color: "text-emerald-500", bg: "bg-white border-slate-200 hover:border-emerald-300", label: "Healthy", badge: "default" },
  degraded: { icon: AlertTriangle, color: "text-amber-500", bg: "bg-white border-slate-200 hover:border-amber-300", label: "Degraded", badge: "secondary" },
  down: { icon: XCircle, color: "text-red-500", bg: "bg-white border-slate-200 hover:border-red-300", label: "Down", badge: "destructive" },
  unknown: { icon: Activity, color: "text-gray-500", bg: "bg-white border-slate-200 hover:border-gray-300", label: "Unknown", badge: "outline" },
};

interface ServiceItem {
  id: string;
  name: string;
  status: Status;
  latency?: number;
  errorRate?: number;
  uptime?: number;
  region: string;
  version: string;
  cpu: number;
  memory: number;
}

export function ServiceHealthGrid() {
  const [filter, setFilter] = useState<Status | "all">("all");
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  // Fetch real-time health data
  const { data: apiHealth, isLoading, refetch, isFetching } = useQuery({
    queryKey: ["services-health"],
    queryFn: () => adminApi.getServicesHealth(),
    refetchInterval: 10000, // Refresh every 10 seconds
  });

  // Convert API response to our format
  const services: ServiceItem[] = (() => {
    if (apiHealth?.data && apiHealth.data.length > 0) {
      return apiHealth.data.map((s: ApiServiceHealth) => ({
        id: s.id,
        name: s.name,
        status: s.status as Status,
        latency: s.latency,
        errorRate: s.errorRate,
        uptime: s.uptime,
        region: s.region || "vn-sg",
        version: s.version || "1.0.0",
        cpu: s.cpu || 0,
        memory: s.memory || 0,
      }));
    }
    // Fallback to mock data if API fails
    return mockServiceHealth.map((s) => ({
      id: s.id,
      name: s.name,
      status: s.status as Status,
      latency: s.latency,
      errorRate: s.errorRate,
      uptime: s.uptime,
      region: s.region,
      version: s.version,
      cpu: s.cpu,
      memory: s.memory,
    }));
  })();

  useEffect(() => {
    if (!isFetching) {
      setLastRefresh(new Date());
    }
  }, [isFetching]);

  const filteredServices = services.filter((s) => filter === "all" || s.status === filter);
  const counts = {
    healthy: services.filter((s) => s.status === "healthy").length,
    degraded: services.filter((s) => s.status === "degraded").length,
    down: services.filter((s) => s.status === "down").length,
    total: services.length,
  };

  return (
    <div className="space-y-4">
      {/* Filter bar */}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <FilterChip label="All" count={counts.total} active={filter === "all"} onClick={() => setFilter("all")} />
          <FilterChip label="Healthy" count={counts.healthy} active={filter === "healthy"} onClick={() => setFilter("healthy")} variant="emerald" />
          <FilterChip label="Degraded" count={counts.degraded} active={filter === "degraded"} onClick={() => setFilter("degraded")} variant="amber" />
          <FilterChip label="Down" count={counts.down} active={filter === "down"} onClick={() => setFilter("down")} variant="red" />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">
            Updated {format(lastRefresh, "HH:mm:ss")}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            disabled={isFetching}
            className="gap-1.5"
          >
            <RefreshCw className={`w-3 h-3 ${isFetching ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>
      </div>

      {/* Service grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {isLoading ? (
          // Loading skeleton
          Array.from({ length: 8 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardContent className="p-4 space-y-3">
                <div className="h-4 bg-slate-200 rounded w-3/4" />
                <div className="h-6 bg-slate-200 rounded" />
                <div className="h-8 bg-slate-200 rounded" />
              </CardContent>
            </Card>
          ))
        ) : (
          filteredServices.map((service) => {
            const config = statusConfig[service.status];
            const Icon = config.icon;

            return (
              <Card
                key={service.id}
                className={cn(
                  "transition-all hover:shadow-md",
                  config.bg,
                )}
              >
              <CardContent className="p-4">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2 min-w-0">
                    <Server className="w-4 h-4 text-slate-400 shrink-0" />
                    <span className="font-semibold text-sm truncate">{service.name}</span>
                  </div>
                  <Icon className={cn("w-5 h-5 shrink-0", config.color)} />
                </div>

                <div className="space-y-2.5">
                  <Badge variant={config.badge} className="text-xs">
                    {config.label}
                  </Badge>

                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <div className="flex items-center gap-1.5 text-slate-500">
                      <Activity className="w-3 h-3" />
                      <span>{service.latency ? `${service.latency}ms` : "—"}</span>
                    </div>
                    <div className="flex items-center gap-1.5 text-slate-500">
                      <Cpu className="w-3 h-3" />
                      <span>{service.cpu ? `${service.cpu}%` : "—"}</span>
                    </div>
                    <div className="flex items-center gap-1.5 text-slate-500">
                      <MemoryStick className="w-3 h-3" />
                      <span>{service.memory ? `${service.memory}%` : "—"}</span>
                    </div>
                    <div className="flex items-center gap-1.5 text-slate-500">
                      <span className="font-mono">v{service.version}</span>
                    </div>
                  </div>

                  {service.errorRate !== undefined && service.errorRate > 0 && (
                    <div className="pt-2 border-t">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-slate-500">Error rate</span>
                        <span
                          className={cn(
                            "font-semibold",
                            service.errorRate > 50
                              ? "text-red-600"
                              : service.errorRate > 5
                                ? "text-amber-600"
                                : "text-slate-700",
                          )}
                        >
                          {service.errorRate}%
                        </span>
                      </div>
                    </div>
                  )}

                  <div className="pt-1 text-[10px] text-slate-400 uppercase tracking-wider">
                    {service.region}
                  </div>
                </div>
              </CardContent>
            </Card>
            );
          })
        )}
      </div>
    </div>
  );
}

interface FilterChipProps {
  label: string;
  count: number;
  active: boolean;
  onClick: () => void;
  variant?: "default" | "emerald" | "amber" | "red";
}

function FilterChip({ label, count, active, onClick, variant = "default" }: FilterChipProps) {
  const variantClasses: Record<NonNullable<FilterChipProps["variant"]>, string> = {
    default: "border-slate-200 bg-white text-slate-700",
    emerald: "border-emerald-200 bg-emerald-50 text-emerald-700",
    amber: "border-amber-200 bg-amber-50 text-amber-700",
    red: "border-red-200 bg-red-50 text-red-700",
  };
  const activeClasses: Record<NonNullable<FilterChipProps["variant"]>, string> = {
    default: "border-slate-900 bg-slate-900 text-white",
    emerald: "border-emerald-600 bg-emerald-600 text-white",
    amber: "border-amber-600 bg-amber-600 text-white",
    red: "border-red-600 bg-red-600 text-white",
  };
  return (
    <button
      onClick={onClick}
      className={cn(
        "px-3 py-1.5 rounded-full text-xs font-medium border transition-all",
        active ? activeClasses[variant] : variantClasses[variant],
      )}
    >
      {label} <span className="ml-1 opacity-70">({count})</span>
    </button>
  );
}

export default ServiceHealthGrid;
