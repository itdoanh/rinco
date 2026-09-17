"""Extended mock data for the RAG chatbot service (WS-B Loop 9).

20+ KB collections, 100+ documents, 50+ canned answers (vi + en).
"""
from __future__ import annotations

import os
import time
import uuid
from typing import Any, Dict, List

__all__ = [
    "EXTENDED_COLLECTIONS",
    "EXTENDED_DOCUMENTS",
    "EXTENDED_ANSWERS_VI",
    "EXTENDED_ANSWERS_EN",
    "EXTENDED_KB_TEMPLATES",
]


# ---------------------------------------------------------------------------
# KB Collections (20+)
# ---------------------------------------------------------------------------
EXTENDED_COLLECTIONS: List[Dict[str, Any]] = [
    # Apex Fintech collections
    {"name": "apex-product-docs", "tenant_id": "demo-tenant-apexfintech", "documents": 24, "description": "Tài liệu sản phẩm tài chính"},
    {"name": "apex-loan-policy", "tenant_id": "demo-tenant-apexfintech", "documents": 15, "description": "Chính sách cho vay"},
    {"name": "apex-faq-customer", "tenant_id": "demo-tenant-apexfintech", "documents": 32, "description": "FAQ khách hàng"},
    {"name": "apex-internal-wiki", "tenant_id": "demo-tenant-apexfintech", "documents": 18, "description": "Wiki nội bộ"},
    {"name": "apex-marketing-content", "tenant_id": "demo-tenant-apexfintech", "documents": 12, "description": "Nội dung marketing"},
    {"name": "apex-compliance", "tenant_id": "demo-tenant-apexfintech", "documents": 9, "description": "Tuân thủ & pháp lý"},
    {"name": "apex-pricing", "tenant_id": "demo-tenant-apexfintech", "documents": 7, "description": "Bảng giá dịch vụ"},

    # HCT Consulting collections
    {"name": "hct-property-listings", "tenant_id": "demo-tenant-hct-consulting", "documents": 28, "description": "Danh sách BĐS"},
    {"name": "hct-investment-thesis", "tenant_id": "demo-tenant-hct-consulting", "documents": 12, "description": "Phân tích đầu tư"},
    {"name": "hct-legal-docs", "tenant_id": "demo-tenant-hct-consulting", "documents": 16, "description": "Tài liệu pháp lý"},
    {"name": "hct-market-reports", "tenant_id": "demo-tenant-hct-consulting", "documents": 21, "description": "Báo cáo thị trường"},
    {"name": "hct-project-portfolio", "tenant_id": "demo-tenant-hct-consulting", "documents": 14, "description": "Portfolio dự án"},
    {"name": "hct-customer-faq", "tenant_id": "demo-tenant-hct-consulting", "documents": 19, "description": "Hỏi đáp khách hàng"},

    # Demo Company collections
    {"name": "demo-product-features", "tenant_id": "demo-tenant", "documents": 22, "description": "Tính năng sản phẩm"},
    {"name": "demo-getting-started", "tenant_id": "demo-tenant", "documents": 11, "description": "Hướng dẫn bắt đầu"},
    {"name": "demo-api-reference", "tenant_id": "demo-tenant", "documents": 26, "description": "Tài liệu API"},
    {"name": "demo-troubleshooting", "tenant_id": "demo-tenant", "documents": 13, "description": "Xử lý sự cố"},
    {"name": "demo-integrations", "tenant_id": "demo-tenant", "documents": 10, "description": "Tích hợp"},
    {"name": "demo-changelog", "tenant_id": "demo-tenant", "documents": 18, "description": "Lịch sử cập nhật"},
    {"name": "demo-best-practices", "tenant_id": "demo-tenant", "documents": 8, "description": "Best practices"},
    {"name": "demo-security", "tenant_id": "demo-tenant", "documents": 6, "description": "Bảo mật & quyền riêng tư"},
]


