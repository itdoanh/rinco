/**
 * Landing API client.
 *
 * - Pages: landing:8086
 * - Leads: lead:8085
 * - CAPI tracking: meta-capi:8098
 */
import { httpUrl } from "@rinco/ui";

const LANDING_API = httpUrl("landing");
const LEAD_API = httpUrl("lead");
const META_CAPI_API = httpUrl("meta-capi");

interface FetchOptions extends RequestInit {
  tenantSlug?: string;
  timeout?: number;
}

class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function fetchWithTimeout(
  url: string,
  options: RequestInit & { timeout?: number },
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

async function request<T>(
  baseUrl: string,
  endpoint: string,
  options: FetchOptions = {},
): Promise<T> {
  const { tenantSlug, headers = {}, ...fetchOptions } = options;
  const requestHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
    ...(headers as Record<string, string>),
  };
  if (tenantSlug) requestHeaders["X-Tenant-Slug"] = tenantSlug;
  const url = `${baseUrl}${endpoint}`;

  try {
    const response = await fetchWithTimeout(url, {
      ...fetchOptions,
      headers: requestHeaders,
    });
    const data = await response.json().catch(() => null);
    if (!response.ok) {
      throw new ApiError(
        (data as { message?: string } | null)?.message || `HTTP ${response.status}`,
        response.status,
        data,
      );
    }
    return data as T;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if (error instanceof Error && error.name === "AbortError") {
      throw new ApiError("Request timeout", 408);
    }
    throw new ApiError(
      error instanceof Error ? error.message : "Network error",
      0,
    );
  }
}

export const api = {
  /** Submit lead form to lead-service */
  submitLead: (data: Record<string, unknown>, tenantSlug?: string) =>
    request(LANDING_API, "/v1/leads", {
      method: "POST",
      body: JSON.stringify({ ...data, tenantSlug }),
      tenantSlug,
    }),

  /** Get CMS page from landing-service */
  getPage: (tenantSlug: string, pageSlug: string) =>
    request<{ data?: unknown } | unknown>(LANDING_API, `/v1/landing/pages/${tenantSlug}/${pageSlug}`, {
      tenantSlug,
    }),

  /** Get tenant metadata */
  getTenant: (tenantSlug: string) =>
    request(LANDING_API, `/v1/landing/tenants/${tenantSlug}`, { tenantSlug }),

  /** Track client-side event (analytics) */
  trackEvent: (event: Record<string, unknown>, tenantSlug?: string) =>
    request(LEAD_API, "/v1/track", {
      method: "POST",
      body: JSON.stringify(event),
      tenantSlug,
    }),

  /** Send server-side CAPI conversion to meta-capi:8098 */
  sendCapiEvent: (event: Record<string, unknown>, tenantSlug?: string) =>
    request(META_CAPI_API, "/v1/capi/events", {
      method: "POST",
      body: JSON.stringify(event),
      tenantSlug,
    }),
};

export { ApiError };
