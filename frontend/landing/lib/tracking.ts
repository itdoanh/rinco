import { api } from "./api";
import { trackLeadCAPI, trackFormSubmitCAPI } from "./capti";

interface TrackEvent {
  event: string;
  properties?: Record<string, unknown>;
  timestamp?: number;
  tenant_slug?: string;
  page_slug?: string;
  utm?: {
    source?: string;
    medium?: string;
    campaign?: string;
    content?: string;
    term?: string;
  };
  user?: {
    name?: string;
    phone?: string;
    email?: string;
  };
  pixel_event?: string;
  capi_event?: boolean;
}

/**
 * Track client-side event
 */
export async function trackEvent(eventData: TrackEvent): Promise<void> {
  const event = {
    ...eventData,
    timestamp: eventData.timestamp || Date.now(),
    properties: {
      ...eventData.properties,
      url: typeof window !== "undefined" ? window.location.href : "",
      user_agent: typeof navigator !== "undefined" ? navigator.userAgent : "",
      referrer: typeof document !== "undefined" ? document.referrer : "",
    },
  };

  try {
    // Send to our tracking API
    await api.trackEvent(event);
  } catch (error) {
    console.error("Tracking error:", error);
  }
}

/**
 * Track page view
 */
export async function trackPageView(
  tenantSlug?: string,
  pageSlug?: string
): Promise<void> {
  const utm = getUTMFromStorage();
  
  await trackEvent({
    event: "page_view",
    tenant_slug: tenantSlug,
    page_slug: pageSlug,
    utm,
  });
}

/**
 * Track form submission
 */
export async function trackFormSubmit(
  formName: string,
  userData: { name?: string; phone?: string; email?: string },
  tenantSlug?: string,
  pageSlug?: string
): Promise<void> {
  const utm = getUTMFromStorage();
  
  // Track with our API
  await trackEvent({
    event: "form_submit",
    properties: { form_name: formName },
    tenant_slug: tenantSlug,
    page_slug: pageSlug,
    utm,
    user: userData,
    pixel_event: "Lead",
    capi_event: true,
  });

  // Send CAPI event (server-side)
  await trackFormSubmitCAPI(formName, userData);
}

/**
 * Track button click
 */
export async function trackClick(
  buttonLabel: string,
  location?: string,
  tenantSlug?: string
): Promise<void> {
  const utm = getUTMFromStorage();
  
  await trackEvent({
    event: "button_click",
    properties: {
      button_label: buttonLabel,
      location,
    },
    tenant_slug: tenantSlug,
    utm,
  });
}

/**
 * Get UTM parameters from localStorage
 */
export function getUTMFromStorage(): TrackEvent["utm"] {
  if (typeof window === "undefined") return {};
  
  try {
    const stored = localStorage.getItem("rinco_utm");
    if (stored) {
      return JSON.parse(stored);
    }
  } catch {
    // Ignore parse errors
  }
  
  return {};
}

/**
 * Save UTM parameters to localStorage
 */
export function saveUTMToStorage(utm: Record<string, string>): void {
  if (typeof window === "undefined") return;
  
  try {
    localStorage.setItem("rinco_utm", JSON.stringify(utm));
  } catch {
    // Ignore storage errors
  }
}
