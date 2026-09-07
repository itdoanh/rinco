"use client";

import { useEffect } from "react";
import { initPixel, trackLead, trackPageView, trackCustomEvent } from "@/lib/pixel";

export function usePixel() {
  useEffect(() => {
    // Initialize on mount
    initPixel();
  }, []);

  return {
    trackLead,
    trackPageView,
    trackCustomEvent,
  };
}

export default usePixel;
