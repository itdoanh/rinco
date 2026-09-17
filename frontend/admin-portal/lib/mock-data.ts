/**
 * Mock data for admin-portal — used as graceful degradation when
 * backend APIs are unavailable. In production these values come from
 * admin-bff endpoints; the dashboard falls back to these so the UI
 * remains demoable without a backend.
 */

export type TenantStatus = "active" | "trial" | "suspended" | "pending";
export type TenantPlan = "free" | "starter" | "pro" | "enterprise";

export interface MockTenant {
  id: string;
  slug: string;
  name: string;
  plan: TenantPlan;
  status: TenantStatus;
  region: string;
  users: number;
  leads: number;
  revenue: number;
  conversionRate: number;
  createdAt: string;
  domains: string[];
  logo: string;
  color: string;
  isolationMode: "shared" | "isolated";
  vpsNodeId?: string;
  primaryContact: { name: string; email: string; phone: string };
}

export const mockTenants: MockTenant[] = [
  {
    id: "tn_001",
    slug: "apexfintech",
    name: "Apex Fintech",
    plan: "enterprise",
    status: "active",
    region: "vn-sg",
    users: 145,
    leads: 2847,
    revenue: 125_000_000,
    conversionRate: 12.5,
    createdAt: "2025-01-15T00:00:00Z",
    domains: ["apexfintech.com", "tuvan.apexfintech.vn"],
    logo: "/logos/apexfintech.png",
    color: "#0066CC",
    isolationMode: "isolated",
    vpsNodeId: "vps_hcm_001",
    primaryContact: { name: "Nguyen Van A", email: "ops@apex.vn", phone: "0987654321" },
  },
  {
    id: "tn_002",
    slug: "vietnamrealty",
    name: "Vietnam Realty",
    plan: "pro",
    status: "active",
    region: "vn-hn",
    users: 87,
    leads: 1924,
    revenue: 48_000_000,
    conversionRate: 9.8,
    createdAt: "2025-03-22T00:00:00Z",
    domains: ["vietnamrealty.vn", "batdongsan.vietnamrealty.vn"],
    logo: "/logos/vietnamrealty.png",
    color: "#F59E0B",
    isolationMode: "shared",
    primaryContact: { name: "Tran Thi B", email: "hello@vietnamrealty.vn", phone: "0909123456" },
  },
  {
    id: "tn_003",
    slug: "saigonhealth",
    name: "Saigon Health Group",
    plan: "enterprise",
    status: "active",
    region: "vn-sg",
    users: 312,
    leads: 5621,
    revenue: 220_000_000,
    conversionRate: 15.2,
    createdAt: "2024-11-08T00:00:00Z",
    domains: ["saigonhealth.vn", "datlich.saigonhealth.vn"],
    logo: "/logos/saigonhealth.png",
    color: "#10B981",
    isolationMode: "isolated",
    vpsNodeId: "vps_hcm_002",
    primaryContact: { name: "Le Van C", email: "admin@saigonhealth.vn", phone: "0912345678" },
  },
  {
    id: "tn_004",
    slug: "eduviet",
    name: "EduViet Academy",
    plan: "pro",
    status: "trial",
    region: "vn-hn",
    users: 45,
    leads: 412,
    revenue: 0,
    conversionRate: 6.4,
    createdAt: "2026-08-30T00:00:00Z",
    domains: ["eduviet.edu.vn"],
    logo: "/logos/eduviet.png",
    color: "#8B5CF6",
    isolationMode: "shared",
    primaryContact: { name: "Pham Thi D", email: "support@eduviet.edu.vn", phone: "0934567890" },
  },
  {
    id: "tn_005",
    slug: "megashop",
    name: "MegaShop Vietnam",
    plan: "enterprise",
    status: "active",
    region: "vn-sg",
    users: 528,
    leads: 12450,
    revenue: 380_000_000,
    conversionRate: 18.7,
    createdAt: "2024-06-14T00:00:00Z",
    domains: ["megashop.vn", "shop.megashop.vn"],
    logo: "/logos/megashop.png",
    color: "#EF4444",
    isolationMode: "isolated",
    vpsNodeId: "vps_hcm_003",
    primaryContact: { name: "Hoang Van E", email: "tech@megashop.vn", phone: "0945678901" },
  },
  {
    id: "tn_006",
    slug: "greentech",
    name: "GreenTech Solutions",
    plan: "starter",
    status: "pending",
    region: "vn-hn",
    users: 12,
    leads: 89,
    revenue: 2_900_000,
    conversionRate: 3.1,
    createdAt: "2026-09-10T00:00:00Z",
    domains: ["greentech.vn"],
    logo: "/logos/greentech.png",
    color: "#06B6D4",
    isolationMode: "shared",
    primaryContact: { name: "Vu Thi F", email: "hello@greentech.vn", phone: "0956789012" },
  },
  {
    id: "tn_007",
    slug: "logistics-pro",
    name: "Logistics Pro",
    plan: "pro",
    status: "active",
    region: "vn-sg",
    users: 67,
    leads: 892,
    revenue: 32_000_000,
    conversionRate: 7.8,
    createdAt: "2025-09-05T00:00:00Z",
    domains: ["logistics-pro.vn"],
    logo: "/logos/logistics-pro.png",
    color: "#F97316",
    isolationMode: "shared",
    primaryContact: { name: "Do Van G", email: "ops@logistics-pro.vn", phone: "0967890123" },
  },
  {
    id: "tn_008",
    slug: "fintech-hub",
    name: "Fintech Hub Asia",
    plan: "enterprise",
    status: "suspended",
    region: "vn-sg",
    users: 198,
    leads: 3201,
    revenue: 95_000_000,
    conversionRate: 11.2,
    createdAt: "2025-05-18T00:00:00Z",
    domains: ["fintechhub.asia"],
    logo: "/logos/fintech-hub.png",
    color: "#6366F1",
    isolationMode: "isolated",
    vpsNodeId: "vps_hcm_004",
    primaryContact: { name: "Bui Thi H", email: "admin@fintechhub.asia", phone: "0978901234" },
  },
];

