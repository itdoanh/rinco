import { create } from 'zustand';

export interface TenantBranding {
  tenant_id: string;
  tenant_name: string;
  logo_url?: string;
  favicon_url?: string;
  primary_color?: string;
  secondary_color?: string;
  font_family?: string;
  custom_css?: string;
  contact_email?: string;
  contact_phone?: string;
  social_links?: {
    facebook?: string;
    twitter?: string;
    linkedin?: string;
    instagram?: string;
  };
  footer_text?: string;
}

interface BrandingState {
  branding: TenantBranding | null;
  isLoading: boolean;
  error: string | null;
  setBranding: (branding: TenantBranding) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useBrandingStore = create<BrandingState>((set) => ({
  branding: null,
  isLoading: false,
  error: null,
  setBranding: (branding) => set({ branding, error: null }),
  setLoading: (loading) => set({ isLoading: loading }),
  setError: (error) => set({ error, isLoading: false }),
}));

// Default branding values
export const defaultBranding: Partial<TenantBranding> = {
  primary_color: '#0ea5e9',
  secondary_color: '#0369a1',
  font_family: 'Inter, sans-serif',
};
