"use client";

import { Header } from "@/components/layout/Header";
import { Footer } from "@/components/layout/Footer";
import { StickyCTA } from "@/components/layout/StickyCTA";
import { LeadModal } from "@/components/layout/LeadModal";
import { BlockRenderer } from "@/components/blocks/BlockRenderer";
import { openLeadModal } from "@/components/layout/LeadModal";

// Default blocks from chiase_cu content
const defaultBlocks = [
  {
    id: "hero",
    type: "hero",
    data: {
      title: "Tối Ưu Dòng Tiền 2026 Với",
      subtitle: "Kênh Đầu Tư Hàng Hóa Phái Sinh",
      description:
        "Giải mã cơ chế giao dịch T+0 & sinh lời 2 chiều linh hoạt. Giữ vé tham dự Webinar độc quyền cùng Chuyên gia và nhận ngay Bộ 10 Ebook Thực Chiến Đầu tư hôm nay!",
      cta_text: "ĐĂNG KÝ NGAY",
      cta_label: "hero-cta",
      image: "/anh/anhchandung_chuyengia_nguyentuananh.webp",
      badges: [
        { icon: "🏆", text: "TOP 1", subtext: "Thị trường Q2/2026" },
        { icon: "⚡", text: "T+0", subtext: "Thanh toán tức thì" },
        { icon: "🛡", text: "MXV", subtext: "Thành viên 080" },
      ],
      event_time: "20:00 | 07/09/2026",
      event_location: "Zoom Online",
    },
  },
  {
    id: "logos",
    type: "logo_cloud",
    data: {
      title: "Được cấp phép chính thức và liên thông giao dịch quốc tế",
      logos: [
        { name: "MXV" },
        { name: "CQG" },
        { name: "NYMEX" },
        { name: "CBOT" },
        { name: "LME" },
      ],
    },
  },
  {
    id: "features",
    type: "feature_grid",
    data: {
      features: [
        {
          icon: "⚡",
          title: "Thanh toán T+0 Linh hoạt",
          subtitle:
            "Cho phép đóng/mở vị thế giao dịch tức thì ngay trong phiên. Tiền bán cộng ngay vào tài khoản.",
          value:
            "Vòng xoay vốn liên tục, không bị ngâm vốn, chủ động chốt lời hoặc quản trị rủi ro lập tức.",
        },
        {
          icon: "⇅",
          title: "Cơ chế Giao dịch 2 Chiều",
          subtitle:
            "Cung cấp cả lệnh Mua (Long - kỳ vọng giá tăng) và lệnh Bán (Short - kỳ vọng giá giảm).",
          value:
            "Tìm kiếm cơ hội sinh lời linh hoạt ngay cả khi thị trường đi xuống.",
        },
        {
          icon: "💰",
          title: "Tỷ lệ Ký quỹ Margin Tối ưu",
          subtitle:
            "Chỉ cần ký quỹ một tỷ lệ phần trăm nhỏ theo quy định MXV để giao dịch hợp đồng tiêu chuẩn.",
          value:
            "Tối ưu hóa quy mô sử dụng vốn, gia tăng hiệu suất đầu tư mà không cần 100% tiền mặt.",
        },
        {
          icon: "🌐",
          title: "Minh bạch & Liên thông Quốc tế",
          subtitle:
            "Khớp lệnh Real-time trực tiếp tới các Sở giao dịch hàng hóa lớn nhất thế giới (CBOT, NYMEX, LME…).",
          value:
            "Giá vận hành chuẩn xác theo Cung - Cầu toàn cầu, chống thao túng giá từ cá nhân/tổ chức.",
        },
      ],
      cta_text: "ĐĂNG KÝ GIỮ VÉ ZOOM THAM GIA BUỔI CHIA SẺ",
      cta_label: "section-features",
    },
  },
  {
    id: "speaker",
    type: "speaker",
    data: {
      speakers: [
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
      ],
      cta_text: "ĐĂNG KÝ HỌC CÙNG CHUYÊN GIA NGUYỄN TUẤN ANH",
      cta_label: "section-speaker",
    },
  },
  {
    id: "trust",
    type: "trust",
    data: {
      top1_banner: "/anh/apex_top1_thiphan_quy2_2026.webp",
      badges: [
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
      ],
      cta_text: "TÔI MUỐN THAM GIA WEBINAR CỦA APEX FINTECH",
      cta_label: "section-trust",
    },
  },
  {
    id: "form-registration",
    type: "form",
    data: {
      form_id: "hero-multi",
      title: "🚀 ĐĂNG KÝ GIỮ VÉ ZOOM & NHẬN EBOOK",
      multi_step: true,
      event_time: "20:00 | 07/09/2026",
      event_location: "Zoom Online",
      benefits: [
        "Miễn phí 100% – Không phát sinh chi phí",
        "Tặng Bộ 10 Ebook Thực Chiến Đầu Tư",
        "Tư vấn 1:1 bởi chuyên gia có chứng chỉ MXV",
      ],
    },
  },
  {
    id: "risk",
    type: "risk_warning",
    data: {
      company_info: {
        address: "B-TT8-3 Him Lam, Vạn Phúc, Phường Hà Đông, Thành phố Hà Nội",
        hotline: "0984386538",
        website: "https://hanghoaphaisinh.net/",
      },
    },
  },
];

export default function LandingPage() {
  const handleCtaClick = () => {
    if (typeof window !== "undefined") {
      openLeadModal("button");
    }
  };

  return (
    <main className="min-h-screen">
      {/* Header */}
      <Header onCtaClick={handleCtaClick} />

      {/* Main Content - Block Renderer */}
      <BlockRenderer blocks={defaultBlocks as any} />

      {/* Footer */}
      <Footer />

      {/* Sticky CTA */}
      <StickyCTA onCtaClick={handleCtaClick} />

      {/* Lead Modal */}
      <LeadModal
        formName="modal-form"
        ctaLabel="modal"
      />
    </main>
  );
}
