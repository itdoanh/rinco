/**
 * Lightweight fetch helper for talking to the RINCO backend.
 *
 * The meeting UI typically only needs to fetch meeting metadata
 * (titles, scheduled start times, etc.) before the user joins.  Once
 * the room is live the entire UX is driven by WebRTC and WebSocket
 * — no HTTP polling required.
 *
 * The helper adds a base URL (from ``NEXT_PUBLIC_API_URL``) and a
 * JSON Content-Type by default.  Errors are surfaced as ``ApiError``
 * instances so callers can branch on ``status`` if needed.
 */

const DEFAULT_BASE_URL =
  typeof process !== "undefined"
    ? process.env.NEXT_PUBLIC_API_URL
    : undefined;

export class ApiError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message);
    this.name = "ApiError";
  }
}

export interface ApiOptions {
  baseUrl?: string;
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

export interface MeetingInfo {
  id: string;
  title: string;
  startsAt?: string;
  endsAt?: string;
  hostName?: string;
}

/** Fetch JSON from the backend, raising ApiError on non-2xx responses. */
export async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
  options: ApiOptions = {},
): Promise<T> {
  const baseUrl = options.baseUrl ?? DEFAULT_BASE_URL ?? "";
  const url = path.startsWith("http")
    ? path
    : `${baseUrl.replace(/\/$/, "")}${path.startsWith("/") ? path : `/${path}`}`;

  const headers: Record<string, string> = {
    "content-type": "application/json",
    ...(options.headers ?? {}),
    ...((init.headers as Record<string, string> | undefined) ?? {}),
  };

  const response = await fetch(url, { ...init, headers, signal: options.signal });
  const text = await response.text();
  if (!response.ok) {
    throw new ApiError(response.status, text || response.statusText);
  }
  if (!text) return undefined as T;
  try {
    return JSON.parse(text) as T;
  } catch {
    return text as unknown as T;
  }
}

/** Fetch meeting metadata (title, schedule, host).  Returns ``null``
 *  when the backend is unreachable so the UI can degrade gracefully. */
export async function getMeeting(
  roomId: string,
  options: ApiOptions = {},
): Promise<MeetingInfo | null> {
  try {
    return await apiFetch<MeetingInfo>(`/v1/meetings/${encodeURIComponent(roomId)}`, {
      method: "GET",
    }, options);
  } catch {
    return null;
  }
}
