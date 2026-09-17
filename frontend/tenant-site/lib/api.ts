/**
 * Tenant site API client.
 *
 * Resolves CMS pages from landing:8086 and tenant metadata from tenant:8082.
 * Falls back to local mock data on network failure so the page still renders
 * during demo / offline development.
 */
import type { Page, Tenant } from "./schema";
import { getMockPage, getMockTenant } from "./mock-data";
import { httpUrl } from "@rinco/ui";

const TENANT_API = httpUrl("tenant");
const LANDING_API = httpUrl("landing");

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

async function request<T>(
  baseUrl: string,
  endpoint: string,
  options: FetchOptions = {},
): Promise<T | null> {
  const { tenantSlug, headers = {}, ...fetchOptions } = options;
  const requestHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
    ...(headers as Record<string, string>),
  };
  if (tenantSlug) requestHeaders["X-Tenant-Slug"] = tenantSlug;

  try {
    const ctrl = new AbortController();
    const t = setTimeout(() => ctrl.abort(), 8000);
    const res = await fetch(`${baseUrl}${endpoint}`, {
      ...fetchOptions,
      headers: requestHeaders,
      signal: ctrl.signal,
    });
    clearTimeout(t);
    if (!res.ok) {
      throw new ApiError(`HTTP ${res.status}`, res.status);
    }
    const data = (await res.json().catch(() => null)) as T | null;
    return data ?? null;
  } catch (err) {
    if (err instanceof ApiError) throw err;
    throw new ApiError(err instanceof Error ? err.message : "Network error", 0);
  }
}

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
      const result = await request<Tenant>(TENANT_API, `/v1/tenants/${tenantSlug}`, {
        tenantSlug,
      });
      return result ?? (getMockTenant(tenantSlug) as Tenant);
    } catch {
      return getMockTenant(tenantSlug) as Tenant;
    }
  },

  getTenantPages: async (tenantSlug: string): Promise<Page[]> => {
    try {
      const result = await request<{ data?: Page[] } | Page[]>(
        LANDING_API,
        `/v1/landing/pages/${tenantSlug}`,
        { tenantSlug },
      );
      const pages = Array.isArray(result) ? result : result?.data ?? [];
      if (pages.length > 0) return pages;
      const fallback = getMockTenant(tenantSlug);
      return fallback?.pages ?? [];
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
      const result = await request<{ data?: Page } | Page>(
        LANDING_API,
        `/v1/landing/pages/${tenantSlug}/${pageSlug}`,
        { tenantSlug },
      );
      const page = (result as { data?: Page } | null)?.data ?? (result as Page | null);
      return page ?? getMockPage(tenantSlug, pageSlug) ?? null;
    } catch {
      return getMockPage(tenantSlug, pageSlug) ?? null;
    }
  },

  /** Submits lead via Next.js rewrite (`/api/leads` → lead:8085). */
  submitLead: async (data: Record<string, unknown>, tenantSlug: string) => {
    const res = await fetch("/api/leads", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Tenant-Slug": tenantSlug,
      },
      body: JSON.stringify(data),
    });
    if (!res.ok) {
      throw new ApiError(`HTTP ${res.status}`, res.status);
    }
    return res.json().catch(() => ({}));
  },
};
