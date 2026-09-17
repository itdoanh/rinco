import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { tenantApi } from "@/lib/api";
import { PageRenderer } from "@/components/renderer/PageRenderer";
import { getMockPage, getMockTenant } from "@/lib/mock-data";
import type { Tenant, Page } from "@/lib/schema";

interface PageProps {
  params: Promise<{ tenant: string }>;
}

async function loadHome(tenantSlug: string): Promise<{ tenant: Tenant | null; page: Page | null }> {
  const [tenant, page] = await Promise.all([
    tenantApi.getTenant(tenantSlug),
    tenantApi.getPage(tenantSlug, "home"),
  ]);
  return {
    tenant: tenant ?? getMockTenant(tenantSlug) ?? null,
    page: page ?? getMockPage(tenantSlug, "home") ?? null,
  };
}

export default async function TenantPage({ params }: PageProps) {
  const { tenant: tenantSlug } = await params;
  const { tenant, page } = await loadHome(tenantSlug);

  if (!tenant || !page) notFound();

  return <PageRenderer page={page} tenant={tenant} />;
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { tenant: tenantSlug } = await params;

  // Use real API for SEO metadata, fallback to mock on failure
  let tenant = await tenantApi.getTenant(tenantSlug).catch(() => null);
  let page = await tenantApi.getPage(tenantSlug, "home").catch(() => null);
  if (!tenant) tenant = getMockTenant(tenantSlug) ?? null;
  if (!page) page = getMockPage(tenantSlug, "home") ?? null;

  const seo = page?.seo;
  const tenantName = tenant?.name ?? tenantSlug;
  return {
    title: seo?.title ?? page?.title ?? `${tenantName} | RINCO`,
    description: seo?.description ?? page?.description ?? `Welcome to ${tenantName}`,
    openGraph: {
      title: seo?.title ?? page?.title ?? tenantName,
      description: seo?.description ?? page?.description ?? undefined,
      images: seo?.image ? [{ url: seo.image }] : undefined,
      type: "website",
    },
    twitter: {
      card: "summary_large_image",
      title: seo?.title ?? page?.title ?? tenantName,
      description: seo?.description ?? page?.description ?? undefined,
      images: seo?.image ? [seo.image] : undefined,
    },
    alternates: {
      canonical: `/${tenantSlug}`,
    },
  };
}
