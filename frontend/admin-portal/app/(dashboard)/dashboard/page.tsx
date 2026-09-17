"use client";

import { useQuery } from "@tanstack/react-query";
import { adminApi, type AdminStats } from "@/lib/api";
import { mockStats, mockAnalytics, mockTenants, mockServiceHealth } from "@/lib/mock-data";
import { MetricCard } from "@/components/analytics/MetricCard";
import { Chart } from "@/components/analytics/Chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { useAutoRefresh } from "@/hooks/useRealtime";
import {
  Building2,
  Users,
  TrendingUp,
  DollarSign,
  Target,
  Eye,
  Activity,
  Server,
  Zap,
  ArrowUpRight,
  RefreshCw,
} from "lucide-react";
import { format } from "date-fns";

type AdminStatsPayload = AdminStats & { data?: Record<string, number | string | undefined> };

export default function DashboardPage() {
  const { data: statsData, isLoading, refetch } = useQuery({
    queryKey: ["admin-stats"],
    queryFn: () => adminApi.getStats(),
    staleTime: 30000,
  });

  // Real-time refresh hook
  const { lastUpdated, isRefreshing } = useAutoRefresh(
    async () => {
      await refetch();
      return true;
    },
    { interval: 30000, pauseOnBlur: true }
  );

  const apiPayload = statsData as AdminStatsPayload | undefined;
  const totalTenants = apiPayload?.data?.totalTenants ?? mockStats.totalTenants;
  const totalUsers = apiPayload?.data?.totalUsers ?? mockStats.totalUsers;
  const totalLeads = apiPayload?.data?.totalLeads ?? mockStats.totalLeads;
  const mrr = apiPayload?.data?.mrr ?? mockStats.monthlyRevenue;
  const leadsToday = apiPayload?.data?.leadsToday ?? 342;
  const conversionRate = apiPayload?.data?.conversionRate ?? mockStats.conversionRate;
  const pageviewsToday = apiPayload?.data?.pageviewsToday ?? mockStats.totalPageviews;
  const apiCallsMin = apiPayload?.data?.apiCallsMin ?? mockStats.apiCallsPerMin;

  const healthyServices = mockServiceHealth.filter((s) => s.status === "healthy").length;
  const totalServices = mockServiceHealth.length;

  return (
    <div className="p-6 space-y-6 bg-gradient-to-br from-slate-50 to-slate-100 min-h-screen">
      {/* Header */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
          <p className="text-muted-foreground mt-1">
            Real-time overview · {format(new Date(), "PPpp")}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {lastUpdated && (
            <span className="text-xs text-muted-foreground">
              Updated {format(lastUpdated, "HH:mm:ss")}
            </span>
          )}
          <Badge variant="outline" className="gap-1.5 px-3 py-1">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            All systems operational
          </Badge>
          <button
            onClick={() => refetch()}
            className={`p-2 rounded-lg hover:bg-slate-200 transition-colors ${isRefreshing ? "animate-spin" : ""}`}
            title="Refresh data"
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
          loading={isLoading}
        />
        <MetricCard
          title="Total Users"
          value={totalUsers.toLocaleString()}
          change={{ value: 8.2, isPositive: true }}
          icon={Users}
          iconColor="text-emerald-500"
          loading={isLoading}
        />
        <MetricCard
          title="Total Leads"
          value={totalLeads.toLocaleString()}
          change={{ value: 15.3, isPositive: true }}
          icon={Target}
          iconColor="text-orange-500"
          loading={isLoading}
        />
        <MetricCard
          title="MRR"
          value={new Intl.NumberFormat("vi-VN", {
            style: "currency",
            currency: "VND",
            notation: "compact",
          }).format(mrr)}
          change={{ value: 18.7, isPositive: true }}
          icon={DollarSign}
          iconColor="text-purple-500"
          loading={isLoading}
        />
      </div>

      {/* Secondary metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              Leads Today
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{leadsToday}</div>
            <p className="text-xs text-emerald-600 mt-1 flex items-center gap-1">
              <ArrowUpRight className="w-3 h-3" /> +23 from yesterday
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              Pageviews Today
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">
              {Number(pageviewsToday).toLocaleString()}
            </div>
            <p className="text-xs text-emerald-600 mt-1 flex items-center gap-1">
              <ArrowUpRight className="w-3 h-3" /> +15% WoW
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              Conversion Rate
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{conversionRate}%</div>
            <p className="text-xs text-emerald-600 mt-1 flex items-center gap-1">
              <ArrowUpRight className="w-3 h-3" /> +0.5% last week
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium">
              API Calls / min
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">
              {Number(apiCallsMin).toLocaleString()}
            </div>
            <p className="text-xs text-amber-600 mt-1 flex items-center gap-1">
              <Zap className="w-3 h-3" /> High load
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Chart
          title="Leads (Last 7 Days)"
          data={mockAnalytics.leadsByDay}
          dataKey="value"
          color="#F97316"
          type="area"
        />
        <Chart
          title="Revenue Trend (VND)"
          data={mockAnalytics.revenueByMonth}
          dataKey="value"
          color="#10B981"
          type="area"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Chart
          title="Conversion by Source"
          data={mockAnalytics.conversionBySource}
          dataKey="value"
          color="#8B5CF6"
          type="bar"
        />
        <Chart
          title="Traffic (24h)"
          data={mockAnalytics.trafficByHour}
          dataKey="value"
          color="#06B6D4"
          type="line"
        />
        <Chart
          title="Top Tenants by Leads"
          data={mockAnalytics.leadsByTenant.slice(0, 5)}
          dataKey="value"
          color="#EF4444"
          type="bar"
        />
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
            <div className="space-y-4">
              {[
                { time: "2 min ago", action: "New tenant registered", detail: "TechCorp Vietnam", icon: Building2 },
                { time: "15 min ago", action: "Lead submitted", detail: "Demo Request from HCM City", icon: Target },
                { time: "32 min ago", action: "User upgraded plan", detail: "Fintech Solutions → Pro", icon: TrendingUp },
                { time: "1 hour ago", action: "API spike detected", detail: "chat-engine: 2x normal traffic", icon: Zap },
                { time: "2 hours ago", action: "Quorum approved", detail: "tenant.delete fintech-hub", icon: Server },
              ].map((activity, idx) => (
                <div key={idx} className="flex items-center gap-4 text-sm">
                  <div className="w-9 h-9 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                    <activity.icon className="w-4 h-4 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium truncate">{activity.action}</div>
                    <div className="text-muted-foreground text-xs truncate">{activity.detail}</div>
                  </div>
                  <span className="text-muted-foreground text-xs whitespace-nowrap">{activity.time}</span>
                </div>
              ))}
            </div>
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
            <div className="space-y-3">
              {mockServiceHealth.slice(0, 6).map((service) => (
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
          </CardContent>
        </Card>
      </div>

      {/* Top Tenants Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Building2 className="w-5 h-5 text-primary" />
            Top Tenants
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-muted-foreground border-b">
                  <th className="pb-2 font-medium">Tenant</th>
                  <th className="pb-2 font-medium">Plan</th>
                  <th className="pb-2 font-medium">Users</th>
                  <th className="pb-2 font-medium">Leads</th>
                  <th className="pb-2 font-medium">Revenue</th>
                  <th className="pb-2 font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {mockTenants.slice(0, 5).map((t) => (
                  <tr key={t.id} className="border-b last:border-0 hover:bg-muted/30 transition">
                    <td className="py-3">
                      <div className="flex items-center gap-2">
                        <div
                          className="w-7 h-7 rounded-md flex items-center justify-center text-white text-xs font-bold"
                          style={{ backgroundColor: t.color }}
                        >
                          {t.name[0]}
                        </div>
                        <div>
                          <div className="font-medium">{t.name}</div>
                          <div className="text-xs text-muted-foreground">{t.slug}</div>
                        </div>
                      </div>
                    </td>
                    <td className="py-3 capitalize">{t.plan}</td>
                    <td className="py-3">{t.users}</td>
                    <td className="py-3">{t.leads.toLocaleString()}</td>
                    <td className="py-3">
                      {new Intl.NumberFormat("vi-VN", {
                        style: "currency",
                        currency: "VND",
                        notation: "compact",
                      }).format(t.revenue)}
                    </td>
                    <td className="py-3">
                      <Badge
                        variant={
                          t.status === "active"
                            ? "default"
                            : t.status === "trial"
                              ? "secondary"
                              : "destructive"
                        }
                        className="capitalize"
                      >
                        {t.status}
                      </Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
