"use client";

import { Chart } from "@/components/analytics/Chart";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { MetricCard } from "@/components/analytics/MetricCard";
import { Activity, Zap, Clock, TrendingUp } from "lucide-react";

// Sample data
const cpuData = [
  { name: "00:00", value: 35 },
  { name: "04:00", value: 28 },
  { name: "08:00", value: 52 },
  { name: "12:00", value: 78 },
  { name: "16:00", value: 65 },
  { name: "20:00", value: 45 },
];

const memoryData = [
  { name: "00:00", value: 42 },
  { name: "04:00", value: 38 },
  { name: "08:00", value: 55 },
  { name: "12:00", value: 68 },
  { name: "16:00", value: 72 },
  { name: "20:00", value: 58 },
];

const requestsData = [
  { name: "00:00", value: 1200 },
  { name: "04:00", value: 800 },
  { name: "08:00", value: 3500 },
  { name: "12:00", value: 4200 },
  { name: "16:00", value: 3800 },
  { name: "20:00", value: 2500 },
];

export default function SystemMetricsPage() {
  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">System Metrics</h1>
        <p className="text-gray-500">Performance metrics and resource usage</p>
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
        <Chart title="API Requests/min" data={requestsData} dataKey="value" color="#F97316" type="area" />
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Top Services by Traffic</CardTitle>
          </CardHeader>
          <CardContent>
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
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
