"use client";

import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { MetricCard } from "@/components/analytics/MetricCard";
import { Chart } from "@/components/analytics/Chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Building2,
  Users,
  TrendingUp,
  DollarSign,
  Target,
  Eye,
} from "lucide-react";
import { format } from "date-fns";

// Sample data for demo
const leadsData = [
  { name: "Mon", value: 45 },
  { name: "Tue", value: 52 },
  { name: "Wed", value: 38 },
  { name: "Thu", value: 65 },
  { name: "Fri", value: 72 },
  { name: "Sat", value: 58 },
  { name: "Sun", value: 42 },
];

const revenueData = [
  { name: "Jan", value: 45000000 },
  { name: "Feb", value: 52000000 },
  { name: "Mar", value: 48000000 },
  { name: "Apr", value: 61000000 },
  { name: "May", value: 55000000 },
  { name: "Jun", value: 72000000 },
];

export default function DashboardPage() {
  const { data: statsData, isLoading } = useQuery({
    queryKey: ["admin-stats"],
    queryFn: () => adminApi.getStats(),
  });

  const stats = statsData?.data || {
    totalTenants: 156,
    totalUsers: 2847,
    totalLeads: 125847,
    mrr: 156000000,
    leadsToday: 342,
    conversionRate: 8.5,
    pageviewsToday: 125847,
    apiCallsMin: 4521,
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-gray-500">
          Overview of your platform - {format(new Date(), "PPpp")}
        </p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Total Tenants"
          value={stats.totalTenants.toLocaleString()}
          change={{ value: 12.5, isPositive: true }}
          icon={Building2}
          iconColor="text-blue-500"
          loading={isLoading}
        />
        <MetricCard
          title="Total Users"
          value={stats.totalUsers.toLocaleString()}
          change={{ value: 8.2, isPositive: true }}
          icon={Users}
          iconColor="text-emerald-500"
          loading={isLoading}
        />
        <MetricCard
          title="Total Leads"
          value={stats.totalLeads.toLocaleString()}
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
          }).format(stats.mrr)}
          change={{ value: 18.7, isPositive: true }}
          icon={DollarSign}
          iconColor="text-purple-500"
          loading={isLoading}
        />
      </div>

      {/* Real-time metrics */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-gray-500">
              Leads Today
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{stats.leadsToday}</div>
            <p className="text-xs text-emerald-600 mt-1">+23 from yesterday</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-gray-500">
              Pageviews Today
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">
              {stats.pageviewsToday.toLocaleString()}
            </div>
            <p className="text-xs text-emerald-600 mt-1">+15% from yesterday</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-gray-500">
              Conversion Rate
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{stats.conversionRate}%</div>
            <p className="text-xs text-emerald-600 mt-1">+0.5% from last week</p>
          </CardContent>
        </Card>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Chart
          title="Leads (Last 7 Days)"
          data={leadsData}
          dataKey="value"
          color="#F97316"
          type="area"
        />
        <Chart
          title="Revenue Trend (VND)"
          data={revenueData}
          dataKey="value"
          color="#10B981"
          type="area"
        />
      </div>

      {/* Recent Activity */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Activity</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[
              { time: "2 min ago", action: "New tenant registered", detail: "TechCorp Vietnam" },
              { time: "15 min ago", action: "Lead submitted", detail: "Demo Request from Ho Chi Minh City" },
              { time: "32 min ago", action: "User upgraded plan", detail: "Fintech Solutions - Pro Plan" },
              { time: "1 hour ago", action: "API spike detected", detail: "chat-engine: 2x normal traffic" },
            ].map((activity, idx) => (
              <div key={idx} className="flex items-center gap-4 text-sm">
                <div className="w-2 h-2 rounded-full bg-primary" />
                <span className="text-gray-500 w-20">{activity.time}</span>
                <span className="font-medium">{activity.action}</span>
                <span className="text-gray-500">- {activity.detail}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
