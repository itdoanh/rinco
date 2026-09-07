"use client";

import { useState } from "react";
import { DynamicForm } from "./DynamicForm";
import { Button } from "@/components/ui/button";

interface FormBlockProps {
  data: {
    form_id?: string;
    title?: string;
    description?: string;
    multi_step?: boolean;
    cta_label?: string;
    show_progress?: boolean;
    progress_given?: number;
    progress_total?: number;
    event_time?: string;
    event_location?: string;
    benefits?: string[];
    style?: string;
  };
  tenantSlug?: string;
  pageSlug?: string;
}

export function FormBlock({ data, tenantSlug, pageSlug }: FormBlockProps) {
  const [channel, setChannel] = useState<"zalo" | "phone" | "">("");
  const [step, setStep] = useState(1);

  const ctaLabel = data.cta_label || "GIỮ VÉ ZOOM & NHẬN EBOOK MIỄN PHÍ";

  if (data.multi_step) {
    return (
      <div className="form-card max-w-md mx-auto">
        <div className="form-card-header px-6 py-4">
          <span className="text-white font-bold text-center block text-sm uppercase tracking-wide">
            🚀 {data.title || "ĐĂNG KÝ GIỮ VÉ ZOOM & NHẬN EBOOK"}
          </span>
        </div>
        <div className="p-6">
          {/* Step indicator */}
          <div className="step-indicator mb-6">
            <div className={`step-item ${step >= 1 ? "is-active" : ""} ${step > 1 ? "is-done" : ""}`}>
              <div className="step-circle">{step > 1 ? "✓" : "1"}</div>
              <div className="step-label">Chọn kênh</div>
            </div>
            <div className="step-line" />
            <div className={`step-item ${step >= 2 ? "is-active" : ""}`}>
              <div className="step-circle">2</div>
              <div className="step-label">Thông tin</div>
            </div>
          </div>

          {step === 1 && (
            <div>
              <h3 className="form-step-title">
                Bạn muốn nhận suất tham gia buổi chia sẻ qua kênh nào?
              </h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-5">
                <button
                  type="button"
                  className="channel-btn"
                  onClick={() => {
                    setChannel("zalo");
                    setStep(2);
                  }}
                >
                  <span className="channel-emoji">📲</span>
                  <span className="channel-name">Nhận qua Zalo</span>
                  <span className="channel-sub">Nhanh & tiện lợi nhất</span>
                </button>
                <button
                  type="button"
                  className="channel-btn"
                  onClick={() => {
                    setChannel("phone");
                    setStep(2);
                  }}
                >
                  <span className="channel-emoji">📞</span>
                  <span className="channel-name">Nhận qua SĐT</span>
                  <span className="channel-sub">Chuyên viên gọi tư vấn</span>
                </button>
              </div>
            </div>
          )}

          {step === 2 && (
            <div>
              <h3 className="form-step-title">
                Nhập thông tin để Hệ thống Gửi Vé Zoom & Ebook Miễn Phí
              </h3>
              <DynamicForm
                formName="hero-multi"
                tenantSlug={tenantSlug}
                pageSlug={pageSlug}
              />
              <button
                type="button"
                onClick={() => setStep(1)}
                className="text-sm text-gray-500 hover:text-orange font-semibold mt-3 w-full text-center"
              >
                ← Quay lại chọn kênh
              </button>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="form-card max-w-md">
      {data.title && (
        <div className="form-card-header px-6 py-4">
          <span className="text-white font-bold text-center block text-sm uppercase tracking-wide">
            {data.title}
          </span>
        </div>
      )}

      <div className="p-6">
        {data.show_progress && data.progress_total && (
          <div className="bg-gradient-to-r from-orange-50 to-amber-50 border border-orange-200 rounded-xl px-4 py-3 mb-4">
            <div className="flex items-center justify-between text-xs font-bold text-navy-900 mb-2">
              <span>
                🎁 Đã tặng {data.progress_given || 0}/{data.progress_total} Bộ Ebook
              </span>
              <span className="text-orange">
                {data.progress_total - (data.progress_given || 0)} suất còn lại
              </span>
            </div>
            <Progress
              value={((data.progress_given || 0) / data.progress_total) * 100}
              className="h-2 bg-orange-100"
            />
            <div className="text-[11px] text-gray-500 mt-1.5 text-center">
              cho nhà đầu tư đăng ký tuần này
            </div>
          </div>
        )}

        <DynamicForm
          formName={`${data.form_id || "default"}-form`}
          tenantSlug={tenantSlug}
          pageSlug={pageSlug}
        />

        {(data.event_time || data.event_location) && (
          <div className="flex flex-wrap gap-2 justify-center pt-4 border-t border-gray-100 mt-4">
            {data.event_time && (
              <div className="flex items-center gap-1.5 bg-gray-50 px-3 py-1.5 rounded-lg text-xs">
                <span>📅</span>
                <span>
                  <strong>{data.event_time}</strong>
                </span>
              </div>
            )}
            {data.event_location && (
              <div className="flex items-center gap-1.5 bg-gray-50 px-3 py-1.5 rounded-lg text-xs">
                <span>💻</span>
                <span>{data.event_location}</span>
              </div>
            )}
          </div>
        )}

        {data.benefits && data.benefits.length > 0 && (
          <ul className="space-y-2 mt-4 pt-4 border-t border-gray-100">
            {data.benefits.map((benefit, idx) => (
              <li
                key={idx}
                className="flex items-center gap-2 text-gray-700 text-sm font-semibold"
              >
                <span className="check-icon">✓</span>
                {benefit}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

export default FormBlock;
