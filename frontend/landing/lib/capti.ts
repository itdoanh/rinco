import { api } from "./api";

interface CAPIEvent {
  event_name: string;
  event_time?: number;
  user_data?: {
    email?: string;
    phone?: string;
    name?: string;
  };
  custom_data?: Record<string, unknown>;
  data_processing_options?: string[];
  data_processing_options_state?: number;
}

const CAPI_ACCESS_TOKEN = process.env.NEXT_PUBLIC_CAPI_ACCESS_TOKEN || "";
const CAPI_PIXEL_ID = process.env.NEXT_PUBLIC_CAPI_PIXEL_ID || "";
const IS_BROWSER = typeof window !== "undefined";

/**
 * Send server-side event to Facebook Conversions API
 */
export async function sendCAPIEvent(event: CAPIEvent): Promise<boolean> {
  if (!CAPI_ACCESS_TOKEN || !CAPI_PIXEL_ID) {
    console.warn("CAPI credentials not configured");
    return false;
  }

  try {
    // Send through our API proxy
    await api.sendCapiEvent({
      access_token: CAPI_ACCESS_TOKEN,
      pixel_id: CAPI_PIXEL_ID,
      ...event,
      event_time: event.event_time || Math.floor(Date.now() / 1000),
    });
    return true;
  } catch (error) {
    console.error("CAPI error:", error);
    return false;
  }
}

/**
 * Track lead with CAPI (server-side)
 */
export async function trackLeadCAPI(
  name?: string,
  phone?: string,
  email?: string,
  value?: number
): Promise<boolean> {
  return sendCAPIEvent({
    event_name: "Lead",
    user_data: {
      name,
      phone,
      email,
    },
    custom_data: {
      currency: "VND",
      value,
    },
  });
}

/**
 * Track page view with CAPI (server-side)
 */
export async function trackPageViewCAPI(
  pageUrl?: string,
  userData?: { email?: string; phone?: string }
): Promise<boolean> {
  return sendCAPIEvent({
    event_name: "PageView",
    user_data: userData,
    custom_data: {
      page_url: pageUrl || (IS_BROWSER ? window.location.href : ""),
    },
  });
}

/**
 * Track form submission with CAPI
 */
export async function trackFormSubmitCAPI(
  formName: string,
  userData?: { name?: string; phone?: string; email?: string }
): Promise<boolean> {
  return sendCAPIEvent({
    event_name: "Lead",
    user_data: userData,
    custom_data: {
      content_name: formName,
      currency: "VND",
    },
  });
}
