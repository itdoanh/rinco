import { useEffect, useCallback, useRef } from 'react';
import { trackPageView } from '@/lib/tracking';
import { getCookie } from '@/lib/utils';

interface UsePageViewOptions {
  tenantId: string;
  pageId: string;
  enabled?: boolean;
}

interface PageViewData {
  url?: string;
  referrer?: string;
  title?: string;
}

export function usePageView({
  tenantId,
  pageId,
  enabled = true,
}: UsePageViewOptions) {
  const hasTracked = useRef(false);
  const sessionId = useRef(getCookie('_rinco_sid') || generateSessionId());

  const track = useCallback(
    (data?: PageViewData) => {
      if (!enabled || hasTracked.current) return;

      const utmParams = getUTMFromCookie();

      trackPageView({
        tenant_id: tenantId,
        page_id: pageId,
        url: data?.url || (typeof window !== 'undefined' ? window.location.href : ''),
        referrer: data?.referrer || document.referrer || undefined,
        title: data?.title || document.title || undefined,
        session_id: sessionId.current,
        ...utmParams,
      });

      hasTracked.current = true;
    },
    [tenantId, pageId, enabled]
  );

  // Track on mount
  useEffect(() => {
    if (enabled) {
      track();
    }

    return () => {
      hasTracked.current = false;
    };
  }, [enabled, track]);

  // Track on route change (for SPAs)
  useEffect(() => {
    if (!enabled || typeof window === 'undefined') return;

    const handlePopState = () => {
      hasTracked.current = false;
      track({ url: window.location.href });
    };

    // Listen for history changes
    const originalPushState = window.history.pushState;
    window.history.pushState = function (...args) {
      originalPushState.apply(this, args);
      hasTracked.current = false;
      track({ url: window.location.href });
    };

    window.addEventListener('popstate', handlePopState);

    return () => {
      window.removeEventListener('popstate', handlePopState);
      window.history.pushState = originalPushState;
    };
  }, [enabled, track]);

  return { track };
}

function generateSessionId(): string {
  const sid = `${Date.now()}-${Math.random().toString(36).substring(2, 15)}`;
  if (typeof document !== 'undefined') {
    document.cookie = `_rinco_sid=${sid}; path=/; max-age=86400; SameSite=Lax`;
  }
  return sid;
}

function getUTMFromCookie(): Record<string, string> {
  if (typeof document === 'undefined') return {};

  const getCookie = (name: string) => {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()?.split(';').shift() || '';
    return '';
  };

  return {
    utm_source: getCookie('_rinco_utm_source'),
    utm_medium: getCookie('_rinco_utm_medium'),
    utm_campaign: getCookie('_rinco_utm_campaign'),
    utm_content: getCookie('_rinco_utm_content'),
    utm_term: getCookie('_rinco_utm_term'),
  };
}
