import axios, { AxiosError } from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      // Handle unauthorized
      if (typeof window !== 'undefined') {
        console.error('Unauthorized request');
      }
    }
    return Promise.reject(error);
  }
);

export interface Page {
  id: string;
  slug: string;
  tenant_id: string;
  title: string;
  meta_description?: string;
  blocks: Block[];
  created_at: string;
  updated_at: string;
}

export interface Block {
  id: string;
  type: string;
  data: Record<string, unknown>;
  order: number;
}

export const pagesApi = {
  async getPage(tenantSlug: string, pageSlug: string): Promise<Page> {
    const response = await api.get(`/api/pages/${tenantSlug}/${pageSlug}`);
    return response.data;
  },

  async getTenantPages(tenantId: string): Promise<Page[]> {
    const response = await api.get(`/api/pages`, {
      params: { tenant_id: tenantId },
    });
    return response.data;
  },
};

export const leadsApi = {
  async submitLead(data: {
    name: string;
    email: string;
    phone: string;
    tenant_id: string;
    page_id: string;
    note?: string;
  }): Promise<void> {
    await api.post('/api/leads', data);
  },
};

export const trackingApi = {
  async trackEvent(data: {
    event: string;
    tenant_id: string;
    page_id: string;
    properties?: Record<string, unknown>;
  }): Promise<void> {
    // Don't await - fire and forget
    api.post('/api/track', data).catch(() => {});
  },
};