export interface MockStats {
  totalTenants: number;
  totalUsers: number;
  totalLeads: number;
  monthlyRevenue: number;
  growthRate: number;
  activeUsers: number;
  systemHealth: "healthy" | "degraded" | "down";
  totalPageviews: number;
  apiCallsPerMin: number;
  conversionRate: number;
}

export const mockStats: MockStats = {
  totalTenants: 12,
  totalUsers: 1247,
  totalLeads: 24819,
  monthlyRevenue: 458_000_000,
  growthRate: 18.5,
  activeUsers: 892,
  systemHealth: "healthy",
  totalPageviews: 184_532,
  apiCallsPerMin: 4521,
  conversionRate: 12.4,
};

export interface TimeSeriesPoint {
  name: string;
  value: number;
  [key: string]: string | number;
}

export const mockAnalytics = {
  leadsByDay: [
    { name: "T2", value: 145 },
    { name: "T3", value: 178 },
    { name: "T4", value: 132 },
    { name: "T5", value: 198 },
    { name: "T6", value: 234 },
    { name: "T7", value: 189 },
    { name: "CN", value: 156 },
  ] as TimeSeriesPoint[],
  revenueByMonth: [
    { name: "T1", value: 245_000_000 },
    { name: "T2", value: 278_000_000 },
    { name: "T3", value: 312_000_000 },
    { name: "T4", value: 298_000_000 },
    { name: "T5", value: 356_000_000 },
    { name: "T6", value: 402_000_000 },
    { name: "T7", value: 458_000_000 },
    { name: "T8", value: 478_000_000 },
    { name: "T9", value: 512_000_000 },
  ] as TimeSeriesPoint[],
  conversionBySource: [
    { name: "Facebook", value: 28 },
    { name: "Google", value: 35 },
    { name: "TikTok", value: 18 },
    { name: "Zalo", value: 12 },
    { name: "Direct", value: 7 },
  ] as TimeSeriesPoint[],
  trafficByHour: Array.from({ length: 24 }, (_, i) => ({
    name: `${i}h`,
    value: Math.round(50 + Math.sin((i / 24) * Math.PI * 2) * 30 + Math.random() * 20),
  })) as TimeSeriesPoint[],
  leadsByTenant: mockTenants
    .filter((t) => t.status === "active")
    .map((t) => ({ name: t.name, value: t.leads })) as TimeSeriesPoint[],
};

export interface MockAuditLog {
  id: string;
  timestamp: string;
  actorEmail: string;
  actorRole: string;
  action: string;
  targetType: string;
  targetId: string;
  tenantSlug: string;
  ipAddress: string;
  status: "success" | "failure";
}