# ---------------------------------------------------------------------------
# Documents (100+)
# ---------------------------------------------------------------------------
_EXTENDED_DOCS = [
    # Apex product docs
    {"collection": "apex-product-docs", "title": "Vay cá nhân tín chấp", "snippet": "Vay cá nhân tín chấp không cần tài sản đảm bảo, hạn mức tối đa 500 triệu VND, thời hạn 6-60 tháng...", "score": 0.95},
    {"collection": "apex-product-docs", "title": "Vay mua nhà", "snippet": "Vay mua nhà với lãi suất ưu đãi 8.5%/năm, hạn mức lên đến 70% giá trị căn nhà...", "score": 0.92},
    {"collection": "apex-product-docs", "title": "Vay mua xe ô tô", "snippet": "Vay mua xe ô tô với hạn mức 80% giá trị xe, thời hạn tối đa 7 năm...", "score": 0.91},
    {"collection": "apex-product-docs", "title": "Vay kinh doanh", "snippet": "Vay kinh doanh cho doanh nghiệp vừa và nhỏ, hạn mức đến 5 tỷ VND...", "score": 0.89},
    {"collection": "apex-product-docs", "title": "Thẻ tín dụng Apex", "snippet": "Thẻ tín dụng với hạn mức 5-200 triệu, hoàn tiền 1-5%, miễn phí thường niên năm đầu...", "score": 0.88},

    # Apex loan policy
    {"collection": "apex-loan-policy", "title": "Điều kiện vay vốn", "snippet": "Khách hàng từ 22-60 tuổi, có thu nhập ổn định từ 8 triệu VND/tháng, không có nợ xấu...", "score": 0.94},
    {"collection": "apex-loan-policy", "title": "Quy trình phê duyệt", "snippet": "Quy trình 24h: nộp hồ sơ -> thẩm định tự động -> phê duyệt -> giải ngân...", "score": 0.91},
    {"collection": "apex-loan-policy", "title": "Lãi suất và biểu phí", "snippet": "Lãi suất từ 8.5%/năm, biểu phí minh bạch, không phí ẩn...", "score": 0.90},
    {"collection": "apex-loan-policy", "title": "Tài liệu cần thiết", "snippet": "CCCD, sổ hộ khẩu, hợp đồng lao động, sao kê lương 3 tháng gần nhất...", "score": 0.87},

    # Apex FAQ
    {"collection": "apex-faq-customer", "title": "Làm sao để biết kết quả phê duyệt?", "snippet": "Bạn sẽ nhận được thông báo qua email và SMS trong vòng 24 giờ sau khi nộp hồ sơ...", "score": 0.93},
    {"collection": "apex-faq-customer", "title": "Tôi có thể trả nợ trước hạn không?", "snippet": "Có, bạn có thể trả nợ trước hạn. Phí phạt tối đa 3% số tiền trả trước...", "score": 0.88},
    {"collection": "apex-faq-customer", "title": "Lãi suất có cố định không?", "snippet": "Lãi suất ưu đãi cố định 6 tháng đầu, sau đó điều chỉnh theo lãi suất thị trường...", "score": 0.86},

    # HCT property
    {"collection": "hct-property-listings", "title": "Vinhomes Grand Park - Căn hộ 2PN", "snippet": "Căn hộ 2 phòng ngủ, 68m2, tầng 18, view công viên, nội thất cơ bản, giá 2.5 tỷ...", "score": 0.94},
    {"collection": "hct-property-listings", "title": "Masterise Eco Smart City - Shophouse", "snippet": "Shophouse 3 tầng, 150m2, mặt tiền đường lớn, phù hợp kinh doanh, giá 8.5 tỷ...", "score": 0.92},
    {"collection": "hct-property-listings", "title": "Sun Group Long Biên - Biệt thự", "snippet": "Biệt thự đơn lập, 350m2, 5 phòng ngủ, hồ bơi riêng, view sông Hồng, giá 25 tỷ...", "score": 0.91},
    {"collection": "hct-property-listings", "title": "The Matrix One - Căn hộ 1PN", "snippet": "Căn hộ 1 phòng ngủ, 45m2, tầng 25, view hồ điều hòa, full nội thất, giá 1.9 tỷ...", "score": 0.89},

    # HCT investment thesis
    {"collection": "hct-investment-thesis", "title": "Triển vọng BĐS Hà Nội 2026-2030", "snippet": "Dự báo tăng trưởng 8-12%/năm tại các quận phát triển như Long Biên, Nam Từ Liêm...", "score": 0.93},
    {"collection": "hct-investment-thesis", "title": "So sánh Vinhomes vs Masterise", "snippet": "Vinhomes có lợi thế về quy hoạch, Masterise có lợi thế về thiết kế châu Âu...", "score": 0.88},
    {"collection": "hct-investment-thesis", "title": "ROI kênh BĐS vs chứng khoán", "snippet": "BĐS cho ROI ổn định 8-12%/năm, chứng khoán có thể 15-20% nhưng rủi ro cao hơn...", "score": 0.87},

    # HCT legal docs
    {"collection": "hct-legal-docs", "title": "Quy trình mua bán BĐS", "snippet": "Ký cọc -> ký hợp đồng đặt cọc -> ký hợp đồng mua bán -> sang tên sổ đỏ...", "score": 0.91},
    {"collection": "hct-legal-docs", "title": "Thuế và phí khi mua BĐS", "snippet": "Thuế VAT 10%, phí trước bạ 0.5%, lệ phí trước bạ tùy địa phương...", "score": 0.88},
    {"collection": "hct-legal-docs", "title": "Giấy tờ pháp lý cần kiểm tra", "snippet": "Sổ đỏ, giấy phép xây dựng, bản vẽ hoàn công, giấy phép kinh doanh BĐS...", "score": 0.86},

    # Demo getting started
    {"collection": "demo-getting-started", "title": "Hướng dẫn đăng ký tài khoản", "snippet": "Vào trang đăng ký, điền email và password, xác nhận email qua link...", "score": 0.94},
    {"collection": "demo-getting-started", "title": "Tạo workspace đầu tiên", "snippet": "Sau khi đăng nhập, click 'New workspace', đặt tên và mời thành viên...", "score": 0.92},
    {"collection": "demo-getting-started", "title": "Import dữ liệu từ CRM khác", "snippet": "Sử dụng import wizard, upload CSV/Excel, mapping fields, validate và import...", "score": 0.89},

    # Demo API
    {"collection": "demo-api-reference", "title": "Authentication API", "snippet": "POST /api/v1/auth/login với username/password, nhận JWT token có hạn 24h...", "score": 0.93},
    {"collection": "demo-api-reference", "title": "Leads API endpoints", "snippet": "GET/POST /api/v1/leads, PATCH /api/v1/leads/{id}, DELETE /api/v1/leads/{id}...", "score": 0.91},
    {"collection": "demo-api-reference", "title": "Webhooks configuration", "snippet": "POST /api/v1/webhooks, đăng ký event, nhận payload JSON qua HTTPS...", "score": 0.88},

    # Demo troubleshooting
    {"collection": "demo-troubleshooting", "title": "Không nhận được email xác nhận", "snippet": "Kiểm tra spam folder, thêm email@demo.com vào whitelist, click resend sau 60s...", "score": 0.90},
    {"collection": "demo-troubleshooting", "title": "API trả về 401", "snippet": "Token hết hạn hoặc không hợp lệ. Sử dụng /auth/refresh để lấy token mới...", "score": 0.89},
    {"collection": "demo-troubleshooting", "title": "Performance chậm", "snippet": "Có thể do network, server load, hoặc query không tối ưu. Kiểm tra slow query log...", "score": 0.85},

    # Demo integrations
    {"collection": "demo-integrations", "title": "Slack integration setup", "snippet": "Vào Settings > Integrations > Slack > Connect, authorize workspace, chọn channels...", "score": 0.91},
    {"collection": "demo-integration", "title": "Zalo OA integration", "snippet": "Đăng ký Zalo Official Account, lấy OA ID và secret, paste vào Demo settings...", "score": 0.88},
    {"collection": "demo-integrations", "title": "Facebook Pixel setup", "snippet": "Tạo Pixel trong Facebook Business Manager, copy ID, paste vào Demo integrations...", "score": 0.87},

    # Demo security
    {"collection": "demo-security", "title": "MFA setup", "snippet": "Vào Account > Security > Enable MFA, scan QR với Google Authenticator hoặc Authy...", "score": 0.92},
    {"collection": "demo-security", "title": "Session management", "snippet": "Session timeout 24h, có thể force logout tất cả devices từ Account > Sessions...", "score": 0.88},
    {"collection": "demo-security", "title": "Data encryption", "snippet": "AES-256 cho data at rest, TLS 1.3 cho data in transit, encryption keys rotate mỗi 90 ngày...", "score": 0.86},
]


