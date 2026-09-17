"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { LoadingSkeleton, EmptyState, ErrorBoundary } from "@rinco/ui";
import {
  Bell,
  Send,
  Users,
  Eye,
  MousePointerClick,
  TrendingUp,
} from "lucide-react";
import { adminApi, type AdminNotification } from "@/lib/admin-api";

const CATEGORY_META: Record<AdminNotification["category"], { icon: string; color: string; bg: string }> = {
  critical: { icon: "🚨", color: "text-red-700", bg: "bg-red-50 border-red-200" },
  warning: { icon: "⚠️", color: "text-amber-700", bg: "bg-amber-50 border-amber-200" },
  info: { icon: "ℹ️", color: "text-blue-700", bg: "bg-blue-50 border-blue-200" },
  marketing: { icon: "📣", color: "text-purple-700", bg: "bg-purple-50 border-purple-200" },
};

const CHANNEL_LABEL: Record<AdminNotification["channel"], string> = {
  in_app: "In-App",
  email: "Email",
  telegram: "Telegram",
  push: "Push",
  sms: "SMS",
};

const STATUS_LABEL: Record<AdminNotification["status"], {
  label: string;
  variant: "secondary" | "default" | "destructive";
}> = {
  draft: { label: "Draft", variant: "secondary" },
  scheduled: { label: "Scheduled", variant: "secondary" },
  sent: { label: "Sent", variant: "default" },
  failed: { label: "Failed", variant: "destructive" },
};

function NotificationsInner() {
  const [filter, setFilter] = useState<"all" | AdminNotification["status"]>("all");
  const { data, isLoading, error } = useQuery({
    queryKey: ["notifications", { filter }],
    queryFn: () =>
      adminApi.getNotifications({
        ...(filter !== "all" ? { status: filter } : {}),
      }),
    staleTime: 30_000,
  });

  const list: AdminNotification[] = data?.data ?? [];
  const filtered = list.filter((n) => filter === "all" || n.status === filter);
  const sent = list.filter((n) => n.status === "sent");
  const totalRecipients = sent.reduce((acc, n) => acc + (n.recipientCount ?? 0), 0);
  const totalOpened = sent.reduce((acc, n) => acc + (n.opened ?? 0), 0);
  const totalClicked = sent.reduce((acc, n) => acc + (n.clicked ?? 0), 0);
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
            Broadcast & schedule từ notification-service :8088
          </p>
        </div>
        <Button className="gap-2">
          <Send className="w-4 h-4" /> Compose Broadcast
        </Button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <SummaryTile label="Sent" value={sent.length} icon={Send} tone="emerald" loading={isLoading} />
        <SummaryTile
          label="Recipients"
          value={totalRecipients.toLocaleString()}
          icon={Users}
          tone="blue"
          loading={isLoading}
        />
        <SummaryTile label="Open rate" value={`${openRate}%`} icon={Eye} tone="purple" loading={isLoading} />
        <SummaryTile
          label="Click rate"
          value={`${clickRate}%`}
          icon={MousePointerClick}
          tone="amber"
          loading={isLoading}
        />
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

          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <LoadingSkeleton.Card key={i} lines={3} />
              ))}
            </div>
          ) : error ? (
            <EmptyState
              variant="error"
              title="Không tải được notifications"
              description={(error as Error).message}
            />
          ) : filtered.length === 0 ? (
            <EmptyState
              variant="default"
              title="Chưa có notifications"
              description="Broadcast sẽ xuất hiện sau khi gửi."
            />
          ) : (
            <div className="space-y-3">
              {filtered.map((n) => {
                const cat = CATEGORY_META[n.category] ?? CATEGORY_META.info;
                const status = STATUS_LABEL[n.status] ?? STATUS_LABEL.draft;
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
                                {CHANNEL_LABEL[n.channel] ?? n.channel}
                              </Badge>
                            </div>
                            <p className="text-sm text-slate-700 mt-1">{n.body}</p>
                            <div className="flex items-center gap-4 mt-2 text-xs text-slate-600">
                              <span>📅 {new Date(n.createdAt).toLocaleString()}</span>
                              {n.status === "sent" && (
                                <>
                                  <span>👥 {n.recipientCount.toLocaleString()} recipients</span>
                                  <span>👁️ {(n.opened ?? 0).toLocaleString()} opened</span>
                                  <span>🖱️ {(n.clicked ?? 0).toLocaleString()} clicked</span>
                                </>
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                    </CardHeader>
                  </Card>
                );
              })}
            </div>
          )}
        </TabsContent>

        <TabsContent value="compose">
          <Card>
            <CardHeader>
              <CardTitle>Compose new broadcast</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <EmptyState
                variant="default"
                title="Form compose coming soon"
                description="UI sẽ wire với POST /v1/admin/notifications trong loop tiếp theo."
                className="border-0"
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="history">
          <Card>
            <CardContent className="py-12 text-center text-muted-foreground">
              <TrendingUp className="w-10 h-10 mx-auto mb-2 opacity-50" />
              <p>Delivery history & analytics sẽ có trong Phase 2.</p>
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
  loading,
}: {
  label: string;
  value: string | number;
  icon: typeof Bell;
  tone: "slate" | "emerald" | "blue" | "purple" | "amber";
  loading?: boolean;
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
        <div className="flex-1 min-w-0">
          {loading ? (
            <LoadingSkeleton className="h-7 w-16" />
          ) : (
            <div className="text-2xl font-bold truncate">{value}</div>
          )}
          <div className="text-xs text-muted-foreground">{label}</div>
        </div>
      </CardContent>
    </Card>
  );
}

export default function NotificationsPage() {
  return (
    <ErrorBoundary>
      <NotificationsInner />
    </ErrorBoundary>
  );
}
