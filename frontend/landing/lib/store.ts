import { create } from 'zustand';

export interface BlockData {
  id: string;
  type: 'hero' | 'feature_grid' | 'testimonial' | 'faq' | 'form' | 'cta' | 'pricing' | 'stats';
  data: Record<string, unknown>;
  order: number;
  visibility?: {
    desktop?: boolean;
    mobile?: boolean;
    conditions?: {
      referrer?: string[];
      utm_source?: string[];
      cookie_value?: string;
    };
  };
}

export interface LandingPageData {
  id: string;
  slug: string;
  tenant_id: string;
  title: string;
  meta_description?: string;
  blocks: BlockData[];
  created_at?: string;
  updated_at?: string;
}

interface LandingStore {
  pageData: LandingPageData | null;
  isLoading: boolean;
  error: string | null;
  setPageData: (data: LandingPageData) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useLandingStore = create<LandingStore>((set) => ({
  pageData: null,
  isLoading: false,
  error: null,
  setPageData: (data) => set({ pageData: data, error: null }),
  setLoading: (loading) => set({ isLoading: loading }),
  setError: (error) => set({ error, isLoading: false }),
}));
