import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

// =============================================================================
// Class-name helpers
// =============================================================================

/**
 * Merge Tailwind CSS class names with conflict resolution.
 *
 * Wraps `clsx` (for conditional classes) and `tailwind-merge` (for
 * resolving conflicting Tailwind utilities, e.g. `px-2` vs `px-4`).
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

// =============================================================================
// Number / currency / date formatting
// =============================================================================

/** Sentinel string returned by formatters when input is invalid. */
export const INVALID_FORMAT_PLACEHOLDER = "—";

/** `true` when `value` is a finite, non-NaN number. */
function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

/**
 * Format a number as a localised currency string.
 *
 * Defaults to VND (Vietnamese Dong) which is the primary market for
 * RINCO.  Non-finite values (NaN, Infinity, null, undefined) render as
 * a sentinel so the UI never displays "NaN đ" or "₫NaN".
 */
export function formatCurrency(
  amount: number,
  currency = "VND",
  locale = "vi-VN",
): string {
  if (!isFiniteNumber(amount)) return INVALID_FORMAT_PLACEHOLDER;
  try {
    return new Intl.NumberFormat(locale, {
      style: "currency",
      currency,
      minimumFractionDigits: 0,
    }).format(amount);
  } catch {
    // Unknown currency code (e.g. test stub) — fall back to plain number.
    return `${amount.toLocaleString(locale)} ${currency}`;
  }
}

/** Format a number using thousand separators.  Returns the sentinel
 * for non-finite input. */
export function formatNumber(num: number, locale = "vi-VN"): string {
  if (!isFiniteNumber(num)) return INVALID_FORMAT_PLACEHOLDER;
  return new Intl.NumberFormat(locale).format(num);
}

/**
 * `true` if the `Date` instance represents a real point in time (i.e.
 * `getTime()` is a finite number).  Useful to detect the result of
 * `new Date("garbage")` which is still a valid object but produces
 * `NaN` from `getTime()`.
 */
function isValidDate(d: Date): boolean {
  return d instanceof Date && Number.isFinite(d.getTime());
}

function toDate(date: Date | string | number | null | undefined): Date | null {
  if (date === null || date === undefined || date === "") return null;
  if (date instanceof Date) return isValidDate(date) ? date : null;
  const d = new Date(date);
  return isValidDate(d) ? d : null;
}

/**
 * Format a date as ``dd/MM/yyyy`` in the Vietnamese locale.  Returns
 * the sentinel for null/undefined/invalid input.
 */
