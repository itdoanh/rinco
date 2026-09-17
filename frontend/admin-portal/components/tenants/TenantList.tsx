"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { adminApi, type AdminTenant } from "@/lib/admin-api";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Building2,
  Search,
  Plus,
  MoreHorizontal,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { LoadingSkeleton, EmptyState, ErrorBoundary } from "@rinco/ui";
import { format } from "date-fns";

const STATUS_BADGE: Record<NonNullable<AdminTenant["status"]>, {
  variant: "default" | "secondary" | "destructive";
  label: string;
  className: string;
}> = {
  active: {
    variant: "default",
    label: "Active",
    className: "bg-emerald-100 text-emerald-700 hover:bg-emerald-100",
  },
  trial: {
    variant: "secondary",
    label: "Trial",
    className: "bg-amber-100 text-amber-700 hover:bg-amber-100",
  },
  pending: {
    variant: "secondary",
    label: "Pending",
    className: "bg-blue-100 text-blue-700 hover:bg-blue-100",
  },
  suspended: { variant: "destructive", label: "Suspended", className: "" },
};

type SortKey = "name" | "users" | "leads" | "revenue" | "createdAt";

function TenantListInner() {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [planFilter, setPlanFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<SortKey>("name");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");
  const [page, setPage] = useState(1);
  const pageSize = 10;

  const { data: tenantsResp, isLoading, error } = useQuery({
    queryKey: ["tenants"],
    queryFn: () => adminApi.getTenants({ limit: 100 }),
    staleTime: 30_000,
  });

  const tenants: AdminTenant[] = tenantsResp?.data ?? [];

  const filtered = useMemo(() => {
    const out = tenants.filter((t) => {
      const matchSearch =
        !search ||
        t.name.toLowerCase().includes(search.toLowerCase()) ||
        t.slug.toLowerCase().includes(search.toLowerCase());
      const matchStatus = statusFilter === "all" || t.status === statusFilter;
      const matchPlan = planFilter === "all" || t.plan === planFilter;
      return matchSearch && matchStatus && matchPlan;
    });
    out.sort((a, b) => {
      const av = (a[sortBy] ?? "") as string | number;
      const bv = (b[sortBy] ?? "") as string | number;
      const cmp =
        typeof av === "string" && typeof bv === "string"
          ? av.localeCompare(bv)
          : Number(av) - Number(bv);
      return sortDir === "asc" ? cmp : -cmp;
    });
    return out;
  }, [tenants, search, statusFilter, planFilter, sortBy, sortDir]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const paged = filtered.slice((page - 1) * pageSize, page * pageSize);

  const toggleSort = (key: SortKey) => {
    if (sortBy === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortBy(key);
      setSortDir("asc");
    }
  };

  const createMut = useMutation({
    mutationFn: (slug: string) =>
      adminApi.createTenant({ slug, name: slug, plan: "free" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tenants"] }),
  });

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Tenants</h1>
          <p className="text-muted-foreground mt-1">
            {tenants.length} tenants · {filtered.length} matching filters
          </p>
        </div>
        <Button
          className="gap-2"
          onClick={() => createMut.mutate(`new-${Date.now()}`)}
          disabled={createMut.isPending}
        >
          <Plus className="w-4 h-4" /> New Tenant
        </Button>
      </div>

      {/* Filter bar */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <Input
                placeholder="Search by name or slug..."
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
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="trial">Trial</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="suspended">Suspended</SelectItem>
              </SelectContent>
            </Select>
            <Select value={planFilter} onValueChange={setPlanFilter}>
              <SelectTrigger className="w-32">
                <SelectValue placeholder="Plan" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Plans</SelectItem>
                <SelectItem value="free">Free</SelectItem>
                <SelectItem value="starter">Starter</SelectItem>
                <SelectItem value="pro">Pro</SelectItem>
                <SelectItem value="enterprise">Enterprise</SelectItem>
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
                title="Không tải được tenants"
                description={(error as Error).message}
              />
            </div>
          ) : paged.length === 0 ? (
            <div className="p-6">
              <EmptyState
                variant="search"
                title="Không có tenant nào khớp filter"
                description="Thử bỏ bộ lọc hoặc tạo tenant mới."
              />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
                    <th className="px-4 py-3">
                      <button
                        onClick={() => toggleSort("name")}
                        className="font-medium hover:text-slate-900"
                      >
                        Tenant {sortBy === "name" && (sortDir === "asc" ? "↑" : "↓")}
                      </button>
                    </th>
                    <th className="px-4 py-3 font-medium">Status</th>
                    <th className="px-4 py-3 font-medium">Plan</th>
                    <th className="px-4 py-3">
                      <button
                        onClick={() => toggleSort("users")}
                        className="font-medium hover:text-slate-900"
                      >
                        Users {sortBy === "users" && (sortDir === "asc" ? "↑" : "↓")}
                      </button>
                    </th>
                    <th className="px-4 py-3">
                      <button
                        onClick={() => toggleSort("leads")}
                        className="font-medium hover:text-slate-900"
                      >
                        Leads {sortBy === "leads" && (sortDir === "asc" ? "↑" : "↓")}
                      </button>
                    </th>
                    <th className="px-4 py-3">
                      <button
                        onClick={() => toggleSort("revenue")}
                        className="font-medium hover:text-slate-900"
                      >
                        Revenue {sortBy === "revenue" && (sortDir === "asc" ? "↑" : "↓")}
                      </button>
                    </th>
                    <th className="px-4 py-3 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {paged.map((t) => {
                    const status =
                      STATUS_BADGE[(t.status ?? "pending") as keyof typeof STATUS_BADGE] ??
                      STATUS_BADGE.pending;
                    const initials = (t.name ?? t.slug ?? "?").charAt(0).toUpperCase();
                    return (
                      <tr
                        key={t.id}
                        className="border-b hover:bg-slate-50/60 dark:hover:bg-slate-900/40 transition"
                      >
                        <td className="px-4 py-3">
                          <Link
                            href={`/tenants/${t.slug ?? t.id}`}
                            className="flex items-center gap-3 group"
                          >
                            <div
                              className="w-9 h-9 rounded-lg flex items-center justify-center text-white font-bold shrink-0"
                              style={{ backgroundColor: t.color ?? "#0EA5E9" }}
                            >
                              {initials}
                            </div>
                            <div className="min-w-0">
                              <div className="font-semibold group-hover:text-primary transition-colors truncate">
                                {t.name ?? t.slug}
                              </div>
                              <div className="text-xs text-muted-foreground font-mono truncate">
                                {t.slug}
                              </div>
                            </div>
                          </Link>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant={status.variant} className={status.className}>
                            {status.label}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 capitalize">{t.plan ?? "—"}</td>
                        <td className="px-4 py-3 tabular-nums">{t.users?.toLocaleString() ?? "—"}</td>
                        <td className="px-4 py-3 tabular-nums">
                          {t.leads?.toLocaleString() ?? "—"}
                        </td>
                        <td className="px-4 py-3 tabular-nums">
                          {t.revenue
                            ? new Intl.NumberFormat("vi-VN", {
                                style: "currency",
                                currency: "VND",
                                notation: "compact",
                              }).format(t.revenue)
                            : "—"}
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
                Page {page} of {totalPages}
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

export function TenantList() {
  return (
    <ErrorBoundary>
      <TenantListInner />
    </ErrorBoundary>
  );
}

export default TenantList;
