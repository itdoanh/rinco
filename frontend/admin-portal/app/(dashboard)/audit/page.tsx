"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Search, Shield, AlertTriangle, Info, CheckCircle } from "lucide-react";
import { format } from "date-fns";

interface AuditEntry {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  resource: string;
  ip: string;
  level: "info" | "warning" | "error" | "success";
}

const sampleAudit: AuditEntry[] = [
  {
    id: "1",
    timestamp: new Date().toISOString(),
    actor: "admin@rinco.app",
    action: "tenant.create",
    resource: "tenant/apex-fintech",
    ip: "192.168.1.42",
    level: "success",
  },
  {
    id: "2",
    timestamp: new Date(Date.now() - 3600_000).toISOString(),
    actor: "ops@rinco.app",
    action: "config.update",
    resource: "config/feature-flags",
    ip: "10.0.0.4",
    level: "info",
  },
  {
    id: "3",
    timestamp: new Date(Date.now() - 7200_000).toISOString(),
    actor: "admin@rinco.app",
    action: "tenant.suspend",
    resource: "tenant/demo-corp",
    ip: "192.168.1.42",
    level: "warning",
  },
  {
    id: "4",
    timestamp: new Date(Date.now() - 10800_000).toISOString(),
    actor: "system",
    action: "auth.login.failed",
    resource: "user/unknown",
    ip: "203.0.113.42",
    level: "error",
  },
  {
    id: "5",
    timestamp: new Date(Date.now() - 14400_000).toISOString(),
    actor: "admin@rinco.app",
    action: "user.invite",
    resource: "user/eng@rinco.app",
    ip: "192.168.1.42",
    level: "info",
  },
]

const levelMeta = {
  info: { icon: Info, color: "text-blue-600", label: "INFO" },
  warning: { icon: AlertTriangle, color: "text-amber-600", label: "WARN" },
  error: { icon: AlertTriangle, color: "text-red-600", label: "ERROR" },
  success: { icon: CheckCircle, color: "text-emerald-600", label: "OK" },
}

export default function AuditPage() {
  const [search, setSearch] = useState("")
  const [level, setLevel] = useState<string>("all")

  const filtered = sampleAudit.filter((e) => {
    const matchesSearch =
      e.actor.toLowerCase().includes(search.toLowerCase()) ||
      e.action.toLowerCase().includes(search.toLowerCase()) ||
      e.resource.toLowerCase().includes(search.toLowerCase())
    const matchesLevel = level === "all" || e.level === level
    return matchesSearch && matchesLevel
  })

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Shield className="w-6 h-6" />
          Audit Log
        </h1>
        <p className="text-gray-500">
          Immutable record of admin and system actions
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder="Search by actor, action, resource..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={level} onValueChange={setLevel}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="All levels" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All levels</SelectItem>
                <SelectItem value="info">Info</SelectItem>
                <SelectItem value="warning">Warning</SelectItem>
                <SelectItem value="error">Error</SelectItem>
                <SelectItem value="success">Success</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground text-xs uppercase">
                  <th className="text-left py-2 px-3">Time</th>
                  <th className="text-left py-2 px-3">Actor</th>
                  <th className="text-left py-2 px-3">Action</th>
                  <th className="text-left py-2 px-3">Resource</th>
                  <th className="text-left py-2 px-3">IP</th>
                  <th className="text-left py-2 px-3">Level</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((entry) => {
                  const meta = levelMeta[entry.level]
                  const Icon = meta.icon
                  return (
                    <tr key={entry.id} className="border-b hover:bg-muted/40">
                      <td className="py-2 px-3 font-mono text-xs">
                        {format(new Date(entry.timestamp), "yyyy-MM-dd HH:mm:ss")}
                      </td>
                      <td className="py-2 px-3 font-medium">{entry.actor}</td>
                      <td className="py-2 px-3 font-mono text-xs">{entry.action}</td>
                      <td className="py-2 px-3 font-mono text-xs">{entry.resource}</td>
                      <td className="py-2 px-3 font-mono text-xs">{entry.ip}</td>
                      <td className="py-2 px-3">
                        <Badge variant="outline" className={meta.color}>
                          <Icon className="w-3 h-3 mr-1" />
                          {meta.label}
                        </Badge>
                      </td>
                    </tr>
                  )
                })}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={6} className="text-center py-8 text-muted-foreground">
                      No matching entries.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
