'use client';

import { useEffect } from 'react';
import { initPixel } from '@/lib/pixel';
import { getCookie } from '@/lib/utils';

interface PixelInitProps {
  pixelId?: string;
  tenantId?: string;
}

export function PixelInit({ pixelId, tenantId = '' }: PixelInitProps) {
  useEffect(() => {
    if (!pixelId) return;

    // Initialize Meta Pixel
    initPixel(pixelId);

    // Track PageView
    if (typeof window !== 'undefined' && (window as Window & { fbq?: Function }).fbq) {
      (window as Window & { fbq: Function }).fbq('track', 'PageView', {
        tenant_id: tenantId,
      });
    }
  }, [pixelId, tenantId]);

  return null;
}

// Type declarations for Meta Pixel
declare global {
  interface Window {
    fbq: ((action: string, event: string, data?: Record<string, unknown>) => void) & {
      queue: Array<[string, string, Record<string, unknown>?]>;
    };
    _fbq: typeof fbq;
  }
}
