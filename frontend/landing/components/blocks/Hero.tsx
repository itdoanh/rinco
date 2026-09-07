"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { trackPageView } from "@/lib/tracking";
import { trackClick } from "@/lib/pixel";

interface HeroProps {
  data: {
    title?: string;
    subtitle?: string;
    description?: string;
    cta_text?: string;
    cta_label?: string;
    image?: string;
    badges?: Array<{ icon: string; text: string; subtext?: string }>;
    event_time?: string;
    event_date?: string;
    urgency_text?: string;
  };
  tenantSlug?: string;
  pageSlug?: string;
  onCtaClick?: () => void;
}

export function Hero({ data, onCtaClick }: HeroProps) {
  const [seatsLeft, setSeatsLeft] = useState(50);
  const heroRef = useRef<HTMLElement>(null);

  useEffect(() => {
    // Animate entrance
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add("animate-fade-in");
          }
        });
      },
      { threshold: 0.1 }
    );

    if (heroRef.current) {
      observer.observe(heroRef.current);
    }

    return () => observer.disconnect();
  }, []);

  return (
    <section
      ref={heroRef}
      className="relative overflow-hidden bg-gradient-to-b from-white to-gray-50 pt-10 pb-16 md:pt-16 md:pb-24"
    >
      {/* Background shapes */}
      <div className="float-shape float-shape-1" />
      <div className="float-shape float-shape-2" />
      <div className="float-shape float-shape-3" />

      <div className="relative z-10 max-w-7xl mx-auto px-4">
        <div className="flex flex-col lg:flex-row items-center gap-10 lg:gap-12">
          {/* LEFT: Headline / Content */}
          <div className="flex-1 w-full">
            {/* Webinar Tag */}
            <div className="webinar-tag inline-flex items-center gap-2 px-4 py-2 rounded-full text-white text-xs md:text-sm font-bold uppercase tracking-wide mb-5">
              <span className="webinar-tag-dot" />
              CHIA SẺ TRỰC TUYẾN MIỄN PHÍ QUA ZOOM ONLINE
            </div>

            {/* Main Headline */}
            <h1 className="font-heading text-3xl sm:text-4xl md:text-5xl lg:text-5xl font-black leading-tight mb-4">
              <span className="block text-navy-900">{data.title || "Tối Ưu Dòng Tiền 2026 Với"}</span>
              <span className="block gradient-text">
                {data.subtitle || "Kênh Đầu Tư Hàng Hóa Phái Sinh"}
              </span>
              <span className="block text-base md:text-lg lg:text-xl text-gray-500 font-bold mt-3">
                (Chính Thống - Quản Lý Bởi Bộ Công Thương & MXV)
              </span>
            </h1>

            {/* Sub-headline */}
            <p className="text-gray-600 text-base md:text-lg leading-relaxed mb-5 max-w-xl">
              {data.description ||
                "Giải mã cơ chế giao dịch T+0 & sinh lời 2 chiều linh hoạt. Giữ vé tham dự Webinar độc quyền cùng Chuyên gia và nhận ngay Bộ 10 Ebook Thực Chiến Đầu Tư hôm nay!"}
            </p>

            {/* Urgency */}
            <div className="flex items-center gap-2 text-red-600 text-sm font-bold mb-6 bg-red-50 border border-red-200 rounded-full px-4 py-2 inline-flex">
              <svg
                className="w-4 h-4 animate-pulse flex-shrink-0"
                fill="currentColor"
                viewBox="0 0 24 24"
              >
                <path d="M13 2L4 14h7l-2 8 9-12h-7l2-8z" />
              </svg>
              <span>
                ⚡ Số lượng <strong>vé đăng ký</strong> có hạn, hãy đăng ký trong thời gian sớm nhất!
              </span>
            </div>

            {/* CTA Button */}
            <Button
              variant="cta"
              size="xl"
              onClick={() => {
                trackClick(data.cta_label || "hero-cta", "hero", "top");
                onCtaClick?.();
              }}
              className="mb-4"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                viewBox="0 0 24 24"
              >
                <path d="M5 13l4 4L19 7" />
              </svg>
              {data.cta_text || "ĐĂNG KÝ NGAY"}
            </Button>
          </div>

          {/* RIGHT: Hero image */}
          <div className="hidden lg:block flex-shrink-0 w-full max-w-md">
            <div className="hero-image-wrap">
              <div className="relative">
                <div className="absolute inset-0 bg-gradient-to-br from-orange/20 to-amber/20 rounded-3xl blur-3xl" />
                <img
                  src={data.image || "/anh/anhchandung_chuyengia_nguyentuananh.webp"}
                  alt="Chuyên gia APEX Fintech"
                  className="relative w-full rounded-3xl shadow-2xl"
                  style={{ maxHeight: "600px", objectFit: "cover" }}
                />
                {/* Badges */}
                {data.badges?.map((badge, idx) => (
                  <div
                    key={idx}
                    className={`hero-chip hero-chip-${idx + 1}`}
                  >
                    <span className="text-2xl">{badge.icon}</span>
                    <div>
                      {badge.text}
                      {badge.subtext && (
                        <span className="text-orange">{badge.subtext}</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

export default Hero;
