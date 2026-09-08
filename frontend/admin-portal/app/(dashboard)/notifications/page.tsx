"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Bell, Check, X, Info, AlertTriangle, CheckCircle2, AlertCircle } from "lucide-react";

interface Notification {
  id: string
  type: "info" | "warning" | "error" | "success"
  category: "system" | "billing" | "security" | "tenant"
  title: string
  message: string
  timestamp: string
  read: boolean
}

const sampleNotifications: Notification[] = [
  {
    id: "1",
    type: "error",
    category: "system",
    title: "Service Down",
    message: "chat-engine-3 pod is failing health checks. K3s will restart automatically.",
    timestamp: new Date(Date.now() - 120_000).toISOString(),
    read: false,
  },
  {
    id: "2",
    type: "warning",
    category: "billing",
    title: "Payment Failed",
    message: "Tenant 'demo-corp' (PRO plan) payment failed. Retry in 3 days.",
    timestamp: new Date(Date.now() - 1800_000).toISOString(),
    read: false,
  },
  {
    id: "3",
    type: "success",
    category: "tenant",
    title: "New Tenant Provisioned",
    message: "Tenant 'acme-fintech' has been created and assigned to cluster vn-sg-1.",
    timestamp: new Date(Date.now() - 3600_000).toISOString(),
    read: true,
  },
  {
    id: "4",
    type: "info",
    category: "system",
    title: "Scheduled Maintenance",
    message: "Database maintenance scheduled for Sept 15 at 02:00 UTC. Estimated downtime: 15 min.",
    timestamp: new Date(Date.now() - 7200_000).toISOString(),
    read: true,
  },
  {
    id: "5",
    type: "error",
    category: "security",
    title: "Brute Force Detection",
    message: "10+ failed login attempts for super_admin 'owner@rinco.app' from IP 203.0.113.42.",
    timestamp: new Date(Date.now() - 14400_000).toISOString(),
    read: false,
  },
]

const typeMeta = {
  info: { icon: Info, color: "text-blue-600", bg: "bg-blue-50" },
  warning: { icon: AlertTriangle, color: "text-amber-600", bg: "bg-amber-50" },
  error: { icon: AlertCircle, color: "text-red-600", bg: "bg-red-50" },
  success: { icon: CheckCircle2, color: "text-emerald-600", bg: "bg-emerald-50" },
}

export default function NotificationsPage() {
  const [notifications, setNotifications] = useState(sampleNotifications)
  const [filter, setFilter] = useState<"all" | "unread">("all")

  const filtered = filter === "unread"
    ? notifications.filter(n => !n.read)
    : notifications

  const unreadCount = notifications.filter(n => !n.read).length

  function markRead(id: string) {
    setNotifications(prev =>
      prev.map(n => n.id === id ? { ...n, read: true } : n)
    )
  }

  function markAllRead() {
    setNotifications(prev => prev.map(n => ({ ...n, read: true })))
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Bell className="w-6 h-6" />
            Notifications
          </h1>
          <p className="text-gray-500">
            {unreadCount > 0
              ? `You have ${unreadCount} unread notification${unreadCount > 1 ? "s" : ""}`
              : "All caught up!"
            }
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant={filter === "all" ? "default" : "outline"}
            size="sm"
            onClick={() => setFilter("all")}
          >
            All ({notifications.length})
          </Button>
          <Button
            variant={filter === "unread" ? "default" : "outline"}
            size="sm"
            onClick={() => setFilter("unread")}
          >
            Unread ({unreadCount})
          </Button>
          {unreadCount > 0 && (
            <Button variant="outline" size="sm" onClick={markAllRead}>
              <Check className="w-4 h-4 mr-1" />
              Mark all read
            </Button>
          )}
        </div>
      </div>

      <div className="space-y-3">
        {filtered.length === 0 ? (
          <Card>
            <CardContent className="py-12 text-center text-muted-foreground">
              No notifications to display
            </CardContent>
          </Card>
        ) : (
          filtered.map(n => {
            const meta = typeMeta[n.type]
            const Icon = meta.icon
            return (
              <Card key={n.id} className={n.read ? "opacity-60" : "border-l-4 border-l-primary"}>
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-start gap-3 flex-1">
                      <div className={`w-10 h-10 rounded-lg ${meta.bg} flex items-center justify-center flex-shrink-0`}>
                        <Icon className={`w-5 h-5 ${meta.color}`} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <CardTitle className="text-base">{n.title}</CardTitle>
                          <Badge variant="outline" className="text-xs">{n.category}</Badge>
                          {!n.read && (
                            <span className="w-2 h-2 bg-primary rounded-full inline-block" />
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground mt-1">
                          {n.message}
                        </p>
                        <p className="text-xs text-muted-foreground mt-2">
                          {new Date(n.timestamp).toLocaleString()}
                        </p>
                      </div>
                    </div>
                    {!n.read && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => markRead(n.id)}
                      >
                        <Check className="w-4 h-4" />
                      </Button>
                    )}
                  </div>
                </CardHeader>
              </Card>
            )
          })
        )}
      </div>
    </div>
  )
}
