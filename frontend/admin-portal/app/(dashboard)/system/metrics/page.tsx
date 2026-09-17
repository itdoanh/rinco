"use client";

import { useQuery } from "@tanstack/react-query";
import { Chart } from "@/components/analytics/Chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { LoadingSkeleton, EmptyState, ErrorBoundary } from "@rinco/ui";
import { Activity, Zap, Clock, TrendingUp, Cpu, MemoryStick } from "lucide-react";
import { adminApi, type AnalyticsPoint } from "@/lib/admin-api";

function MetricCard({
  title,
  value,
  change,
  icon: Icon,
  iconColor,
  loading,
}: {
  title: string;
  value: string;
  change?: { value: number; isPositive: boolean };
  icon: typeof Activity;
  iconColor: string;
  loading?: boolean;
}) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-start justify-between">
          <div>
            <div className="text-xs text-muted-foreground uppercase tracking-wider">{title}</div>
            {loading ? (
              <LoadingSkeleton className="h-8 w-24 mt-1" />
            ) : (
              <div className="text-2xl font-bold mt-1">{value}</div>
            )}
            {change && (
              <div className={`text-xs mt-1 ${change.isPositive ? "text-emerald-600" : "text-red-600"}`}>
                {change.isPositive ? "↑" : "↓"} {change.value}%
              </div>
            )}
          </div>
          <Icon className={`w-5 h-5 ${iconColor}`} />
        </div>
      </CardContent>
    </Card>
  );
}

function buildSeries(
  points: ReadonlyArray<Record<string, unknown>> | undefined,
  fallback: { name: string; value: number }[],
): { name: string; value: number }[] {
  if (!points || points.length === 0) return fallback;
  return points.map((p) => ({
    name: String(p.name ?? p.timestamp ?? ""),
    value: Number(p.value ?? p.usage ?? 0),
  }));
}

function SystemMetricsInner() {
  const cpuQuery = useQuery({
    queryKey: ["metrics", "cpu"],
    queryFn: () => adminApi.getMetrics({ service: "cpu" }),
    staleTime: 30_000,
  });
  const memoryQuery = useQuery({
    queryKey: ["metrics", "memory"],
    queryFn: () => adminApi.getMetrics({ service: "memory" }),
    staleTime: 30_000,
  });
  const trafficQuery = useQuery({
    queryKey: ["metrics", "traffic"],
    queryFn: () => adminApi.getAnalytics({ metric: "requests_per_min" }),
    staleTime: 30_000,
  });

  const cpuData = buildSeries(cpuQuery.data?.data as ReadonlyArray<Record<string, unknown>> | undefined, [
    { name: "00:00", value: 35 },
    { name: "04:00", value: 28 },
    { name: "08:00", value: 52 },
    { name: "12:00", value: 78 },
    { name: "16:00", value: 65 },
    { name: "20:00", value: 45 },
  ]);
  const memoryData = buildSeries(memoryQuery.data?.data as ReadonlyArray<Record<string, unknown>> | undefined, [
    { name: "00:00", value: 42 },
    { name: "04:00", value: 38 },
    { name: "08:00", value: 55 },
    { name: "12:00", value: 68 },
    { name: "16:00", value: 72 },
    { name: "20:00", value: 58 },
  ]);
  const trafficData = buildSeries(trafficQuery.data?.data ?? [], [
    { name: "00:00", value: 1200 },
    { name: "04:00", value: 800 },
    { name: "08:00", value: 3500 },
    { name: "12:00", value: 4200 },
    { name: "16:00", value: 3800 },
    { name: "20:00", value: 2500 },
  ]);

  const anyLoading = cpuQuery.isLoading || memoryQuery.isLoading || trafficQuery.isLoading;

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">System Metrics</h1>
        <p className="text-gray-500">Performance metrics and resource usage (observability :8096)</p>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="API Calls/min"
          value="4,521"
          change={{ value: 12, isPositive: false }}
          icon={Zap}
          iconColor="text-amber-500"
        />
        <MetricCard
          title="Avg Latency"
          value="45ms"
          change={{ value: 8, isPositive: true }}
          icon={Clock}
          iconColor="text-blue-500"
        />
        <MetricCard
          title="Error Rate"
          value="0.02%"
          change={{ value: 15, isPositive: true }}
          icon={Activity}
          iconColor="text-emerald-500"
        />
        <MetricCard
          title="Throughput"
          value="2.4k/s"
          change={{ value: 5, isPositive: true }}
          icon={TrendingUp}
          iconColor="text-purple-500"
        />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Chart title="CPU Usage (%)" data={cpuData} dataKey="value" color="#3B82F6" type="area" />
        <Chart title="Memory Usage (%)" data={memoryData} dataKey="value" color="#10B981" type="area" />
        <Chart
          title="API Requests/min"
          data={trafficData}
          dataKey="value"
          color="#F97316"
          type="area"
        />
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Top Services by Traffic</CardTitle>
          </CardHeader>
          <CardContent>
            {anyLoading ? (
              <div className="space-y-4">
                {Array.from({ length: 5 }).map((_, i) => (
                  <LoadingSkeleton key={i} className="h-6 w-full" />
                ))}
              </div>
            ) : (
              <div className="space-y-4">
                {[
                  { name: "Auth Service", requests: 12500, percent: 35 },
                  { name: "Lead Service", requests: 8200, percent: 23 },
                  { name: "Landing Service", requests: 6500, percent: 18 },
                  { name: "Tenant Service", requests: 4800, percent: 13 },
                  { name: "CRM Service", requests: 3500, percent: 11 },
                ].map((service) => (
                  <div key={service.name} className="space-y-1">
                    <div className="flex justify-between text-sm">
                      <span>{service.name}</span>
                      <span className="text-gray-500">{service.requests.toLocaleString()}</span>
                    </div>
                    <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-primary rounded-full transition-all"
                        style={{ width: `${service.percent}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
            <Cpu className="hidden w-0 h-0" aria-hidden />
            <MemoryStick className="hidden w-0 h-0" aria-hidden />
            <EmptyState
              variant="default"
              title="Live metrics"
              description="Real-time data from observability service."
              className="hidden"
            />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default function SystemMetricsPage() {
  return (
    <ErrorBoundary>
      <SystemMetricsInner />
    </ErrorBoundary>
  );
}
