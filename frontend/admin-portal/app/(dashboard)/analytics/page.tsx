"use client";

import { useQuery } from "@tanstack/react-query";
import { MetricCard } from "@/components/analytics/MetricCard";
import { Chart } from "@/components/analytics/Chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  TrendingUp,
  Eye,
  MousePointerClick,
  Users,
  Globe,
  Clock,
} from "lucide-react";

const trafficData = [
  { name: "Mon", value: 4520 },
  { name: "Tue", value: 5210 },
  { name: "Wed", value: 4815 },
  { name: "Thu", value: 6230 },
  { name: "Fri", value: 5910 },
  { name: "Sat", value: 3820 },
  { name: "Sun", value: 3245 },
]

const conversionData = [
  { name: "Direct", value: 8.2 },
  { name: "FB Ads", value: 12.5 },
  { name: "Google", value: 9.8 },
  { name: "Email", value: 15.2 },
  { name: "Referral", value: 6.4 },
]

const trafficSources = [
  { source: "Facebook Ads", visitors: 12453, leads: 1456, color: "bg-blue-500" },
  { source: "Google Ads", visitors: 9823, leads: 982, color: "bg-emerald-500" },
  { source: "Organic", visitors: 7621, leads: 412, color: "bg-amber-500" },
  { source: "Email", visitors: 3214, leads: 489, color: "bg-purple-500" },
  { source: "Direct", visitors: 5621, leads: 312, color: "bg-orange-500" },
]

export default function AnalyticsPage() {
  const { isLoading } = useQuery({
    queryKey: ["analytics"],
    queryFn: async () => ({ data: { ok: true } }),
  })

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Analytics</h1>
        <p className="text-gray-500">
          Platform-wide traffic, conversion & engagement metrics
        </p>
      </div>

      {/* Top KPI */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Total Visitors"
          value="48,231"
          change={{ value: 12.4, isPositive: true }}
          icon={Users}
          iconColor="text-blue-500"
          loading={isLoading}
        />
        <MetricCard
          title="Pageviews"
          value="162,432"
          change={{ value: 8.7, isPositive: true }}
          icon={Eye}
          iconColor="text-emerald-500"
          loading={isLoading}
        />
        <MetricCard
          title="Click Rate"
          value="24.8%"
          change={{ value: 2.1, isPositive: true }}
          icon={MousePointerClick}
          iconColor="text-orange-500"
          loading={isLoading}
        />
        <MetricCard
          title="Avg. Session"
          value="3m 42s"
          change={{ value: 5.3, isPositive: false }}
          icon={Clock}
          iconColor="text-purple-500"
          loading={isLoading}
        />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Chart
          title="Daily Traffic (last 7 days)"
          data={trafficData}
          dataKey="value"
          color="#3B82F6"
          type="area"
        />
        <Chart
          title="Conversion Rate by Channel (%)"
          data={conversionData}
          dataKey="value"
          color="#10B981"
          type="bar"
        />
      </div>

      {/* Traffic Sources Table */}
      <Card>
        <CardHeader>
          <CardTitle>Traffic Sources</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="text-left py-2 px-3 font-medium">Source</th>
                    <th className="text-left py-2 px-3 font-medium">Visitors</th>
                    <th className="text-left py-2 px-3 font-medium">Leads</th>
                    <th className="text-left py-2 px-3 font-medium">CR</th>
                    <th className="text-left py-2 px-3 font-medium">Share</th>
                  </tr>
                </thead>
                <tbody>
                  {trafficSources.map((src) => {
                    const cr = ((src.leads / src.visitors) * 100).toFixed(1)
                    const total = trafficSources.reduce((a, s) => a + s.visitors, 0)
                    const share = ((src.visitors / total) * 100).toFixed(0)
                    return (
                      <tr key={src.source} className="border-b hover:bg-muted/50">
                        <td className="py-2 px-3 flex items-center gap-2">
                          <div className={`w-2 h-2 rounded-full ${src.color}`} />
                          {src.source}
                        </td>
                        <td className="py-2 px-3">{src.visitors.toLocaleString()}</td>
                        <td className="py-2 px-3">{src.leads.toLocaleString()}</td>
                        <td className="py-2 px-3 font-semibold text-emerald-600">
                          {cr}%
                        </td>
                        <td className="py-2 px-3">
                          <div className="flex items-center gap-2">
                            <div className="w-24 bg-muted rounded-full h-2">
                              <div
                                className={`h-2 rounded-full ${src.color}`}
                                style={{ width: `${share}%` }}
                              />
                            </div>
                            <span className="text-xs text-muted-foreground">{share}%</span>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
