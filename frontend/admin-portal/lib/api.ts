const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

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

export async function apiClient<T>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<T> {
  const { tenantId, headers = {}, ...fetchOptions } = options;

  const requestHeaders: HeadersInit = {
    "Content-Type": "application/json",
    ...headers,
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

// Admin API
export const adminApi = {
  // Dashboard
  getStats: () => apiClient("/api/v1/admin/stats"),
  getAnalytics: (params?: Record<string, string>) => {
    const searchParams = new URLSearchParams(params);
    return apiClient(`/api/v1/admin/analytics?${searchParams}`);
  },

  // Tenants
  getTenants: (params?: { page?: number; limit?: number; search?: string; status?: string }) => {
    const searchParams = new URLSearchParams(
      Object.entries(params || {}).reduce((acc, [k, v]) => {
        if (v) acc[k] = v;
        return acc;
      }, {} as Record<string, string>)
    );
    return apiClient(`/api/v1/admin/tenants?${searchParams}`);
  },
  getTenant: (id: string) => apiClient(`/api/v1/admin/tenants/${id}`),
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
  getUsers: (params?: { page?: number; limit?: number; search?: string; role?: string }) => {
    const searchParams = new URLSearchParams(
      Object.entries(params || {}).reduce((acc, [k, v]) => {
        if (v) acc[k] = v;
        return acc;
      }, {} as Record<string, string>)
    );
    return apiClient(`/api/v1/admin/users?${searchParams}`);
  },
  getUser: (id: string) => apiClient(`/api/v1/admin/users/${id}`),
  createUser: (data: Record<string, unknown>) =>
    apiClient("/api/v1/admin/users", { method: "POST", body: JSON.stringify(data) }),
  updateUser: (id: string, data: Record<string, unknown>) =>
    apiClient(`/api/v1/admin/users/${id}`, { method: "PATCH", body: JSON.stringify(data) }),
  deleteUser: (id: string) =>
    apiClient(`/api/v1/admin/users/${id}`, { method: "DELETE" }),

  // System Health
  getServicesHealth: () => apiClient("/api/v1/admin/services/health"),
  getServiceLogs: (params?: { service?: string; level?: string; from?: string; to?: string; limit?: number }) => {
    const searchParams = new URLSearchParams(
      Object.entries(params || {}).reduce((acc, [k, v]) => {
        if (v) acc[k] = v;
        return acc;
      }, {} as Record<string, string>)
    );
    return apiClient(`/api/v1/admin/services/logs?${searchParams}`);
  },
  getMetrics: (params?: { from?: string; to?: string }) => {
    const searchParams = new URLSearchParams(params || {});
    return apiClient(`/api/v1/admin/metrics?${searchParams}`);
  },

  // Audit Logs
  getAuditLogs: (params?: { page?: number; limit?: number; user_id?: string; action?: string; from?: string; to?: string }) => {
    const searchParams = new URLSearchParams(
      Object.entries(params || {}).reduce((acc, [k, v]) => {
        if (v) acc[k] = v;
        return acc;
      }, {} as Record<string, string>)
    );
    return apiClient(`/api/v1/admin/audit-logs?${searchParams}`);
  },

  // Leads Analytics
  getLeadsAnalytics: (params?: { from?: string; to?: string; tenant_id?: string }) => {
    const searchParams = new URLSearchParams(
      Object.entries(params || {}).reduce((acc, [k, v]) => {
        if (v) acc[k] = v;
        return acc;
        }, {} as Record<string, string>)
    );
    return apiClient(`/api/v1/admin/analytics/leads?${searchParams}`);
  },

  // Traffic Analytics
  getTrafficAnalytics: (params?: { from?: string; to?: string }) => {
    const searchParams = new URLSearchParams(params || {});
    return apiClient(`/api/v1/admin/analytics/traffic?${searchParams}`);
  },
};

export { ApiError };
