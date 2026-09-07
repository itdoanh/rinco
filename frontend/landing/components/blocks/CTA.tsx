"use client";

import { Button } from "@/components/ui/button";
import { trackClick } from "@/lib/pixel";

interface CTAProps {
  data: {
    title?: string;
    subtitle?: string;
    cta_text?: string;
    cta_label?: string;
    benefits?: string[];
    style?: string;
  };
  onCtaClick?: () => void;
}

export function CTA({ data, onCtaClick }: CTAProps) {
  const defaultBenefits = [
    "Miễn phí 100% – Không phát sinh chi phí",
    "Tặng Bộ 10 Ebook Thực Chiến Đầu Tư",
    "Tư vấn 1:1 bởi chuyên gia có chứng chỉ MXV",
  ];

  const benefits = data.benefits?.length ? data.benefits : defaultBenefits;

  return (
    <section className="bg-gradient-to-br from-gray-50 via-white to-orange-50 py-16 md:py-24">
      <div className="max-w-5xl mx-auto px-4">
        <div className="bg-white rounded-3xl shadow-2xl overflow-hidden border border-gray-100">
          <div className="grid grid-cols-1 lg:grid-cols-2">
            {/* LEFT: Info */}
            <div className="bg-gradient-to-br from-navy-900 to-navy-700 p-8 md:p-12 text-white relative overflow-hidden">
              <div className="absolute inset-0 grid-pattern opacity-50" />
              <div className="relative z-10">
                <div className="webinar-tag inline-flex items-center gap-2 px-4 py-2 rounded-full text-white text-xs font-bold uppercase tracking-wide mb-6">
                  <span className="webinar-tag-dot" />
                  BUỔI CHIA SẺ ĐỘC QUYỀN
                </div>

                <h2 className="font-heading text-2xl md:text-3xl lg:text-4xl font-black mb-4">
                  {data.title || "GIỮ SUẤT THAM DỰ BUỔI CHIA SẺ ZOOM"}
                  <br />
                  <span className="gradient-text">& NHẬN BỘ TÀI LIỆU</span>
                </h2>

                <p className="text-red-400 font-bold text-sm md:text-base mb-8">
                  ⚡ SỐ LƯỢNG VÉ MIỄN PHÍ <strong className="text-xl">GIỚI HẠN!</strong>
                </p>

                {/* Benefits */}
                <ul className="space-y-3">
                  {benefits.map((benefit, idx) => (
                    <li
                      key={idx}
                      className="flex items-center gap-3 text-white/90 font-semibold text-sm"
                    >
                      <span className="check-icon">✓</span>
                      {benefit}
                    </li>
                  ))}
                </ul>
              </div>
            </div>

            {/* RIGHT: CTA Button */}
            <div className="p-8 md:p-12 flex flex-col items-center justify-center">
              <Button
                variant="cta"
                size="xl"
                onClick={() => {
                  trackClick(data.cta_label || "section-cta", "cta", "bottom");
                  onCtaClick?.();
                }}
                className="w-full max-w-sm"
              >
                🚀 {data.cta_text || "ĐĂNG KÝ NGAY"}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

export default CTA;
