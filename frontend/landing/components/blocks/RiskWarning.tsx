"use client";

import { Button } from "@/components/ui/button";

interface RiskWarningProps {
  data: {
    title?: string;
    content?: string;
    company_name?: string;
    company_info?: {
      address?: string;
      hotline?: string;
      website?: string;
    };
  };
}

export function RiskWarning({ data }: RiskWarningProps) {
  return (
    <section className="bg-navy-900 py-16 md:py-24">
      <div className="max-w-4xl mx-auto px-4">
        {/* Risk Warning Box */}
        <div className="risk-box p-8 md:p-12 mb-10">
          <div className="flex items-center gap-4 mb-6">
            <div className="w-14 h-14 bg-gradient-to-br from-gold to-orange rounded-full flex items-center justify-center flex-shrink-0 text-2xl">
              ⚠
            </div>
            <h3 className="text-xl md:text-2xl font-heading font-black text-orange">
              {data.title || "CẢNH BÁO RỦI RO THỊ TRƯỜNG"}
            </h3>
          </div>
          <p className="text-white/80 leading-relaxed text-sm md:text-base">
            {data.content ||
              "Giao dịch Hàng hóa Phái sinh là hoạt động đầu tư tài chính có sử dụng đòn bẩy ký quỹ, chứa đựng cơ hội lợi nhuận đi kèm với rủi ro biến động giá. APEX Fintech khuyến cáo Nhà đầu tư cần trang bị đầy đủ kiến thức, hiểu rõ cơ chế vận hành thị trường và quản lý vốn kỷ luật. Chúng tôi không đưa ra các cam kết hay đảm bảo về mức lợi nhuận cố định dưới mọi hình thức."}
          </p>
        </div>

        {/* Company Info */}
        <div className="company-info p-8 md:p-10">
          <div className="flex flex-col md:flex-row gap-6 md:gap-10 items-start">
            <div className="flex items-center gap-4 flex-shrink-0">
              <img
                src="/assets/img/logo_APEX_K_NEN.webp"
                alt="APEX"
                className="w-14 h-14 rounded-xl"
              />
              <div>
                <div className="font-heading font-bold text-white text-base md:text-lg">
                  {data.company_name || "CÔNG TY CỔ PHẦN CÔNG NGHỆ TÀI CHÍNH APEX"}
                </div>
                <div className="text-white/50 text-xs md:text-sm mt-1">
                  Thành viên Kinh doanh Số 080 - Sở Giao dịch Hàng hóa Việt Nam (MXV)
                </div>
              </div>
            </div>

            <div className="flex-1 space-y-3">
              {data.company_info?.address && (
                <div className="flex items-start gap-3 text-white/80 text-sm">
                  <span className="text-orange flex-shrink-0 mt-0.5">📍</span>
                  <span>{data.company_info.address}</span>
                </div>
              )}
              {data.company_info?.hotline && (
                <div className="flex items-start gap-3 text-white/80 text-sm">
                  <span className="text-orange flex-shrink-0 mt-0.5">📞</span>
                  <span>
                    Hotline/Zalo:{" "}
                    <strong className="text-gold-light">{data.company_info.hotline}</strong>
                  </span>
                </div>
              )}
              {data.company_info?.website && (
                <div className="flex items-start gap-3 text-white/80 text-sm">
                  <span className="text-orange flex-shrink-0 mt-0.5">🌐</span>
                  <a
                    href={data.company_info.website}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline hover:text-gold transition-colors"
                  >
                    {data.company_info.website}
                  </a>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

export default RiskWarning;
