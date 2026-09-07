const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface FetchOptions extends RequestInit {
  tenantSlug?: string;
}

export async function apiClient<T>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<T> {
  const { tenantSlug, headers = {}, ...fetchOptions } = options;

  const requestHeaders: HeadersInit = {
    "Content-Type": "application/json",
    ...headers,
  };

  if (tenantSlug) {
    requestHeaders["X-Tenant-Slug"] = tenantSlug;
  }

  const url = `${API_BASE_URL}${endpoint}`;

  try {
    const response = await fetch(url, {
      ...fetchOptions,
      headers: requestHeaders,
    });

    const data = await response.json().catch(() => null);

    if (!response.ok) {
      throw new Error(data?.message || "An error occurred");
    }

    return data as T;
  } catch (error) {
    throw new Error(
      error instanceof Error ? error.message : "Network error"
    );
  }
}

export const tenantApi = {
  getTenant: (tenantSlug: string) =>
    apiClient(`/tenants/${tenantSlug}`, { tenantSlug }),

  getTenantPages: (tenantSlug: string) =>
    apiClient(`/landing/pages/${tenantSlug}`, { tenantSlug }),

  getPage: (tenantSlug: string, pageSlug: string) =>
    apiClient(`/landing/pages/${tenantSlug}/${pageSlug}`, { tenantSlug }),

  submitLead: (data: Record<string, unknown>, tenantSlug: string) =>
    apiClient("/api/leads", {
      method: "POST",
      body: JSON.stringify(data),
      tenantSlug,
    }),
};
