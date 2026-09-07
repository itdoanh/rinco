'use client';

import { useEffect } from 'react';
import { setCookie } from '@/lib/utils';

interface UTMCaptureProps {
  expiresDays?: number;
}

export function UTMCapture({ expiresDays = 30 }: UTMCaptureProps) {
  useEffect(() => {
    if (typeof window === 'undefined') return;

    const params = new URLSearchParams(window.location.search);

    // Capture UTM parameters
    const utmParams = [
      'utm_source',
      'utm_medium',
      'utm_campaign',
      'utm_content',
      'utm_term',
    ];

    utmParams.forEach((param) => {
      const value = params.get(param);
      if (value) {
        setCookie(`_rinco_${param}`, value, expiresDays);
      }
    });

    // Capture first-touch attribution
    const firstTouchSource = params.get('utm_source') || getCookie('_rinco_ft_source');
    if (firstTouchSource && !getCookie('_rinco_ft_source')) {
      setCookie('_rinco_ft_source', firstTouchSource, expiresDays);
      setCookie('_rinco_ft_medium', params.get('utm_medium') || '', expiresDays);
      setCookie('_rinco_ft_campaign', params.get('utm_campaign') || '', expiresDays);
    }

    // Capture landing page
    setCookie('_rinco_landing_url', window.location.href, expiresDays);
  }, [expiresDays]);

  return null;
}

function getCookie(name: string): string {
  if (typeof document === 'undefined') return '';
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(';').shift() || '';
  return '';
}