export function formatDate(
  date: Date | string | number | null | undefined,
  locale = "vi-VN",
): string {
  const d = toDate(date);
  if (!d) return INVALID_FORMAT_PLACEHOLDER;
  return new Intl.DateTimeFormat(locale, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(d);
}

/**
 * Format a date+time as ``dd/MM/yyyy HH:mm`` in the Vietnamese locale.
 */
export function formatDateTime(
  date: Date | string | number | null | undefined,
  locale = "vi-VN",
): string {
  const d = toDate(date);
  if (!d) return INVALID_FORMAT_PLACEHOLDER;
  return new Intl.DateTimeFormat(locale, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

// =============================================================================
// String helpers
// =============================================================================

/**
 * Convert a human-readable string into a URL-safe slug.  Handles
 * Vietnamese diacritics (NFD decomposition + combining-mark strip),
 * collapses non-alphanumeric runs into single hyphens, and trims
 * leading/trailing hyphens.  Empty input returns empty string.
 */
export function slugify(str: string | null | undefined): string {
  if (!str) return "";
  return str
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)+/g, "");
}

/**
 * Truncate a string to at most ``length`` characters and append an
 * ellipsis.  Edge cases:
 *
 *   * ``length <= 0`` — returns "" (no point truncating to a negative
 *     length).
 *   * ``length >= str.length`` — returns the input untouched.
 *   * ``str === null/undefined`` — returns "".
 */
export function truncate(
  str: string | null | undefined,
  length: number,
  suffix = "...",
): string {
  if (str === null || str === undefined) return "";
  if (!Number.isFinite(length) || length <= 0) return "";
  if (str.length <= length) return str;
  return str.slice(0, length) + suffix;
}

/**
 * Extract the first character of each whitespace-separated word and
 * uppercase the result.  Always returns at most 2 characters.
 *
 *   * Empty/whitespace-only input — "".
 *   * Single word — first letter uppercased.
 *   * Multiple words — first letter of first two words.
 */
export function getInitials(name: string | null | undefined): string {
  if (!name) return "";
  const parts = name.trim().split(/\s+/);
  return parts
    .map((n) => n[0] ?? "")
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

// =============================================================================
// Async / scheduling helpers
// =============================================================================

/** Resolve after ``ms`` milliseconds. */
export function sleep(ms: number): Promise<void> {
  if (!Number.isFinite(ms) || ms < 0) {
    return Promise.resolve();
  }
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export interface DebouncedFn<T extends (...args: unknown[]) => unknown> {
  (...args: Parameters<T>): void;
  /** Cancel the pending invocation. */
  cancel(): void;
  /** Immediately invoke the underlying function with the last args,
   *  cancelling any pending call. */
  flush(): void;
}

/**
 * Debounce a function so it only fires after ``wait`` milliseconds of
 * inactivity.  The returned wrapper exposes ``cancel`` and ``flush``
 * for explicit lifecycle control.
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
  func: T,
  wait: number,
): DebouncedFn<T> {
  let timeout: ReturnType<typeof setTimeout> | null = null;
  let lastArgs: Parameters<T> | null = null;

  const debounced = (...args: Parameters<T>): void => {
    lastArgs = args;
    if (timeout) clearTimeout(timeout);
    timeout = setTimeout(() => {
      timeout = null;
      if (lastArgs) {
        func(...lastArgs);
        lastArgs = null;
      }
    }, wait);
  };

  debounced.cancel = (): void => {
    if (timeout) {
      clearTimeout(timeout);
      timeout = null;
    }
    lastArgs = null;
  };

  debounced.flush = (): void => {
    if (timeout) {
      clearTimeout(timeout);
      timeout = null;
    }
    if (lastArgs) {
      func(...lastArgs);
      lastArgs = null;
    }
  };

  return debounced as DebouncedFn<T>;
}

// =============================================================================
// ID generation
// =============================================================================

/**
 * Generate a short non-cryptographic ID.  Suitable for React ``key``
 * props and DOM IDs but NOT for security tokens (use ``crypto.randomUUID``
 * or a similar CSPRNG for those).
 */
export function generateId(): string {
  return (
    Math.random().toString(36).substring(2) + Date.now().toString(36)
  );
}

/**
 * Cryptographically random UUID v4 using the browser/Node
 * ``crypto.randomUUID`` API.  Throws if the runtime doesn't support
 * the API (very old browsers).
 */
export function generateUuid(): string {
  if (
    typeof globalThis !== "undefined" &&
    typeof globalThis.crypto?.randomUUID === "function"
  ) {
    return globalThis.crypto.randomUUID();
  }
  // RFC 4122 v4 fallback.  Not as strong as the native API but
  // sufficient for non-security contexts.
  const bytes = new Uint8Array(16);
  if (
    typeof globalThis !== "undefined" &&
    typeof globalThis.crypto?.getRandomValues === "function"
  ) {
    globalThis.crypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < 16; i++) bytes[i] = Math.floor(Math.random() * 256);
  }
  // version + variant bits per RFC 4122 §4.4
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0"));
  return (
    hex.slice(0, 4).join("") +
    "-" +
    hex.slice(4, 6).join("") +
    "-" +
    hex.slice(6, 8).join("") +
    "-" +
    hex.slice(8, 10).join("") +
    "-" +
    hex.slice(10, 16).join("")
  );
}

// =============================================================================
// Numeric helpers
// =============================================================================

/** Clamp ``n`` between ``min`` and ``max``.  Non-finite inputs are
 *  clamped to whichever bound is closer (or to ``min`` if they're
 *  equidistant). */
export function clamp(n: number, min: number, max: number): number {
  if (!Number.isFinite(n)) {
    // Treat NaN as min, Infinity as max, -Infinity as min.
    if (n === Infinity) return max;
    return min;
  }
  if (n < min) return min;
  if (n > max) return max;
  return n;
}

/** Convert a string to a finite number, returning ``fallback`` if the
 *  string is not parseable. */
export function parseFloatSafe(
  str: string | null | undefined,
  fallback = 0,
): number {
  if (str === null || str === undefined || str === "") return fallback;
  const n = Number(str);
  return Number.isFinite(n) ? n : fallback;
}
