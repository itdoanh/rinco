"use client";

import { useState, useMemo } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
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
  Search,
  Plus,
  MoreHorizontal,
  ChevronLeft,
  ChevronRight,
  Shield,
  Mail,
  Building2,
  UserPlus,
  TreePine,
} from "lucide-react";
import { format } from "date-fns";

const mockUsers = [
  {
    id: "u_001",
    email: "nguyen.van.a@apexfintech.vn",
    name: "Nguyen Van A",
    role: "admin",
    tenant: "Apex Fintech",
    tenantSlug: "apexfintech",
    status: "active",
    lastLogin: "2026-09-17T10:30:00Z",
    createdAt: "2025-01-15T00:00:00Z",
    avatar: null,
  },
  {
    id: "u_002",
    email: "tran.thi.b@vietnamrealty.vn",
    name: "Tran Thi B",
    role: "manager",
    tenant: "Vietnam Realty",
    tenantSlug: "vietnamrealty",
    status: "active",
    lastLogin: "2026-09-17T08:15:00Z",
    createdAt: "2025-03-22T00:00:00Z",
    avatar: null,
  },
  {
    id: "u_003",
    email: "le.van.c@saigonhealth.vn",
    name: "Le Van C",
    role: "user",
    tenant: "Saigon Health Group",
    tenantSlug: "saigonhealth",
    status: "active",
    lastLogin: "2026-09-16T14:22:00Z",
    createdAt: "2024-11-08T00:00:00Z",
    avatar: null,
  },
  {
    id: "u_004",
    email: "pham.thi.d@eduviet.edu.vn",
    name: "Pham Thi D",
    role: "user",
    tenant: "EduViet Academy",
    tenantSlug: "eduviet",
    status: "active",
    lastLogin: "2026-09-15T16:45:00Z",
    createdAt: "2026-08-30T00:00:00Z",
    avatar: null,
  },
  {
    id: "u_005",
    email: "hoang.van.e@megashop.vn",
    name: "Hoang Van E",
    role: "admin",
    tenant: "MegaShop Vietnam",
    tenantSlug: "megashop",
    status: "active",
    lastLogin: "2026-09-17T09:00:00Z",
    createdAt: "2024-06-14T00:00:00Z",
    avatar: null,
  },
  {
    id: "u_006",
    email: "vu.thi.f@greentech.vn",
    name: "Vu Thi F",
    role: "user",
    tenant: "GreenTech Solutions",
    tenantSlug: "greentech",
    status: "pending",
    lastLogin: null,
    createdAt: "2026-09-10T00:00:00Z",
    avatar: null,
  },
  {
    id: "u_007",
    email: "do.van.g@logistics-pro.vn",
    name: "Do Van G",
    role: "manager",
    tenant: "Logistics Pro",
    tenantSlug: "logistics-pro",
    status: "active",
    lastLogin: "2026-09-17T11:20:00Z",
    createdAt: "2025-09-05T00:00:00Z",
    avatar: null,
  },
  {
    id: "u_008",
    email: "bui.thi.h@fintechhub.asia",
    name: "Bui Thi H",
    role: "admin",
    tenant: "Fintech Hub Asia",
    tenantSlug: "fintech-hub",
    status: "suspended",
    lastLogin: "2026-09-10T08:00:00Z",
    createdAt: "2025-05-18T00:00:00Z",
    avatar: null,
  },
];

const ROLE_META = {
  admin: { color: "bg-purple-100 text-purple-700", label: "Admin" },
  manager: { color: "bg-blue-100 text-blue-700", label: "Manager" },
  user: { color: "bg-gray-100 text-gray-700", label: "User" },
};

const STATUS_META = {
  active: { variant: "default" as const, label: "Active" },
  pending: { variant: "secondary" as const, label: "Pending" },
  suspended: { variant: "destructive" as const, label: "Suspended" },
};

