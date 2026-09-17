/**
 * Mock data for tenant-site — used as graceful degradation when
 * backend APIs are unavailable. Each tenant has 4 pages with full
 * block content (hero, features, stats, contact, testimonials).
 */

import type { Block, Branding, Page, Tenant } from "./schema";

interface MockTenantFull extends Tenant {
  pages: Page[];
}

const heroBlock = (title: string, subtitle: string, ctaText = "Liên hệ ngay"): Block => ({
  id: "hero",
  type: "hero",
  data: { title, subtitle, ctaText },
});

const featuresBlock = (
  title: string,
  items: { icon: string; title: string; description: string }[],
): Block => ({
  id: "features",
  type: "features",
  data: { title, items },
});

const statsBlock = (stats: { value: string; label: string }[]): Block => ({
  id: "stats",
  type: "stats",
  data: { stats },
});

const contactBlock = (tenantSlug: string): Block => ({
  id: "contact",
  type: "contact",
  data: { tenantSlug },
});

const testimonialsBlock = (
  title: string,
  items: { name: string; role: string; quote: string; avatar?: string }[],
): Block => ({
  id: "testimonials",
  type: "testimonials",
  data: { title, items },
});

const ctaBlock = (title: string, subtitle: string, ctaText: string): Block => ({
  id: "cta",
  type: "cta",
  data: { title, subtitle, ctaText },
});

const teamBlock = (
  title: string,
  members: { name: string; role: string; avatar?: string }[],
): Block => ({
  id: "team",
  type: "team",
  data: { title, members },
});

const pricingBlock = (
  title: string,
  plans: { name: string; price: string; features: string[]; highlighted?: boolean }[],
): Block => ({
  id: "pricing",
  type: "pricing",
  data: { title, plans },
});

const faqBlock = (
  title: string,
  items: { q: string; a: string }[],
): Block => ({
  id: "faq",
  type: "faq",
  data: { title, items },
});

