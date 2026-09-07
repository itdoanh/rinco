"use client";

import Image from "next/image";
import { Button } from "@/components/ui/button";
import { trackClick } from "@/lib/pixel";

interface TrustBadge {
  icon: string;
  title: string;
  description: string;
}

interface TrustSectionProps {
  data: {
    title?: string;
    subtitle?: string;
    top1_banner?: string;
    badges?: TrustBadge[];
    cta_text?: string;
    cta_label?: string;
  };
  onCtaClick?: () => void;
}

export function TrustSection({ data, onCtaClick }: TrustSectionProps) {
  const defaultBadges: TrustBadge[] = [
    {
      icon: "🏆",
      title: "TOP 1 Thị Phần Giao Dịch",
      description:
        "APEX Fintech tự hào xếp hạng TOP 1 về Khối lượng giao dịch và Số lượng hợp đồng tại MXV trong Quý II/2026.",
    },
    {
      icon: "📜",
      title: "Pháp Lý Minh Bạch",
      description:
        "Được MXV và Bộ Công Thương cấp phép hoạt động kinh doanh chính thức, công khai, minh bạch.",
    },
    {
      icon: "💻",
      title: "Hạ Tầng CQG Chuẩn Quốc Tế",
      description:
        "Kết nối phần mềm giao dịch hàng đầu thế giới CQG (Desktop, Mobile, Web) mượt mà, khớp lệnh siêu tốc.",
    },
    {
      icon: "🌍",
      title: "Dữ Liệu Thời Gian Thực",
      description:
        "Kết nối trực tiếp dữ liệu real-time từ CBOT, NYMEX, LME, OSE, TOCOM…",
    },
  ];

  const badges = data.badges?.length ? data.badges : defaultBadges;

  return (
    <section className="bg-white py-16 md:py-24">
      <div className="max-w-7xl mx-auto px-4">
        {/* Header */}
        <div className="text-center mb-12">
          <span className="inline-flex items-center gap-2 bg-orange/10 border-2 border-orange text-orange px-4 py-2 rounded-full text-xs font-bold uppercase tracking-widest mb-5">
            BẢO CHỨNG UY TÍN
          </span>
          <h2 className="font-heading text-3xl md:text-4xl lg:text-5xl font-black text-navy-900 mb-4">
            APEX FINTECH<br />
            <span className="gradient-text">THÀNH VIÊN KINH DOANH SỐ 080 (MXV)</span>
          </h2>
        </div>

        {/* TOP 1 Banner */}
        {data.top1_banner && (
          <div className="top1-banner mb-10">
            <Image
              src={data.top1_banner}
              alt="APEX TOP 1 Thị phần Q2/2026"
              width={1200}
              height={400}
              className="w-full h-auto rounded-2xl shadow-2xl"
            />
          </div>
        )}

        {/* Trust Badges Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {badges.map((badge, idx) => (
            <div key={idx} className="trust-badge">
              <div className="trust-icon">
                <span className="text-3xl">{badge.icon}</span>
              </div>
              <h3 className="font-heading font-bold text-navy-900 text-lg mb-3">
                {badge.title}
              </h3>
              <p className="text-gray-600 text-sm leading-relaxed">
                {badge.description}
              </p>
            </div>
          ))}
        </div>

        {/* CTA */}
        {data.cta_text && (
          <div className="text-center mt-10">
            <Button
              variant="cta"
              size="xl"
              onClick={() => {
                trackClick(data.cta_label || "section-trust", "trust", "bottom");
                onCtaClick?.();
              }}
            >
              👉 {data.cta_text}
            </Button>
          </div>
        )}
      </div>
    </section>
  );
}

export default TrustSection;
