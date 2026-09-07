import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ChevronLeft, Building2, Calendar, Users, Target } from "lucide-react";
import { format } from "date-fns";
import { notFound } from "next/navigation";

// Demo data — replaced at runtime by tenant-svc
const DEMO_TENANTS: Record<
  string,
  {
    id: string;
    slug: string;
    name: string;
    status: "active" | "suspended" | "pending";
    plan: string;
    created_at: string;
    contacts: { name: string; email: string; phone: string };
    stats: { leads: number; mrr: number; users: number; conversionRate: number };
  }
> = {
  "apex-fintech": {
    id: "tn_001",
    slug: "apex-fintech",
    name: "APEX Fintech",
    status: "active",
    plan: "Enterprise",
    created_at: "2025-08-12T00:00:00Z",
    contacts: { name: "Nguyen Van A", email: "ops@apex.vn", phone: "0987654321" },
    stats: { leads: 125_847, mrr: 156_000_000, users: 247, conversionRate: 8.5 },
  },
  "demo-corp": {
    id: "tn_002",
    slug: "demo-corp",
    name: "Demo Corp",
    status: "pending",
    plan: "Starter",
    created_at: "2026-01-15T00:00:00Z",
    contacts: { name: "Tran Thi B", email: "hello@demo.vn", phone: "0909123456" },
    stats: { leads: 1_482, mrr: 2_900_000, users: 12, conversionRate: 3.2 },
  },
}

export default async function TenantDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params
  const tenant = DEMO_TENANTS[id]

  if (!tenant) {
    // In production this is `notFound()` after a tenant-svc fetch attempt
    return (
      <div className="p-6 space-y-4">
        <Link
          href="/tenants"
          className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="w-4 h-4 mr-1" />
          Back to Tenants
        </Link>
        <Card>
          <CardHeader>
            <CardTitle>Tenant not found</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground">
              We could not find a tenant with id <code>{id}</code>. The
              record may have been deleted.
            </p>
            <Button asChild className="mt-4">
              <Link href="/tenants">Return to list</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <Link
        href="/tenants"
        className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronLeft className="w-4 h-4 mr-1" />
        All Tenants
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <Building2 className="w-7 h-7 text-primary" />
            <h1 className="text-2xl font-bold">{tenant.name}</h1>
            <Badge
              variant={tenant.status === "active" ? "default" : "secondary"}
              className="ml-2"
            >
              {tenant.status}
            </Badge>
          </div>
          <p className="text-muted-foreground mt-1">
            <code>slug:</code> <code className="font-mono">{tenant.slug}</code>
            {" · "}
            <code>plan:</code> <code>{tenant.plan}</code>
          </p>
        </div>

        <div className="flex gap-2">
          <Button variant="outline">Suspend</Button>
          <Button variant="outline">Reset Password</Button>
          <Button>Edit</Button>
        </div>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase text-muted-foreground">Leads</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{tenant.stats.leads.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
              <Target className="w-3 h-3" /> All time
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase text-muted-foreground">MRR</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {new Intl.NumberFormat("vi-VN", {
                style: "currency",
                currency: "VND",
                notation: "compact",
              }).format(tenant.stats.mrr)}
            </div>
            <p className="text-xs text-muted-foreground mt-1">recurring</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase text-muted-foreground">Users</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{tenant.stats.users}</div>
            <p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
              <Users className="w-3 h-3" /> Active
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-xs uppercase text-muted-foreground">
              Conversion
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{tenant.stats.conversionRate}%</div>
            <p className="text-xs text-muted-foreground mt-1">last 30 days</p>
          </CardContent>
        </Card>
      </div>

      {/* Contact / Meta */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Primary Contact</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div>
              <span className="text-muted-foreground">Name:</span>{" "}
              <span className="font-medium">{tenant.contacts.name}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Email:</span>{" "}
              <span className="font-mono">{tenant.contacts.email}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Phone:</span>{" "}
              <span className="font-mono">{tenant.contacts.phone}</span>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Metadata</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex items-center gap-2">
              <Calendar className="w-4 h-4 text-muted-foreground" />
              <span className="text-muted-foreground">Created:</span>{" "}
              <span>{format(new Date(tenant.created_at), "PPpp")}</span>
            </div>
            <div>
              <span className="text-muted-foreground">ID:</span>{" "}
              <code className="font-mono text-xs">{tenant.id}</code>
            </div>
            <div>
              <span className="text-muted-foreground">Plan:</span>{" "}
              <Badge variant="outline">{tenant.plan}</Badge>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
