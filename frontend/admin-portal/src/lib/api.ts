import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor to add auth token
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    if (typeof window !== 'undefined') {
      const token = localStorage.getItem('access_token');
      if (token && config.headers) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    if (error.response?.status === 401) {
      // Token expired, try to refresh
      const refreshToken = localStorage.getItem('refresh_token');
      if (refreshToken) {
        try {
          const response = await axios.post(`${API_BASE_URL}/api/auth/refresh`, {
            refresh_token: refreshToken,
          });

          const { access_token } = response.data;
          localStorage.setItem('access_token', access_token);

          // Retry the original request
          if (error.config && error.config.headers) {
            error.config.headers.Authorization = `Bearer ${access_token}`;
            return axios(error.config);
          }
        } catch (refreshError) {
          // Refresh failed, clear tokens and redirect to login
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
          if (typeof window !== 'undefined') {
            window.location.href = '/(auth)/login';
          }
        }
      } else {
        // No refresh token, redirect to login
        if (typeof window !== 'undefined') {
          window.location.href = '/(auth)/login';
        }
      }
    }
    return Promise.reject(error);
  }
);

// API endpoints
export const apiEndpoints = {
  // Auth
  login: '/api/auth/login',
  logout: '/api/auth/logout',
  refresh: '/api/auth/refresh',

  // Tenants
  tenants: '/api/tenants',
  tenant: (id: string) => `/api/tenants/${id}`,

  // Users
  users: '/api/users',
  user: (id: string) => `/api/users/${id}`,

  // Leads
  leads: '/api/leads',
  lead: (id: string) => `/api/leads/${id}`,

  // Pages
  pages: '/api/pages',
  page: (id: string) => `/api/pages/${id}`,

  // Analytics
  analytics: '/api/analytics',
  analyticsOverview: '/api/analytics/overview',
  analyticsLeads: '/api/analytics/leads',
  analyticsTraffic: '/api/analytics/traffic',

  // System
  health: '/api/health',
  metrics: '/api/metrics',
};

// API functions
export const adminApi = {
  // Auth
  async login(email: string, password: string) {
    const response = await api.post(apiEndpoints.login, { email, password });
    return response.data;
  },

  async logout() {
    const response = await api.post(apiEndpoints.logout);
    return response.data;
  },

  // Tenants
  async getTenants(params?: { page?: number; limit?: number; search?: string; status?: string }) {
    const response = await api.get(apiEndpoints.tenants, { params });
    return response.data;
  },

  async getTenant(id: string) {
    const response = await api.get(apiEndpoints.tenant(id));
    return response.data;
  },

  async createTenant(data: Record<string, unknown>) {
    const response = await api.post(apiEndpoints.tenants, data);
    return response.data;
  },

  async updateTenant(id: string, data: Record<string, unknown>) {
    const response = await api.put(apiEndpoints.tenant(id), data);
    return response.data;
  },

  async deleteTenant(id: string) {
    const response = await api.delete(apiEndpoints.tenant(id));
    return response.data;
  },

  // Users
  async getUsers(params?: { page?: number; limit?: number; tenant_id?: string }) {
    const response = await api.get(apiEndpoints.users, { params });
    return response.data;
  },

  async getUser(id: string) {
    const response = await api.get(apiEndpoints.user(id));
    return response.data;
  },

  async createUser(data: Record<string, unknown>) {
    const response = await api.post(apiEndpoints.users, data);
    return response.data;
  },

  async updateUser(id: string, data: Record<string, unknown>) {
    const response = await api.put(apiEndpoints.user(id), data);
    return response.data;
  },

  async deleteUser(id: string) {
    const response = await api.delete(apiEndpoints.user(id));
    return response.data;
  },

  // Leads
  async getLeads(params?: { page?: number; limit?: number; tenant_id?: string; status?: string }) {
    const response = await api.get(apiEndpoints.leads, { params });
    return response.data;
  },

  async getLead(id: string) {
    const response = await api.get(apiEndpoints.lead(id));
    return response.data;
  },

  async updateLead(id: string, data: Record<string, unknown>) {
    const response = await api.put(apiEndpoints.lead(id), data);
    return response.data;
  },

  // Analytics
  async getAnalyticsOverview() {
    const response = await api.get(apiEndpoints.analyticsOverview);
    return response.data;
  },

  async getLeadsAnalytics(params?: { start_date?: string; end_date?: string }) {
    const response = await api.get(apiEndpoints.analyticsLeads, { params });
    return response.data;
  },

  async getTrafficAnalytics(params?: { start_date?: string; end_date?: string }) {
    const response = await api.get(apiEndpoints.analyticsTraffic, { params });
    return response.data;
  },

  // System
  async getHealth() {
    const response = await api.get(apiEndpoints.health);
    return response.data;
  },

  async getMetrics() {
    const response = await api.get(apiEndpoints.metrics);
    return response.data;
  },
};
