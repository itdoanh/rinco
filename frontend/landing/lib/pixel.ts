declare global {
  interface Window {
    fbq: (action: string, eventName: string, data?: Record<string, unknown>) => void;
    gtag: (...args: unknown[]) => void;
    dataLayer: unknown[];
  }
}

const PIXEL_ID = process.env.NEXT_PUBLIC_FB_PIXEL_ID || "";
const IS_BROWSER = typeof window !== "undefined";

/**
 * Initialize Meta Pixel
 */
export function initPixel(pixelId: string = PIXEL_ID): void {
  if (!IS_BROWSER || !pixelId) return;
  
  // Load Pixel script if not already loaded
  if (!(window as Window).fbq) {
    const script = document.createElement("script");
    script.async = true;
    script.src = "https://connect.facebook.net/en_US/fbevents.js";
    document.head.appendChild(script);
  }
  
  window.fbq("init", pixelId);
  window.fbq("track", "PageView");
}

/**
 * Track page view
 */
export function trackPageView(customData?: Record<string, unknown>): void {
  if (!IS_BROWSER) return;
  initPixel();
  window.fbq("track", "PageView", customData);
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
  window.fbq("track", "Lead", {
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
    window.fbq("track", "Lead", {
      content_name: formName,
    });
  } else {
    window.fbq("trackCustom", "FormError", {
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
  window.fbq("trackCustom", "ButtonClick", {
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
  window.fbq("trackCustom", eventName, data);
}
