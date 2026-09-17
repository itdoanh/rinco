/**
 * Tenant site API client.
 *
 * Provides typed wrappers around the tenant-site BFF (port 8082).
 * Every call resolves tenant context from middleware (x-tenant-slug)
 * and gracefully falls back to local mock data on network failure so
 * the page still renders during demo / offline development.
 */
import type { Page, Tenant } from "./schema";
import { getMockPage, getMockTenant } from "./mock-data";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

interface FetchOptions extends RequestInit {
  tenantSlug?: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public data?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function apiClient<T = unknown>(
  endpoint: string,
  options: FetchOptions = {},
): Promise<T> {
  const { tenantSlug, headers = {}, ...fetchOptions } = options;

  const requestHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    ...(headers as Record<string, string>),
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
      throw new ApiError(
        (data as { message?: string } | null)?.message || "An error occurred",
        response.status,
        data,
      );
    }

    return data as T;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    throw new ApiError(
      error instanceof Error ? error.message : "Network error",
      0,
    );
  }
}

// --- Types ---
export type TenantBranding = {
  primary: string;
  secondary: string;
  accent: string;
  logo?: string;
  fontFamily?: string;
};

export type TenantFull = Tenant;

export type TenantPageData = Page;

export const tenantApi = {
  getTenant: async (tenantSlug: string): Promise<Tenant> => {
    try {
      const result = await apiClient<Tenant>(
        `/tenants/${tenantSlug}`,
        { tenantSlug },
      );
      return (result as Tenant | null) ?? (getMockTenant(tenantSlug) as Tenant);
    } catch {
      return getMockTenant(tenantSlug) as Tenant;
    }
  },

  getTenantPages: async (tenantSlug: string): Promise<Page[]> => {
    try {
      const result = await apiClient<Page[]>(
        `/landing/pages/${tenantSlug}`,
        { tenantSlug },
      );
      const fallback = getMockTenant(tenantSlug);
      return result ?? fallback?.pages ?? [];
    } catch {
      const fallback = getMockTenant(tenantSlug);
      return fallback?.pages ?? [];
    }
  },

  getPage: async (
    tenantSlug: string,
    pageSlug: string,
  ): Promise<Page | null> => {
    try {
      const result = await apiClient<Page>(
        `/landing/pages/${tenantSlug}/${pageSlug}`,
        { tenantSlug },
      );
      return result ?? getMockPage(tenantSlug, pageSlug) ?? null;
    } catch {
      return getMockPage(tenantSlug, pageSlug) ?? null;
    }
  },

  submitLead: async (data: Record<string, unknown>, tenantSlug: string) => {
    return apiClient("/api/leads", {
      method: "POST",
      body: JSON.stringify(data),
      tenantSlug,
    });
  },
};
