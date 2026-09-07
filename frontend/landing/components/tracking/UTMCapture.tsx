"use client";

import { useEffect } from "react";
import { usePathname, useSearchParams } from "next/navigation";
import { saveUTMToStorage } from "@/lib/tracking";

export function UTMCapture() {
  const pathname = usePathname();
  const searchParams = useSearchParams();

  useEffect(() => {
    // Capture UTM parameters from URL
    const utmParams = [
      "utm_source",
      "utm_medium",
      "utm_campaign",
      "utm_content",
      "utm_term",
    ];

    const utmData: Record<string, string> = {};

    utmParams.forEach((param) => {
      const value = searchParams.get(param);
      if (value) {
        utmData[param] = value;
      }
    });

    // Save to localStorage if any UTM params exist
    if (Object.keys(utmData).length > 0) {
      saveUTMToStorage(utmData);
    }
  }, [pathname, searchParams]);

  return null;
}

export default UTMCapture;
