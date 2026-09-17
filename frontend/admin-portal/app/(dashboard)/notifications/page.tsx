"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bell, Check, Send, Users, Eye, MousePointerClick, TrendingUp } from "lucide-react";
import { mockNotifications } from "@/lib/mock-data";

const CATEGORY_META = {
  critical: { icon: "🚨", color: "text-red-700", bg: "bg-red-50 border-red-200" },
  warning: { icon: "⚠️", color: "text-amber-700", bg: "bg-amber-50 border-amber-200" },
  info: { icon: "ℹ️", color: "text-blue-700", bg: "bg-blue-50 border-blue-200" },
  marketing: { icon: "📣", color: "text-purple-700", bg: "bg-purple-50 border-purple-200" },
};

const CHANNEL_LABEL = {
  in_app: "In-App",
  email: "Email",
  telegram: "Telegram",
  push: "Push",
  sms: "SMS",
};

const STATUS_LABEL = {
  draft: { label: "Draft", variant: "secondary" as const },
  scheduled: { label: "Scheduled", variant: "secondary" as const },
  sent: { label: "Sent", variant: "default" as const },
  failed: { label: "Failed", variant: "destructive" as const },
};

export default function NotificationsPage() {
  const [filter, setFilter] = useState<"all" | "draft" | "scheduled" | "sent">("all");

  const filtered = mockNotifications.filter((n) => filter === "all" || n.status === filter);

  const sent = mockNotifications.filter((n) => n.status === "sent");
  const totalRecipients = sent.reduce((acc, n) => acc + n.recipientCount, 0);
  const totalOpened = sent.reduce((acc, n) => acc + n.opened, 0);
  const totalClicked = sent.reduce((acc, n) => acc + n.clicked, 0);
  const openRate = totalRecipients ? ((totalOpened / totalRecipients) * 100).toFixed(1) : "0";
  const clickRate = totalOpened ? ((totalClicked / totalOpened) * 100).toFixed(1) : "0";

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Bell className="w-7 h-7 text-primary" />
            Notifications
          </h1>
          <p className="text-muted-foreground mt-1">
            Broadcast, schedule và theo dõi hiệu quả thông báo
          </p>
        </div>
        <Button className="gap-2">
          <Send className="w-4 h-4" /> Compose Broadcast
        </Button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <SummaryTile label="Sent" value={sent.length} icon={Send} tone="emerald" />
        <SummaryTile label="Recipients" value={totalRecipients.toLocaleString()} icon={Users} tone="blue" />
        <SummaryTile label="Open rate" value={`${openRate}%`} icon={Eye} tone="purple" />
        <SummaryTile label="Click rate" value={`${clickRate}%`} icon={MousePointerClick} tone="amber" />
      </div>

      <Tabs defaultValue="inbox" className="w-full">
        <TabsList>
          <TabsTrigger value="inbox">Inbox</TabsTrigger>
          <TabsTrigger value="compose">Compose</TabsTrigger>
          <TabsTrigger value="history">History</TabsTrigger>
        </TabsList>

        <TabsContent value="inbox" className="space-y-4">
          <div className="flex gap-2 flex-wrap">
            {(["all", "draft", "scheduled", "sent"] as const).map((f) => (
              <Button
                key={f}
                variant={filter === f ? "default" : "outline"}
                size="sm"
                onClick={() => setFilter(f)}
              >
                {f === "all" ? "All" : f.charAt(0).toUpperCase() + f.slice(1)}
              </Button>
            ))}
          </div>

          <div className="space-y-3">
            {filtered.length === 0 ? (
              <Card>
                <CardContent className="py-12 text-center text-muted-foreground">
                  No notifications to display
                </CardContent>
              </Card>
            ) : (
              filtered.map((n) => {
                const cat = CATEGORY_META[n.category];
                const status = STATUS_LABEL[n.status];
                return (
                  <Card key={n.id} className={`${cat.bg} border`}>
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex items-start gap-3 flex-1">
                          <div className="text-2xl">{cat.icon}</div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 flex-wrap">
                              <CardTitle className="text-base">{n.title}</CardTitle>
                              <Badge variant="outline" className="text-xs capitalize">
                                {n.category}
                              </Badge>
                              <Badge variant={status.variant} className="text-xs">
                                {status.label}
                              </Badge>
                              <Badge variant="outline" className="text-xs">
                                {CHANNEL_LABEL[n.channel]}
                              </Badge>
                            </div>
                            <p className="text-sm text-slate-700 mt-1">{n.body}</p>
                            <div className="flex items-center gap-4 mt-2 text-xs text-slate-600">
                              <span>📅 {new Date(n.createdAt).toLocaleString()}</span>
                              {n.status === "sent" && (
                                <>
                                  <span>👥 {n.recipientCount.toLocaleString()} recipients</span>
                                  <span>👁️ {n.opened.toLocaleString()} opened</span>
                                  <span>🖱️ {n.clicked.toLocaleString()} clicked</span>
                                </>
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                    </CardHeader>
                  </Card>
                );
              })
            )}
          </div>
        </TabsContent>

        <TabsContent value="compose">
          <Card>
            <CardHeader>
              <CardTitle>Compose new broadcast</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-sm font-medium block mb-1">Title</label>
                <input
                  type="text"
                  placeholder="e.g. Scheduled maintenance on Oct 5"
                  className="w-full rounded-lg border-2 border-slate-200 bg-white px-4 py-2.5 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary outline-none"
                />
              </div>
              <div>
                <label className="text-sm font-medium block mb-1">Body</label>
                <textarea
                  rows={4}
                  placeholder="Write your message here..."
                  className="w-full rounded-lg border-2 border-slate-200 bg-white px-4 py-2.5 text-sm focus:ring-2 focus:ring-primary/20 focus:border-primary outline-none"
                />
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label className="text-sm font-medium block mb-1">Category</label>
                  <select className="w-full rounded-lg border-2 border-slate-200 bg-white px-4 py-2.5 text-sm">
                    <option>Info</option>
                    <option>Warning</option>
                    <option>Critical</option>
                    <option>Marketing</option>
                  </select>
                </div>
                <div>
                  <label className="text-sm font-medium block mb-1">Channel</label>
                  <select className="w-full rounded-lg border-2 border-slate-200 bg-white px-4 py-2.5 text-sm">
                    <option>In-App</option>
                    <option>Email</option>
                    <option>Telegram</option>
                    <option>Push</option>
                  </select>
                </div>
                <div>
                  <label className="text-sm font-medium block mb-1">Audience</label>
                  <select className="w-full rounded-lg border-2 border-slate-200 bg-white px-4 py-2.5 text-sm">
                    <option>All super admins</option>
                    <option>Tenant owners</option>
                    <option>Specific tenant</option>
                  </select>
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button variant="outline">Save as draft</Button>
                <Button>Schedule</Button>
                <Button>Send now</Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="history">
          <Card>
            <CardContent className="py-12 text-center text-muted-foreground">
              <TrendingUp className="w-10 h-10 mx-auto mb-2 opacity-50" />
              <p>Delivery history và analytics sẽ có trong Phase 2.</p>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function SummaryTile({
  label,
  value,
  icon: Icon,
  tone,
}: {
  label: string;
  value: string | number;
  icon: typeof Bell;
  tone: "slate" | "emerald" | "blue" | "purple" | "amber";
}) {
  const tones = {
    slate: "bg-slate-100 text-slate-700",
    emerald: "bg-emerald-100 text-emerald-700",
    blue: "bg-blue-100 text-blue-700",
    purple: "bg-purple-100 text-purple-700",
    amber: "bg-amber-100 text-amber-700",
  };
  return (
    <Card>
      <CardContent className="p-4 flex items-center gap-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${tones[tone]}`}>
          <Icon className="w-5 h-5" />
        </div>
        <div>
          <div className="text-2xl font-bold">{value}</div>
          <div className="text-xs text-muted-foreground">{label}</div>
        </div>
      </CardContent>
    </Card>
  );
}
