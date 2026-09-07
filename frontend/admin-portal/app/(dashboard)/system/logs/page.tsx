"use client";

import { LogViewer } from "@/components/system/LogViewer";

export default function SystemLogsPage() {
  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">System Logs</h1>
        <p className="text-gray-500">View and search system logs from all services</p>
      </div>

      <LogViewer />
    </div>
  );
}