const now = Date.now();
const hours = (n: number) => new Date(now - n * 60 * 60 * 1000).toISOString();

export const mockAuditLogs: MockAuditLog[] = Array.from({ length: 50 }, (_, i) => {
  const actions = [
    { action: "tenant.create", targetType: "tenant" },
    { action: "tenant.update", targetType: "tenant" },
    { action: "tenant.suspend", targetType: "tenant" },
    { action: "user.invite", targetType: "user" },
    { action: "user.role_change", targetType: "user" },
    { action: "feature_flag.toggle", targetType: "feature_flag" },
    { action: "notification.broadcast", targetType: "notification" },
    { action: "quorum.create", targetType: "quorum" },
    { action: "quorum.sign", targetType: "quorum" },
    { action: "admin.login", targetType: "session" },
    { action: "domain.verify", targetType: "domain" },
    { action: "service.restart", targetType: "service" },
  ];
  const actors = [
    { email: "owner@rinco.app", role: "OWNER" },
    { email: "sre@rinco.app", role: "SRE_ADMIN" },
    { email: "security@rinco.app", role: "SECURITY_ADMIN" },
    { email: "finance@rinco.app", role: "FINANCE_ADMIN" },
    { email: "support@rinco.app", role: "SUPPORT_ADMIN" },
  ];
  const pick = <T,>(arr: T[], n: number) => arr[n % arr.length];
  const a = pick(actions, i);
  const actor = pick(actors, i);
  return {
    id: `log_${i.toString().padStart(4, "0")}`,
    timestamp: hours(i * 0.5),
    actorEmail: actor.email,
    actorRole: actor.role,
    action: a.action,
    targetType: a.targetType,
    targetId: `tgt_${(i * 13).toString(36)}`,
    tenantSlug: pick(mockTenants, i).slug,
    ipAddress: `10.${i % 256}.${(i * 7) % 256}.${(i * 11) % 256}`,
    status: i % 9 === 0 ? "failure" : "success",
  };
});

export interface MockNotification {
  id: string;
  title: string;
  body: string;
  category: "critical" | "warning" | "info" | "marketing";
  channel: "in_app" | "email" | "telegram" | "push" | "sms";
  recipientCount: number;
  opened: number;
  clicked: number;
  status: "draft" | "scheduled" | "sent" | "failed";
  createdAt: string;
  sentAt?: string;
}

export const mockNotifications: MockNotification[] = [
  {
    id: "nt_001",
    title: "Bảo trì hệ thống định kỳ",
    body: "Hệ thống sẽ bảo trì từ 02:00 - 04:00 ngày mai.",
    category: "warning",
    channel: "email",
    recipientCount: 1247,
    opened: 892,
    clicked: 145,
    status: "sent",
    createdAt: hours(12),
    sentAt: hours(11),
  },
  {
    id: "nt_002",
    title: "Tính năng mới: AI Insights",
    body: "Chúng tôi vừa ra mắt tính năng phân tích AI cho báo cáo.",
    category: "marketing",
    channel: "in_app",
    recipientCount: 1247,
    opened: 678,
    clicked: 234,
    status: "sent",
    createdAt: hours(36),
    sentAt: hours(35),
  },
  {
    id: "nt_003",
    title: "Cảnh báo: CPU cao",
    body: "Service auth-service đang sử dụng 92% CPU.",
    category: "critical",
    channel: "telegram",
    recipientCount: 12,
    opened: 12,
    clicked: 8,
    status: "sent",
    createdAt: hours(2),
    sentAt: hours(2),
  },
  {
    id: "nt_004",
    title: "Khuyến mãi cuối năm",
    body: "Giảm 30% gói Enterprise cho đăng ký mới.",
    category: "marketing",
    channel: "email",
    recipientCount: 0,
    opened: 0,
    clicked: 0,
    status: "draft",
    createdAt: hours(1),
  },
  {
    id: "nt_005",
    title: "Quorum pending approval",
    body: "Yêu cầu xóa tenant apexfintech cần 1 chữ ký nữa.",
    category: "critical",
    channel: "in_app",
    recipientCount: 3,
    opened: 3,
    clicked: 2,
    status: "sent",
    createdAt: hours(0.5),
    sentAt: hours(0.5),
  },
  ...Array.from({ length: 15 }, (_, i) => ({
    id: `nt_${(i + 6).toString().padStart(3, "0")}`,
    title: `Thông báo ${i + 6}`,
    body: "Nội dung thông báo mẫu cho demo.",
    category: (["info", "warning", "marketing", "critical"] as const)[i % 4],
    channel: (["in_app", "email", "telegram", "push"] as const)[i % 4],
    recipientCount: Math.floor(Math.random() * 1000) + 50,
    opened: Math.floor(Math.random() * 800) + 20,
    clicked: Math.floor(Math.random() * 200),
    status: (["sent", "sent", "sent", "scheduled", "draft"] as const)[i % 5],
    createdAt: hours(i * 2),
    sentAt: hours(i * 2 - 0.5),
  })),
];

