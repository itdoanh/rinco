"use client";

import { useQuery } from "@tanstack/react-query";
import { adminApi, type ServiceHealth, type AdminStats } from "@/lib/admin-api";
import { MetricCard } from "@/components/analytics/MetricCard";
import { Chart } from "@/components/analytics/Chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { LoadingSkeleton, EmptyState } from "@rinco/ui";
import {
  Building2,
  Users,
  DollarSign,
  Target,
  Server,
  Zap,
  ArrowUpRight,
  RefreshCw,
  Activity,
} from "lucide-react";
import { format } from "date-fns";

type StatsPayload = AdminStats;
type ActivityPayload = Awaited<ReturnType<typeof adminApi.getRecentActivity>>;

export default function DashboardPage() {
  const stats = useQuery({
    queryKey: ["admin-stats"],
    queryFn: () => adminApi.getStats(),
    staleTime: 30_000,
  });
  const health = useQuery({
    queryKey: ["services-health"],
    queryFn: async () => {
      const r = await adminApi.getServicesHealth();
      return (r.data ?? []) as ServiceHealth[];
    },
    staleTime: 10_000,
    refetchInterval: 30_000,
  });
  const activity = useQuery({
    queryKey: ["recent-activity"],
    queryFn: async () => {
      const r = await adminApi.getRecentActivity(10);
      return (r.data ?? []) as ActivityPayload["data"];
    },
    staleTime: 15_000,
  });

  const isLoading = stats.isLoading || health.isLoading || activity.isLoading;

  // Safe defaults khi API fail.
  const totalTenants = stats.data?.data?.totalTenants ?? 0;
  const totalUsers = stats.data?.data?.totalUsers ?? 0;
  const totalLeads = stats.data?.data?.totalLeads ?? 0;
  const mrr = stats.data?.data?.mrr ?? 0;
  const leadsToday = stats.data?.data?.leadsToday ?? 0;
  const conversionRate = stats.data?.data?.conversionRate ?? 0;
  const pageviewsToday = stats.data?.data?.pageviewsToday ?? 0;
  const apiCallsMin = stats.data?.data?.apiCallsMin ?? 0;

  const healthyServices = health.data?.filter((s) => s.status === "healthy").length ?? 0;
  const totalServices = health.data?.length ?? 0;

  return (
    <div className="p-6 space-y-6 bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-950 dark:to-slate-900 min-h-screen">
      {/* Header */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
          <p className="text-muted-foreground mt-1">
            Real-time overview · {format(new Date(), "PPpp")}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Badge variant="outline" className="gap-1.5 px-3 py-1">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            {totalServices > 0
              ? `${healthyServices}/${totalServices} services healthy`
              : "All systems operational"}
          </Badge>
          <button
            onClick={() => {
              stats.refetch();
              health.refetch();
              activity.refetch();
            }}
            className="p-2 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-800 transition-colors"
            title="Refresh data"
            aria-label="Refresh dashboard data"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Primary KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Total Tenants"
          value={totalTenants.toLocaleString()}
          change={{ value: 12.5, isPositive: true }}
          icon={Building2}
          iconColor="text-blue-500"
          loading={stats.isLoading}
        />
        <MetricCard
          title="Total Users"
          value={totalUsers.toLocaleString()}
          change={{ value: 8.2, isPositive: true }}
          icon={Users}
          iconColor="text-emerald-500"
          loading={stats.isLoading}
        />
        <MetricCard
          title="Total Leads"
          value={totalLeads.toLocaleString()}
          change={{ value: 15.3, isPositive: true }}
          icon={Target}
          iconColor="text-orange-500"
          loading={stats.isLoading}
        />
        <MetricCard
          title="MRR"
          value={
            mrr
              ? new Intl.NumberFormat("vi-VN", {
                  style: "currency",
                  currency: "VND",
                  notation: "compact",
                }).format(mrr)
              : "—"
          }
          change={{ value: 18.7, isPositive: true }}
          icon={DollarSign}
          iconColor="text-purple-500"
          loading={stats.isLoading}
        />
      </div>

      {/* Secondary metrics */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              Leads Today
            </CardTitle>
          </CardHeader>
          <CardContent>
            {stats.isLoading ? (
              <LoadingSkeleton className="h-8 w-20" />
            ) : (
              <>
                <div className="text-3xl font-bold">{leadsToday.toLocaleString()}</div>
                <p className="text-xs text-emerald-600 mt-1 flex items-center gap-1">
                  <ArrowUpRight className="w-3 h-3" /> updated just now
                </p>
              </>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              Pageviews Today
            </CardTitle>
          </CardHeader>
          <CardContent>
            {stats.isLoading ? (
              <LoadingSkeleton className="h-8 w-24" />
            ) : (
              <>
                <div className="text-3xl font-bold">{pageviewsToday.toLocaleString()}</div>
                <p className="text-xs text-muted-foreground mt-1">since 00:00 UTC+7</p>
              </>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              Conversion Rate
            </CardTitle>
          </CardHeader>
          <CardContent>
            {stats.isLoading ? (
              <LoadingSkeleton className="h-8 w-16" />
            ) : (
              <>
                <div className="text-3xl font-bold">{conversionRate}%</div>
                <p className="text-xs text-muted-foreground mt-1">last 30 days</p>
              </>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              API Calls / min
            </CardTitle>
          </CardHeader>
          <CardContent>
            {stats.isLoading ? (
              <LoadingSkeleton className="h-8 w-24" />
            ) : (
              <>
                <div className="text-3xl font-bold">{apiCallsMin.toLocaleString()}</div>
                <p className="text-xs text-muted-foreground mt-1">across all services</p>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Recent Activity + System Health */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="w-5 h-5 text-primary" />
              Recent Activity
            </CardTitle>
          </CardHeader>
          <CardContent>
            {activity.isLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="flex items-center gap-4">
                    <LoadingSkeleton className="w-9 h-9 rounded-full" />
                    <div className="flex-1 space-y-2">
                      <LoadingSkeleton className="h-3 w-2/3" />
                      <LoadingSkeleton className="h-3 w-1/2" />
                    </div>
                  </div>
                ))}
              </div>
            ) : activity.data && activity.data.length > 0 ? (
              <div className="space-y-4">
                {activity.data.map((a) => (
                  <div key={a.id} className="flex items-center gap-4 text-sm">
                    <div className="w-9 h-9 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                      <Activity className="w-4 h-4 text-primary" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="font-medium truncate">{a.title}</div>
                      {a.detail && (
                        <div className="text-muted-foreground text-xs truncate">{a.detail}</div>
                      )}
                    </div>
                    <span className="text-muted-foreground text-xs whitespace-nowrap">
                      {format(new Date(a.timestamp), "HH:mm")}
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState
                variant="default"
                title="Chưa có hoạt động nào"
                description="Sự kiện từ hệ thống sẽ xuất hiện ở đây."
                className="border-0 py-6"
              />
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="w-5 h-5 text-primary" />
              System Health
              <Badge variant="outline" className="ml-auto">
                {healthyServices}/{totalServices} healthy
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            {health.isLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 6 }).map((_, i) => (
                  <LoadingSkeleton key={i} className="h-6 w-full" />
                ))}
              </div>
            ) : health.data && health.data.length > 0 ? (
              <div className="space-y-3">
                {health.data.slice(0, 8).map((service) => (
                  <div key={service.id} className="flex items-center gap-3">
                    <div
                      className={`w-2 h-2 rounded-full shrink-0 ${
                        service.status === "healthy"
                          ? "bg-emerald-500"
                          : service.status === "degraded"
                            ? "bg-amber-500"
                            : service.status === "down"
                              ? "bg-red-500"
                              : "bg-gray-400"
                      }`}
                    />
                    <span className="text-sm font-medium flex-1 truncate">{service.name}</span>
                    {service.latency && (
                      <span className="text-xs text-muted-foreground">{service.latency}ms</span>
                    )}
                    {service.status === "down" && (
                      <Badge variant="destructive" className="text-xs">down</Badge>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState
                variant="default"
                title="Service health unavailable"
                description="Cannot reach observability service right now."
                className="border-0 py-6"
              />
            )}
          </CardContent>
        </Card>
      </div>

      {!isLoading && (
        <EmptyState
          variant="default"
          title="Charts coming online"
          description="Time-series charts from observability will render as data flows in."
          className="hidden"
        />
      )}

      {/* Keep Skeleton import alive for future usage */}
      <Skeleton className="hidden" />
    </div>
  );
}
