import { z } from "zod";

export const loginSchema = z.object({
  email: z.string().email("Email không hợp lệ"),
  password: z.string().min(6, "Mật khẩu phải có ít nhất 6 ký tự"),
});

export type LoginFormData = z.infer<typeof loginSchema>;

export const tenantSchema = z.object({
  name: z.string().min(2, "Tên tenant phải có ít nhất 2 ký tự"),
  slug: z
    .string()
    .min(2, "Slug phải có ít nhất 2 ký tự")
    .regex(/^[a-z0-9-]+$/, "Slug chỉ chứa chữ thường, số và dấu gạch ngang"),
  email: z.string().email("Email không hợp lệ"),
  phone: z.string().optional(),
  domain: z.string().optional(),
  branding: z
    .object({
      primary: z.string().optional(),
      secondary: z.string().optional(),
      accent: z.string().optional(),
      fontFamily: z.string().optional(),
      logoUrl: z.string().url().optional().or(z.literal("")),
    })
    .optional(),
  settings: z
    .object({
      maxUsers: z.number().optional(),
      maxLeads: z.number().optional(),
      features: z.array(z.string()).optional(),
    })
    .optional(),
});

export type TenantFormData = z.infer<typeof tenantSchema>;

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  email: string;
  phone?: string;
  domain?: string;
  status: "active" | "suspended" | "pending" | "trial";
  branding?: {
    primary?: string;
    secondary?: string;
    accent?: string;
    fontFamily?: string;
    logoUrl?: string;
  };
  settings?: {
    maxUsers?: number;
    maxLeads?: number;
    features?: string[];
  };
  stats?: {
    users: number;
    leads: number;
    mrr: number;
  };
  created_at: string;
  updated_at: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: "admin" | "user" | "viewer";
  tenant_id?: string;
  created_at: string;
  last_login?: string;
}

export interface AuditLog {
  id: string;
  user_id: string;
  user_name: string;
  action: string;
  resource: string;
  resource_id?: string;
  metadata?: Record<string, unknown>;
  ip: string;
  created_at: string;
}
