"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import { LoadingSkeleton, EmptyState, ErrorBoundary } from "@rinco/ui";
import {
  Users,
  Search,
  UserPlus,
  MoreHorizontal,
  ChevronLeft,
  ChevronRight,
  Shield,
  Mail,
  Building2,
} from "lucide-react";
import { format } from "date-fns";
import { adminApi, type AdminUser } from "@/lib/admin-api";

const ROLE_META: Record<string, { color: string; label: string }> = {
  admin: { color: "bg-purple-100 text-purple-700", label: "Admin" },
  manager: { color: "bg-blue-100 text-blue-700", label: "Manager" },
  user: { color: "bg-gray-100 text-gray-700", label: "User" },
};

const STATUS_META: Record<string, { variant: "default" | "secondary" | "destructive"; label: string }> = {
  active: { variant: "default", label: "Active" },
  pending: { variant: "secondary", label: "Pending" },
  suspended: { variant: "destructive", label: "Suspended" },
};

function UsersPageInner() {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [page, setPage] = useState(1);
  const pageSize = 10;

  const { data, isLoading, error } = useQuery({
    queryKey: ["users"],
    queryFn: () => adminApi.getUsers({ limit: 200 }),
    staleTime: 30_000,
  });

  const inviteMut = useMutation({
    mutationFn: (p: { email: string; name: string; role: string }) =>
      adminApi.inviteUser(p),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["users"] }),
  });

  const users: AdminUser[] = data?.data ?? [];
  const tenants = Array.from(
    new Set(users.map((u) => u.tenantName ?? u.tenantSlug ?? "").filter(Boolean)),
  ).sort();

  const filtered = users.filter((u) => {
    const matchSearch =
      !search ||
      (u.fullName ?? u.name ?? "").toLowerCase().includes(search.toLowerCase()) ||
      u.email.toLowerCase().includes(search.toLowerCase());
    const matchRole = roleFilter === "all" || u.role === roleFilter;
    const matchStatus = statusFilter === "all" || u.status === statusFilter;
    return matchSearch && matchRole && matchStatus;
  });

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);

  const stats = {
    total: users.length,
    active: users.filter((u) => u.status === "active").length,
    pending: users.filter((u) => u.status === "pending").length,
    suspended: users.filter((u) => u.status === "suspended").length,
  };

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
            {stats.total} users · {stats.active} active · {stats.pending} pending ·{" "}
            {stats.suspended} suspended
          </p>
        </div>
        <div className="flex gap-2">
          <Dialog>
            <DialogTrigger asChild>
              <Button className="gap-2">
                <UserPlus className="w-4 h-4" />
                Invite User
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Invite user</DialogTitle>
                <DialogDescription>
                  Gửi email mời tham gia platform. Người được mời sẽ nhận link đăng ký.
                </DialogDescription>
              </DialogHeader>
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  const form = new FormData(e.currentTarget);
                  inviteMut.mutate({
                    email: String(form.get("email") ?? ""),
                    name: String(form.get("name") ?? ""),
                    role: String(form.get("role") ?? "user"),
                  });
                }}
                className="space-y-4"
              >
                <div>
                  <Label htmlFor="email">Email</Label>
                  <Input id="email" name="email" type="email" required />
                </div>
                <div>
                  <Label htmlFor="name">Full name</Label>
                  <Input id="name" name="name" required />
                </div>
                <div>
                  <Label htmlFor="role">Role</Label>
                  <select
                    id="role"
                    name="role"
                    className="w-full border-2 border-slate-200 bg-white rounded-md px-3 py-2 mt-1 text-sm"
                    defaultValue="user"
                  >
                    <option value="admin">Admin</option>
                    <option value="manager">Manager</option>
                    <option value="user">User</option>
                  </select>
                </div>
                <DialogFooter>
                  <Button type="submit" disabled={inviteMut.isPending}>
                    {inviteMut.isPending ? "Sending…" : "Send invitation"}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard title="Total Users" value={stats.total} icon={Users} color="bg-slate-100 text-slate-700" loading={isLoading} />
        <StatCard title="Active" value={stats.active} icon={Shield} color="bg-emerald-100 text-emerald-700" loading={isLoading} />
        <StatCard title="Pending" value={stats.pending} icon={Mail} color="bg-amber-100 text-amber-700" loading={isLoading} />
        <StatCard title="Suspended" value={stats.suspended} icon={Building2} color="bg-red-100 text-red-700" loading={isLoading} />
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
          {isLoading ? (
            <div className="p-4">
              <LoadingSkeleton.Table rows={pageSize} columns={6} />
            </div>
          ) : error ? (
            <div className="p-6">
              <EmptyState
                variant="error"
                title="Không tải được users"
                description={(error as Error).message}
              />
            </div>
          ) : paged.length === 0 ? (
            <div className="p-6">
              <EmptyState
                variant="search"
                title="Không có user nào khớp filter"
                description="Thử bỏ bộ lọc hoặc mời user mới."
              />
            </div>
          ) : (
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
                  {paged.map((user) => {
                    const roleMeta = ROLE_META[user.role] ?? ROLE_META.user;
                    const statusMeta = STATUS_META[user.status] ?? STATUS_META.pending;
                    const initials = (user.fullName ?? user.name ?? user.email)
                      .split(" ")
                      .map((n) => n[0])
                      .join("")
                      .toUpperCase();
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
                              <div className="font-semibold">{user.fullName ?? user.name ?? "—"}</div>
                              <div className="text-xs text-muted-foreground">{user.email}</div>
                            </div>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant="outline" className="text-xs">
                            {user.tenantName ?? user.tenantSlug ?? "—"}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          <Badge className={`${roleMeta.color} text-xs`}>{roleMeta.label}</Badge>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant={statusMeta.variant} className="text-xs">
                            {statusMeta.label}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 text-xs text-muted-foreground">
                          {user.lastLoginAt
                            ? format(new Date(user.lastLoginAt), "yyyy-MM-dd HH:mm")
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
                  })}
                </tbody>
              </table>
            </div>
          )}

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
  loading,
}: {
  title: string;
  value: number;
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
            <LoadingSkeleton className="h-7 w-12" />
          ) : (
            <div className="text-2xl font-bold">{value}</div>
          )}
          <div className="text-xs text-muted-foreground">{title}</div>
        </div>
      </CardContent>
    </Card>
  );
}

export default function UsersPage() {
  return (
    <ErrorBoundary>
      <UsersPageInner />
    </ErrorBoundary>
  );
}
