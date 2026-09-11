declare global {
  interface Window {
    fbq: ((...args: unknown[]) => void) & { callMethod?: (...args: unknown[]) => void; queue?: unknown[]; push?: (...args: unknown[]) => void; loaded?: boolean };
    _fbq?: typeof window.fbq;
    gtag: (...args: unknown[]) => void;
    dataLayer: unknown[];
  }
}

const PIXEL_ID = process.env.NEXT_PUBLIC_FB_PIXEL_ID || "";
const IS_BROWSER = typeof window !== "undefined";

// Guard flag so we never call window.fbq without the stub being installed.
// When PIXEL_ID is empty we still install a no-op stub so all track* calls
// succeed silently — this matches Meta Pixel's official snippet behaviour.
let pixelReady = false;

/**
 * No-op fbq stub matching Meta Pixel's official loader signature.
 * All call sites are guarded so missing configuration never throws.
 */
function installFbqStub(): void {
  if (!IS_BROWSER) return;
  if (window.fbq && typeof window.fbq === "function" && window._fbq) {
    // Real Meta Pixel stub already in place.
    pixelReady = true;
    return;
  }
  const n: unknown[] = [];
  const f: Window["fbq"] = Object.assign(
    function (...args: unknown[]) {
      n.push(args);
    } as Window["fbq"],
    {
      queue: n,
      callMethod(this: unknown, ...args: unknown[]) {
        n.push(args);
      },
      push(this: unknown, ...args: unknown[]) {
        n.push(args);
      },
      loaded: false,
    }
  );
  window.fbq = f;
  window._fbq = f;
  pixelReady = true;
}

/**
 * Initialize Meta Pixel using the official snippet pattern.
 * Safe to call multiple times — subsequent calls are no-ops.
 */
export function initPixel(pixelId: string = PIXEL_ID): void {
  if (!IS_BROWSER) return;
  installFbqStub();
  if (!pixelId) {
    // No pixel configured — stub absorbs all calls so callers never throw.
    return;
  }
  if (window.fbq.loaded) return;
  const script = document.createElement("script");
  script.async = true;
  script.src = "https://connect.facebook.net/en_US/fbevents.js";
  document.head.appendChild(script);
  window.fbq("init", pixelId);
  window.fbq("track", "PageView");
  window.fbq.loaded = true;
}

/**
 * Safe wrapper around window.fbq. Never throws when pixel isn't configured.
 */
function safeFbqCall(...args: unknown[]): void {
  if (!IS_BROWSER) return;
  try {
    if (typeof window.fbq !== "function") installFbqStub();
    window.fbq(...args);
  } catch (err) {
    // Last-resort guard: log + swallow so tracking never breaks the app.
    console.warn("[pixel] tracking call failed:", err);
  }
}

/**
 * Track page view
 */
export function trackPageView(customData?: Record<string, unknown>): void {
  if (!IS_BROWSER) return;
  initPixel();
  safeFbqCall("track", "PageView", customData);
}

/**
 * Track lead generation
 */
export function trackLead(
  contentName?: string,
  contentCategory?: string,
  currency: string = "VND",
  value?: number
): void {
  if (!IS_BROWSER) return;
  initPixel();
  safeFbqCall("track", "Lead", {
    content_name: contentName,
    content_category: contentCategory,
    currency,
    value,
  });
}

/**
 * Track form submission
 */
export function trackForm(
  formName: string,
  success: boolean = true
): void {
  if (!IS_BROWSER) return;
  initPixel();
  if (success) {
    safeFbqCall("track", "Lead", {
      content_name: formName,
    });
  } else {
    safeFbqCall("trackCustom", "FormError", {
      form_name: formName,
    });
  }
}

/**
 * Track button click
 */
export function trackClick(
  buttonLabel: string,
  ctaLabel?: string,
  location?: string
): void {
  if (!IS_BROWSER) return;
  initPixel();
  safeFbqCall("trackCustom", "ButtonClick", {
    button_label: buttonLabel,
    cta_label: ctaLabel,
    location,
  });
}

/**
 * Track custom event
 */
export function trackCustomEvent(
  eventName: string,
  data?: Record<string, unknown>
): void {
  if (!IS_BROWSER) return;
  initPixel();
  safeFbqCall("trackCustom", eventName, data);
}

/**
 * Reset the pixel — only used in tests.
 */
export function _resetPixelForTests(): void {
  pixelReady = false;
  if (IS_BROWSER) {
    delete window.fbq;
    delete window._fbq;
  }
}

/** True when pixel has been initialised. Useful for tests. */
export function _isPixelReady(): boolean {
  return pixelReady;
}
