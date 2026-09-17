const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

interface FetchOptions extends RequestInit {
  tenantId?: string;
  timeout?: number;
}

class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: unknown
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function apiClient<T = unknown>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<T> {
  const { headers = {}, ...fetchOptions } = options;

  const requestHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    ...(headers as Record<string, string>),
  };

  // Add auth token if available
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("auth_token");
    if (token) {
      requestHeaders["Authorization"] = `Bearer ${token}`;
    }
  }

  const url = `${API_BASE_URL}${endpoint}`;

  try {
    const response = await fetch(url, {
      ...fetchOptions,
      headers: requestHeaders,
    });

    const data = await response.json().catch(() => null);

    if (!response.ok) {
      throw new ApiError(
        data?.message || "An error occurred",
        response.status,
        data
      );
    }

    return data as T;
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    throw new ApiError(
      error instanceof Error ? error.message : "Network error",
      0
    );
  }
}

// Types
export type AdminStatsResponse = {
  data?: {
    totalTenants?: number;
    totalUsers?: number;
    totalLeads?: number;
    mrr?: number;
    leadsToday?: number;
    conversionRate?: number;
    pageviewsToday?: number;
    apiCallsMin?: number;
  };
};

export type PaginatedResponse<T> = {
  data: T[];
  total?: number;
  page?: number;
  limit?: number;
};

export type ServiceHealth = {
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
};

function buildSearchParams(params?: Record<string, string | number | undefined | null>): string {
  if (!params) return "";
  const sp = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== "") sp.set(k, String(v));
  });
  const s = sp.toString();
  return s ? `?${s}` : "";
}

// Legacy alias kept for backward compat with existing call sites
export type AdminStats = AdminStatsResponse;

export const adminApi = {
  // Dashboard
  getStats: () => apiClient<AdminStatsResponse>("/api/v1/admin/stats"),

  // Service Logs
  getServiceLogs: (params?: { service?: string; level?: string; from?: string; to?: string; limit?: number }) =>
    apiClient<PaginatedResponse<Record<string, unknown>>>(`/api/v1/admin/services/logs${buildSearchParams(params as Record<string, string | number | undefined>)}`),
  getAnalytics: (params?: Record<string, string>) =>
    apiClient<PaginatedResponse<Record<string, unknown>>>(`/api/v1/admin/analytics${buildSearchParams(params)}`),

  // Tenants
  getTenants: (params?: { page?: number; limit?: number; search?: string; status?: string }) =>
    apiClient<PaginatedResponse<Record<string, unknown>>>(`/api/v1/admin/tenants${buildSearchParams(params as Record<string, string | number | undefined>)}`),
  getTenant: (id: string) => apiClient<{ data: Record<string, unknown> }>(`/api/v1/admin/tenants/${id}`),
  createTenant: (data: Record<string, unknown>) =>
    apiClient("/api/v1/admin/tenants", { method: "POST", body: JSON.stringify(data) }),
  updateTenant: (id: string, data: Record<string, unknown>) =>
    apiClient(`/api/v1/admin/tenants/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  suspendTenant: (id: string, reason: string) =>
    apiClient(`/api/v1/admin/tenants/${id}/suspend`, {
      method: "POST",
      body: JSON.stringify({ reason }),
    }),
  activateTenant: (id: string) =>
    apiClient(`/api/v1/admin/tenants/${id}/activate`, { method: "POST" }),
  deleteTenant: (id: string) =>
    apiClient(`/api/v1/admin/tenants/${id}`, { method: "DELETE" }),

  // Users
  getUsers: (params?: { page?: number; limit?: number; search?: string; role?: string }) =>
    apiClient<PaginatedResponse<Record<string, unknown>>>(`/api/v1/admin/users${buildSearchParams(params as Record<string, string | number | undefined>)}`),
  getUser: (id: string) => apiClient<{ data: Record<string, unknown> }>(`/api/v1/admin/users/${id}`),
  createUser: (data: Record<string, unknown>) =>
    apiClient("/api/v1/admin/users", { method: "POST", body: JSON.stringify(data) }),
  updateUser: (id: string, data: Record<string, unknown>) =>
    apiClient(`/api/v1/admin/users/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deleteUser: (id: string) =>
    apiClient(`/api/v1/admin/users/${id}`, { method: "DELETE" }),

  // System Health
  getServicesHealth: () => apiClient<{ data: ServiceHealth[] }>("/api/v1/admin/services/health"),
  getMetrics: (params?: { from?: string; to?: string }) =>
    apiClient<{ data: Record<string, unknown>[] }>(`/api/v1/admin/metrics${buildSearchParams(params)}`),

  // Audit Logs
  getAuditLogs: (params?: { page?: number; limit?: number; user_id?: string; action?: string; from?: string; to?: string }) =>
    apiClient<PaginatedResponse<Record<string, unknown>>>(`/api/v1/admin/audit-logs${buildSearchParams(params as Record<string, string | number | undefined>)}`),

  // Leads Analytics
  getLeadsAnalytics: (params?: { from?: string; to?: string; tenant_id?: string }) =>
    apiClient<{ data: Record<string, unknown>[] }>(`/api/v1/admin/analytics/leads${buildSearchParams(params)}`),

  // Traffic Analytics
  getTrafficAnalytics: (params?: { from?: string; to?: string }) =>
    apiClient<{ data: Record<string, unknown>[] }>(`/api/v1/admin/analytics/traffic${buildSearchParams(params)}`),
};

export { ApiError };
