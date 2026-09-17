"use client";

import { useState } from "react";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  Users,
  Building2,
  DollarSign,
  Calendar,
  Plus,
  Search,
  MoreHorizontal,
  ChevronLeft,
  ChevronRight,
  Phone,
  Mail,
  MapPin,
  Briefcase,
  TrendingUp,
  Activity,
  Target,
} from "lucide-react";
import { format } from "date-fns";

// Mock CRM data
const mockContacts = [
  {
    id: "c_001",
    name: "Tran Quoc Viet",
    email: "viet.tq@company-a.vn",
    phone: "0981234567",
    company: "Company A Corp",
    position: "CEO",
    status: "qualified",
    source: "website",
    tenant: "Apex Fintech",
    createdAt: "2025-08-15T00:00:00Z",
    lastActivity: "2026-09-16T10:30:00Z",
  },
  {
    id: "c_002",
    name: "Nguyen Thi Mai Lan",
    email: "mai.lan@company-b.vn",
    phone: "0912345678",
    company: "Company B Ltd",
    position: "Marketing Director",
    status: "contacted",
    source: "referral",
    tenant: "Vietnam Realty",
    createdAt: "2025-09-20T00:00:00Z",
    lastActivity: "2026-09-15T14:20:00Z",
  },
  {
    id: "c_003",
    name: "Le Van Sung",
    email: "sung.lv@company-c.vn",
    phone: "0934567890",
    company: "Company C JSC",
    position: "CTO",
    status: "new",
    source: "linkedin",
    tenant: "Saigon Health Group",
    createdAt: "2026-01-10T00:00:00Z",
    lastActivity: "2026-09-17T09:00:00Z",
  },
  {
    id: "c_004",
    name: "Pham Thi Hoa",
    email: "hoa.pt@company-d.vn",
    phone: "0945678901",
    company: "Company D Group",
    position: "Sales Manager",
    status: "negotiation",
    source: "google_ads",
    tenant: "MegaShop Vietnam",
    createdAt: "2025-06-05T00:00:00Z",
    lastActivity: "2026-09-17T11:00:00Z",
  },
  {
    id: "c_005",
    name: "Hoang Van Duc",
    email: "duc.hv@company-e.vn",
    phone: "0956789012",
    company: "Company E Corp",
    position: "Procurement Head",
    status: "qualified",
    source: "website",
    tenant: "Logistics Pro",
    createdAt: "2025-11-12T00:00:00Z",
    lastActivity: "2026-09-14T16:45:00Z",
  },
];

const mockCompanies = [
  { id: "co_001", name: "Company A Corp", contacts: 12, deals: 5, revenue: 125_000_000, status: "customer" },
  { id: "co_002", name: "Company B Ltd", contacts: 8, deals: 3, revenue: 48_000_000, status: "customer" },
  { id: "co_003", name: "Company C JSC", contacts: 15, deals: 7, revenue: 220_000_000, status: "customer" },
  { id: "co_004", name: "Company D Group", contacts: 20, deals: 12, revenue: 380_000_000, status: "customer" },
  { id: "co_005", name: "Company E Corp", contacts: 6, deals: 2, revenue: 32_000_000, status: "prospect" },
];

const mockDeals = [
  { id: "d_001", name: "Enterprise Package", value: 50_000_000, stage: "negotiation", contact: "Tran Quoc Viet", probability: 75, expectedClose: "2026-10-15", tenant: "Apex Fintech" },
  { id: "d_002", name: "Pro Upgrade", value: 25_000_000, stage: "proposal", contact: "Nguyen Thi Mai Lan", probability: 50, expectedClose: "2026-10-30", tenant: "Vietnam Realty" },
  { id: "d_003", name: "Annual Contract", value: 120_000_000, stage: "qualified", contact: "Le Van Sung", probability: 80, expectedClose: "2026-11-15", tenant: "Saigon Health" },
  { id: "d_004", name: "Multi-seat License", value: 75_000_000, stage: "closed_won", contact: "Pham Thi Hoa", probability: 100, expectedClose: "2026-09-01", tenant: "MegaShop" },
  { id: "d_005", name: "Starter Package", value: 15_000_000, stage: "new", contact: "Hoang Van Duc", probability: 20, expectedClose: "2026-12-01", tenant: "Logistics Pro" },
];

const STATUS_META = {
  new: { color: "bg-blue-100 text-blue-700", label: "New" },
  contacted: { color: "bg-amber-100 text-amber-700", label: "Contacted" },
  qualified: { color: "bg-emerald-100 text-emerald-700", label: "Qualified" },
  negotiation: { color: "bg-purple-100 text-purple-700", label: "Negotiation" },
  closed_won: { color: "bg-green-100 text-green-700", label: "Won" },
  closed_lost: { color: "bg-red-100 text-red-700", label: "Lost" },
};

