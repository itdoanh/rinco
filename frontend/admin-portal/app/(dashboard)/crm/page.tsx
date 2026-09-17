"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LoadingSkeleton, EmptyState, ErrorBoundary } from "@rinco/ui";
import {
  Users,
  Building2,
  DollarSign,
  Plus,
  Search,
  MoreHorizontal,
  Phone,
  Mail,
  TrendingUp,
  Activity,
  Target,
} from "lucide-react";
import { format } from "date-fns";
import { adminApi, type CrmContact, type CrmDeal } from "@/lib/admin-api";

const STATUS_META: Record<string, { color: string; label: string }> = {
  new: { color: "bg-blue-100 text-blue-700", label: "New" },
  contacted: { color: "bg-amber-100 text-amber-700", label: "Contacted" },
  qualified: { color: "bg-emerald-100 text-emerald-700", label: "Qualified" },
  negotiation: { color: "bg-purple-100 text-purple-700", label: "Negotiation" },
  closed_won: { color: "bg-green-100 text-green-700", label: "Won" },
  closed_lost: { color: "bg-red-100 text-red-700", label: "Lost" },
};

const STAGE_META: Record<string, { color: string; order: number }> = {
  new: { color: "bg-slate-100 text-slate-700", order: 1 },
  qualified: { color: "bg-blue-100 text-blue-700", order: 2 },
  proposal: { color: "bg-amber-100 text-amber-700", order: 3 },
  negotiation: { color: "bg-purple-100 text-purple-700", order: 4 },
  closed_won: { color: "bg-green-100 text-green-700", order: 5 },
  closed_lost: { color: "bg-red-100 text-red-700", order: 6 },
};

function fmtVND(n: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
    notation: "compact",
  }).format(n);
}

