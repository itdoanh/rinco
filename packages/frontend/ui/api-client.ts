/**
 * RINCO shared HTTP client — typed wrapper around fetch() dùng cho mọi
 * frontend app. Features:
 *   - JSON Content-Type mặc định
 *   - Auth token tự động đính kèm từ localStorage ("auth_token")
 *   - AbortController timeout mặc định 15s
 *   - Network errors trả về ApiError(status=0)
 *   - JSON parse failures fail-open (return undefined)
 *   - Có thể set custom headers / signal / timeout per call
 */

import { httpUrl, type ServiceName } from "./services-registry";

export interface ApiErrorData {
  message?: string;
  code?: string;
  [key: string]: unknown;
}

export class ApiError extends Error {
  public readonly status: number;
  public readonly data?: ApiErrorData;
  constructor(message: string, status: number, data?: ApiErrorData) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.data = data;
  }
}

export interface FetchOptions extends Omit<RequestInit, "body"> {
  /** Service đích — sẽ dùng port từ registry. */
  service?: ServiceName;
  /** Override base URL (khi muốn bypass registry, e.g. relative path trong Next.js BFF). */
  baseUrl?: string;
  /** Headers bổ sung. */
  headers?: Record<string, string>;
  /** Timeout ms; mặc định 15_000. */
  timeout?: number;
  /** Body sẽ được JSON.stringify nếu là object. */
  body?: unknown;
  /** Signal bổ sung (e.g. AbortSignal từ React Query). */
  signal?: AbortSignal;
  /** Bỏ qua auto-attached Authorization header. */
  skipAuth?: boolean;
}

function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage.getItem("auth_token");
  } catch {
    return null;
  }
}

function resolveBaseUrl(opts: FetchOptions): string {
  if (opts.baseUrl) return opts.baseUrl.replace(/\/$/, "");
  if (opts.service) return httpUrl(opts.service);
  return "";
}

export async function apiFetch<T = unknown>(
  endpoint: string,
  options: FetchOptions = {},
): Promise<T> {
  const {
    service,
    baseUrl,
    headers: extraHeaders,
    timeout = 15_000,
    body,
    signal: externalSignal,
    skipAuth,
    ...rest
  } = options;

  const base = resolveBaseUrl({ service, baseUrl });
  const url = endpoint.startsWith("http")
    ? endpoint
    : `${base}${endpoint.startsWith("/") ? endpoint : `/${endpoint}`}`;

  const headers: Record<string, string> = {
    Accept: "application/json",
    ...(extraHeaders ?? {}),
  };

  // JSON Content-Type mặc định khi body là object.
  const isJsonBody = body !== undefined && body !== null && !(body instanceof FormData);
  if (isJsonBody && !headers["Content-Type"] && !headers["content-type"]) {
    headers["Content-Type"] = "application/json";
  }

  // Auto-attach Authorization nếu có token.
  if (!skipAuth) {
    const token = getAuthToken();
    if (token && !headers["Authorization"]) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  // Build body theo kiểu.
  const fetchInit: RequestInit = { ...rest, headers };
  if (body !== undefined && body !== null) {
    fetchInit.body = isJsonBody ? JSON.stringify(body) : (body as BodyInit);
  }

  // Compose timeout + external signal.
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeout);
  if (externalSignal) {
    if (externalSignal.aborted) controller.abort();
    else externalSignal.addEventListener("abort", () => controller.abort(), { once: true });
  }
  fetchInit.signal = controller.signal;

  try {
    const res = await fetch(url, fetchInit);

    // Parse JSON an toàn.
    let parsed: unknown = undefined;
    const text = await res.text();
    if (text) {
      try {
        parsed = JSON.parse(text);
      } catch {
        parsed = undefined;
      }
    }

    if (!res.ok) {
      const data = (parsed ?? {}) as ApiErrorData;
      throw new ApiError(
        data.message || res.statusText || `Request failed (${res.status})`,
        res.status,
        data,
      );
    }
    return parsed as T;
  } catch (err) {
    if (err instanceof ApiError) throw err;
    if (err instanceof DOMException && err.name === "AbortError") {
      throw new ApiError("Request timeout or aborted", 0);
    }
    throw new ApiError(
      err instanceof Error ? err.message : "Network error",
      0,
    );
  } finally {
    clearTimeout(timer);
  }
}

/** Shorthand: GET request. */
export const apiGet = <T>(endpoint: string, options?: FetchOptions) =>
  apiFetch<T>(endpoint, { ...options, method: "GET" });

/** Shorthand: POST request. */
export const apiPost = <T>(endpoint: string, body?: unknown, options?: FetchOptions) =>
  apiFetch<T>(endpoint, { ...options, method: "POST", body });

/** Shorthand: PUT request. */
export const apiPut = <T>(endpoint: string, body?: unknown, options?: FetchOptions) =>
  apiFetch<T>(endpoint, { ...options, method: "PUT", body });

/** Shorthand: PATCH request. */
export const apiPatch = <T>(endpoint: string, body?: unknown, options?: FetchOptions) =>
  apiFetch<T>(endpoint, { ...options, method: "PATCH", body });

/** Shorthand: DELETE request. */
export const apiDelete = <T>(endpoint: string, options?: FetchOptions) =>
  apiFetch<T>(endpoint, { ...options, method: "DELETE" });