export interface MockServiceHealth {
  id: string;
  name: string;
  status: "healthy" | "degraded" | "down" | "unknown";
  latency?: number;
  errorRate?: number;
  uptime?: number;
  region: string;
  version: string;
  cpu: number;
  memory: number;
}

export const mockServiceHealth: MockServiceHealth[] = [
  { id: "auth", name: "Auth Service", status: "healthy", latency: 45, uptime: 99.99, region: "vn-sg", version: "1.4.2", cpu: 32, memory: 48 },
  { id: "tenant", name: "Tenant Service", status: "healthy", latency: 32, uptime: 99.95, region: "vn-sg", version: "1.3.0", cpu: 28, memory: 41 },
  { id: "crm", name: "CRM Service", status: "healthy", latency: 28, uptime: 99.90, region: "vn-sg", version: "2.1.0", cpu: 45, memory: 56 },
  { id: "lead", name: "Lead Service", status: "healthy", latency: 55, uptime: 99.85, region: "vn-hn", version: "1.7.1", cpu: 38, memory: 52 },
  { id: "landing", name: "Landing Service", status: "healthy", latency: 42, uptime: 99.92, region: "vn-sg", version: "1.2.4", cpu: 22, memory: 35 },
  { id: "chat", name: "Chat Engine", status: "degraded", latency: 180, errorRate: 2.5, uptime: 99.20, region: "vn-sg", version: "3.0.0", cpu: 78, memory: 82 },
  { id: "webrtc", name: "WebRTC SFU", status: "healthy", latency: 35, uptime: 99.98, region: "vn-hn", version: "2.4.0", cpu: 52, memory: 64 },
  { id: "recording", name: "Recording Service", status: "healthy", latency: 48, uptime: 99.88, region: "vn-sg", version: "1.1.2", cpu: 41, memory: 58 },
  { id: "lead-scoring", name: "Lead Scoring", status: "healthy", latency: 120, uptime: 99.50, region: "vn-hn", version: "0.9.5", cpu: 65, memory: 72 },
  { id: "rag", name: "RAG Chatbot", status: "healthy", latency: 250, uptime: 99.70, region: "vn-sg", version: "1.0.1", cpu: 58, memory: 68 },
  { id: "ai-sre", name: "AI SRE", status: "healthy", latency: 180, uptime: 99.60, region: "vn-sg", version: "0.5.0", cpu: 48, memory: 62 },
  { id: "stt", name: "STT Service", status: "down", errorRate: 100, uptime: 0, region: "vn-hn", version: "1.3.0", cpu: 0, memory: 0 },
  { id: "admin-gateway", name: "Admin Gateway", status: "healthy", latency: 22, uptime: 99.99, region: "vn-sg", version: "2.0.0", cpu: 18, memory: 28 },
];

export interface MockFeatureFlag {
  key: string;
  description: string;
  enabled: boolean;
  rolloutPercentage: number;
  category: "core" | "experimental" | "beta" | "deprecated";
  updatedAt: string;
  updatedBy: string;
  tenantOverrides: number;
}

