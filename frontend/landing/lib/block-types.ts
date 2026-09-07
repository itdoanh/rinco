import { BlockData } from '@/lib/store';

export interface HeroBlockData {
  headline: string;
  subheadline?: string;
  cta_text?: string;
  cta_link?: string;
  background_image?: string;
  background_video?: string;
  alignment?: 'left' | 'center' | 'right';
  theme?: 'light' | 'dark';
}

export interface FeatureGridBlockData {
  title?: string;
  subtitle?: string;
  columns?: 2 | 3 | 4;
  features: Array<{
    icon?: string;
    title: string;
    description?: string;
    link?: string;
  }>;
}

export interface TestimonialBlockData {
  title?: string;
  testimonials: Array<{
    name: string;
    role?: string;
    company?: string;
    avatar?: string;
    quote: string;
    rating?: number;
  }>;
  theme?: 'light' | 'dark';
}

export interface FAQBlockData {
  title?: string;
  subtitle?: string;
  items: Array<{
    question: string;
    answer: string;
  }>;
  defaultOpen?: number;
}

export interface FormBlockData {
  title?: string;
  description?: string;
  fields: Array<{
    name: string;
    type: 'text' | 'email' | 'phone' | 'textarea' | 'select';
    label: string;
    placeholder?: string;
    required?: boolean;
    options?: string[];
  }>;
  submit_text?: string;
  success_message?: string;
  webhook_url?: string;
  redirect_url?: string;
}

export interface CTABlockData {
  title: string;
  description?: string;
  button_text: string;
  button_link?: string;
  secondary_button_text?: string;
  secondary_button_link?: string;
  theme?: 'light' | 'dark' | 'gradient';
}

export interface PricingBlockData {
  title?: string;
  subtitle?: string;
  plans: Array<{
    name: string;
    price: number;
    period?: string;
    currency?: string;
    description?: string;
    features: string[];
    cta_text: string;
    cta_link?: string;
    highlighted?: boolean;
    badge?: string;
  }>;
}

export interface StatsBlockData {
  title?: string;
  stats: Array<{
    value: string;
    label: string;
    prefix?: string;
    suffix?: string;
  }>;
  theme?: 'light' | 'dark';
}

export type BlockType = BlockData['type'];

export function isHeroBlock(data: BlockData): data is BlockData & { type: 'hero' } {
  return data.type === 'hero';
}

export function isFeatureGridBlock(data: BlockData): data is BlockData & { type: 'feature_grid' } {
  return data.type === 'feature_grid';
}

export function isTestimonialBlock(data: BlockData): data is BlockData & { type: 'testimonial' } {
  return data.type === 'testimonial';
}

export function isFAQBlock(data: BlockData): data is BlockData & { type: 'faq' } {
  return data.type === 'faq';
}

export function isFormBlock(data: BlockData): data is BlockData & { type: 'form' } {
  return data.type === 'form';
}

export function isCTABlock(data: BlockData): data is BlockData & { type: 'cta' } {
  return data.type === 'cta';
}

export function isPricingBlock(data: BlockData): data is BlockData & { type: 'pricing' } {
  return data.type === 'pricing';
}

export function isStatsBlock(data: BlockData): data is BlockData & { type: 'stats' } {
  return data.type === 'stats';
}
