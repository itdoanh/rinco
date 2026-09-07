"use client";

import Image from "next/image";
import { Button } from "@/components/ui/button";
import { trackClick } from "@/lib/pixel";

interface Speaker {
  name: string;
  title: string;
  organization: string;
  image: string;
  bio: string[];
  badge?: string;
}

interface SpeakerSectionProps {
  data: {
    speakers?: Speaker[];
    cta_text?: string;
    cta_label?: string;
  };
  onCtaClick?: () => void;
}

export function SpeakerSection({ data, onCtaClick }: SpeakerSectionProps) {
  const defaultSpeakers: Speaker[] = [
    {
      name: "NGUYỄN TUẤN ANH",
      title: "Giám đốc Phát triển Thị trường",
      organization: "APEX Fintech",
      image: "/anh/anhchandung_chuyengia_nguyentuananh.webp",
      badge: "🏆 Diễn giả chính",
      bio: [
        "Nhiều năm kinh nghiệm phân tích tài chính phái sinh và tư vấn chiến lược phân bổ vốn cho nhà đầu tư cá nhân.",
        "Được đào tạo và cấp Chứng chỉ chính thức bởi Sở Giao dịch Hàng hóa Việt Nam (MXV).",
        "Chuyên gia cấu trúc giải pháp phòng ngừa rủi ro giá cho các doanh nghiệp xuất nhập khẩu hàng hóa lớn.",
        "Đồng hành tư vấn 1:1, hỗ trợ kỹ thuật và chiến lược cho nhà đầu tư trong và sau chương trình.",
      ],
    },
  ];

  const speakers = data.speakers?.length ? data.speakers : defaultSpeakers;

  return (
    <section className="relative bg-gradient-to-br from-navy-900 to-navy-800 py-16 md:py-24 overflow-hidden">
      {/* Background pattern */}
      <div className="absolute inset-0 grid-pattern" />
      <div className="absolute w-96 h-96 bg-orange/10 rounded-full blur-3xl -bottom-48 -left-48" />

      <div className="relative z-10 max-w-7xl mx-auto px-4">
        {/* Header */}
        <div className="text-center mb-12">
          <span className="inline-flex items-center gap-2 bg-orange/20 border-2 border-gold text-gold px-4 py-2 rounded-full text-xs font-bold uppercase tracking-widest mb-5">
            DIỄN GIẢ CHÍNH
          </span>
          <h2 className="font-heading text-3xl md:text-4xl lg:text-5xl font-black text-white mb-3">
            NGƯỜI ĐỒNG HÀNH CỦA BẠN<br />
            <span className="gradient-text">TẠI BUỔI CHIA SẺ</span>
          </h2>
        </div>

        {/* Speaker Cards */}
        <div className="space-y-8 max-w-4xl mx-auto">
          {speakers.map((speaker, idx) => (
            <div key={idx} className="speaker-card rounded-3xl p-6 md:p-10">
              <div className="flex flex-col md:flex-row gap-8 md:gap-12 items-center md:items-start">
                {/* Image */}
                <div className="flex-shrink-0 relative">
                  <Image
                    src={speaker.image}
                    alt={speaker.name}
                    width={256}
                    height={320}
                    className="w-64 md:w-80 rounded-2xl shadow-2xl object-cover"
                    style={{ aspectRatio: "4/5" }}
                  />
                  {speaker.badge && (
                    <div className="speaker-badge absolute -bottom-4 left-1/2 -translate-x-1/2 px-5 py-2 rounded-full text-white text-xs font-bold flex items-center gap-2 whitespace-nowrap">
                      {speaker.badge}
                    </div>
                  )}
                </div>

                {/* Info */}
                <div className="flex-1 text-center md:text-left">
                  <div className="font-heading text-3xl md:text-4xl font-black text-white mb-2">
                    {speaker.name}
                  </div>
                  <div className="text-gold-light font-bold text-lg mb-1">
                    {speaker.title}
                  </div>
                  <div className="text-white/50 text-sm mb-6 pb-6 border-b border-white/10">
                    {speaker.organization}
                  </div>

                  {/* Bio */}
                  <ul className="flex flex-col gap-4">
                    {speaker.bio.map((item, bioIdx) => (
                      <li
                        key={bioIdx}
                        className="flex items-start gap-3 text-white/85 text-sm md:text-base"
                      >
                        <span className="check-icon">✓</span>
                        <span>{item}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
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
                trackClick(data.cta_label || "section-speaker", "speaker", "bottom");
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

export default SpeakerSection;