export const mockTenants: Record<string, MockTenantFull> = {
  apexfintech: {
    id: "tn_001",
    slug: "apexfintech",
    name: "Apex Fintech",
    logo: "/logos/apexfintech.png",
    branding: {
      primary: "#0066CC",
      secondary: "#F59E0B",
      accent: "#10B981",
      fontFamily: "Inter",
    },
    pages: [
      {
        id: "p1",
        slug: "home",
        title: "Apex Fintech — Giải pháp tài chính thông minh",
        description: "Nền tảng fintech hàng đầu cho doanh nghiệp Việt Nam",
        blocks: [
          heroBlock(
            "Tài chính thông minh cho doanh nghiệp hiện đại",
            "Apex Fintech giúp doanh nghiệp của bạn tối ưu vận hành tài chính với công nghệ AI và blockchain tiên tiến.",
          ),
          statsBlock([
            { value: "500+", label: "Doanh nghiệp tin dùng" },
            { value: "98%", label: "Khách hàng hài lòng" },
            { value: "24/7", label: "Hỗ trợ kỹ thuật" },
            { value: "10 tỷ+", label: "Giao dịch mỗi tháng" },
          ]),
          featuresBlock(
            "Tại sao chọn Apex Fintech",
            [
              { icon: "🚀", title: "Xử lý tức thì", description: "Giao dịch hoàn tất trong vòng 2 giây với hạ tầng hiệu năng cao." },
              { icon: "🔒", title: "Bảo mật chuẩn quốc tế", description: "Tuân thủ PCI-DSS, ISO 27001, mã hóa AES-256 end-to-end." },
              { icon: "🤖", title: "AI phân tích rủi ro", description: "Machine learning giúp phát hiện gian lận và đánh giá tín dụng tự động." },
              { icon: "🌐", title: "Tích hợp dễ dàng", description: "RESTful API + SDK đầy đủ cho 8 ngôn ngữ lập trình." },
            ],
          ),
          testimonialsBlock(
            "Khách hàng nói gì",
            [
              { name: "Nguyen Van A", role: "CEO, TechCorp", quote: "Apex Fintech giúp chúng tôi tiết kiệm 60% chi phí xử lý thanh toán." },
              { name: "Tran Thi B", role: "CFO, MegaShop", quote: "Dashboard trực quan, đội ngũ support phản hồi trong 5 phút." },
            ],
          ),
          contactBlock("apexfintech"),
        ],
      },
      {
        id: "p2",
        slug: "about",
        title: "Về chúng tôi — Apex Fintech",
        description: "Câu chuyện và đội ngũ của Apex Fintech",
        blocks: [
          heroBlock("Câu chuyện của chúng tôi", "Được thành lập năm 2020 với sứ mệnh democratize tài chính cho SMEs Việt Nam."),
          teamBlock("Đội ngũ lãnh đạo", [
            { name: "Nguyen Van Founder", role: "CEO & Co-Founder" },
            { name: "Tran Thi CTO", role: "CTO & Co-Founder" },
            { name: "Le Van COO", role: "COO" },
            { name: "Pham Thi CFO", role: "CFO" },
          ]),
          statsBlock([
            { value: "50+", label: "Nhân viên" },
            { value: "3", label: "Văn phòng" },
            { value: "5 năm", label: "Kinh nghiệm" },
            { value: "20+", label: "Đối tác chiến lược" },
          ]),
          contactBlock("apexfintech"),
        ],
      },
      {
        id: "p3",
        slug: "services",
        title: "Dịch vụ — Apex Fintech",
        description: "Các sản phẩm và dịch vụ tài chính",
        blocks: [
          heroBlock("Dịch vụ của chúng tôi", "Bộ giải pháp tài chính toàn diện cho doanh nghiệp"),
          featuresBlock(
            "Sản phẩm nổi bật",
            [
              { icon: "💳", title: "Payment Gateway", description: "Xử lý thanh toán đa kênh với phí cạnh tranh." },
              { icon: "📊", title: "Credit Scoring", description: "Đánh giá tín dụng tự động với AI chính xác 95%." },
              { icon: "💱", title: "FX Engine", description: "Tỷ giá realtime cho 40+ đồng tiền." },
              { icon: "🛡️", title: "Fraud Detection", description: "Phát hiện gian lận trong 100ms." },
              { icon: "📈", title: "Analytics", description: "Dashboard phân tích chi tiết cho CFO." },
              { icon: "🔗", title: "Open Banking", description: "Kết nối 30+ ngân hàng Việt Nam." },
            ],
          ),
          ctaBlock("Sẵn sàng bắt đầu?", "Đăng ký demo miễn phí và khám phá sức mạnh của Apex Fintech.", "Đăng ký ngay"),
        ],
      },
      {
        id: "p4",
        slug: "contact",
        title: "Liên hệ — Apex Fintech",
        description: "Liên hệ với đội ngũ Apex Fintech",
        blocks: [
          heroBlock("Liên hệ với chúng tôi", "Chúng tôi sẵn sàng hỗ trợ bạn 24/7."),
          contactBlock("apexfintech"),
        ],
      },
    ],
  },

  vietnamrealty: {
    id: "tn_002",
    slug: "vietnamrealty",
    name: "Vietnam Realty",
    logo: "/logos/vietnamrealty.png",
    branding: {
      primary: "#F59E0B",
      secondary: "#1F2937",
      accent: "#10B981",
      fontFamily: "Inter",
    },
    pages: [
      {
        id: "p1",
        slug: "home",
        title: "Vietnam Realty — Bất động sản cho mọi người",
        description: "Nền tảng BĐS uy tín hàng đầu",
        blocks: [
          heroBlock(
            "Tìm ngôi nhà mơ ước của bạn",
            "Hơn 10,000 căn hộ, nhà phố, biệt thự được cập nhật mỗi ngày.",
          ),
          statsBlock([
            { value: "10K+", label: "Tin đăng" },
            { value: "50K+", label: "Khách hàng" },
            { value: "98%", label: "Hài lòng" },
            { value: "15+", label: "Tỉnh thành" },
          ]),
          featuresBlock(
            "Tại sao chọn Vietnam Realty",
            [
              { icon: "🔍", title: "Tìm kiếm thông minh", description: "AI gợi ý BĐS phù hợp với ngân sách và vị trí." },
              { icon: "📋", title: "Pháp lý minh bạch", description: "Xác minh 100% giấy tờ pháp lý trước khi đăng." },
              { icon: "🤝", title: "Môi giới chuyên nghiệp", description: "Đội ngũ 500+ môi giới được đào tạo bài bản." },
              { icon: "💰", title: "Hỗ trợ tài chính", description: "Liên kết 20+ ngân hàng, hỗ trợ vay lên đến 80%." },
            ],
          ),
          testimonialsBlock(
            "Câu chuyện khách hàng",
            [
              { name: "Anh Minh", role: "Kỹ sư IT", quote: "Tìm được căn hộ ưng ý tại Quận 2 chỉ trong 2 tuần nhờ Vietnam Realty." },
              { name: "Chị Lan", role: "Giáo viên", quote: "Môi giới nhiệt tình, hỗ trợ thủ tục pháp lý tận tình." },
            ],
          ),
          contactBlock("vietnamrealty"),
        ],
      },
      {
        id: "p2",
        slug: "about",
        title: "Về Vietnam Realty",
        blocks: [
          heroBlock("Về chúng tôi", "Vietnam Realty — Kết nối BĐS Việt"),
          statsBlock([
            { value: "5 năm", label: "Hoạt động" },
            { value: "100+", label: "Nhân viên" },
            { value: "10K+", label: "Giao dịch thành công" },
          ]),
          contactBlock("vietnamrealty"),
        ],
      },
      {
        id: "p3",
        slug: "services",
        title: "Dịch vụ — Vietnam Realty",
        blocks: [
          heroBlock("Dịch vụ của chúng tôi", "Mua bán, cho thuê, đầu tư BĐS"),
          pricingBlock(
            "Gói dịch vụ",
            [
              { name: "Cơ bản", price: "Miễn phí", features: ["Đăng 3 tin", "Tìm kiếm cơ bản", "Hỗ trợ email"] },
              { name: "Pro", price: "2.9tr/tháng", features: ["Đăng không giới hạn", "AI matching", "Hỗ trợ 24/7", "Báo cáo thị trường"], highlighted: true },
              { name: "Enterprise", price: "Liên hệ", features: ["Mọi tính năng Pro", "API riêng", "Account manager", "Whitelabel"] },
            ],
          ),
          faqBlock(
            "Câu hỏi thường gặp",
            [
              { q: "Phí dịch vụ là bao nhiêu?", a: "Gói Cơ bản miễn phí, gói Pro 2.9 triệu/tháng." },
              { q: "Có hỗ trợ vay ngân hàng?", a: "Có, chúng tôi liên kết 20+ ngân hàng với lãi suất ưu đãi." },
              { q: "Thời gian đăng tin?", a: "Tin được duyệt trong vòng 2 giờ làm việc." },
            ],
          ),
        ],
      },
      {
        id: "p4",
        slug: "contact",
        title: "Liên hệ — Vietnam Realty",
        blocks: [
          heroBlock("Liên hệ với chúng tôi", "Hotline: 1900-6868"),
          contactBlock("vietnamrealty"),
        ],
      },
    ],
  },

  saigonhealth: {
    id: "tn_003",
    slug: "saigonhealth",
    name: "Saigon Health Group",
    logo: "/logos/saigonhealth.png",
    branding: {
      primary: "#10B981",
      secondary: "#0EA5E9",
      accent: "#F59E0B",
      fontFamily: "Inter",
    },
    pages: [
      {
        id: "p1",
        slug: "home",
        title: "Saigon Health Group",
        description: "Chăm sóc sức khỏe toàn diện",
        blocks: [
          heroBlock(
            "Chăm sóc sức khỏe hàng đầu Việt Nam",
            "Hệ thống y tế hiện đại với 12 bệnh viện và phòng khám trên toàn quốc.",
          ),
          statsBlock([
            { value: "12", label: "Cơ sở y tế" },
            { value: "200+", label: "Bác sĩ" },
            { value: "500K+", label: "Bệnh nhân/năm" },
            { value: "98%", label: "Hài lòng" },
          ]),
          featuresBlock(
            "Dịch vụ y tế",
            [
              { icon: "🏥", title: "Khám tổng quát", description: "Gói khám sức khỏe toàn diện cho cá nhân và doanh nghiệp." },
              { icon: "⚕️", title: "Chuyên khoa", description: "30+ chuyên khoa với bác sĩ đầu ngành." },
              { icon: "📅", title: "Đặt lịch online", description: "Đặt lịch khám trong 30 giây qua app hoặc website." },
              { icon: "💊", title: "Nhà thuốc", description: "Hệ thống nhà thuốc đạt chuẩn GPP." },
            ],
          ),
          contactBlock("saigonhealth"),
        ],
      },
      {
        id: "p2",
        slug: "about",
        title: "Về Saigon Health Group",
        blocks: [
          heroBlock("Về chúng tôi", "Saigon Health Group — Vì sức khỏe cộng đồng"),
          statsBlock([
            { value: "15 năm", label: "Hoạt động" },
            { value: "2000+", label: "Nhân viên y tế" },
            { value: "12", label: "Cơ sở" },
          ]),
          contactBlock("saigonhealth"),
        ],
      },
      {
        id: "p3",
        slug: "services",
        title: "Dịch vụ — Saigon Health",
        blocks: [
          heroBlock("Dịch vụ y tế", "Từ khám tổng quát đến điều trị chuyên sâu"),
          featuresBlock(
            "Các gói dịch vụ",
            [
              { icon: "🩺", title: "Khám sức khỏe doanh nghiệp", description: "Gói khám định kỳ cho nhân viên." },
              { icon: "🏠", title: "Khám tại nhà", description: "Bác sĩ đến tận nơi theo yêu cầu." },
              { icon: "🧬", title: "Xét nghiệm ADN", description: "Xét nghiệm gen, huyết thống chính xác." },
              { icon: "💉", title: "Tiêm chủng", description: "Vaccine chất lượng cao cho mọi lứa tuổi." },
            ],
          ),
          ctaBlock("Đặt lịch khám ngay", "Đặt lịch trong 30 giây, nhận ưu đãi 20% cho lần khám đầu tiên.", "Đặt lịch"),
        ],
      },
      {
        id: "p4",
        slug: "contact",
        title: "Liên hệ — Saigon Health",
        blocks: [
          heroBlock("Liên hệ với chúng tôi", "Hotline: 1900-6060"),
          contactBlock("saigonhealth"),
        ],
      },
    ],
  },

  megashop: {
    id: "tn_005",
    slug: "megashop",
    name: "MegaShop Vietnam",
    logo: "/logos/megashop.png",
    branding: {
      primary: "#EF4444",
      secondary: "#FBBF24",
      accent: "#06B6D4",
      fontFamily: "Inter",
    },
    pages: [
      {
        id: "p1",
        slug: "home",
        title: "MegaShop Vietnam",
        description: "Siêu thị trực tuyến hàng đầu",
        blocks: [
          heroBlock(
            "Mua sắm thông minh, tiết kiệm hơn",
            "Hơn 1 triệu sản phẩm từ 5,000+ thương hiệu uy tín.",
          ),
          statsBlock([
            { value: "1M+", label: "Sản phẩm" },
            { value: "5K+", label: "Thương hiệu" },
            { value: "5M+", label: "Khách hàng" },
            { value: "24h", label: "Giao hàng nhanh" },
          ]),
          featuresBlock(
            "Ưu đãi đặc biệt",
            [
              { icon: "🚚", title: "Free ship 24h", description: "Miễn phí vận chuyển cho đơn từ 200K trong nội thành." },
              { icon: "💳", title: "Trả góp 0%", description: "Trả góp qua thẻ tín dụng với 0% lãi suất." },
              { icon: "🔄", title: "Đổi trả dễ dàng", description: "Đổi trả miễn phí trong 30 ngày." },
              { icon: "🎁", title: "Tích điểm thưởng", description: "Đổi điểm lấy quà tặng giá trị." },
            ],
          ),
          contactBlock("megashop"),
        ],
      },
      {
        id: "p2",
        slug: "about",
        title: "Về MegaShop",
        blocks: [
          heroBlock("Về MegaShop", "10 năm đồng hành cùng người tiêu dùng Việt"),
          contactBlock("megashop"),
        ],
      },
      {
        id: "p3",
        slug: "services",
        title: "Dịch vụ — MegaShop",
        blocks: [
          heroBlock("Dịch vụ của chúng tôi", "Từ mua sắm đến hậu mãi"),
          featuresBlock(
            "Tiện ích",
            [
              { icon: "📦", title: "Giao hàng nhanh", description: "Trong ngày tại HCM, HN." },
              { icon: "🛡️", title: "Bảo hành chính hãng", description: "Bảo hành 12-24 tháng." },
              { icon: "💬", title: "Hỗ trợ 24/7", description: "Chat trực tuyến mọi lúc." },
            ],
          ),
        ],
      },
      {
        id: "p4",
        slug: "contact",
        title: "Liên hệ — MegaShop",
        blocks: [
          heroBlock("Liên hệ", "Hotline: 1900-6868"),
          contactBlock("megashop"),
        ],
      },
    ],
  },

  eduviet: {
    id: "tn_004",
    slug: "eduviet",
    name: "EduViet Academy",
    logo: "/logos/eduviet.png",
    branding: {
      primary: "#8B5CF6",
      secondary: "#F59E0B",
      accent: "#10B981",
      fontFamily: "Inter",
    },
    pages: [
      {
        id: "p1",
        slug: "home",
        title: "EduViet Academy",
        description: "Nền tảng giáo dục trực tuyến hàng đầu",
        blocks: [
          heroBlock(
            "Học tập mọi lúc, mọi nơi",
            "Hơn 500 khóa học từ các chuyên gia hàng đầu Việt Nam và thế giới.",
          ),
          statsBlock([
            { value: "500+", label: "Khóa học" },
            { value: "100K+", label: "Học viên" },
            { value: "50+", label: "Giảng viên" },
            { value: "4.8★", label: "Đánh giá" },
          ]),
          featuresBlock(
            "Lợi ích khi học tại EduViet",
            [
              { icon: "🎓", title: "Chứng chỉ uy tín", description: "Được công nhận bởi 100+ doanh nghiệp." },
              { icon: "👨‍🏫", title: "Giảng viên top đầu", description: "Chuyên gia từ Google, Microsoft, VinAI." },
              { icon: "📱", title: "Học trên mọi thiết bị", description: "iOS, Android, Web — đồng bộ hoàn toàn." },
              { icon: "🏆", title: "Cộng đồng sôi động", description: "Forum, group học tập, mentor 1-1." },
            ],
          ),
          contactBlock("eduviet"),
        ],
      },
      {
        id: "p2",
        slug: "about",
        title: "Về EduViet Academy",
        blocks: [
          heroBlock("Về chúng tôi", "Sứ mệnh giáo dục cho mọi người Việt"),
          contactBlock("eduviet"),
        ],
      },
      {
        id: "p3",
        slug: "services",
        title: "Khóa học — EduViet",
        blocks: [
          heroBlock("Khóa học nổi bật", "Lập trình, Marketing, Ngoại ngữ, Kinh doanh"),
          pricingBlock(
            "Bảng giá",
            [
              { name: "Free", price: "0đ", features: ["20 khóa học miễn phí", "Cộng đồng", "Học thử"] },
              { name: "Pro", price: "299K/tháng", features: ["Mọi khóa học", "Chứng chỉ", "Mentor 1-1"], highlighted: true },
              { name: "Team", price: "Liên hệ", features: ["Mọi tính năng Pro", "Dashboard doanh nghiệp", "API"] },
            ],
          ),
        ],
      },
      {
        id: "p4",
        slug: "contact",
        title: "Liên hệ — EduViet",
        blocks: [
          heroBlock("Liên hệ", "Hotline: 1900-6868"),
          contactBlock("eduviet"),
        ],
      },
    ],
  },

  demo: {
    id: "tn_demo",
    slug: "demo",
    name: "Demo Company",
    branding: {
      primary: "#0A192F",
      secondary: "#F5A623",
      accent: "#FF6B00",
    },
    pages: [
      {
        id: "p1",
        slug: "home",
        title: "Demo Company",
        description: "Demo landing page",
        blocks: [
          heroBlock(
            "Giải pháp số toàn diện",
            "Chúng tôi giúp doanh nghiệp của bạn phát triển bền vững với công nghệ hiện đại.",
          ),
          featuresBlock(
            "Tại sao chọn chúng tôi",
            [
              { icon: "🚀", title: "Nhanh chóng", description: "Triển khai trong thời gian ngắn nhất." },
              { icon: "💎", title: "Chất lượng", description: "Sản phẩm đạt chuẩn quốc tế." },
              { icon: "🤝", title: "Hỗ trợ 24/7", description: "Đội ngũ chuyên nghiệp luôn sẵn sàng." },
            ],
          ),
          statsBlock([
            { value: "500+", label: "Khách hàng" },
            { value: "98%", label: "Hài lòng" },
            { value: "24/7", label: "Hỗ trợ" },
            { value: "10+", label: "Năm kinh nghiệm" },
          ]),
          contactBlock("demo"),
        ],
      },
      {
        id: "p2",
        slug: "about",
        title: "About — Demo",
        blocks: [heroBlock("About Demo Company", "Câu chuyện của chúng tôi"), contactBlock("demo")],
      },
      {
        id: "p3",
        slug: "services",
        title: "Services — Demo",
        blocks: [heroBlock("Our Services", "What we offer"), contactBlock("demo")],
      },
      {
        id: "p4",
        slug: "contact",
        title: "Contact — Demo",
        blocks: [heroBlock("Contact Us", "Get in touch"), contactBlock("demo")],
      },
    ],
  },
};

export function getMockTenant(slug: string): MockTenantFull | undefined {
  return mockTenants[slug] ?? mockTenants.demo;
}

export function getMockTenantSlugs(): string[] {
  return Object.keys(mockTenants);
}

export function getMockPage(slug: string, pageSlug: string): Page | undefined {
  const tenant = getMockTenant(slug);
  return tenant?.pages.find((p) => p.slug === pageSlug);
}
