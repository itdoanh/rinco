"use client";

import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
  Activity,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Server,
} from "lucide-react";

interface ServiceHealth {
  name: string;
  status: "healthy" | "degraded" | "down" | "unknown";
  latency?: number;
  errorRate?: number;
  uptime?: number;
}

export function ServiceHealthGrid() {
  const { data, isLoading } = useQuery({
    queryKey: ["services-health"],
    queryFn: () => adminApi.getServicesHealth(),
    refetchInterval: 30000, // Refresh every 30s
  });

  const services: ServiceHealth[] = data?.data || [
    { name: "Auth Service", status: "healthy", latency: 45, uptime: 99.99 },
    { name: "Tenant Service", status: "healthy", latency: 32, uptime: 99.95 },
    { name: "CRM Service", status: "healthy", latency: 28, uptime: 99.90 },
    { name: "Lead Service", status: "healthy", latency: 55, uptime: 99.85 },
    { name: "Landing Service", status: "healthy", latency: 42, uptime: 99.92 },
    { name: "Chat Engine", status: "degraded", latency: 180, errorRate: 2.5 },
    { name: "WebRTC SFU", status: "healthy", latency: 35, uptime: 99.98 },
    { name: "Recording Service", status: "healthy", latency: 48, uptime: 99.88 },
    { name: "Lead Scoring", status: "healthy", latency: 120, uptime: 99.50 },
    { name: "RAG Chatbot", status: "healthy", latency: 250, uptime: 99.70 },
    { name: "AI SRE", status: "healthy", latency: 180, uptime: 99.60 },
    { name: "STT Service", status: "down", errorRate: 100 },
  ];

  const statusConfig = {
    healthy: {
      icon: CheckCircle2,
      color: "text-emerald-500",
      bg: "bg-emerald-50 border-emerald-200",
      label: "Healthy",
    },
    degraded: {
      icon: AlertTriangle,
      color: "text-amber-500",
      bg: "bg-amber-50 border-amber-200",
      label: "Degraded",
    },
    down: {
      icon: XCircle,
      color: "text-red-500",
      bg: "bg-red-50 border-red-200",
      label: "Down",
    },
    unknown: {
      icon: Activity,
      color: "text-gray-500",
      bg: "bg-gray-50 border-gray-200",
      label: "Unknown",
    },
  };

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="p-4 border rounded-xl">
            <Skeleton className="h-20" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {services.map((service) => {
        const config = statusConfig[service.status];
        const Icon = config.icon;

        return (
          <div
            key={service.name}
            className={cn(
              "p-4 border rounded-xl transition-all hover:shadow-md",
              config.bg
            )}
          >
            <div className="flex items-start justify-between mb-3">
              <div className="flex items-center gap-2">
                <Server className="w-4 h-4 text-gray-400" />
                <span className="font-medium text-sm">{service.name}</span>
              </div>
              <Icon className={cn("w-5 h-5", config.color)} />
            </div>

            <div className="space-y-2">
              <Badge
                variant={
                  service.status === "healthy"
                    ? "success"
                    : service.status === "degraded"
                    ? "warning"
                    : service.status === "down"
                    ? "destructive"
                    : "secondary"
                }
                className="text-xs"
              >
                {config.label}
              </Badge>

              {service.status === "healthy" && (
                <div className="flex items-center justify-between text-xs text-gray-500">
                  <span>Latency</span>
                  <span className="font-medium">{service.latency}ms</span>
                </div>
              )}

              {service.status === "degraded" && (
                <div className="flex items-center justify-between text-xs text-gray-500">
                  <span>Error Rate</span>
                  <span className="font-medium text-amber-600">
                    {service.errorRate}%
                  </span>
                </div>
              )}

              {service.status === "down" && (
                <div className="flex items-center justify-between text-xs text-gray-500">
                  <span>Error Rate</span>
                  <span className="font-medium text-red-600">
                    {service.errorRate}%
                  </span>
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default ServiceHealthGrid;
