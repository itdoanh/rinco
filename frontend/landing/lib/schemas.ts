import { z } from 'zod';

export const leadSchema = z.object({
  name: z.string().min(2, 'Tên phải có ít nhất 2 ký tự'),
  email: z.string().email('Email không hợp lệ'),
  phone: z.string().min(10, 'Số điện thoại không hợp lệ'),
  company: z.string().optional(),
  note: z.string().optional(),
  tenant_id: z.string(),
  page_id: z.string(),
  utm_source: z.string().optional(),
  utm_medium: z.string().optional(),
  utm_campaign: z.string().optional(),
  utm_content: z.string().optional(),
  utm_term: z.string().optional(),
  referrer: z.string().optional(),
});

export type LeadFormData = z.infer<typeof leadSchema>;

export const subscriptionSchema = z.object({
  email: z.string().email('Email không hợp lệ'),
  tenant_id: z.string(),
  page_id: z.string(),
});

export type SubscriptionFormData = z.infer<typeof subscriptionSchema>;
