/**
 * Admin Portal API client — typed wrappers around the RINCO platform
 * services. Mỗi call resolve base URL từ service-registry, attach
 * Authorization token từ localStorage nếu có.
 *
 * Mục tiêu: gom tất cả endpoint mà admin-portal gọi vào 1 file để
 * dễ audit & swap mock → real.
 */

import {
  apiGet,
  apiPost,
  apiPatch,
  apiPut,
  apiDelete,
  ApiError,
  type FetchOptions,
} from "@rinco/ui";
import type { ServiceName } from "@rinco/ui";

// ============================================================
// Dashboard / stats
// ============================================================
export interface AdminStats {
  totalTenants?: number;
  totalUsers?: number;
  totalLeads?: number;
  mrr?: number;
  leadsToday?: number;
  conversionRate?: number;
  pageviewsToday?: number;
  apiCallsMin?: number;
}

export interface ServiceHealth {
  id: string;
  name: string;
  status: "healthy" | "degraded" | "down" | "unknown";
  latency?: number;
  errorRate?: number;
  uptime?: number;
  region?: string;
  version?: string;
  cpu?: number;
  memory?: number;
}

export interface RecentActivity {
  id: string;
  timestamp: string;
  type: string;
  title: string;
  detail?: string;
  icon?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total?: number;
  page?: number;
  limit?: number;
}

// ============================================================
// Tenants (tenant-service:8082)
// ============================================================
export interface AdminTenant {
  id: string;
  slug: string;
  name: string;
  plan?: "free" | "starter" | "pro" | "enterprise";
  status?: "active" | "trial" | "suspended" | "pending";
  region?: string;
  users?: number;
  leads?: number;
  revenue?: number;
  conversionRate?: number;
  createdAt?: string;
  domains?: string[];
  color?: string;
  logo?: string;
  isolationMode?: "shared" | "isolated";
  vpsNodeId?: string;
  primaryContact?: {
    name?: string;
    email?: string;
    phone?: string;
  };
  settings?: Record<string, unknown>;
  branding?: Record<string, unknown>;
}

export interface CreateTenantPayload {
  slug: string;
  name: string;
  plan?: string;
  region?: string;
  primaryContact?: { name: string; email: string; phone?: string };
}

export interface UpdateTenantPayload extends Partial<CreateTenantPayload> {
  status?: string;
  branding?: Record<string, unknown>;
  settings?: Record<string, unknown>;
}

// ============================================================
// Users (crm-service:8083 + auth:8081)
// ============================================================
export interface AdminUser {
  id: string;
  email: string;
  name?: string;
  fullName?: string;
  role: string;
  tenantId?: string;
  tenantSlug?: string;
  tenantName?: string;
  status: "active" | "pending" | "suspended" | "disabled" | "locked";
  mfaEnabled?: boolean;
  lastLoginAt?: string | null;
  lastLoginIp?: string;
  createdAt: string;
}

export interface InviteUserPayload {
  email: string;
  name: string;
  role: string;
  tenantId?: string;
}

// ============================================================
// CRM (crm-service:8083)
// ============================================================
export interface CrmContact {
  id: string;
  name: string;
  email: string;
  phone?: string;
  company?: string;
  position?: string;
  status: string;
  source?: string;
  tenantId?: string;
  tenantName?: string;
  createdAt?: string;
  lastActivity?: string;
}

export interface CrmDeal {
  id: string;
  name: string;
  value: number;
  stage: string;
  contact?: string;
  probability?: number;
  expectedClose?: string;
  tenantId?: string;
  tenantName?: string;
}

// ============================================================
// Lead (lead-service:8085)
// ============================================================
export interface Lead {
  id: string;
  name: string;
  email?: string;
  phone?: string;
  source?: string;
  status?: string;
  score?: number;
  tenantId?: string;
  tenantSlug?: string;
  utm?: Record<string, string>;
  createdAt?: string;
}

