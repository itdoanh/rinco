"use client";

import { MetricCard } from "@/components/analytics/MetricCard";
import { Chart } from "@/components/analytics/Chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Users,
  Eye,
  MousePointerClick,
  Clock,
  Globe,
  Smartphone,
  Monitor,
  Tablet,
} from "lucide-react";
import { mockAnalytics } from "@/lib/mock-data";

const trafficData = [
  { name: "T2", value: 4520 },
  { name: "T3", value: 5210 },
  { name: "T4", value: 4815 },
  { name: "T5", value: 6230 },
  { name: "T6", value: 5910 },
  { name: "T7", value: 3820 },
  { name: "CN", value: 3245 },
];

const conversionData = mockAnalytics.conversionBySource;

const trafficSources = [
  { source: "Facebook Ads", visitors: 12453, leads: 1456, color: "bg-blue-500" },
  { source: "Google Ads", visitors: 9823, leads: 982, color: "bg-emerald-500" },
  { source: "Organic", visitors: 7621, leads: 412, color: "bg-amber-500" },
  { source: "Email", visitors: 3214, leads: 489, color: "bg-purple-500" },
  { source: "Direct", visitors: 5621, leads: 312, color: "bg-orange-500" },
];

const deviceData = [
  { name: "Mobile", value: 58, icon: Smartphone },
  { name: "Desktop", value: 32, icon: Monitor },
  { name: "Tablet", value: 10, icon: Tablet },
];

export default function AnalyticsPage() {
  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Analytics</h1>
        <p className="text-muted-foreground mt-1">
          Platform-wide traffic, conversion & engagement metrics
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard title="Total Visitors" value="48,231" change={{ value: 12.4, isPositive: true }} icon={Users} iconColor="text-blue-500" />
        <MetricCard title="Pageviews" value="162,432" change={{ value: 8.7, isPositive: true }} icon={Eye} iconColor="text-emerald-500" />
        <MetricCard title="Click Rate" value="24.8%" change={{ value: 2.1, isPositive: true }} icon={MousePointerClick} iconColor="text-orange-500" />
        <MetricCard title="Avg. Session" value="3m 42s" change={{ value: 5.3, isPositive: false }} icon={Clock} iconColor="text-purple-500" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Chart title="Daily Traffic (last 7 days)" data={trafficData} dataKey="value" color="#3B82F6" type="area" />
        <Chart title="Conversion Rate by Channel (%)" data={conversionData} dataKey="value" color="#10B981" type="bar" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Chart title="Traffic 24h" data={mockAnalytics.trafficByHour} dataKey="value" color="#06B6D4" type="line" />
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Smartphone className="w-4 h-4" />
              Devices
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {deviceData.map((d) => (
                <div key={d.name} className="flex items-center gap-3">
                  <d.icon className="w-4 h-4 text-slate-500" />
                  <span className="text-sm flex-1">{d.name}</span>
                  <div className="flex-1 max-w-32 bg-slate-100 rounded-full h-2 overflow-hidden">
                    <div className="bg-primary h-full rounded-full" style={{ width: `${d.value}%` }} />
                  </div>
                  <span className="text-sm font-semibold tabular-nums w-10 text-right">{d.value}%</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Globe className="w-4 h-4" />
              Top Countries
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {[
                { c: "🇻🇳 Vietnam", v: 18432 },
                { c: "🇺🇸 USA", v: 9214 },
                { c: "🇸🇬 Singapore", v: 4128 },
                { c: "🇯🇵 Japan", v: 3128 },
                { c: "🇰🇷 Korea", v: 2104 },
              ].map((row) => (
                <div key={row.c} className="flex justify-between items-center text-sm">
                  <span>{row.c}</span>
                  <span className="tabular-nums text-slate-600">{row.v.toLocaleString()}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Traffic Sources</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
                  <th className="py-3 px-3 font-medium">Source</th>
                  <th className="py-3 px-3 font-medium">Visitors</th>
                  <th className="py-3 px-3 font-medium">Leads</th>
                  <th className="py-3 px-3 font-medium">CR</th>
                  <th className="py-3 px-3 font-medium">Share</th>
                </tr>
              </thead>
              <tbody>
                {trafficSources.map((src) => {
                  const cr = ((src.leads / src.visitors) * 100).toFixed(1);
                  const total = trafficSources.reduce((a, s) => a + s.visitors, 0);
                  const share = ((src.visitors / total) * 100).toFixed(0);
                  return (
                    <tr key={src.source} className="border-b hover:bg-slate-50/60">
                      <td className="py-2.5 px-3 flex items-center gap-2">
                        <div className={`w-2 h-2 rounded-full ${src.color}`} />
                        {src.source}
                      </td>
                      <td className="py-2.5 px-3 tabular-nums">{src.visitors.toLocaleString()}</td>
                      <td className="py-2.5 px-3 tabular-nums">{src.leads.toLocaleString()}</td>
                      <td className="py-2.5 px-3 font-semibold text-emerald-600 tabular-nums">{cr}%</td>
                      <td className="py-2.5 px-3">
                        <div className="flex items-center gap-2">
                          <div className="w-32 bg-slate-100 rounded-full h-2 overflow-hidden">
                            <div className={`h-full rounded-full ${src.color}`} style={{ width: `${share}%` }} />
                          </div>
                          <span className="text-xs text-slate-500 tabular-nums">{share}%</span>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
