"use client";

import { useEffect } from "react";
import { trackPageView as trackPageViewEvent } from "@/lib/tracking";

export function usePageView(tenantSlug?: string, pageSlug?: string) {
  useEffect(() => {
    trackPageViewEvent(tenantSlug, pageSlug);
  }, [tenantSlug, pageSlug]);
}

export default usePageView;
