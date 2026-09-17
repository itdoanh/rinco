"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ServiceHealthGrid } from "@/components/system/ServiceHealthGrid";
import { LogViewer } from "@/components/system/LogViewer";
import { Chart } from "@/components/analytics/Chart";
import { MetricCard } from "@/components/analytics/MetricCard";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Activity,
  Server,
  Zap,
  Clock,
  TrendingUp,
  HardDrive,
  Wifi,
} from "lucide-react";

// Sample metrics data
const latencyData = [
  { name: "00:00", value: 35 },
  { name: "04:00", value: 28 },
  { name: "08:00", value: 52 },
  { name: "12:00", value: 48 },
  { name: "16:00", value: 45 },
  { name: "20:00", value: 42 },
];

const throughputData = [
  { name: "00:00", value: 1200 },
  { name: "04:00", value: 800 },
  { name: "08:00", value: 3500 },
  { name: "12:00", value: 4200 },
  { name: "16:00", value: 3800 },
  { name: "20:00", value: 2500 },
];

export default function SystemPage() {
  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <Activity className="w-7 h-7 text-primary" />
          System
        </h1>
        <p className="text-muted-foreground mt-1">
          Monitor and manage platform infrastructure
        </p>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard title="Services Healthy" value="11/12" icon={Server} iconColor="text-emerald-500" />
        <MetricCard title="Avg Latency" value="45ms" change={{ value: 8, isPositive: true }} icon={Clock} iconColor="text-blue-500" />
        <MetricCard title="API Calls/min" value="4,521" icon={Zap} iconColor="text-amber-500" />
        <MetricCard title="Active Connections" value="2,847" icon={Wifi} iconColor="text-purple-500" />
      </div>

      {/* System Tabs */}
      <Tabs defaultValue="health" className="w-full">
        <TabsList>
          <TabsTrigger value="health">Health</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="metrics">Metrics</TabsTrigger>
        </TabsList>

        <TabsContent value="health" className="space-y-6">
          <ServiceHealthGrid />
        </TabsContent>

        <TabsContent value="logs">
          <LogViewer />
        </TabsContent>

        <TabsContent value="metrics" className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Chart title="API Latency (ms)" data={latencyData} dataKey="value" color="#3B82F6" type="area" />
            <Chart title="Request Throughput" data={throughputData} dataKey="value" color="#10B981" type="area" />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
