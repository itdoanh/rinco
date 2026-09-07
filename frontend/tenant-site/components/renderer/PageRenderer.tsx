"use client";

import type { Block, Page } from "@/lib/schema";
import { Header } from "@/components/branding/Header";
import { Footer } from "@/components/branding/Footer";
import { ContactForm } from "@/components/renderer/ContactForm";
import type { Branding, Tenant } from "@/lib/schema";

interface PageRendererProps {
  page: Page;
  tenant: Tenant;
}

export function PageRenderer({ page, tenant }: PageRendererProps) {
  const branding = tenant.branding as Branding | undefined;

  return (
    <div className="min-h-screen flex flex-col">
      <Header
        logo={branding?.logoUrl || tenant.logo}
        logoAlt={tenant.name}
        companyName={tenant.name}
        branding={branding}
      />

      <main className="flex-1">
        {/* Page content from blocks */}
        {page.blocks.map((block) => (
          <BlockComponent key={block.id} block={block} branding={branding} />
        ))}
      </main>

      <Footer companyName={tenant.name} branding={branding} />
    </div>
  );
}

function BlockComponent({
  block,
  branding,
}: {
  block: Block;
  branding?: Branding;
}) {
  const primary = branding?.primary || "#0A192F";
  const secondary = branding?.secondary || "#F5A623";

  switch (block.type) {
    case "hero":
      return (
        <section className="bg-gradient-to-b from-white to-gray-50 py-20">
          <div className="max-w-7xl mx-auto px-4 text-center">
            <h1 className="text-4xl md:text-5xl font-black mb-6" style={{ color: primary }}>
              {block.data.title as string}
            </h1>
            <p className="text-xl text-gray-600 max-w-2xl mx-auto">
              {block.data.subtitle as string}
            </p>
          </div>
        </section>
      );

    case "features":
      return (
        <section className="py-20 bg-white">
          <div className="max-w-7xl mx-auto px-4">
            <h2 className="text-3xl font-bold text-center mb-12" style={{ color: primary }}>
              {block.data.title as string || "Dịch vụ của chúng tôi"}
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              {((block.data.items as Array<{ icon: string; title: string; description: string }>) || []).map((item, idx) => (
                <div key={idx} className="text-center p-6 rounded-2xl border hover:shadow-lg transition-shadow">
                  <div className="text-4xl mb-4">{item.icon}</div>
                  <h3 className="text-xl font-bold mb-2">{item.title}</h3>
                  <p className="text-gray-600">{item.description}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
      );

    case "contact":
      return (
        <section className="py-20 bg-gray-50" id="contact">
          <div className="max-w-3xl mx-auto px-4">
            <h2 className="text-3xl font-bold text-center mb-8" style={{ color: primary }}>
              Liên hệ với chúng tôi
            </h2>
            <div className="bg-white rounded-2xl p-8 shadow-lg">
              <ContactForm tenantSlug={block.data.tenantSlug as string || ""} branding={branding} />
            </div>
          </div>
        </section>
      );

    case "stats":
      return (
        <section className="py-16" style={{ backgroundColor: primary }}>
          <div className="max-w-7xl mx-auto px-4">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-8 text-white text-center">
              {((block.data.stats as Array<{ value: string; label: string }>) || []).map((stat, idx) => (
                <div key={idx}>
                  <div className="text-4xl font-black">{stat.value}</div>
                  <div className="text-white/70 mt-2">{stat.label}</div>
                </div>
              ))}
            </div>
          </div>
        </section>
      );

    default:
      return null;
  }
}

export default PageRenderer;