// ============================================================
// Audit (observability:8096 + crm:8083)
// ============================================================
export interface AuditLogEntry {
  id: string;
  timestamp: string;
  actorEmail?: string;
  actorRole?: string;
  action: string;
  targetType?: string;
  targetId?: string;
  tenantSlug?: string;
  ipAddress?: string;
  status?: "success" | "failure";
  traceId?: string;
  metadata?: Record<string, unknown>;
}

// ============================================================
// Notifications (notification-service:8088)
// ============================================================
export interface AdminNotification {
  id: string;
  title: string;
  body: string;
  category: "critical" | "warning" | "info" | "marketing";
  channel: "in_app" | "email" | "telegram" | "push" | "sms";
  recipientCount: number;
  opened?: number;
  clicked?: number;
  status: "draft" | "scheduled" | "sent" | "failed";
  createdAt: string;
  sentAt?: string;
}

export interface NotificationTemplate {
  id: string;
  code: string;
  category: AdminNotification["category"];
  channel: AdminNotification["channel"];
  subject: string;
  bodyTemplate: string;
  variables?: { name: string; type: string; required: boolean }[];
  updatedAt: string;
}

// ============================================================
// Feature flags (dynamic-model-service:8084)
// ============================================================
export interface FeatureFlag {
  key: string;
  description: string;
  enabled: boolean;
  rolloutPercentage: number;
  category: "core" | "experimental" | "beta" | "deprecated";
  updatedAt: string;
  updatedBy?: string;
  tenantOverrides?: number;
}

// ============================================================
// Quorum (tenant-service:8082)
// ============================================================
export interface QuorumRequest {
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

// ============================================================
// System / observability (observability-service:8096)
// ============================================================
export interface SystemLogEntry {
  id: string;
  timestamp: string;
  level: "info" | "warn" | "error" | "debug";
  service: string;
  message: string;
  traceId?: string;
  metadata?: Record<string, unknown>;
}

export interface SystemMetric {
  name: string;
  timestamp: string;
  value: number;
  labels?: Record<string, string>;
}

export interface AnalyticsPoint {
  name: string;
  value: number;
  [key: string]: string | number | undefined;
}

// ============================================================
// Helper: build query string
// ============================================================
function qs(params?: Record<string, string | number | boolean | undefined | null>): string {
  if (!params) return "";
  const sp = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== "") sp.set(k, String(v));
  });
  const s = sp.toString();
  return s ? `?${s}` : "";
}

function opts(service: ServiceName, extra?: FetchOptions): FetchOptions {
  return { service, ...extra };
}

