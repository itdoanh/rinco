"use client";

import { ServiceHealthGrid } from "@/components/system/ServiceHealthGrid";

export default function SystemHealthPage() {
  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">System Health</h1>
        <p className="text-gray-500">Real-time status of all platform services</p>
      </div>

      <ServiceHealthGrid />
    </div>
  );
}
