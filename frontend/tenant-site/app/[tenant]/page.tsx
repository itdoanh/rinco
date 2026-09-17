import { notFound } from "next/navigation";
import { tenantApi } from "@/lib/api";
import { PageRenderer } from "@/components/renderer/PageRenderer";
import { getMockPage, getMockTenant } from "@/lib/mock-data";
import type { Tenant, Page } from "@/lib/schema";

interface PageProps {
  params: Promise<{ tenant: string }>;
}

export default async function TenantPage({ params }: PageProps) {
  const { tenant: tenantSlug } = await params;

  let tenant: Tenant | undefined;
  let page: Page | undefined;

  try {
    const [tenantResult, pageResult] = await Promise.all([
      tenantApi.getTenant(tenantSlug),
      tenantApi.getPage(tenantSlug, "home"),
    ]);
    tenant = tenantResult;
    page = pageResult ?? undefined;
  } catch {
    tenant = getMockTenant(tenantSlug);
    page = getMockPage(tenantSlug, "home");
  }

  if (!tenant || !page) {
    notFound();
  }

  return <PageRenderer page={page} tenant={tenant} />;
}

export async function generateMetadata({ params }: PageProps) {
  const { tenant: tenantSlug } = await params;
  const tenant = getMockTenant(tenantSlug);
  const home = getMockPage(tenantSlug, "home");
  return {
    title: home?.title ?? `${tenant?.name ?? tenantSlug} | RINCO`,
    description: home?.description ?? `Welcome to ${tenant?.name ?? tenantSlug}`,
  };
}
