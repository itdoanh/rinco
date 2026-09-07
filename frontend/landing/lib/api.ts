const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface FetchOptions extends RequestInit {
  tenantSlug?: string;
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

async function fetchWithTimeout(
  url: string,
  options: RequestInit & { timeout?: number }
): Promise<Response> {
  const { timeout = 30000, ...fetchOptions } = options;
  
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const response = await fetch(url, {
      ...fetchOptions,
      signal: controller.signal,
    });
    return response;
  } finally {
    clearTimeout(timeoutId);
  }
}

export async function apiClient<T>(
  endpoint: string,
  options: FetchOptions = {}
): Promise<T> {
  const {
    tenantSlug,
    headers = {},
    ...fetchOptions
  } = options;

  const requestHeaders: HeadersInit = {
    "Content-Type": "application/json",
    ...headers,
  };

  if (tenantSlug) {
    requestHeaders["X-Tenant-Slug"] = tenantSlug;
  }

  const url = `${API_BASE_URL}${endpoint}`;

  try {
    const response = await fetchWithTimeout(url, {
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
    
    if (error instanceof Error && error.name === "AbortError") {
      throw new ApiError("Request timeout", 408);
    }
    
    throw new ApiError(
      error instanceof Error ? error.message : "Network error",
      0
    );
  }
}

// API methods
export const api = {
  // Lead endpoints
  submitLead: (data: Record<string, unknown>, tenantSlug?: string) =>
    apiClient("/api/leads", {
      method: "POST",
      body: JSON.stringify(data),
      tenantSlug,
    }),

  // Page endpoints
  getPage: (tenantSlug: string, pageSlug: string) =>
    apiClient(`/landing/pages/${tenantSlug}/${pageSlug}`, {
      tenantSlug,
    }),

  // Tenant endpoints
  getTenant: (tenantSlug: string) =>
    apiClient(`/tenants/${tenantSlug}`),

  // Tracking endpoints
  trackEvent: (event: Record<string, unknown>) =>
    apiClient("/api/track", {
      method: "POST",
      body: JSON.stringify(event),
    }),

  // CAPI endpoints
  sendCapiEvent: (event: Record<string, unknown>) =>
    apiClient("/api/capi", {
      method: "POST",
      body: JSON.stringify(event),
    }),
};

export { ApiError };