export default function UsersPage() {
  const [search, setSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [tenantFilter, setTenantFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [page, setPage] = useState(1);
  const pageSize = 10;

  const tenants = useMemo(
    () => Array.from(new Set(mockUsers.map((u) => u.tenant))).sort(),
    [],
  );

  const filtered = useMemo(() => {
    return mockUsers.filter((u) => {
      const matchSearch =
        !search ||
        u.name.toLowerCase().includes(search.toLowerCase()) ||
        u.email.toLowerCase().includes(search.toLowerCase());
      const matchRole = roleFilter === "all" || u.role === roleFilter;
      const matchTenant = tenantFilter === "all" || u.tenantSlug === tenantFilter;
      const matchStatus = statusFilter === "all" || u.status === statusFilter;
      return matchSearch && matchRole && matchTenant && matchStatus;
    });
  }, [search, roleFilter, tenantFilter, statusFilter]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);

  const stats = useMemo(() => ({
    total: mockUsers.length,
    active: mockUsers.filter((u) => u.status === "active").length,
    pending: mockUsers.filter((u) => u.status === "pending").length,
    suspended: mockUsers.filter((u) => u.status === "suspended").length,
  }), []);

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Users className="w-7 h-7 text-primary" />
            Users
          </h1>
          <p className="text-muted-foreground mt-1">
            {stats.total} users · {stats.active} active · {stats.pending} pending · {stats.suspended} suspended
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" className="gap-2">
            <TreePine className="w-4 h-4" />
            Tree View
          </Button>
          <Button className="gap-2">
            <UserPlus className="w-4 h-4" />
            Invite User
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard title="Total Users" value={stats.total} icon={Users} color="bg-slate-100 text-slate-700" />
        <StatCard title="Active" value={stats.active} icon={Shield} color="bg-emerald-100 text-emerald-700" />
        <StatCard title="Pending" value={stats.pending} icon={Mail} color="bg-amber-100 text-amber-700" />
        <StatCard title="Suspended" value={stats.suspended} icon={Building2} color="bg-red-100 text-red-700" />
      </div>

      {/* Filter bar */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <Input
                placeholder="Search by name or email..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={roleFilter} onValueChange={setRoleFilter}>
              <SelectTrigger className="w-32">
                <SelectValue placeholder="Role" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Roles</SelectItem>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="manager">Manager</SelectItem>
                <SelectItem value="user">User</SelectItem>
              </SelectContent>
            </Select>
            <Select value={tenantFilter} onValueChange={setTenantFilter}>
              <SelectTrigger className="w-44">
                <SelectValue placeholder="Tenant" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Tenants</SelectItem>
                {tenants.map((t) => (
                  <SelectItem key={t} value={t.toLowerCase().replace(/\s+/g, "-")}>
                    {t}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-32">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Statuses</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="suspended">Suspended</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
                  <th className="px-4 py-3 font-medium">User</th>
                  <th className="px-4 py-3 font-medium">Tenant</th>
                  <th className="px-4 py-3 font-medium">Role</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3 font-medium">Last Login</th>
                  <th className="px-4 py-3 font-medium">Created</th>
                  <th className="px-4 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {paged.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="text-center py-12 text-muted-foreground">
                      No users match the current filters
                    </td>
                  </tr>
                ) : (
                  paged.map((user) => {
                    const roleMeta = ROLE_META[user.role as keyof typeof ROLE_META];
                    const statusMeta = STATUS_META[user.status as keyof typeof STATUS_META];
                    const initials = user.name.split(" ").map((n) => n[0]).join("").toUpperCase();

                    return (
                      <tr key={user.id} className="border-b hover:bg-slate-50/60 transition">
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-3">
                            <Avatar className="w-9 h-9">
                              <AvatarFallback className="bg-primary/10 text-primary text-xs">
                                {initials}
                              </AvatarFallback>
                            </Avatar>
                            <div>
                              <div className="font-semibold">{user.name}</div>
                              <div className="text-xs text-muted-foreground">{user.email}</div>
                            </div>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant="outline" className="text-xs">
                            {user.tenant}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          <Badge className={`${roleMeta.color} text-xs`}>
                            {roleMeta.label}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant={statusMeta.variant} className="text-xs">
                            {statusMeta.label}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground">
                          {user.lastLogin
                            ? format(new Date(user.lastLogin), "yyyy-MM-dd HH:mm")
                            : "Never"}
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground">
                          {format(new Date(user.createdAt), "yyyy-MM-dd")}
                        </td>
                        <td className="px-4 py-3">
                          <Button variant="ghost" size="icon" aria-label="Actions">
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {filtered.length > 0 && (
            <div className="flex items-center justify-between px-4 py-3 border-t">
              <div className="text-sm text-muted-foreground">
                Page {page} of {totalPages} ({filtered.length} users)
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  <ChevronLeft className="w-4 h-4" /> Prev
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                >
                  Next <ChevronRight className="w-4 h-4" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
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
  value: number;
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