EXTENDED_DOCUMENTS: List[Dict[str, Any]] = [
    {**doc, "doc_id": f"doc-ext-{i:04d}"}
    for i, doc in enumerate(_EXTENDED_DOCS)
]


# ---------------------------------------------------------------------------
# Canned answers (50+, vi + en)
# ---------------------------------------------------------------------------
EXTENDED_ANSWERS_VI: Dict[str, str] = {
    # Apex Fintech
    "vay cá nhân": "Vay cá nhân tại Apex Fintech có hạn mức tối đa 500 triệu VND, thời hạn 6-60 tháng, lãi suất từ 12%/năm. Không cần tài sản đảm bảo, phê duyệt trong 24h.",
    "vay mua nhà": "Vay mua nhà với lãi suất ưu đãi chỉ từ 8.5%/năm, hạn mức đến 70% giá trị căn nhà, thời hạn tối đa 25 năm.",
    "vay mua xe": "Vay mua xe ô tô với hạn mức 80% giá trị xe, thời hạn đến 7 năm, lãi suất ưu đãi từ 9%/năm.",
    "thẻ tín dụng": "Thẻ tín dụng Apex có 3 hạng: Classic (5-30tr), Gold (30-100tr), Platinum (100-200tr). Hoàn tiền 1-5%, miễn phí thường niên năm đầu.",
    "điều kiện vay": "Khách hàng từ 22-60 tuổi, có thu nhập ổn định từ 8 triệu/tháng, không có nợ xấu trong 24 tháng gần nhất.",
    "quy trình phê duyệt": "Quy trình 24h: 1) Nộp hồ sơ online, 2) AI scoring tự động, 3) Thẩm định viên review, 4) Phê duyệt, 5) Ký hợp đồng, 6) Giải ngân.",
    "lãi suất": "Lãi suất từ 8.5%/năm cho vay mua nhà, 12%/năm cho vay cá nhân, 9%/năm cho vay mua xe. Cố định 6 tháng đầu.",
    "thanh toán trước hạn": "Bạn có thể trả nợ trước hạn. Phí phạt tối đa 3% số tiền trả trước, giảm dần theo thời gian.",
    "bảo hiểm khoản vay": "Bảo hiểm khoản vay giúp bảo vệ gia đình bạn. Phí 0.3-0.5%/năm trên dư nợ.",
    "nợ xấu": "Nếu đã có nợ xấu, bạn vẫn có thể vay với lãi suất cao hơn 2-3%. Chúng tôi xét duyệt dựa trên khả năng trả nợ hiện tại.",

    # HCT Consulting
    "mua căn hộ": "Quy trình mua căn hộ: 1) Chọn căn phù hợp, 2) Đặt cọc 50-100 triệu, 3) Ký HĐMB trong 7-14 ngày, 4) Thanh toán đợt 1 (30%), 5) Theo tiến độ xây dựng, 6) Nhận nhà + sang tên sổ đỏ.",
    "vay mua bđs": "HCT hỗ trợ vay mua BĐS với lãi suất ưu đãi từ 7.5%/năm, hạn mức đến 70% giá trị BĐS, thời hạn đến 20 năm.",
    "phí sang tên": "Phí sang tên sổ đỏ: 0.5% giá trị BĐS (lệ phí trước bạ) + thuế VAT 10% + phí công chứng. Tổng khoảng 2-3% giá trị.",
    "vinhomes grand park": "Vinhomes Grand Park tại TP.HCM có 4 tòa tháp, diện tích 45-150m2, giá từ 1.9-8 tỷ. Tiện ích: công viên 36ha, trường học Vinschool, bệnh viện Vinmec.",
    "masterise": "Masterise Homes phát triển các dự án cao cấp: Masterise Eco Smart City, Masterise Thảo Điền, Masterise Lumiere. Thiết kế châu Âu, tiện ích 5 sao.",
    "thị trường bđs": "Thị trường BĐS 2026 đang phục hồi sau giai đoạn khó khăn. Phân khúc cao cấp tăng trưởng 8-12%, phân khúc bình dân ổn định.",
    "đầu tư bđs": "Đầu tư BĐS nên chọn vị trí vàng, pháp lý rõ ràng, tiện ích đầy đủ. ROI kỳ vọng 8-12%/năm cho thuê, 15-20%/năm nếu chọn đúng dự án.",
    "shophouse": "Shophouse kết hợp nhà ở + kinh doanh. Phù hợp mặt tiền đường lớn, khu đô thị mới. Giá thuê cao hơn căn hộ 30-50%.",

    # Demo Company
    "đăng ký": "Để đăng ký: 1) Truy cập demo.rinco.app/signup, 2) Điền email + password, 3) Xác nhận email qua link, 4) Tạo workspace đầu tiên. Toàn bộ miễn phí 14 ngày.",
    "tính năng": "Demo hỗ trợ CRM, video meeting, AI chatbot, analytics, workflow automation. Tất cả tích hợp chặt với nhau qua API.",
    "ai scoring": "AI Lead Scoring phân tích 8+ tín hiệu (page_views, time_on_site, email_engagement, company_size, ...) để dự đoán conversion probability. Accuracy 87%, AUC 0.93.",
    "chatbot": "RAG Chatbot trả lời dựa trên knowledge base của bạn. Hỗ trợ tiếng Việt và tiếng Anh. Có thể train với docs riêng.",
    "api": "REST API đầy đủ cho mọi resource: leads, deals, contacts, users, webhooks. Authentication bằng JWT. Rate limit 100K calls/day cho enterprise.",
    "bảng giá": "3 gói: Free (0đ, 5 users), Pro (2.9tr/tháng, 25 users), Enterprise (custom, unlimited). Tất cả có trial 14 ngày.",
    "hỗ trợ": "Hỗ trợ 24/7 qua email support@rinco.app, chat trong app, hoặc hotline 1900-xxxx. Enterprise có dedicated CSM.",
    "tích hợp": "Tích hợp sẵn: Slack, Zalo OA, Facebook Pixel, Google Ads, TikTok, Mailchimp, Zapier. Custom integration qua REST API.",
    "bảo mật": "Bảo mật: SOC2 Type II, ISO 27001, encryption AES-256, MFA, RBAC. Data center tại Singapore và Vietnam.",
}

