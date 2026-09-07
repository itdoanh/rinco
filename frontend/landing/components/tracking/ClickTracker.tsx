"use client";

import { useEffect, useRef } from "react";
import { trackClick } from "@/lib/pixel";
import { trackClick as trackClickEvent } from "@/lib/tracking";

interface ClickTrackerProps {
  selector?: string;
  dataAttr?: string;
  tenantSlug?: string;
}

export function ClickTracker({
  selector = "[data-track]",
  dataAttr = "data-track",
  tenantSlug,
}: ClickTrackerProps) {
  const tracked = useRef<Set<Element>>(new Set());

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      
      // Find closest element with tracking attribute
      const trackedElement = target.closest(`[${dataAttr}]`) as HTMLElement | null;
      
      if (trackedElement && !tracked.current.has(trackedElement)) {
        tracked.current.add(trackedElement);
        
        const label = trackedElement.getAttribute(dataAttr);
        const ctaLabel = trackedElement.dataset.ctaLabel;
        const location = trackedElement.dataset.location;
        
        // Track with pixel
        if (label) {
          trackClick(label, ctaLabel || undefined, location || undefined);
        }
        
        // Track with our API
        if (label) {
          trackClickEvent(label, location || undefined, tenantSlug);
        }
      }
    };

    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, [dataAttr, tenantSlug]);

  return null;
}

export default ClickTracker;
