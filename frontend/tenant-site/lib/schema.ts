import { z } from "zod";

export const leadSchema = z.object({
  name: z.string().min(2, "Họ và tên phải có ít nhất 2 ký tự"),
  phone: z.string().min(10, "Số điện thoại không hợp lệ"),
  email: z.string().email("Email không hợp lệ").optional().or(z.literal("")),
});

export type LeadFormData = z.infer<typeof leadSchema>;

export interface Branding {
  primary?: string;
  secondary?: string;
  accent?: string;
  fontFamily?: string;
  logoUrl?: string;
  favicon?: string;
}

export interface Tenant {
  id: string;
  slug: string;
  name: string;
  logo?: string;
  branding?: Branding;
  settings?: {
    maxUsers?: number;
    maxLeads?: number;
    features?: string[];
  };
}

export interface Block {
  id: string;
  type: string;
  data: Record<string, unknown>;
}

export interface Page {
  id: string;
  slug: string;
  title: string;
  description?: string;
  blocks: Block[];
  seo?: {
    title?: string;
    description?: string;
    image?: string;
  };
}
