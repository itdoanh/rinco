import { create } from "zustand";

interface Tenant {
  id: string;
  slug: string;
  name: string;
  status: "active" | "suspended" | "pending";
  created_at: string;
  mrr: number;
  users_count: number;
  leads_count: number;
}

interface AdminState {
  tenants: Tenant[];
  selectedTenant: Tenant | null;
  filters: {
    search: string;
    status: string;
    sortBy: string;
    sortOrder: "asc" | "desc";
  };
  setTenants: (tenants: Tenant[]) => void;
  setSelectedTenant: (tenant: Tenant | null) => void;
  setFilters: (filters: Partial<AdminState["filters"]>) => void;
  addTenant: (tenant: Tenant) => void;
  updateTenant: (id: string, updates: Partial<Tenant>) => void;
  removeTenant: (id: string) => void;
}

export const useAdminStore = create<AdminState>((set) => ({
  tenants: [],
  selectedTenant: null,
  filters: {
    search: "",
    status: "",
    sortBy: "created_at",
    sortOrder: "desc",
  },
  setTenants: (tenants) => set({ tenants }),
  setSelectedTenant: (tenant) => set({ selectedTenant: tenant }),
  setFilters: (filters) =>
    set((state) => ({ filters: { ...state.filters, ...filters } })),
  addTenant: (tenant) =>
    set((state) => ({ tenants: [...state.tenants, tenant] })),
  updateTenant: (id, updates) =>
    set((state) => ({
      tenants: state.tenants.map((t) =>
        t.id === id ? { ...t, ...updates } : t
      ),
    })),
  removeTenant: (id) =>
    set((state) => ({
      tenants: state.tenants.filter((t) => t.id !== id),
    })),
}));

// Analytics store
interface AnalyticsData {
  totalTenants: number;
  totalUsers: number;
  totalLeads: number;
  mrr: number;
  leadsToday: number;
  conversionRate: number;
  pageviewsToday: number;
  apiCallsMin: number;
}

interface AnalyticsState {
  data: AnalyticsData | null;
  isLoading: boolean;
  setData: (data: AnalyticsData) => void;
  setLoading: (loading: boolean) => void;
}

export const useAnalyticsStore = create<AnalyticsState>((set) => ({
  data: null,
  isLoading: false,
  setData: (data) => set({ data }),
  setLoading: (isLoading) => set({ isLoading }),
}));

// UI store
interface UIState {
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: true,
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
}));
