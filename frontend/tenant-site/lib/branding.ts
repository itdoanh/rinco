"use client";

import { useEffect } from "react";
import type { Branding } from "@/lib/schema";

/**
 * Convert hex color to HSL for CSS variables
 */
export function hexToHsl(hex: string): string {
  // Remove # if present
  hex = hex.replace(/^#/, "");

  // Parse hex
  const r = parseInt(hex.substring(0, 2), 16) / 255;
  const g = parseInt(hex.substring(2, 4), 16) / 255;
  const b = parseInt(hex.substring(4, 6), 16) / 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h = 0;
  let s = 0;
  const l = (max + min) / 2;

  if (max !== min) {
    const d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);

    switch (max) {
      case r:
        h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
        break;
      case g:
        h = ((b - r) / d + 2) / 6;
        break;
      case b:
        h = ((r - g) / d + 4) / 6;
        break;
    }
  }

  return `${Math.round(h * 360)} ${Math.round(s * 100)}% ${Math.round(l * 100)}%`;
}

/**
 * Apply tenant branding to the page
 * Injects CSS variables for dynamic theming
 */
export function useBranding(branding?: Branding) {
  useEffect(() => {
    if (!branding) return;

    const root = document.documentElement;

    // Set CSS custom properties
    if (branding.primary) {
      root.style.setProperty("--brand-primary", branding.primary);
      root.style.setProperty("--primary", hexToHsl(branding.primary));
    }
    if (branding.secondary) {
      root.style.setProperty("--brand-secondary", branding.secondary);
    }
    if (branding.accent) {
      root.style.setProperty("--brand-accent", branding.accent);
      root.style.setProperty("--accent", hexToHsl(branding.accent));
    }

    // Set favicon
    if (branding.favicon) {
      const favicon = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
      if (favicon) {
        favicon.href = branding.favicon;
      }
    }

    // Cleanup on unmount
    return () => {
      if (branding.primary) {
        root.style.removeProperty("--brand-primary");
      }
      if (branding.secondary) {
        root.style.removeProperty("--brand-secondary");
      }
      if (branding.accent) {
        root.style.removeProperty("--brand-accent");
      }
    };
  }, [branding]);
}

/**
 * Generate CSS for dynamic branding
 */
export function getBrandingCSS(branding?: Branding): string {
  if (!branding) return "";

  const styles: string[] = [];

  if (branding.primary) {
    styles.push(`--primary: ${hexToHsl(branding.primary)};`);
  }
  if (branding.accent) {
    styles.push(`--accent: ${hexToHsl(branding.accent)};`);
  }
  if (branding.fontFamily) {
    styles.push(`--font-family: ${branding.fontFamily};`);
  }

  return `:root { ${styles.join(" ")} }`;
}
