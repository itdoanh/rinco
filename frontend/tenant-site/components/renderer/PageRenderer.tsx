"use client";

import type { Block, Page, Branding, Tenant } from "@/lib/schema";
import { Header } from "@/components/branding/Header";
import { Footer } from "@/components/branding/Footer";
import { ContactForm } from "@/components/renderer/ContactForm";
import { ChevronRight, Mail, Phone, MapPin } from "lucide-react";

interface PageRendererProps {
  page: Page;
  tenant: Tenant;
}

export function PageRenderer({ page, tenant }: PageRendererProps) {
  const branding = tenant.branding as Branding | undefined;

  return (
    <div className="min-h-screen flex flex-col bg-white">
      <Header
        logo={branding?.logoUrl || tenant.logo}
        logoAlt={tenant.name}
        companyName={tenant.name}
        branding={branding}
      />

      <main className="flex-1">
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
  const accent = branding?.accent || "#FF6B00";

  switch (block.type) {
    case "hero":
      return (
        <section
          className="relative overflow-hidden py-20 md:py-28"
          style={{
            background: `linear-gradient(135deg, ${primary} 0%, ${primary}cc 50%, ${secondary}33 100%)`,
          }}
        >
          <div className="absolute inset-0 opacity-10" aria-hidden="true">
            <div className="absolute -top-32 -right-32 w-96 h-96 rounded-full" style={{ background: secondary }} />
            <div className="absolute -bottom-32 -left-32 w-96 h-96 rounded-full" style={{ background: accent }} />
          </div>
          <div className="relative max-w-7xl mx-auto px-4 text-center">
            <h1 className="text-4xl md:text-6xl font-black mb-6 text-white leading-tight">
              {(block.data.title as string) ?? ""}
            </h1>
            <p className="text-lg md:text-2xl text-white/85 max-w-3xl mx-auto mb-8">
              {(block.data.subtitle as string) ?? ""}
            </p>
            {Boolean(block.data.ctaText) && (
              <a
                href="#contact"
                className="inline-flex items-center gap-2 px-8 py-3 rounded-full font-semibold text-white shadow-lg hover:shadow-xl transition-all hover:-translate-y-0.5"
                style={{ backgroundColor: accent }}
              >
                {String(block.data.ctaText ?? "")}
                <ChevronRight className="w-4 h-4" />
              </a>
            )}
          </div>
        </section>
      );

    case "features": {
      const items = ((block.data.items as Array<{
        icon?: string;
        title: string;
        description: string;
      }>) || []);
      return (
        <section className="py-20 bg-white">
          <div className="max-w-7xl mx-auto px-4">
            <h2
              className="text-3xl md:text-4xl font-bold text-center mb-4"
              style={{ color: primary }}
            >
              {(block.data.title as string) ?? "Dịch vụ của chúng tôi"}
            </h2>
            <div className="w-20 h-1 mx-auto mb-12 rounded-full" style={{ backgroundColor: accent }} />
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
              {items.map((item, idx) => (
                <div
                  key={idx}
                  className="group text-center p-6 rounded-2xl border border-slate-100 hover:border-transparent hover:shadow-2xl transition-all hover:-translate-y-1"
                  style={{ ["--hover-border" as string]: primary }}
                >
                  <div
                    className="w-16 h-16 rounded-2xl flex items-center justify-center text-3xl mx-auto mb-4 group-hover:scale-110 transition-transform"
                    style={{ backgroundColor: `${primary}15`, color: primary }}
                  >
                    {item.icon || "✨"}
                  </div>
                  <h3 className="text-xl font-bold mb-2">{item.title}</h3>
                  <p className="text-slate-600 leading-relaxed">{item.description}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
      );
    }

    case "contact":
      return (
        <section className="py-20 bg-slate-50" id="contact">
          <div className="max-w-5xl mx-auto px-4">
            <div className="text-center mb-12">
              <h2
                className="text-3xl md:text-4xl font-bold mb-4"
                style={{ color: primary }}
              >
                Liên hệ với chúng tôi
              </h2>
              <p className="text-slate-600 max-w-xl mx-auto">
                Điền form bên dưới hoặc liên hệ trực tiếp qua hotline/email của chúng tôi.
              </p>
            </div>
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
              <div className="lg:col-span-1 space-y-4">
                <InfoCard
                  icon={<Phone className="w-5 h-5" />}
                  title="Hotline"
                  value="1900-6868"
                  color={primary}
                />
                <InfoCard
                  icon={<Mail className="w-5 h-5" />}
                  title="Email"
                  value={`contact@${block.data.tenantSlug}.com`}
                  color={primary}
                />
                <InfoCard
                  icon={<MapPin className="w-5 h-5" />}
                  title="Địa chỉ"
                  value="Hà Nội & TP. HCM"
                  color={primary}
                />
              </div>
              <div className="lg:col-span-2">
                <div className="bg-white rounded-2xl p-8 shadow-lg">
                  <ContactForm
                    tenantSlug={(block.data.tenantSlug as string) ?? ""}
                    branding={branding}
                  />
                </div>
              </div>
            </div>
          </div>
        </section>
      );

    case "stats":
      return (
        <section
          className="py-16 md:py-20 relative overflow-hidden"
          style={{ backgroundColor: primary }}
        >
          <div className="absolute inset-0 opacity-10" aria-hidden="true">
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] rounded-full border-2 border-white" />
            <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[400px] h-[400px] rounded-full border-2 border-white" />
          </div>
          <div className="relative max-w-7xl mx-auto px-4">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-8 text-white text-center">
              {((block.data.stats as Array<{ value: string; label: string }>) || []).map((stat, idx) => (
                <div key={idx} className="group">
                  <div className="text-4xl md:text-5xl font-black mb-2 group-hover:scale-110 transition-transform">
                    {stat.value}
                  </div>
                  <div className="text-white/70 uppercase text-xs tracking-wider">
                    {stat.label}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      );

    case "testimonials": {
      const items = ((block.data.items as Array<{
        name: string;
        role: string;
        quote: string;
      }>) || []);
      return (
        <section className="py-20 bg-gradient-to-b from-white to-slate-50">
          <div className="max-w-7xl mx-auto px-4">
            <h2
              className="text-3xl md:text-4xl font-bold text-center mb-4"
              style={{ color: primary }}
            >
              {(block.data.title as string) ?? "Khách hàng nói gì"}
            </h2>
            <div className="w-20 h-1 mx-auto mb-12 rounded-full" style={{ backgroundColor: accent }} />
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {items.map((item, idx) => (
                <div
                  key={idx}
                  className="bg-white rounded-2xl p-8 shadow-sm hover:shadow-xl transition-shadow border border-slate-100"
                >
                  <div className="text-5xl mb-4 leading-none" style={{ color: accent }}>
                    "
                  </div>
                  <p className="text-slate-700 leading-relaxed mb-6 italic">
                    {item.quote}
                  </p>
                  <div className="flex items-center gap-3 border-t pt-4">
                    <div
                      className="w-10 h-10 rounded-full flex items-center justify-center text-white font-bold"
                      style={{ backgroundColor: primary }}
                    >
                      {item.name[0]}
                    </div>
                    <div>
                      <div className="font-semibold">{item.name}</div>
                      <div className="text-xs text-slate-500">{item.role}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      );
    }

    case "cta":
      return (
        <section
          className="py-16 md:py-20 text-center"
          style={{ background: `linear-gradient(135deg, ${accent} 0%, ${secondary} 100%)` }}
        >
          <div className="max-w-3xl mx-auto px-4">
            <h2 className="text-3xl md:text-4xl font-black text-white mb-4">
              {(block.data.title as string) ?? ""}
            </h2>
            <p className="text-white/85 text-lg mb-8">
              {(block.data.subtitle as string) ?? ""}
            </p>
            {Boolean(block.data.ctaText) && (
              <a
                href="#contact"
                className="inline-flex items-center gap-2 px-8 py-3 rounded-full bg-white font-semibold shadow-lg hover:shadow-xl transition-all hover:-translate-y-0.5"
                style={{ color: primary }}
              >
                {String(block.data.ctaText ?? "")}
                <ChevronRight className="w-4 h-4" />
              </a>
            )}
          </div>
        </section>
      );

    case "team": {
      const members = ((block.data.members as Array<{
        name: string;
        role: string;
      }>) || []);
      return (
        <section className="py-20 bg-white">
          <div className="max-w-7xl mx-auto px-4">
            <h2
              className="text-3xl md:text-4xl font-bold text-center mb-12"
              style={{ color: primary }}
            >
              {(block.data.title as string) ?? "Đội ngũ"}
            </h2>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
              {members.map((m, idx) => (
                <div key={idx} className="text-center group">
                  <div
                    className="w-24 h-24 mx-auto mb-4 rounded-full flex items-center justify-center text-3xl font-bold text-white group-hover:scale-110 transition-transform"
                    style={{ backgroundColor: primary }}
                  >
                    {m.name[0]}
                  </div>
                  <h3 className="font-bold text-base">{m.name}</h3>
                  <p className="text-sm text-slate-500">{m.role}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
      );
    }

    case "pricing": {
      const plans = ((block.data.plans as Array<{
        name: string;
        price: string;
        features: string[];
        highlighted?: boolean;
      }>) || []);
      return (
        <section className="py-20 bg-slate-50">
          <div className="max-w-7xl mx-auto px-4">
            <h2
              className="text-3xl md:text-4xl font-bold text-center mb-12"
              style={{ color: primary }}
            >
              {(block.data.title as string) ?? "Bảng giá"}
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
              {plans.map((plan, idx) => (
                <div
                  key={idx}
                  className={`rounded-2xl p-8 transition-all hover:-translate-y-1 ${
                    plan.highlighted
                      ? "bg-white shadow-2xl border-2 relative"
                      : "bg-white shadow-sm border border-slate-100"
                  }`}
                  style={plan.highlighted ? { borderColor: accent } : {}}
                >
                  {plan.highlighted && (
                    <div
                      className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 rounded-full text-xs font-bold text-white"
                      style={{ backgroundColor: accent }}
                    >
                      PHỔ BIẾN NHẤT
                    </div>
                  )}
                  <h3 className="text-xl font-bold mb-2">{plan.name}</h3>
                  <div className="text-3xl font-black mb-6" style={{ color: primary }}>
                    {plan.price}
                  </div>
                  <ul className="space-y-3 mb-8">
                    {plan.features.map((f, i) => (
                      <li key={i} className="flex items-start gap-2 text-sm">
                        <span className="text-emerald-500 mt-0.5">✓</span>
                        <span>{f}</span>
                      </li>
                    ))}
                  </ul>
                  <a
                    href="#contact"
                    className={`block text-center py-3 rounded-xl font-semibold transition-all ${
                      plan.highlighted
                        ? "text-white shadow-md hover:shadow-lg"
                        : "border-2 hover:bg-slate-50"
                    }`}
                    style={plan.highlighted ? { backgroundColor: primary } : { borderColor: primary, color: primary }}
                  >
                    Chọn gói
                  </a>
                </div>
              ))}
            </div>
          </div>
        </section>
      );
    }

    case "faq": {
      const items = ((block.data.items as Array<{ q: string; a: string }>) || []);
      return (
        <section className="py-20 bg-white">
          <div className="max-w-3xl mx-auto px-4">
            <h2
              className="text-3xl md:text-4xl font-bold text-center mb-12"
              style={{ color: primary }}
            >
              {(block.data.title as string) ?? "FAQ"}
            </h2>
            <div className="space-y-4">
              {items.map((item, idx) => (
                <details
                  key={idx}
                  className="group bg-slate-50 rounded-xl border border-slate-100 overflow-hidden"
                >
                  <summary className="cursor-pointer list-none p-5 flex items-center justify-between font-semibold">
                    <span>{item.q}</span>
                    <span
                      className="ml-4 transition-transform group-open:rotate-45 text-xl"
                      style={{ color: accent }}
                    >
                      +
                    </span>
                  </summary>
                  <div className="px-5 pb-5 text-slate-600 leading-relaxed">
                    {item.a}
                  </div>
                </details>
              ))}
            </div>
          </div>
        </section>
      );
    }

    default:
      return null;
  }
}

function InfoCard({
  icon,
  title,
  value,
  color,
}: {
  icon: React.ReactNode;
  title: string;
  value: string;
  color: string;
}) {
  return (
    <div className="flex items-center gap-4 bg-white rounded-xl p-4 shadow-sm border border-slate-100">
      <div
        className="w-10 h-10 rounded-lg flex items-center justify-center text-white shrink-0"
        style={{ backgroundColor: color }}
      >
        {icon}
      </div>
      <div>
        <div className="text-xs text-slate-500 uppercase tracking-wider">{title}</div>
        <div className="font-semibold">{value}</div>
      </div>
    </div>
  );
}

export default PageRenderer;