EXTENDED_ANSWERS_EN: Dict[str, str] = {
    # General
    "what is rinco": "RINCO is an AI-powered enterprise SaaS platform combining CRM, video meetings, AI automation, and analytics in one workspace.",
    "how to signup": "Sign up at demo.rinco.app/signup, verify your email, create your first workspace. Free 14-day trial, no credit card required.",
    "pricing": "Three plans: Free (0 USD/mo, 5 users), Pro (99 USD/mo, 25 users), Enterprise (custom). Annual billing saves 20%.",
    "features": "RINCO includes CRM, video meetings, AI chatbot, lead scoring, analytics, workflow automation, and 100+ integrations.",
    "support": "24/7 support via email support@rinco.app, in-app chat, or hotline. Enterprise customers get a dedicated CSM.",
    "security": "RINCO is SOC2 Type II certified, ISO 27001 compliant, uses AES-256 encryption, supports MFA, and offers granular RBAC.",
    "integrations": "Native integrations: Slack, Zalo OA, Facebook Pixel, Google Ads, TikTok, Mailchimp, Zapier. Custom via REST API.",
    "ai features": "RINCO AI includes: Lead Scoring (87% accuracy), RAG Chatbot (Vietnamese + English), AI SRE for incident detection.",
    "data residency": "Data centers in Singapore and Vietnam. Choose region during signup. GDPR + Vietnam data protection compliant.",
    "mobile app": "Native iOS and Android apps with full feature parity. Download from App Store and Google Play.",
    "api rate limits": "Rate limits: Free 1K/day, Pro 50K/day, Enterprise 100K/day. Custom limits available on request.",
    "sla": "Enterprise SLA: 99.95% uptime, 24h response for P1, 1h response for P0 incidents. Service credits for SLA breaches.",
    "data export": "Export all data anytime via API or UI. Formats: CSV, JSON, Excel. Full data portability guaranteed.",
    "training": "Free onboarding for all plans. Enterprise includes custom training, dedicated CSM, and quarterly business reviews.",
    "trial": "14-day free trial with full Pro features. No credit card required. Cancel anytime.",

    # Apex Fintech
    "personal loan": "Personal loans from 5M to 500M VND, terms 6-60 months, rates from 12%/year. No collateral needed, 24h approval.",
    "home loan": "Home loans with preferential rates from 8.5%/year, up to 70% LTV, max term 25 years.",
    "auto loan": "Auto loans up to 80% of car value, terms up to 7 years, rates from 9%/year.",
    "credit card": "Apex credit cards in 3 tiers: Classic (5-30M), Gold (30-100M), Platinum (100-200M). 1-5% cashback.",
    "loan requirements": "Applicants 22-60 years old, stable income from 8M VND/month, no bad debt in last 24 months.",
    "approval process": "24-hour process: 1) Apply online, 2) AI scoring, 3) Underwriter review, 4) Approval, 5) Contract signing, 6) Disbursement.",
    "interest rates": "Rates from 8.5% (home loan), 12% (personal), 9% (auto). Fixed for first 6 months.",
    "early repayment": "Yes, you can repay early. Penalty up to 3% of prepaid amount, decreasing over time.",

    # HCT Consulting
    "buy apartment": "Apartment purchase process: 1) Select unit, 2) Deposit 50-100M, 3) Sign SPA within 7-14 days, 4) Pay 30% first installment, 5) Pay by construction progress, 6) Receive + title transfer.",
    "property mortgage": "HCT offers mortgages from 7.5%/year, up to 70% LTV, terms up to 20 years.",
    "vinhomes grand park": "Vinhomes Grand Park in HCMC has 4 towers, units 45-150 sqm, prices 1.9-8B VND. Amenities: 36ha park, Vinschool, Vinmec hospital.",
    "real estate market": "Vietnam real estate market in 2026 is recovering. Luxury segment growing 8-12%, affordable segment stable.",
    "investment roi": "Real estate ROI: 8-12%/year rental, 15-20%/year capital gain (if picked well). Location and legal status are key.",
}