export const mockFeatureFlags: MockFeatureFlag[] = [
  { key: "ai_insights_v2", description: "AI Insights thế hệ 2 với GPT-4 Turbo", enabled: true, rolloutPercentage: 100, category: "core", updatedAt: hours(48), updatedBy: "owner@rinco.app", tenantOverrides: 3 },
  { key: "voice_cloning", description: "Clone giọng nói từ mẫu 30s", enabled: true, rolloutPercentage: 50, category: "experimental", updatedAt: hours(72), updatedBy: "sre@rinco.app", tenantOverrides: 0 },
  { key: "auto_translate_v3", description: "Tự động dịch đa ngôn ngữ với chất lượng cao", enabled: true, rolloutPercentage: 80, category: "beta", updatedAt: hours(24), updatedBy: "owner@rinco.app", tenantOverrides: 5 },
  { key: "realtime_collab", description: "Cộng tác thời gian thực trong CRM", enabled: false, rolloutPercentage: 0, category: "experimental", updatedAt: hours(120), updatedBy: "owner@rinco.app", tenantOverrides: 0 },
  { key: "advanced_analytics", description: "Dashboard analytics nâng cao với cohort", enabled: true, rolloutPercentage: 100, category: "core", updatedAt: hours(168), updatedBy: "owner@rinco.app", tenantOverrides: 2 },
  { key: "multi_currency", description: "Hỗ trợ đa tiền tệ cho billing", enabled: true, rolloutPercentage: 100, category: "core", updatedAt: hours(200), updatedBy: "finance@rinco.app", tenantOverrides: 0 },
  { key: "webrtc_simulcast", description: "WebRTC simulcast cho chất lượng HD", enabled: true, rolloutPercentage: 90, category: "beta", updatedAt: hours(48), updatedBy: "sre@rinco.app", tenantOverrides: 1 },
  { key: "rag_enterprise", description: "RAG với private documents cho Enterprise", enabled: false, rolloutPercentage: 0, category: "experimental", updatedAt: hours(96), updatedBy: "owner@rinco.app", tenantOverrides: 0 },
  { key: "mobile_pwa_v2", description: "Mobile PWA thế hệ 2", enabled: true, rolloutPercentage: 100, category: "core", updatedAt: hours(300), updatedBy: "owner@rinco.app", tenantOverrides: 0 },
  { key: "audit_export_v2", description: "Export audit log ra S3 Glacier", enabled: true, rolloutPercentage: 100, category: "core", updatedAt: hours(72), updatedBy: "security@rinco.app", tenantOverrides: 0 },
  { key: "embedded_widgets", description: "Embed CRM widget vào site khác", enabled: false, rolloutPercentage: 0, category: "experimental", updatedAt: hours(48), updatedBy: "owner@rinco.app", tenantOverrides: 0 },
  { key: "white_label_mobile", description: "White label mobile app", enabled: false, rolloutPercentage: 0, category: "beta", updatedAt: hours(24), updatedBy: "owner@rinco.app", tenantOverrides: 0 },
  { key: "legacy_chat_v1", description: "Chat engine thế hệ cũ (deprecated)", enabled: false, rolloutPercentage: 0, category: "deprecated", updatedAt: hours(720), updatedBy: "owner@rinco.app", tenantOverrides: 0 },
  { key: "sso_oauth", description: "SSO qua OAuth providers", enabled: true, rolloutPercentage: 100, category: "core", updatedAt: hours(168), updatedBy: "security@rinco.app", tenantOverrides: 1 },
  { key: "vector_search_v3", description: "Vector search với HNSW index", enabled: true, rolloutPercentage: 75, category: "beta", updatedAt: hours(36), updatedBy: "sre@rinco.app", tenantOverrides: 2 },
];

export interface MockSystemLog {
  id: string;
  timestamp: string;
  level: "info" | "warn" | "error" | "debug";
  service: string;
  message: string;
  traceId?: string;
  metadata?: Record<string, unknown>;
}

export const mockSystemLogs: MockSystemLog[] = Array.from({ length: 100 }, (_, i) => {
  const services = ["auth-service", "tenant-service", "crm-service", "lead-service", "chat-engine", "webrtc-sfu", "recording-service", "lead-scoring", "rag-chatbot", "stt-service"];
  const levels: MockSystemLog["level"][] = ["info", "info", "info", "info", "debug", "warn", "error", "info"];
  const messages = [
    "User login successful",
    "Lead submitted from contact form",
    "Database query executed",
    "Cache miss, fetching from DB",
    "WebRTC connection established",
    "High latency detected",
    "Connection failed",
    "AI inference completed",
    "Backup completed successfully",
    "Service health check passed",
    "Tenant configuration updated",
    "Feature flag toggled",
  ];
  const svc = services[i % services.length];
  const lvl = levels[i % levels.length];
  return {
    id: `sys_${i.toString().padStart(4, "0")}`,
    timestamp: hours(i * 0.1),
    level: lvl,
    service: svc,
    message: messages[i % messages.length],
    traceId: i % 3 === 0 ? `tr_${(i * 31).toString(36)}${(i * 17).toString(36)}` : undefined,
    metadata:
      lvl === "error"
        ? { error: "Connection refused", code: "ECONNREFUSED", retry_count: 3 }
        : lvl === "warn"
          ? { latency_ms: 480, threshold: 300 }
          : undefined,
  };
});

