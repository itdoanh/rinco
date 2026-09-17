import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { tenantApi } from "@/lib/api";
import { PageRenderer } from "@/components/renderer/PageRenderer";
import { getMockPage, getMockTenant } from "@/lib/mock-data";
import type { Tenant, Page } from "@/lib/schema";

interface PageProps {
  params: Promise<{ tenant: string; page: string }>;
}

async function loadSubpage(
  tenantSlug: string,
  pageSlug: string,
): Promise<{ tenant: Tenant | null; page: Page | null }> {
  const [tenant, page] = await Promise.all([
    tenantApi.getTenant(tenantSlug),
    tenantApi.getPage(tenantSlug, pageSlug),
  ]);
  return {
    tenant: tenant ?? getMockTenant(tenantSlug) ?? null,
    page: page ?? getMockPage(tenantSlug, pageSlug) ?? null,
  };
}

export default async function TenantSubPage({ params }: PageProps) {
  const { tenant: tenantSlug, page: pageSlug } = await params;
  const { tenant, page } = await loadSubpage(tenantSlug, pageSlug);

  if (!tenant || !page) notFound();

  return <PageRenderer page={page} tenant={tenant} />;
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { tenant: tenantSlug, page: pageSlug } = await params;

  let tenant = await tenantApi.getTenant(tenantSlug).catch(() => null);
  let page = await tenantApi.getPage(tenantSlug, pageSlug).catch(() => null);
  if (!tenant) tenant = getMockTenant(tenantSlug) ?? null;
  if (!page) page = getMockPage(tenantSlug, pageSlug) ?? null;

  const seo = page?.seo;
  const tenantName = tenant?.name ?? tenantSlug;
  return {
    title: seo?.title ?? page?.title ?? `${pageSlug} - ${tenantName} | RINCO`,
    description: seo?.description ?? page?.description ?? `${pageSlug} of ${tenantName}`,
    openGraph: {
      title: seo?.title ?? page?.title ?? `${pageSlug} - ${tenantName}`,
      description: seo?.description ?? page?.description ?? undefined,
      images: seo?.image ? [{ url: seo.image }] : undefined,
      type: "website",
    },
    alternates: {
      canonical: `/${tenantSlug}/${pageSlug}`,
    },
  };
}