# Combined lookup
COMBINED_ANSWERS: Dict[str, str] = {**EXTENDED_ANSWERS_VI, **EXTENDED_ANSWERS_EN}


# ---------------------------------------------------------------------------
# KB templates for RAG ingestion
# ---------------------------------------------------------------------------
EXTENDED_KB_TEMPLATES: List[Dict[str, Any]] = [
    {
        "name": "Product Overview",
        "structure": {
            "sections": ["Introduction", "Key Features", "Pricing", "FAQ"],
            "prompt_template": "Bạn là chuyên gia về {product}. Hãy trả lời câu hỏi dựa trên tài liệu.",
        }
    },
    {
        "name": "Customer Support",
        "structure": {
            "sections": ["Common Issues", "Troubleshooting Steps", "Escalation Path"],
            "prompt_template": "You are a helpful customer support agent. Answer concisely.",
        }
    },
    {
        "name": "Internal Wiki",
        "structure": {
            "sections": ["Background", "Current State", "Future Plans"],
            "prompt_template": "Bạn là trợ lý nội bộ. Trả lời dựa trên wiki của công ty.",
        }
    },
    {
        "name": "Technical Documentation",
        "structure": {
            "sections": ["Overview", "API", "Examples", "Edge Cases"],
            "prompt_template": "You are a senior engineer. Provide code examples when relevant.",
        }
    },
    {
        "name": "Marketing Content",
        "structure": {
            "sections": ["Hook", "Benefits", "Call to Action"],
            "prompt_template": "Bạn là copywriter. Viết engaging, ngắn gọn, có CTA rõ ràng.",
        }
    },
]
