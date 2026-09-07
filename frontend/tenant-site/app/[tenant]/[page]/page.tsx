import { notFound } from "next/navigation";
import { tenantApi } from "@/lib/api";
import { PageRenderer } from "@/components/renderer/PageRenderer";

const defaultTenant = {
  id: "demo",
  slug: "demo",
  name: "Demo Company",
  branding: {
    primary: "#0A192F",
    secondary: "#F5A623",
    accent: "#FF6B00",
  },
};

export default async function TenantSubPage({
  params,
}: {
  params: Promise<{ tenant: string; page: string }>;
}) {
  const { tenant: tenantSlug, page: pageSlug } = await params

  let tenant = defaultTenant
  let page = null

  try {
    const [tenantData, pageData] = await Promise.all([
      tenantApi.getTenant(tenantSlug).catch(() => defaultTenant),
      tenantApi.getPage(tenantSlug, pageSlug).catch(() => null),
    ])

    if (tenantData) tenant = tenantData
    if (pageData) page = pageData
  } catch {
    // fall through to defaults
  }

  if (!page) {
    notFound()
  }

  return <PageRenderer page={page} tenant={tenant} />
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ tenant: string; page: string }>;
}) {
  const { tenant, page } = await params
  return {
    title: `${page} - ${tenant} | RINCO Tenant`,
    description: `Page ${page} of ${tenant}`,
  }
}