// ============================================================
// adminApi — single namespace chứa mọi endpoint
// ============================================================
export const adminApi = {
  // --- Dashboard stats ------------------------------------------
  getStats: () =>
    apiGet<{ data?: AdminStats }>("/v1/admin/stats", opts("observability")),

  getRecentActivity: (limit = 10) =>
    apiGet<{ data: RecentActivity[] }>(
      `/v1/admin/activity/recent${qs({ limit })}`,
      opts("crm"),
    ),

  // --- Service health (observability:8096) ---------------------
  getServicesHealth: () =>
    apiGet<{ data: ServiceHealth[] }>("/v1/health/services", opts("observability")),

  // --- Tenants (tenant:8082) -----------------------------------
  getTenants: (params?: { page?: number; limit?: number; search?: string; status?: string }) =>
    apiGet<{ data: AdminTenant[]; total?: number }>(
      `/v1/admin/tenants${qs(params)}`,
      opts("tenant"),
    ),
  getTenant: (id: string) =>
    apiGet<{ data: AdminTenant }>(`/v1/admin/tenants/${id}`, opts("tenant")),
  createTenant: (data: CreateTenantPayload) =>
    apiPost<AdminTenant>("/v1/admin/tenants", data, opts("tenant")),
  updateTenant: (id: string, data: UpdateTenantPayload) =>
    apiPatch<AdminTenant>(`/v1/admin/tenants/${id}`, data, opts("tenant")),
  suspendTenant: (id: string, reason: string) =>
    apiPost<AdminTenant>(`/v1/admin/tenants/${id}/suspend`, { reason }, opts("tenant")),
  activateTenant: (id: string) =>
    apiPost<AdminTenant>(`/v1/admin/tenants/${id}/activate`, undefined, opts("tenant")),
  deleteTenant: (id: string) =>
    apiDelete<{ ok: boolean }>(`/v1/admin/tenants/${id}`, opts("tenant")),

  // --- Users (crm:8083 + auth:8081) -----------------------------
  getUsers: (params?: { page?: number; limit?: number; search?: string; role?: string; status?: string }) =>
    apiGet<{ data: AdminUser[]; total?: number }>(
      `/v1/admin/users${qs(params)}`,
      opts("crm"),
    ),
  inviteUser: (data: InviteUserPayload) =>
    apiPost<AdminUser>("/v1/admin/users/invite", data, opts("auth")),
  revokeUser: (id: string) =>
    apiPost<{ ok: boolean }>(`/v1/admin/users/${id}/revoke`, undefined, opts("auth")),
  deleteUser: (id: string) =>
    apiDelete<{ ok: boolean }>(`/v1/admin/users/${id}`, opts("auth")),

  // --- CRM (crm:8083) -------------------------------------------
  getContacts: (params?: { page?: number; limit?: number; status?: string; tenant_id?: string }) =>
    apiGet<PaginatedResponse<CrmContact>>(
      `/v1/crm/contacts${qs(params)}`,
      opts("crm"),
    ),
  createContact: (data: Partial<CrmContact>) =>
    apiPost<CrmContact>("/v1/crm/contacts", data, opts("crm")),
  updateContact: (id: string, data: Partial<CrmContact>) =>
    apiPatch<CrmContact>(`/v1/crm/contacts/${id}`, data, opts("crm")),
  deleteContact: (id: string) =>
    apiDelete<{ ok: boolean }>(`/v1/crm/contacts/${id}`, opts("crm")),

  getDeals: (params?: { page?: number; limit?: number; stage?: string; tenant_id?: string }) =>
    apiGet<PaginatedResponse<CrmDeal>>(`/v1/crm/deals${qs(params)}`, opts("crm")),

  // --- Leads (lead:8085) ----------------------------------------
  getLeads: (params?: { page?: number; limit?: number; tenant_id?: string }) =>
    apiGet<PaginatedResponse<Lead>>(`/v1/leads${qs(params)}`, opts("lead")),

  // --- Analytics (observability:8096) ---------------------------
  getAnalytics: (params?: { from?: string; to?: string; metric?: string }) =>
    apiGet<{ data: AnalyticsPoint[] }>(`/v1/analytics/query${qs(params)}`, opts("observability")),
  getLeadsAnalytics: (params?: { from?: string; to?: string; tenant_id?: string }) =>
    apiGet<{ data: AnalyticsPoint[] }>(`/v1/analytics/leads${qs(params)}`, opts("observability")),
  getTrafficAnalytics: (params?: { from?: string; to?: string }) =>
    apiGet<{ data: AnalyticsPoint[] }>(`/v1/analytics/traffic${qs(params)}`, opts("observability")),

  // --- Audit (crm:8083 + observability:8096) -------------------
  getAuditLogs: (params?: {
    page?: number;
    limit?: number;
    actor_id?: string;
    action?: string;
    from?: string;
    to?: string;
  }) =>
    apiGet<PaginatedResponse<AuditLogEntry>>(
      `/v1/audit/logs${qs(params)}`,
      opts("crm"),
    ),

  // --- Notifications (notification:8088) -----------------------
  getNotifications: (params?: { status?: string; category?: string }) =>
    apiGet<PaginatedResponse<AdminNotification>>(
      `/v1/admin/notifications${qs(params)}`,
      opts("notification"),
    ),
  createNotification: (data: Partial<AdminNotification>) =>
    apiPost<AdminNotification>("/v1/admin/notifications", data, opts("notification")),

  // --- Notification templates (notification:8088) --------------
  getNotificationTemplates: () =>
    apiGet<{ data: NotificationTemplate[] }>(
      "/v1/admin/notification-templates",
      opts("notification"),
    ),
  createNotificationTemplate: (data: Partial<NotificationTemplate>) =>
    apiPost<NotificationTemplate>(
      "/v1/admin/notification-templates",
      data,
      opts("notification"),
    ),
  updateNotificationTemplate: (id: string, data: Partial<NotificationTemplate>) =>
    apiPatch<NotificationTemplate>(
      `/v1/admin/notification-templates/${id}`,
      data,
      opts("notification"),
    ),
  deleteNotificationTemplate: (id: string) =>
    apiDelete<{ ok: boolean }>(
      `/v1/admin/notification-templates/${id}`,
      opts("notification"),
    ),

  // --- Feature flags (dynamic-model:8084) ---------------------
  getFeatureFlags: () =>
    apiGet<{ data: FeatureFlag[] }>("/v1/admin/feature-flags", opts("dynamic-model")),
  toggleFeatureFlag: (key: string, enabled: boolean) =>
    apiPatch<FeatureFlag>(`/v1/admin/feature-flags/${key}`, { enabled }, opts("dynamic-model")),
  updateFeatureFlagRollout: (key: string, rolloutPercentage: number) =>
    apiPatch<FeatureFlag>(
      `/v1/admin/feature-flags/${key}`,
      { rolloutPercentage },
      opts("dynamic-model"),
    ),
  createFeatureFlag: (data: Partial<FeatureFlag>) =>
    apiPost<FeatureFlag>("/v1/admin/feature-flags", data, opts("dynamic-model")),

  // --- Quorum (tenant:8082) ------------------------------------
  getQuorumRequests: () =>
    apiGet<{ data: QuorumRequest[] }>("/v1/admin/quorum", opts("tenant")),
  createQuorumRequest: (data: Partial<QuorumRequest>) =>
    apiPost<QuorumRequest>("/v1/admin/quorum", data, opts("tenant")),
  signQuorumRequest: (id: string) =>
    apiPost<QuorumRequest>(`/v1/admin/quorum/${id}/sign`, undefined, opts("tenant")),
  rejectQuorumRequest: (id: string) =>
    apiPost<QuorumRequest>(`/v1/admin/quorum/${id}/reject`, undefined, opts("tenant")),

  // --- System / observability (observability:8096) ------------
  getServiceLogs: (params?: {
    service?: string;
    level?: string;
    search?: string;
    limit?: number;
  }) =>
    apiGet<PaginatedResponse<SystemLogEntry>>(
      `/v1/logs/query${qs(params)}`,
      opts("observability"),
    ),
  getMetrics: (params?: { from?: string; to?: string; service?: string }) =>
    apiGet<{ data: SystemMetric[] }>(`/v1/metrics/query${qs(params)}`, opts("observability")),

  // --- Settings (tenant:8082) ---------------------------------
  getPlatformSettings: () =>
    apiGet<{ data: Record<string, unknown> }>("/v1/admin/settings/platform", opts("tenant")),
  updatePlatformSettings: (data: Record<string, unknown>) =>
    apiPut<{ data: Record<string, unknown> }>(
      "/v1/admin/settings/platform",
      data,
      opts("tenant"),
    ),

  // --- Auth (auth:8081) ---------------------------------------
  login: (email: string, password: string) =>
    apiPost<{ token: string }>("/v1/auth/login", { email, password }, opts("auth")),
  logout: () =>
    apiPost<{ ok: boolean }>("/v1/auth/logout", undefined, opts("auth")),
};

export { ApiError };
