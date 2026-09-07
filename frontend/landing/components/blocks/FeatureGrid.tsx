"use client";

import { Button } from "@/components/ui/button";
import { trackClick } from "@/lib/pixel";

interface FeatureCardProps {
  number: string;
  icon: string;
  title: string;
  subtitle: string;
  value: string;
  ctaLabel?: string;
  onCtaClick?: () => void;
}

function FeatureCard({
  number,
  icon,
  title,
  subtitle,
  value,
  ctaLabel,
  onCtaClick,
}: FeatureCardProps) {
  return (
    <div className="feature-card">
      <div className="feature-card__num">{number}</div>
      <div className="feature-card__icon">{icon}</div>
      <h3 className="feature-card__title">{title}</h3>
      <p className="feature-card__sub">{subtitle}</p>
      <div className="feature-card__divider" />
      <p className="feature-card__value">{value}</p>
      {ctaLabel && (
        <Button
          variant="cta"
          size="sm"
          className="mt-4 w-full"
          onClick={() => {
            trackClick(ctaLabel, "features", title);
            onCtaClick?.();
          }}
        >
          ĐĂNG KÝ NGAY
        </Button>
      )}
    </div>
  );
}

interface FeatureGridProps {
  data: {
    title?: string;
    subtitle?: string;
    description?: string;
    features?: Array<{
      icon: string;
      title: string;
      subtitle: string;
      value: string;
    }>;
    cta_text?: string;
    cta_label?: string;
    style?: string;
  };
  onCtaClick?: () => void;
}

export function FeatureGrid({ data, onCtaClick }: FeatureGridProps) {
  const defaultFeatures = [
    {
      icon: "⚡",
      title: "Thanh toán T+0 Linh hoạt",
      subtitle: "Cho phép đóng/mở vị thế giao dịch tức thì ngay trong phiên. Tiền bán cộng ngay vào tài khoản.",
      value: "Vòng xoay vốn liên tục, không bị ngâm vốn, chủ động chốt lời hoặc quản trị rủi ro lập tức.",
    },
    {
      icon: "⇅",
      title: "Cơ chế Giao dịch 2 Chiều",
      subtitle: "Cung cấp cả lệnh Mua (Long - kỳ vọng giá tăng) và lệnh Bán (Short - kỳ vọng giá giảm).",
      value: "Tìm kiếm cơ hội sinh lời linh hoạt ngay cả khi thị trường đi xuống.",
    },
    {
      icon: "💰",
      title: "Tỷ lệ Ký quỹ Margin Tối ưu",
      subtitle: "Chỉ cần ký quỹ một tỷ lệ phần trăm nhỏ theo quy định MXV để giao dịch hợp đồng tiêu chuẩn.",
      value: "Tối ưu hóa quy mô sử dụng vốn, gia tăng hiệu suất đầu tư mà không cần 100% tiền mặt.",
    },
    {
      icon: "🌐",
      title: "Minh bạch & Liên thông Quốc tế",
      subtitle: "Khớp lệnh Real-time trực tiếp tới các Sở giao dịch hàng hóa lớn nhất thế giới (CBOT, NYMEX, LME…).",
      value: "Giá vận hành chuẩn xác theo Cung - Cầu toàn cầu, chống thao túng giá từ cá nhân/tổ chức.",
    },
  ];

  const features = data.features?.length ? data.features : defaultFeatures;

  return (
    <section className="bg-gray-50 py-16 md:py-24">
      <div className="max-w-7xl mx-auto px-4">
        {/* Header */}
        <div className="text-center mb-12">
          <span className="inline-flex items-center gap-2 bg-orange/10 border-2 border-orange text-orange px-4 py-2 rounded-full text-xs font-bold uppercase tracking-widest mb-5">
            BỐI CẢNH THỊ TRƯỜNG 2026
          </span>
          <h2 className="font-heading text-3xl md:text-4xl lg:text-5xl font-black text-navy-900 mb-4">
            TẠI SAO HÀNG HÓA PHÁI SINH<br />
            <span className="gradient-text">ĐANG LÀ KÊNH ĐẦU TƯ BÙNG NỔ?</span>
          </h2>
          <p className="text-gray-600 text-base md:text-lg max-w-3xl mx-auto">
            <strong>Theo dữ liệu chính thức từ MXV:</strong> Thanh khoản bình quân
            thị trường đạt <strong className="text-orange">7.500 tỷ – 17.000 tỷ đồng/ngày</strong>,
            khẳng định sự dịch chuyển dòng tiền mạnh mẽ của các nhà đầu tư thông minh trong năm 2026.
          </p>
        </div>

        {/* Feature Cards Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5">
          {features.map((feature, idx) => (
            <FeatureCard
              key={idx}
              number={`0${idx + 1}`}
              icon={feature.icon}
              title={feature.title}
              subtitle={feature.subtitle}
              value={feature.value}
              ctaLabel={data.cta_label}
              onCtaClick={onCtaClick}
            />
          ))}
        </div>

        {/* CTA */}
        {data.cta_text && (
          <div className="text-center mt-10">
            <Button
              variant="cta"
              size="xl"
              onClick={() => {
                trackClick(data.cta_label || "section-features", "features", "bottom");
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

export default FeatureGrid;
