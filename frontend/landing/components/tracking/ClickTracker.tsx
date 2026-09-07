'use client';

import { useEffect, useRef, useCallback } from 'react';
import { trackEvent } from '@/lib/tracking';

interface ClickTrackerOptions {
  tenantId?: string;
  pageId?: string;
  selector?: string;
  debounceMs?: number;
}

interface TrackedElement {
  element: HTMLElement;
  text: string;
  href?: string;
  tracked: boolean;
}

export function ClickTracker({
  tenantId = '',
  pageId = '',
  selector = '[data-track-click], a, button',
  debounceMs = 500,
}: ClickTrackerOptions) {
  const clickedElements = useRef<Set<HTMLElement>>(new Set());
  const lastClickTime = useRef<number>(0);

  const handleClick = useCallback(
    (event: MouseEvent) => {
      const now = Date.now();
      if (now - lastClickTime.current < debounceMs) return;

      const target = event.target as HTMLElement;
      const element = target.closest(selector) as HTMLElement | null;

      if (!element) return;
      if (clickedElements.current.has(element)) return;

      lastClickTime.current = now;
      clickedElements.current.add(element);

      const text = element.textContent?.trim().slice(0, 100) || '';
      const href = (element as HTMLAnchorElement).href || undefined;
      const dataTrack = element.getAttribute('data-track-click') || undefined;
      const dataCategory = element.getAttribute('data-track-category') || 'click';
      const dataLabel = element.getAttribute('data-track-label') || text;

      trackEvent('click', {
        element_type: element.tagName.toLowerCase(),
        text: dataTrack || text,
        href,
        category: dataCategory,
        label: dataLabel,
        tenant_id: tenantId,
        page_id: pageId,
        x: event.clientX,
        y: event.clientY,
      });
    },
    [tenantId, pageId, selector, debounceMs]
  );

  useEffect(() => {
    document.addEventListener('click', handleClick, true);
    return () => document.removeEventListener('click', handleClick, true);
  }, [handleClick]);

  return null;
}