const STAGE_META = {
  new: { color: "bg-slate-100 text-slate-700", order: 1 },
  qualified: { color: "bg-blue-100 text-blue-700", order: 2 },
  proposal: { color: "bg-amber-100 text-amber-700", order: 3 },
  negotiation: { color: "bg-purple-100 text-purple-700", order: 4 },
  closed_won: { color: "bg-green-100 text-green-700", order: 5 },
  closed_lost: { color: "bg-red-100 text-red-700", order: 6 },
};

export default function CRMPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const filteredContacts = mockContacts.filter((c) => {
    const matchSearch = !search || c.name.toLowerCase().includes(search.toLowerCase()) || c.email.toLowerCase().includes(search.toLowerCase());
    const matchStatus = statusFilter === "all" || c.status === statusFilter;
    return matchSearch && matchStatus;
  });

  const totalDealValue = mockDeals.reduce((sum, d) => sum + d.value, 0);
  const closedDeals = mockDeals.filter((d) => d.stage === "closed_won").reduce((sum, d) => sum + d.value, 0);
  const avgDealValue = totalDealValue / mockDeals.length;

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
            Contacts, Companies, Deals & Activities across all tenants
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
        <StatCard title="Total Contacts" value={mockContacts.length} icon={Users} color="bg-blue-100 text-blue-700" />
        <StatCard title="Companies" value={mockCompanies.length} icon={Building2} color="bg-purple-100 text-purple-700" />
        <StatCard title="Open Deals" value={mockDeals.filter((d) => !d.stage.startsWith("closed")).length} icon={Target} color="bg-amber-100 text-amber-700" />
        <StatCard
          title="Pipeline Value"
          value={new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", notation: "compact" }).format(totalDealValue)}
          icon={TrendingUp}
          color="bg-emerald-100 text-emerald-700"
        />
      </div>

      {/* Main content tabs */}
      <Tabs defaultValue="contacts" className="w-full">
        <TabsList>
          <TabsTrigger value="contacts">Contacts</TabsTrigger>
          <TabsTrigger value="companies">Companies</TabsTrigger>
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
                  <Input placeholder="Search contacts..." value={search} onChange={(e) => setSearch(e.target.value)} className="pl-9" />
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
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
                      <th className="py-3 px-3 font-medium">Name</th>
                      <th className="py-3 px-3 font-medium">Company</th>
                      <th className="py-3 px-3 font-medium">Status</th>
                      <th className="py-3 px-3 font-medium">Source</th>
                      <th className="py-3 px-3 font-medium">Tenant</th>
                      <th className="py-3 px-3 font-medium">Last Activity</th>
                      <th className="py-3 px-3 font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredContacts.map((contact) => {
                      const status = STATUS_META[contact.status as keyof typeof STATUS_META];
                      return (
                        <tr key={contact.id} className="border-b hover:bg-slate-50/60 transition">
                          <td className="py-3 px-3">
                            <div className="flex items-center gap-3">
                              <div className="w-9 h-9 rounded-full bg-primary/10 flex items-center justify-center text-primary text-xs font-bold">
                                {contact.name.split(" ").map((n) => n[0]).join("").toUpperCase()}
                              </div>
                              <div>
                                <div className="font-semibold">{contact.name}</div>
                                <div className="text-xs text-muted-foreground">{contact.position}</div>
                              </div>
                            </div>
                          </td>
                          <td className="py-3 px-3">
                            <div className="flex items-center gap-1.5 text-sm">
                              <Building2 className="w-3.5 h-3.5 text-slate-400" />
                              {contact.company}
                            </div>
                          </td>
                          <td className="py-3 px-3">
                            <Badge className={`${status.color} text-xs`}>{status.label}</Badge>
                          </td>
                          <td className="py-3 px-3 text-xs capitalize">{contact.source.replace("_", " ")}</td>
                          <td className="py-3 px-3">
                            <Badge variant="outline" className="text-xs">{contact.tenant}</Badge>
                          </td>
                          <td className="py-3 px-3 text-xs text-muted-foreground">
                            {format(new Date(contact.lastActivity), "yyyy-MM-dd HH:mm")}
                          </td>
                          <td className="py-3 px-3">
                            <Button variant="ghost" size="icon"><MoreHorizontal className="w-4 h-4" /></Button>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Companies Tab */}
        <TabsContent value="companies" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {mockCompanies.map((company) => (
              <Card key={company.id} className="hover:border-primary/50 transition-colors cursor-pointer">
                <CardContent className="p-4">
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                        <Building2 className="w-5 h-5 text-primary" />
                      </div>
                      <div>
                        <div className="font-semibold">{company.name}</div>
                        <Badge variant={company.status === "customer" ? "default" : "secondary"} className="text-xs mt-1 capitalize">
                          {company.status}
                        </Badge>
                      </div>
                    </div>
                    <Button variant="ghost" size="icon"><MoreHorizontal className="w-4 h-4" /></Button>
                  </div>
                  <div className="grid grid-cols-3 gap-2 text-center">
                    <div className="bg-slate-50 rounded-lg p-2">
                      <div className="text-lg font-bold">{company.contacts}</div>
                      <div className="text-[10px] text-muted-foreground">Contacts</div>
                    </div>
                    <div className="bg-slate-50 rounded-lg p-2">
                      <div className="text-lg font-bold">{company.deals}</div>
                      <div className="text-[10px] text-muted-foreground">Deals</div>
                    </div>
                    <div className="bg-slate-50 rounded-lg p-2">
                      <div className="text-lg font-bold">
                        {new Intl.NumberFormat("vi-VN", { notation: "compact" }).format(company.revenue)}
                      </div>
                      <div className="text-[10px] text-muted-foreground">Revenue</div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
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
              <div className="space-y-3">
                {mockDeals.map((deal) => {
                  const stage = STAGE_META[deal.stage as keyof typeof STAGE_META];
                  return (
                    <div key={deal.id} className="border rounded-lg p-4 hover:bg-slate-50/50 transition">
                      <div className="flex items-start justify-between gap-4 flex-wrap">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <h3 className="font-semibold">{deal.name}</h3>
                            <Badge className={`${stage.color} text-xs`}>
                              {deal.stage.replace("_", " ").replace(/\b\w/g, (l) => l.toUpperCase())}
                            </Badge>
                          </div>
                          <div className="flex items-center gap-4 text-xs text-muted-foreground">
                            <span>Contact: {deal.contact}</span>
                            <span>Tenant: {deal.tenant}</span>
                            <span>Expected: {format(new Date(deal.expectedClose), "yyyy-MM-dd")}</span>
                          </div>
                        </div>
                        <div className="text-right">
                          <div className="text-xl font-bold">
                            {new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", notation: "compact" }).format(deal.value)}
                          </div>
                          <div className="text-xs text-muted-foreground">
                            {deal.probability}% probability
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
                            style={{ width: `${deal.probability}%` }}
                          />
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </CardContent>
          </Card>

          {/* Deal Stats */}
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <Card>
              <CardContent className="p-4 text-center">
                <div className="text-2xl font-bold">
                  {new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", notation: "compact" }).format(totalDealValue)}
                </div>
                <div className="text-xs text-muted-foreground">Total Pipeline</div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4 text-center">
                <div className="text-2xl font-bold text-green-600">
                  {new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", notation: "compact" }).format(closedDeals)}
                </div>
                <div className="text-xs text-muted-foreground">Closed Won</div>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4 text-center">
                <div className="text-2xl font-bold">
                  {new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", notation: "compact" }).format(avgDealValue)}
                </div>
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
              <div className="space-y-4">
                {[
                  { time: "10 min ago", actor: "Tran Quoc Viet", action: "Updated deal status", target: "Enterprise Package", type: "deal" },
                  { time: "25 min ago", actor: "Nguyen Thi Mai Lan", action: "Sent email", target: "Proposal for Pro Upgrade", type: "email" },
                  { time: "1 hour ago", actor: "Le Van Sung", action: "Created contact", target: "New lead from LinkedIn", type: "contact" },
                  { time: "2 hours ago", actor: "Pham Thi Hoa", action: "Won deal", target: "Multi-seat License - 75M VND", type: "deal" },
                  { time: "3 hours ago", actor: "Hoang Van Duc", action: "Scheduled meeting", target: "Discovery call", type: "meeting" },
                ].map((activity, idx) => (
                  <div key={idx} className="flex items-start gap-4 text-sm">
                    <div className="w-9 h-9 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                      <Activity className="w-4 h-4 text-primary" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="font-medium">{activity.actor}</div>
                      <div className="text-muted-foreground text-xs">
                        {activity.action} — <span className="font-medium">{activity.target}</span>
                      </div>
                    </div>
                    <span className="text-xs text-muted-foreground whitespace-nowrap">{activity.time}</span>
                  </div>
                ))}
              </div>
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
}: {
  title: string;
  value: string | number;
  icon: typeof Users;
  color: string;
}) {
  return (
    <Card>
      <CardContent className="p-4 flex items-center gap-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${color}`}>
          <Icon className="w-5 h-5" />
        </div>
        <div>
          <div className="text-2xl font-bold">{value}</div>
          <div className="text-xs text-muted-foreground">{title}</div>
        </div>
      </CardContent>
    </Card>
  );
}
