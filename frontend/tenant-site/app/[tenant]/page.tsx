import { notFound } from "next/navigation";
import { tenantApi } from "@/lib/api";
import { PageRenderer } from "@/components/renderer/PageRenderer";

// Default tenant data for demo
const defaultTenant = {
  id: "demo",
  slug: "demo",
  name: "Demo Company",
  logo: undefined,
  branding: {
    primary: "#0A192F",
    secondary: "#F5A623",
    accent: "#FF6B00",
  },
};

// Default page content
const defaultPage = {
  id: "home",
  slug: "home",
  title: "Chào mừng đến với Demo Company",
  description: "Nền tảng giải pháp tốt nhất cho doanh nghiệp của bạn",
  blocks: [
    {
      id: "hero",
      type: "hero",
      data: {
        title: "Giải pháp số toàn diện",
        subtitle: "Chúng tôi giúp doanh nghiệp của bạn phát triển bền vững với công nghệ hiện đại",
      },
    },
    {
      id: "features",
      type: "features",
      data: {
        title: "Tại sao chọn chúng tôi",
        items: [
          { icon: "🚀", title: "Nhanh chóng", description: "Triển khai trong thời gian ngắn nhất" },
          { icon: "💎", title: "Chất lượng", description: "Sản phẩm đạt chuẩn quốc tế" },
          { icon: "🤝", title: "Hỗ trợ 24/7", description: "Đội ngũ chuyên nghiệp luôn sẵn sàng" },
        ],
      },
    },
    {
      id: "stats",
      type: "stats",
      data: {
        stats: [
          { value: "500+", label: "Khách hàng" },
          { value: "98%", label: "Hài lòng" },
          { value: "24/7", label: "Hỗ trợ" },
          { value: "10+", label: "Năm kinh nghiệm" },
        ],
      },
    },
    {
      id: "contact",
      type: "contact",
      data: {
        tenantSlug: "demo",
      },
    },
  ],
};

export default async function TenantPage({
  params,
}: {
  params: Promise<{ tenant: string }>;
}) {
  const { tenant: tenantSlug } = await params;

  // Try to fetch real data, fall back to defaults
  let tenant = defaultTenant;
  let page = defaultPage;

  try {
    const [tenantData, pageData] = await Promise.all([
      tenantApi.getTenant(tenantSlug).catch(() => defaultTenant),
      tenantApi.getPage(tenantSlug, "home").catch(() => defaultPage),
    ]);

    if (tenantData) tenant = tenantData;
    if (pageData) page = pageData;
  } catch {
    // Use defaults
  }

  return <PageRenderer page={page} tenant={tenant} />;
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ tenant: string }>;
}) {
  const { tenant: tenantSlug } = await params;
  
  return {
    title: `${tenantSlug} - Tenant Site`,
    description: "Dynamic tenant website",
  };
}
