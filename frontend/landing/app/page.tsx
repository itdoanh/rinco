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
    id: "stats",
    type: "stats",
    data: {
      stats: [
        { value: "7.500", label: "TỶ ĐỒNG/NGÀY — Thanh khoản Q2/2026", prefix: "", suffix: "+" },
        { value: "17.000", label: "TỶ ĐỒNG/NGÀY — Đỉnh thanh khoản 2026", prefix: "", suffix: "" },
        { value: "TOP", label: "THỊ PHẦN MXV — Khối lượng giao dịch", prefix: "", suffix: "1" },
        { value: "50", label: "EBOOK MIỄN PHÍ — Tặng NĐT đăng ký sớm", prefix: "+", suffix: "" },
      ],
    },
  },
  {
    id: "features",
    type: "feature_grid",
    data: {
      title: "TẠI SAO HÀNG HÓA PHÁI SINH ĐANG LÀ KÊNH ĐẦU TƯ BÙNG NỔ?",
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
    id: "testimonial-section",
    type: "testimonial",
    data: {
      title: "CẢM NHẬN TỪ NHÀ ĐẦU TƯ ĐÃ THAM GIA",
      testimonials: [
        {
          name: "Anh Nguyễn Văn H.",
          role: "Nhà đầu tư cá nhân — Hà Nội",
          content:
            "Sau buổi chia sẻ, tôi đã nắm rõ cơ chế T+0 và bắt đầu giao dịch thành công với hợp đồng đầu tiên. Đội ngũ APEX hỗ trợ nhiệt tình 24/7.",
          rating: 5,
        },
        {
          name: "Chị Trần Thị M.",
          role: "Doanh nhân xuất nhập khẩu — TP.HCM",
          content:
            "Được tư vấn 1:1 bởi chuyên gia có chứng chỉ MXV, tôi xây dựng được chiến lược phòng ngừa rủi ro giá hiệu quả. Tiết kiệm chi phí rất nhiều cho DN.",
          rating: 5,
        },
        {
          name: "Anh Lê Hoàng P.",
          role: "Nhà đầu tư F0 — Đà Nẵng",
          content:
            "Bộ 10 Ebook Thực Chiến chi tiết, dễ hiểu. Từ một người chưa biết gì, tôi đã tự tin tham gia thị trường hàng hóa phái sinh một cách kỷ luật.",
          rating: 5,
        },
      ],
    },
  },
  {
    id: "faq-section",
    type: "faq",
    data: {
      title: "GIẢI ĐÁP THẮC MẮC",
      items: [
        {
          question: "Hàng hóa phái sinh là gì?",
          answer:
            "Hàng hóa phái sinh là các công cụ tài chính mà giá trị phụ thuộc vào giá tài sản cơ sở (vàng, dầu, nông sản, kim loại…). Cho phép giao dịch với đòn bẩy ký quỹ, thanh toán T+0 và sinh lời cả 2 chiều tăng/giảm.",
        },
        {
          question: "Thanh toán T+0 là gì?",
          answer:
            "T+0 là hình thức thanh toán trong cùng ngày giao dịch. Khi bạn đóng vị thế, tiền bán được cộng ngay vào tài khoản, không cần chờ như chứng khoán truyền thống (T+2).",
        },
        {
          question: "Giao dịch 2 chiều là gì?",
          answer:
            "Giao dịch 2 chiều cho phép đặt lệnh Mua (Long) khi kỳ vọng giá tăng và lệnh Bán (Short) khi kỳ vọng giá giảm. Cả 2 chiều đều có thể sinh lời.",
        },
        {
          question: "Webinar có miễn phí không?",
          answer:
            "Hoàn toàn miễn phí! Đăng ký tham dự Webinar Zoom sẽ nhận ngay Bộ 10 Ebook Thực Chiến và được tư vấn 1:1 bởi chuyên gia có chứng chỉ MXV.",
        },
        {
          question: "Tôi cần chuẩn bị gì để tham gia?",
          answer:
            "Bạn chỉ cần đăng ký qua form, có thiết bị (PC/điện thoại có loa + micro), kết nối Internet ổn định. Không cần mở tài khoản giao dịch trước.",
        },
        {
          question: "APEX Fintech có uy tín không?",
          answer:
            "APEX Fintech là Thành viên Kinh doanh Số 080 của Sở Giao dịch Hàng hóa Việt Nam (MXV) — đơn vị trực thuộc Bộ Công Thương. Top 1 thị phần giao dịch MXV Quý II/2026.",
        },
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

      {/* Lead Modal — accepts any CTA trigger */}
      <LeadModal
        trigger="button"
        formName="modal-form"
        ctaLabel="modal"
      />
      <LeadModal
        trigger="sticky"
        formName="sticky-form"
        ctaLabel="sticky"
      />
    </main>
  );
}
