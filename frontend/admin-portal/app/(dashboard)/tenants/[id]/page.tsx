import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  ChevronLeft,
  Building2,
  Calendar,
  Users,
  Target,
  Activity,
  Globe,
  CreditCard,
  Settings as SettingsIcon,
  Server,
} from "lucide-react";
import { format } from "date-fns";
import { notFound } from "next/navigation";
import { mockTenants } from "@/lib/mock-data";

export default async function TenantDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const tenant = mockTenants.find((t) => t.slug === id || t.id === id);

  if (!tenant) {
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
              We could not find a tenant with id <code>{id}</code>.
            </p>
            <Button asChild className="mt-4">
              <Link href="/tenants">Return to list</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    );
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
      <div className="flex items-start justify-between flex-wrap gap-4">
        <div className="flex items-center gap-4">
          <div
            className="w-16 h-16 rounded-2xl flex items-center justify-center text-white font-bold text-2xl"
            style={{ backgroundColor: tenant.color }}
          >
            {tenant.name[0]}
          </div>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{tenant.name}</h1>
              <Badge variant={tenant.status === "active" ? "default" : "secondary"}>
                {tenant.status}
              </Badge>
            </div>
            <p className="text-muted-foreground mt-1 text-sm">
              <code className="font-mono">{tenant.slug}</code> · {tenant.plan} · {tenant.region}
            </p>
          </div>
        </div>

        <div className="flex gap-2">
          <Button variant="outline">Suspend</Button>
          <Button variant="outline">Reset Password</Button>
          <Button>Edit</Button>
        </div>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard title="Leads" value={tenant.leads.toLocaleString()} icon={Target} hint="All time" />
        <StatCard
          title="MRR"
          value={new Intl.NumberFormat("vi-VN", {
            style: "currency",
            currency: "VND",
            notation: "compact",
          }).format(tenant.revenue)}
          icon={CreditCard}
          hint="Monthly recurring"
        />
        <StatCard title="Users" value={tenant.users.toString()} icon={Users} hint="Active" />
        <StatCard title="Conversion" value={`${tenant.conversionRate}%`} icon={Activity} hint="Last 30 days" />
      </div>

      {/* Tabs */}
      <Tabs defaultValue="overview" className="w-full">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="leads">Leads</TabsTrigger>
          <TabsTrigger value="domains">Domains</TabsTrigger>
          <TabsTrigger value="billing">Billing</TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
          <TabsTrigger value="audit">Audit</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Users className="w-4 h-4" /> Primary Contact
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                <Field label="Name" value={tenant.primaryContact.name} />
                <Field label="Email" value={tenant.primaryContact.email} mono />
                <Field label="Phone" value={tenant.primaryContact.phone} mono />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Server className="w-4 h-4" /> Infrastructure
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                <Field label="Isolation" value={tenant.isolationMode} />
                <Field label="VPS Node" value={tenant.vpsNodeId ?? "Shared cluster"} mono />
                <Field label="Region" value={tenant.region} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Globe className="w-4 h-4" /> Domains
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-1 text-sm font-mono">
                {tenant.domains.map((d) => (
                  <div key={d} className="text-slate-700">· {d}</div>
                ))}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Calendar className="w-4 h-4" /> Metadata
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                <Field label="Created" value={format(new Date(tenant.createdAt), "PPpp")} />
                <Field label="ID" value={tenant.id} mono />
                <Field label="Plan" value={<Badge variant="outline">{tenant.plan}</Badge>} />
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="users">
          <Card>
            <CardHeader>
              <CardTitle>Users ({tenant.users})</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">
                User management UI will be implemented in Phase 2. {tenant.users} active users in this tenant.
              </p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="leads">
          <Card>
            <CardHeader>
              <CardTitle>Leads ({tenant.leads.toLocaleString()})</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">
                {tenant.leads.toLocaleString()} leads generated all-time. View detailed lead analytics in Phase 2.
              </p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="domains">
          <Card>
            <CardHeader>
              <CardTitle>Domains & SSL</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {tenant.domains.map((d) => (
                  <div key={d} className="flex items-center justify-between p-3 rounded-lg border">
                    <span className="font-mono text-sm">{d}</span>
                    <Badge variant="default" className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100">
                      SSL Active
                    </Badge>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="billing">
          <Card>
            <CardHeader>
              <CardTitle>Billing</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3 text-sm">
                <Field label="Plan" value={<Badge variant="outline">{tenant.plan}</Badge>} />
                <Field label="MRR" value={new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND" }).format(tenant.revenue)} />
                <Field label="Status" value={<Badge>Active</Badge>} />
                <Field label="Next billing" value={format(new Date(Date.now() + 30 * 86400_000), "PP")} />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="settings">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <SettingsIcon className="w-4 h-4" /> Tenant Settings
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">
                Per-tenant settings UI coming in Phase 2.
              </p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="audit">
          <Card>
            <CardHeader>
              <CardTitle>Audit Log</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">
                Full audit log query interface in Phase 2.
              </p>
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
  hint,
}: {
  title: string;
  value: string;
  icon: typeof Building2;
  hint?: string;
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground font-medium flex items-center gap-2">
          <Icon className="w-3 h-3" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {hint && <p className="text-xs text-muted-foreground mt-1">{hint}</p>}
      </CardContent>
    </Card>
  );
}

function Field({ label, value, mono = false }: { label: string; value: React.ReactNode; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      <span className={mono ? "font-mono text-xs" : "font-medium"}>{value}</span>
    </div>
  );
}
