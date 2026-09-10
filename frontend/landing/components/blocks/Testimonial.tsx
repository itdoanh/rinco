"use client";

import Image from "next/image";
import { cn } from "@/lib/utils";

interface Testimonial {
  name: string;
  role?: string;
  avatar?: string;
  content: string;
  rating?: number;
}

interface TestimonialSectionProps {
  data: {
    title?: string;
    testimonials?: Testimonial[];
    style?: string;
  };
}

export function TestimonialSection({ data }: TestimonialSectionProps) {
  const defaultTestimonials: Testimonial[] = [
    {
      name: "Nguyễn Văn A",
      role: "Nhà đầu tư cá nhân",
      content:
        "Sau khi tham gia webinar, tôi hiểu rõ hơn về cơ chế T+0 và bắt đầu giao dịch thành công. Đội ngũ APEX hỗ trợ rất nhiệt tình.",
      rating: 5,
    },
    {
      name: "Trần Thị B",
      role: "Doanh nhân",
      content:
        "Được tư vấn 1:1 bởi chuyên gia, tôi đã xây dựng được chiến lược phòng ngừa rủi ro hiệu quả cho việc kinh doanh xuất nhập khẩu.",
      rating: 5,
    },
    {
      name: "Lê Hoàng C",
      role: "Nhà đầu tư F0",
      content:
        "Tài liệu Ebook rất chi tiết, dễ hiểu. Tôi từ người chưa biết gì giờ đã tự tin tham gia thị trường hàng hóa.",
      rating: 5,
    },
  ];

  const testimonials = data.testimonials?.length
    ? data.testimonials
    : defaultTestimonials;

  return (
    <section className="bg-white py-16 md:py-24">
      <div className="max-w-7xl mx-auto px-4">
        {/* Header */}
        <div className="text-center mb-12">
          <span className="inline-flex items-center gap-2 bg-orange/10 border-2 border-orange text-orange px-4 py-2 rounded-full text-xs font-bold uppercase tracking-widest mb-5">
            CẢM NHẬN KHÁCH HÀNG
          </span>
          <h2 className="font-heading text-3xl md:text-4xl font-black text-navy-900">
            {data.title || "Họ Đã Thành Công Cùng APEX"}
          </h2>
        </div>

        {/* Testimonials Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {testimonials.map((testimonial, idx) => (
            <div
              key={idx}
              className={cn(
                "bg-gray-50 rounded-2xl p-6 border border-gray-100",
                "hover:shadow-lg hover:-translate-y-1 transition-all duration-300"
              )}
            >
              {/* Rating */}
              {testimonial.rating && (
                <div className="flex gap-1 mb-4">
                  {Array.from({ length: testimonial.rating }).map((_, i) => (
                    <span key={i} className="text-amber-400 text-lg">
                      ★
                    </span>
                  ))}
                </div>
              )}

              {/* Content */}
              <p className="text-gray-700 leading-relaxed mb-4">
                "{testimonial.content}"
              </p>

              {/* Author */}
              <div className="flex items-center gap-3">
                {testimonial.avatar ? (
                  <Image
                    src={testimonial.avatar}
                    alt={testimonial.name}
                    width={40}
                    height={40}
                    className="w-10 h-10 rounded-full object-cover"
                  />
                ) : (
                  <div className="w-10 h-10 rounded-full bg-gradient-to-br from-orange to-amber flex items-center justify-center text-white font-bold">
                    {testimonial.name[0]}
                  </div>
                )}
                <div>
                  <p className="font-bold text-navy-900 text-sm">
                    {testimonial.name}
                  </p>
                  {testimonial.role && (
                    <p className="text-gray-500 text-xs">{testimonial.role}</p>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// Note: TestimonialSection is the canonical export name.
// The legacy 'Testimonial' name is preserved via index.ts re-export as default.
export default TestimonialSection;