function CrmPageInner() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const contactsQuery = useQuery({
    queryKey: ["crm-contacts", { search, statusFilter }],
    queryFn: () =>
      adminApi.getContacts({
        ...(statusFilter !== "all" ? { status: statusFilter } : {}),
        limit: 100,
      }),
    staleTime: 30_000,
  });

  const dealsQuery = useQuery({
    queryKey: ["crm-deals"],
    queryFn: () => adminApi.getDeals({ limit: 100 }),
    staleTime: 30_000,
  });

  const contacts: CrmContact[] = contactsQuery.data?.data ?? [];
  const deals: CrmDeal[] = dealsQuery.data?.data ?? [];

  const filteredContacts = contacts.filter((c) => {
    const matchSearch =
      !search ||
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.email.toLowerCase().includes(search.toLowerCase());
    const matchStatus = statusFilter === "all" || c.status === statusFilter;
    return matchSearch && matchStatus;
  });

  const totalDealValue = deals.reduce((sum, d) => sum + (d.value ?? 0), 0);
  const closedDeals = deals
    .filter((d) => d.stage === "closed_won")
    .reduce((sum, d) => sum + (d.value ?? 0), 0);
  const avgDealValue = deals.length ? totalDealValue / deals.length : 0;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Users className="w-7 h-7 text-primary" />
            CRM
          </h1>
          <p className="text-muted-foreground mt-1">
            Contacts, Deals & Activities from CRM service :8083
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <Search className="w-4 h-4 mr-2" />
            Search
          </Button>
          <Button className="gap-2">
            <Plus className="w-4 h-4" />
            Add Contact
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard
          title="Total Contacts"
          value={contacts.length}
          icon={Users}
          color="bg-blue-100 text-blue-700"
          loading={contactsQuery.isLoading}
        />
        <StatCard
          title="Open Deals"
          value={deals.filter((d) => !d.stage.startsWith("closed")).length}
          icon={Target}
          color="bg-amber-100 text-amber-700"
          loading={dealsQuery.isLoading}
        />
        <StatCard
          title="Pipeline Value"
          value={fmtVND(totalDealValue)}
          icon={TrendingUp}
          color="bg-emerald-100 text-emerald-700"
          loading={dealsQuery.isLoading}
        />
        <StatCard
          title="Closed Won"
          value={fmtVND(closedDeals)}
          icon={DollarSign}
          color="bg-purple-100 text-purple-700"
          loading={dealsQuery.isLoading}
        />
      </div>

      {/* Tabs */}
      <Tabs defaultValue="contacts" className="w-full">
        <TabsList>
          <TabsTrigger value="contacts">Contacts</TabsTrigger>
          <TabsTrigger value="deals">Deals</TabsTrigger>
          <TabsTrigger value="activities">Activities</TabsTrigger>
        </TabsList>

        {/* Contacts Tab */}
        <TabsContent value="contacts" className="space-y-4">
          <Card>
            <CardHeader className="pb-2">
              <div className="flex flex-wrap items-center gap-3">
                <div className="relative flex-1 min-w-[200px] max-w-sm">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                  <Input
                    placeholder="Search contacts..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-9"
                  />
                </div>
                <Select value={statusFilter} onValueChange={setStatusFilter}>
                  <SelectTrigger className="w-36">
                    <SelectValue placeholder="Status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Statuses</SelectItem>
                    <SelectItem value="new">New</SelectItem>
                    <SelectItem value="contacted">Contacted</SelectItem>
                    <SelectItem value="qualified">Qualified</SelectItem>
                    <SelectItem value="negotiation">Negotiation</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardHeader>
            <CardContent>
              {contactsQuery.isLoading ? (
                <LoadingSkeleton.Table rows={6} columns={5} />
              ) : contactsQuery.error ? (
                <EmptyState
                  variant="error"
                  title="Không tải được contacts"
                  description={(contactsQuery.error as Error).message}
                />
              ) : filteredContacts.length === 0 ? (
                <EmptyState
                  variant="search"
                  title="Chưa có contact nào"
                  description="Khi có leads đủ qualified sẽ xuất hiện ở đây."
                />
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
                        <th className="py-3 px-3 font-medium">Name</th>
                        <th className="py-3 px-3 font-medium">Contact</th>
                        <th className="py-3 px-3 font-medium">Company</th>
                        <th className="py-3 px-3 font-medium">Status</th>
                        <th className="py-3 px-3 font-medium">Last Activity</th>
                        <th className="py-3 px-3 font-medium">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredContacts.map((contact) => {
                        const status =
                          STATUS_META[contact.status] ?? STATUS_META.new;
                        const initials = contact.name
                          .split(" ")
                          .map((n) => n[0])
                          .join("")
                          .toUpperCase();
                        return (
                          <tr
                            key={contact.id}
                            className="border-b hover:bg-slate-50/60 transition"
                          >
                            <td className="py-3 px-3">
                              <div className="flex items-center gap-3">
                                <div className="w-9 h-9 rounded-full bg-primary/10 flex items-center justify-center text-primary text-xs font-bold">
                                  {initials}
                                </div>
                                <div>
                                  <div className="font-semibold">{contact.name}</div>
                                  {contact.position && (
                                    <div className="text-xs text-muted-foreground">
                                      {contact.position}
                                    </div>
                                  )}
                                </div>
                              </div>
                            </td>
                            <td className="py-3 px-3 text-xs">
                              <div className="flex items-center gap-1.5">
                                <Mail className="w-3 h-3 text-slate-400" />
                                <span className="font-mono">{contact.email}</span>
                              </div>
                              {contact.phone && (
                                <div className="flex items-center gap-1.5 text-muted-foreground">
                                  <Phone className="w-3 h-3" />
                                  <span className="font-mono">{contact.phone}</span>
                                </div>
                              )}
                            </td>
                            <td className="py-3 px-3">
                              <div className="flex items-center gap-1.5 text-sm">
                                <Building2 className="w-3.5 h-3.5 text-slate-400" />
                                {contact.company ?? "—"}
                              </div>
                            </td>
                            <td className="py-3 px-3">
                              <Badge className={`${status.color} text-xs`}>{status.label}</Badge>
                            </td>
                            <td className="py-3 px-3 text-xs text-muted-foreground">
                              {contact.lastActivity
                                ? format(new Date(contact.lastActivity), "yyyy-MM-dd HH:mm")
                                : "—"}
                            </td>
                            <td className="py-3 px-3">
                              <Button variant="ghost" size="icon">
                                <MoreHorizontal className="w-4 h-4" />
                              </Button>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Deals Tab */}
        <TabsContent value="deals" className="space-y-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base flex items-center gap-2">
                <DollarSign className="w-4 h-4" />
                Deals Pipeline
              </CardTitle>
            </CardHeader>
            <CardContent>
              {dealsQuery.isLoading ? (
                <div className="space-y-3">
                  {Array.from({ length: 4 }).map((_, i) => (
                    <LoadingSkeleton key={i} className="h-20 w-full" />
                  ))}
                </div>
              ) : deals.length === 0 ? (
                <EmptyState
                  variant="default"
                  title="Chưa có deals"
                  description="Tạo deal mới từ contacts đã qualified."
                />
              ) : (
                <div className="space-y-3">
                  {deals.map((deal) => {
                    const stage = STAGE_META[deal.stage] ?? STAGE_META.new;
                    return (
                      <div
                        key={deal.id}
                        className="border rounded-lg p-4 hover:bg-slate-50/50 transition"
                      >
                        <div className="flex items-start justify-between gap-4 flex-wrap">
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 mb-1">
                              <h3 className="font-semibold">{deal.name}</h3>
                              <Badge className={`${stage.color} text-xs`}>
                                {deal.stage.replace("_", " ").replace(/\b\w/g, (l) =>
                                  l.toUpperCase(),
                                )}
                              </Badge>
                            </div>
                            <div className="flex items-center gap-4 text-xs text-muted-foreground">
                              {deal.contact && <span>Contact: {deal.contact}</span>}
                              {deal.expectedClose && (
                                <span>
                                  Expected: {format(new Date(deal.expectedClose), "yyyy-MM-dd")}
                                </span>
                              )}
                            </div>
                          </div>
                          <div className="text-right">
                            <div className="text-xl font-bold">{fmtVND(deal.value ?? 0)}</div>
                            <div className="text-xs text-muted-foreground">
                              {deal.probability ?? 0}% probability
                            </div>
                          </div>
                        </div>
                        <div className="mt-3 flex items-center gap-2">
                          <div className="flex-1 bg-slate-100 rounded-full h-2">
                            <div
                              className={`h-2 rounded-full ${
                                deal.stage === "closed_won"
                                  ? "bg-green-500"
                                  : deal.stage === "closed_lost"
                                    ? "bg-red-500"
                                    : "bg-primary"
                              }`}
                              style={{ width: `${deal.probability ?? 0}%` }}
                            />
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </CardContent>
          </Card>

          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <Card>
              <CardContent className="p-4 text-center">
                <div className="text-2xl font-bold">{fmtVND(totalDealValue)}</div>
                <div className="text-xs text-muted-foreground">Total Pipeline</div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4 text-center">
                <div className="text-2xl font-bold text-green-600">{fmtVND(closedDeals)}</div>
                <div className="text-xs text-muted-foreground">Closed Won</div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4 text-center">
                <div className="text-2xl font-bold">{fmtVND(avgDealValue)}</div>
                <div className="text-xs text-muted-foreground">Avg Deal Size</div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Activities Tab */}
        <TabsContent value="activities">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Activity className="w-4 h-4" />
                Recent Activities
              </CardTitle>
            </CardHeader>
            <CardContent>
              <EmptyState
                variant="default"
                title="Activity feed coming soon"
                description="Hoạt động CRM sẽ stream realtime trong phase tiếp theo."
                className="border-0"
              />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function StatCard({
  title,
  value,
  icon: Icon,
  color,
  loading,
}: {
  title: string;
  value: string | number;
  icon: typeof Users;
  color: string;
  loading?: boolean;
}) {
  return (
    <Card>
      <CardContent className="p-4 flex items-center gap-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${color}`}>
          <Icon className="w-5 h-5" />
        </div>
        <div className="flex-1 min-w-0">
          {loading ? (
            <LoadingSkeleton className="h-7 w-20" />
          ) : (
            <div className="text-2xl font-bold truncate">{value}</div>
          )}
          <div className="text-xs text-muted-foreground">{title}</div>
        </div>
      </CardContent>
    </Card>
  );
}

export default function CRMPage() {
  return (
    <ErrorBoundary>
      <CrmPageInner />
    </ErrorBoundary>
  );
}