export interface MockNotificationTemplate {
  id: string;
  code: string;
  category: "critical" | "warning" | "info" | "marketing";
  channel: "in_app" | "email" | "telegram" | "push" | "sms";
  subject: string;
  bodyTemplate: string;
  variables: { name: string; type: string; required: boolean }[];
  updatedAt: string;
}

export const mockNotificationTemplates: MockNotificationTemplate[] = [
  {
    id: "tpl_001",
    code: "maintenance.scheduled",
    category: "warning",
    channel: "email",
    subject: "Thông báo bảo trì hệ thống {{.Date}}",
    bodyTemplate: "Hệ thống sẽ bảo trì từ {{.StartTime}} đến {{.EndTime}}. Mong quý khách thông cảm.",
    variables: [
      { name: "Date", type: "date", required: true },
      { name: "StartTime", type: "time", required: true },
      { name: "EndTime", type: "time", required: true },
    ],
    updatedAt: hours(72),
  },
  {
    id: "tpl_002",
    code: "billing.invoice_failed",
    category: "critical",
    channel: "email",
    subject: "Thanh toán không thành công",
    bodyTemplate: "Hóa đơn {{.InvoiceID}} không thể thanh toán. Vui lòng cập nhật phương thức thanh toán.",
    variables: [{ name: "InvoiceID", type: "string", required: true }],
    updatedAt: hours(120),
  },
  {
    id: "tpl_003",
    code: "tenant.welcome",
    category: "info",
    channel: "email",
    subject: "Chào mừng {{.TenantName}} đến với RINCO!",
    bodyTemplate: "Chào {{.AdminName}}, tài khoản của bạn đã sẵn sàng tại {{.URL}}.",
    variables: [
      { name: "TenantName", type: "string", required: true },
      { name: "AdminName", type: "string", required: true },
      { name: "URL", type: "url", required: true },
    ],
    updatedAt: hours(168),
  },
  {
    id: "tpl_004",
    code: "alert.cpu_high",
    category: "critical",
    channel: "telegram",
    subject: "⚠️ CPU cao: {{.ServiceName}}",
    bodyTemplate: "Service {{.ServiceName}} đang sử dụng {{.CPU}}% CPU (threshold: 80%).",
    variables: [
      { name: "ServiceName", type: "string", required: true },
      { name: "CPU", type: "number", required: true },
    ],
    updatedAt: hours(48),
  },
  {
    id: "tpl_005",
    code: "feature.released",
    category: "marketing",
    channel: "in_app",
    subject: "🚀 Tính năng mới: {{.FeatureName}}",
    bodyTemplate: "Khám phá tính năng {{.FeatureName}} - {{.Description}}",
    variables: [
      { name: "FeatureName", type: "string", required: true },
      { name: "Description", type: "text", required: true },
    ],
    updatedAt: hours(24),
  },
  {
    id: "tpl_006",
    code: "quorum.pending",
    category: "critical",
    channel: "in_app",
    subject: "Quorum pending: {{.Action}}",
    bodyTemplate: "{{.Initiator}} yêu cầu phê duyệt cho {{.Action}}. Lý do: {{.Reason}}",
    variables: [
      { name: "Action", type: "string", required: true },
      { name: "Initiator", type: "string", required: true },
      { name: "Reason", type: "text", required: true },
    ],
    updatedAt: hours(12),
  },
];

export interface MockQuorumRequest {
  id: string;
  action: string;
  payload: Record<string, unknown>;
  initiatorEmail: string;
  reason: string;
  requiredSignatures: number;
  collectedSignatures: { email: string; signedAt: string }[];
  status: "pending" | "approved" | "rejected" | "expired";
  expiresAt: string;
  createdAt: string;
}

