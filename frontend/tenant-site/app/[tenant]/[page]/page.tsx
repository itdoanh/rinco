import { notFound } from "next/navigation";
import { tenantApi } from "@/lib/api";
import { PageRenderer } from "@/components/renderer/PageRenderer";
import { getMockPage, getMockTenant } from "@/lib/mock-data";
import type { Tenant, Page } from "@/lib/schema";

interface PageProps {
  params: Promise<{ tenant: string; page: string }>;
}

export default async function TenantSubPage({ params }: PageProps) {
  const { tenant: tenantSlug, page: pageSlug } = await params;

  let tenant: Tenant | undefined;
  let page: Page | undefined;

  try {
    const [tenantResult, pageResult] = await Promise.all([
      tenantApi.getTenant(tenantSlug),
      tenantApi.getPage(tenantSlug, pageSlug),
    ]);
    tenant = tenantResult;
    page = pageResult ?? undefined;
  } catch {
    tenant = getMockTenant(tenantSlug);
    page = getMockPage(tenantSlug, pageSlug);
  }

  if (!tenant || !page) {
    notFound();
  }

  return <PageRenderer page={page} tenant={tenant} />;
}

export async function generateMetadata({ params }: PageProps) {
  const { tenant: tenantSlug, page: pageSlug } = await params;
  const tenant = getMockTenant(tenantSlug);
  const page = getMockPage(tenantSlug, pageSlug);
  return {
    title: page?.title ?? `${pageSlug} - ${tenant?.name ?? tenantSlug} | RINCO`,
    description: page?.description ?? `${pageSlug} of ${tenant?.name ?? tenantSlug}`,
  };
}
