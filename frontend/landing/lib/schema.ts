import { z } from "zod";

export const leadSchema = z.object({
  name: z.string().min(2, "Họ và tên phải có ít nhất 2 ký tự"),
  phone: z.string().min(10, "Số điện thoại không hợp lệ").max(15),
  email: z.string().email("Email không hợp lệ").optional().or(z.literal("")),
  channel: z.enum(["zalo", "phone"]).optional(),
  website: z.string().optional(), // honeypot
  tenant_slug: z.string().optional(),
  page_slug: z.string().optional(),
  utm_source: z.string().optional(),
  utm_medium: z.string().optional(),
  utm_campaign: z.string().optional(),
  utm_content: z.string().optional(),
  utm_term: z.string().optional(),
});

export type LeadFormData = z.infer<typeof leadSchema>;

export const formSchema = z.object({
  fields: z.array(
    z.object({
      name: z.string(),
      type: z.enum([
        "text",
        "email",
        "phone",
        "select",
        "checkbox",
        "radio",
        "date",
        "textarea",
        "password",
      ]),
      label: z.string(),
      placeholder: z.string().optional(),
      required: z.boolean().optional(),
      options: z.array(z.object({
        label: z.string(),
        value: z.string(),
      })).optional(),
      validation: z.object({
        minLength: z.number().optional(),
        maxLength: z.number().optional(),
        pattern: z.string().optional(),
      }).optional(),
    })
  ),
});

export type FormSchema = z.infer<typeof formSchema>;

export interface Block {
  id: string;
  type: string;
  data: Record<string, unknown>;
}

export interface PageConfig {
  id: string;
  slug: string;
  title: string;
  description: string;
  blocks: Block[];
  seo?: {
    title?: string;
    description?: string;
    image?: string;
  };
}

export interface Tenant {
  id: string;
  slug: string;
  name: string;
  logo?: string;
  branding?: {
    primary?: string;
    secondary?: string;
    accent?: string;
    fontFamily?: string;
  };
}
