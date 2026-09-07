'use client';

import { useEffect, useState, createContext, useContext, type ReactNode } from 'react';
import { useBrandingStore, type TenantBranding, defaultBranding } from '@/lib/branding';

interface BrandingContextValue {
  branding: TenantBranding | null;
  isLoading: boolean;
  primaryColor: string;
  secondaryColor: string;
}

const BrandingContext = createContext<BrandingContextValue>({
  branding: null,
  isLoading: true,
  primaryColor: defaultBranding.primary_color || '#0ea5e9',
  secondaryColor: defaultBranding.secondary_color || '#0369a1',
});

export function useBranding() {
  return useContext(BrandingContext);
}

interface BrandingProviderProps {
  tenantSlug: string;
  children: ReactNode;
}

export function BrandingProvider({ tenantSlug, children }: BrandingProviderProps) {
  const { branding, isLoading, setBranding, setLoading, setError } = useBrandingStore();
  const [primaryColor, setPrimaryColor] = useState(defaultBranding.primary_color || '#0ea5e9');
  const [secondaryColor, setSecondaryColor] = useState(defaultBranding.secondary_color || '#0369a1');

  useEffect(() => {
    async function fetchBranding() {
      setLoading(true);
      try {
        const response = await fetch(`/api/branding/${tenantSlug}`);
        if (response.ok) {
          const data = await response.json();
          setBranding(data);
          setPrimaryColor(data.primary_color || defaultBranding.primary_color || '#0ea5e9');
          setSecondaryColor(data.secondary_color || defaultBranding.secondary_color || '#0369a1');
        } else {
          // Use default branding
          setBranding({
            tenant_id: tenantSlug,
            tenant_name: tenantSlug,
            ...defaultBranding,
          } as TenantBranding);
        }
      } catch (err) {
        console.error('Failed to fetch branding:', err);
        setError(err instanceof Error ? err.message : 'Failed to load branding');
      }
    }

    fetchBranding();
  }, [tenantSlug, setBranding, setLoading, setError]);

  return (
    <BrandingContext.Provider value={{ branding, isLoading, primaryColor, secondaryColor }}>
      {children}
    </BrandingContext.Provider>
  );
}
