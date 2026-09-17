import Link from "next/link";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ArrowRight, Building2 } from "lucide-react";
import { mockTenants } from "@/lib/mock-data";
import type { Tenant } from "@/lib/schema";

export default function DemoIndexPage() {
  const tenants = Object.values(mockTenants) as Array<
    Tenant & { pages: Array<{ slug: string; title: string }> }
  >;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100">
      {/* Header */}
      <header className="border-b bg-white/80 backdrop-blur-md sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-600 to-purple-600 flex items-center justify-center text-white font-bold">
              R
            </div>
            <span className="font-bold text-lg">RINCO Demo Hub</span>
          </div>
          <Badge variant="outline">Multi-tenant routing demo</Badge>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-12">
        <div className="text-center max-w-3xl mx-auto mb-12">
          <Badge className="mb-3 bg-blue-100 text-blue-700">v1.0.0 · 11/2026</Badge>
          <h1 className="text-4xl md:text-5xl font-bold tracking-tight">
            Multi-Tenant Demo Sites
          </h1>
          <p className="text-lg text-slate-600 mt-4">
            Click bất kỳ tenant nào để xem landing page thực tế với hero, features,
            stats, testimonials, contact form. Tất cả dùng chung codebase — chỉ
            khác dữ liệu qua middleware routing.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {tenants.map((t) => (
            <Link key={t.slug} href={`/${t.slug}`}>
              <Card className="h-full hover:shadow-xl hover:-translate-y-1 transition-all duration-300 cursor-pointer group overflow-hidden">
                <div
                  className="h-24 relative"
                  style={{
                    background: `linear-gradient(135deg, ${t.branding?.primary || "#0A192F"} 0%, ${t.branding?.secondary || "#F5A623"} 100%)`,
                  }}
                >
                  <div className="absolute inset-0 flex items-center justify-center">
                    <Building2 className="w-10 h-10 text-white/30" />
                  </div>
                </div>
                <CardContent className="p-5">
                  <div className="flex items-start justify-between mb-2">
                    <h3 className="font-bold text-lg">{t.name}</h3>
                    <Badge variant="secondary" className="text-xs">
                      {t.pages.length} pages
                    </Badge>
                  </div>
                  <code className="text-xs text-slate-500 block mb-3">/{t.slug}</code>
                  <p className="text-sm text-slate-600 mb-4">
                    {t.pages.map((p) => p.title).join(" · ")}
                  </p>
                  <Button variant="outline" className="w-full group-hover:bg-primary group-hover:text-white transition-colors">
                    Xem demo <ArrowRight className="w-4 h-4 ml-2" />
                  </Button>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>

        <div className="mt-16 text-center">
          <div className="inline-flex items-center gap-4 p-4 bg-white rounded-2xl shadow-sm border">
            <div className="text-sm text-slate-600">
              <strong className="font-semibold text-slate-900">Tip:</strong> Try{" "}
              <code className="px-1.5 py-0.5 rounded bg-slate-100 font-mono text-xs">/apexfintech/about</code> or{" "}
              <code className="px-1.5 py-0.5 rounded bg-slate-100 font-mono text-xs">/megashop/contact</code>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
