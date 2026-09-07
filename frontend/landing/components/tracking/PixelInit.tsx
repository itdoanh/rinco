"use client";

import { useEffect, useState } from "react";
import { trackClick } from "@/lib/pixel";

interface PixelInitProps {
  pixelId?: string;
}

declare global {
  interface Window {
    fbq: (...args: unknown[]) => void;
  }
}

export function PixelInit({ pixelId }: PixelInitProps) {
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (loaded) return;

    const id = pixelId || process.env.NEXT_PUBLIC_FB_PIXEL_ID;
    if (!id) return;

    // Check if already loaded
    if (typeof window !== "undefined" && window.fbq) {
      setLoaded(true);
      return;
    }

    // Create script
    const script = document.createElement("script");
    script.async = true;
    script.src = "https://connect.facebook.net/en_US/fbevents.js";
    script.onload = () => setLoaded(true);
    document.head.appendChild(script);

    // Initialize pixel
    window.fbq = function () {
      window.fbq.push(arguments);
    };
    window.fbq.push = window.fbq;
    window.fbq.loaded = true;
    window.fbq.version = "2.0";
    window.fbq.queue = [];

    window.fbq("init", id);
    window.fbq("track", "PageView");

    setLoaded(true);
  }, [pixelId, loaded]);

  return null;
}

export default PixelInit;
