'use client';

import { useEffect } from 'react';
import { trackPageView, trackEvent } from '@/lib/tracking';
import { getCookie } from '@/lib/utils';

interface TrackerProps {
  tenantId?: string;
  pageId?: string;
}

export function Tracker({ tenantId = '', pageId = '' }: TrackerProps) {
  useEffect(() => {
    // Track page view on mount
    const sessionId = getCookie('_rinco_sid') || generateSessionId();
    const referrer = document.referrer || undefined;
    const utmParams = getUTMParams();

    trackPageView({
      tenant_id: tenantId,
      page_id: pageId,
      url: window.location.href,
      referrer,
      session_id: sessionId,
      ...utmParams,
    });

    // Track scroll depth
    let maxScroll = 0;
    const handleScroll = () => {
      const scrollHeight = document.documentElement.scrollHeight - window.innerHeight;
      const scrollPercent = Math.round((window.scrollY / scrollHeight) * 100);

      if (scrollPercent > maxScroll) {
        maxScroll = scrollPercent;

        if (scrollPercent >= 25 && scrollPercent < 50) {
          trackEvent('scroll_depth', { depth: '25%', tenant_id: tenantId, page_id: pageId });
        } else if (scrollPercent >= 50 && scrollPercent < 75) {
          trackEvent('scroll_depth', { depth: '50%', tenant_id: tenantId, page_id: pageId });
        } else if (scrollPercent >= 75 && scrollPercent < 100) {
          trackEvent('scroll_depth', { depth: '75%', tenant_id: tenantId, page_id: pageId });
        } else if (scrollPercent >= 100) {
          trackEvent('scroll_depth', { depth: '100%', tenant_id: tenantId, page_id: pageId });
        }
      }
    };

    // Track time on page
    const startTime = Date.now();
    let timeTracked = false;

    const handleTimeTracking = () => {
      if (timeTracked) return;
      const timeOnPage = Math.round((Date.now() - startTime) / 1000);

      if (timeOnPage >= 30 && timeOnPage < 60) {
        trackEvent('time_on_page', { seconds: 30, tenant_id: tenantId, page_id: pageId });
        timeTracked = true;
      }
    };

    // Track exit intent on desktop
    const handleMouseLeave = (e: MouseEvent) => {
      if (e.clientY <= 0) {
        trackEvent('exit_intent', { tenant_id: tenantId, page_id: pageId });
      }
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    window.addEventListener('mouseleave', handleMouseLeave);
    const timeInterval = setInterval(handleTimeTracking, 30000);

    return () => {
      window.removeEventListener('scroll', handleScroll);
      window.removeEventListener('mouseleave', handleMouseLeave);
      clearInterval(timeInterval);
    };
  }, [tenantId, pageId]);

  return null;
}

function generateSessionId(): string {
  const sid = `${Date.now()}-${Math.random().toString(36).substring(2, 15)}`;
  if (typeof document !== 'undefined') {
    document.cookie = `_rinco_sid=${sid}; path=/; max-age=86400; SameSite=Lax`;
  }
  return sid;
}

function getUTMParams(): Record<string, string> {
  if (typeof window === 'undefined') return {};

  const params = new URLSearchParams(window.location.search);
  return {
    utm_source: params.get('utm_source') || '',
    utm_medium: params.get('utm_medium') || '',
    utm_campaign: params.get('utm_campaign') || '',
    utm_content: params.get('utm_content') || '',
    utm_term: params.get('utm_term') || '',
  };
}