export const mockQuorumRequests: MockQuorumRequest[] = [
  {
    id: "qr_001",
    action: "tenant.delete",
    payload: { tenant_id: "tn_008", tenant_slug: "fintech-hub" },
    initiatorEmail: "owner@rinco.app",
    reason: "Tenant vi phạm ToS mục 5.2 — yêu cầu GDPR delete sau 30 ngày grace.",
    requiredSignatures: 2,
    collectedSignatures: [
      { email: "owner@rinco.app", signedAt: hours(0.2) },
    ],
    status: "pending",
    expiresAt: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
    createdAt: hours(0.3),
  },
  {
    id: "qr_002",
    action: "gateway.global_config",
    payload: { config_key: "rate_limit_global", new_value: "10000/min" },
    initiatorEmail: "sre@rinco.app",
    reason: "Tăng rate limit toàn cầu để xử lý traffic spike Black Friday.",
    requiredSignatures: 2,
    collectedSignatures: [
      { email: "sre@rinco.app", signedAt: hours(2) },
      { email: "owner@rinco.app", signedAt: hours(1.5) },
    ],
    status: "approved",
    expiresAt: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
    createdAt: hours(2.5),
  },
  {
    id: "qr_003",
    action: "dns.master_change",
    payload: { from: "ns1.old.rinco.app", to: "ns1.new.rinco.app" },
    initiatorEmail: "security@rinco.app",
    reason: "Rotate DNS master để tăng cường bảo mật theo yêu cầu quarterly.",
    requiredSignatures: 2,
    collectedSignatures: [
      { email: "security@rinco.app", signedAt: hours(8) },
    ],
    status: "pending",
    expiresAt: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
    createdAt: hours(8.5),
  },
  {
    id: "qr_004",
    action: "tenant.delete",
    payload: { tenant_id: "tn_old_044", tenant_slug: "old-demo" },
    initiatorEmail: "owner@rinco.app",
    reason: "Demo account hết hạn, dọn dẹp resource.",
    requiredSignatures: 2,
    collectedSignatures: [
      { email: "owner@rinco.app", signedAt: hours(24) },
      { email: "security@rinco.app", signedAt: hours(22) },
    ],
    status: "approved",
    expiresAt: hours(19),
    createdAt: hours(24),
  },
];

export interface MockAdmin {
  id: string;
  email: string;
  fullName: string;
  role: "OWNER" | "SRE_ADMIN" | "SECURITY_ADMIN" | "SUPPORT_ADMIN" | "FINANCE_ADMIN" | "READONLY_VIEWER";
  status: "active" | "disabled" | "locked";
  mfaEnabled: boolean;
  lastLoginAt?: string;
  lastLoginIp?: string;
  createdAt: string;
}

export const mockAdmins: MockAdmin[] = [
  { id: "ad_001", email: "owner@rinco.app", fullName: "Nguyen Van Owner", role: "OWNER", status: "active", mfaEnabled: true, lastLoginAt: hours(0.5), lastLoginIp: "10.0.0.1", createdAt: "2024-01-01T00:00:00Z" },
  { id: "ad_002", email: "sre@rinco.app", fullName: "Tran Van SRE", role: "SRE_ADMIN", status: "active", mfaEnabled: true, lastLoginAt: hours(2), lastLoginIp: "10.0.0.2", createdAt: "2024-02-15T00:00:00Z" },
  { id: "ad_003", email: "security@rinco.app", fullName: "Le Thi Security", role: "SECURITY_ADMIN", status: "active", mfaEnabled: true, lastLoginAt: hours(4), lastLoginIp: "10.0.0.3", createdAt: "2024-03-20T00:00:00Z" },
  { id: "ad_004", email: "support@rinco.app", fullName: "Pham Van Support", role: "SUPPORT_ADMIN", status: "active", mfaEnabled: true, lastLoginAt: hours(6), lastLoginIp: "10.0.0.4", createdAt: "2024-04-10T00:00:00Z" },
  { id: "ad_005", email: "finance@rinco.app", fullName: "Hoang Thi Finance", role: "FINANCE_ADMIN", status: "active", mfaEnabled: true, lastLoginAt: hours(12), lastLoginIp: "10.0.0.5", createdAt: "2024-05-05T00:00:00Z" },
  { id: "ad_006", email: "viewer@rinco.app", fullName: "Vu Van Viewer", role: "READONLY_VIEWER", status: "active", mfaEnabled: false, lastLoginAt: hours(48), lastLoginIp: "10.0.0.6", createdAt: "2024-06-01T00:00:00Z" },
  { id: "ad_007", email: "disabled@rinco.app", fullName: "Do Van Disabled", role: "SUPPORT_ADMIN", status: "disabled", mfaEnabled: true, createdAt: "2024-07-15T00:00:00Z" },
];
