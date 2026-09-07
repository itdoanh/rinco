"use client";

import { useEffect, useRef } from "react";
import { initPixel, trackPageView as trackPixelPageView } from "@/lib/pixel";
import { trackPageView as trackPageViewEvent } from "@/lib/tracking";

interface TrackerProps {
  pixelId?: string;
  tenantSlug?: string;
  pageSlug?: string;
}

export function Tracker({
  pixelId,
  tenantSlug,
  pageSlug,
}: TrackerProps) {
  const initialized = useRef(false);

  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;

    // Initialize Meta Pixel
    if (pixelId) {
      initPixel(pixelId);
    } else {
      initPixel();
    }

    // Track initial page view
    trackPixelPageView();
    trackPageViewEvent(tenantSlug, pageSlug);
  }, [pixelId, tenantSlug, pageSlug]);

  return null;
}

export default Tracker;
