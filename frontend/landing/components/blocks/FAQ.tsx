"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";

interface FAQ {
  question: string;
  answer: string;
}

interface FAQProps {
  data: {
    title?: string;
    items?: FAQ[];
    style?: string;
  };
}

export function FAQSection({ data }: FAQProps) {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  const defaultItems: FAQ[] = [
    {
      question: "Hàng hóa phái sinh là gì?",
      answer:
        "Hàng hóa phái sinh là các công cụ tài chính mà giá trị của chúng phụ thuộc vào giá của một tài sản cơ sở (như vàng, dầu, nông sản...). Cho phép nhà đầu tư giao dịch với đòn bẩy, thanh toán T+0 và kiếm lời từ cả 2 chiều tăng và giảm giá.",
    },
    {
      question: "Thanh toán T+0 là gì?",
      answer:
        "T+0 là hình thức thanh toán trong cùng ngày giao dịch. Khi bạn đóng vị thế, tiền bán sẽ được cộng ngay vào tài khoản, không cần chờ như chứng khoán truyền thống (T+2).",
    },
    {
      question: "Giao dịch 2 chiều là gì?",
      answer:
        "Giao dịch 2 chiều cho phép bạn đặt lệnh Mua (Long) khi kỳ vọng giá tăng và lệnh Bán (Short) khi kỳ vọng giá giảm. Cả 2 chiều đều có thể sinh lời.",
    },
    {
      question: " Webinar có miễn phí không?",
      answer:
        "Hoàn toàn miễn phí! Đăng ký tham dự Webinar Zoom sẽ nhận ngay Bộ 10 Ebook Thực Chiến Đầu Tư Hàng Hóa Phái Sinh và được tư vấn 1:1 bởi chuyên gia có chứng chỉ MXV.",
    },
    {
      question: "Tôi cần chuẩn bị gì để tham gia?",
      answer:
        "Bạn chỉ cần đăng ký, điều chỉnh thiết bị (máy tính/điện thoại có loa và micro), và tham gia đúng giờ. Không cần tài khoản giao dịch trước.",
    },
  ];

  const items = data.items?.length ? data.items : defaultItems;

  return (
    <section className="bg-gray-50 py-16 md:py-24">
      <div className="max-w-3xl mx-auto px-4">
        {/* Header */}
        <div className="text-center mb-12">
          <span className="inline-flex items-center gap-2 bg-orange/10 border-2 border-orange text-orange px-4 py-2 rounded-full text-xs font-bold uppercase tracking-widest mb-5">
            CÂU HỎI THƯỜNG GẶP
          </span>
          <h2 className="font-heading text-3xl md:text-4xl font-black text-navy-900">
            {data.title || "Giải Đáp Thắc Mắc"}
          </h2>
        </div>

        {/* FAQ Items */}
        <div className="space-y-4">
          {items.map((item, idx) => (
            <div
              key={idx}
              className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden"
            >
              <button
                type="button"
                onClick={() => setOpenIndex(openIndex === idx ? null : idx)}
                className="w-full flex items-center justify-between p-6 text-left"
              >
                <span className="font-bold text-navy-900 pr-4">{item.question}</span>
                <span
                  className={cn(
                    "w-8 h-8 rounded-full bg-orange/10 text-orange flex items-center justify-center flex-shrink-0 transition-transform",
                    openIndex === idx && "rotate-180"
                  )}
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    viewBox="0 0 24 24"
                  >
                    <path d="M19 9l-7 7-7-7" />
                  </svg>
                </span>
              </button>
              {openIndex === idx && (
                <div className="px-6 pb-6 text-gray-600 leading-relaxed">
                  {item.answer}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

export { FAQSection, FAQ as FAQSectionAlias }
export default FAQSection;
