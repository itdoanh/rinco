import { useState, useCallback } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { leadSchema, type LeadFormData } from '@/lib/schemas';

interface UseFormSubmitOptions {
  tenantId: string;
  pageId: string;
  webhookUrl?: string;
  onSuccess?: (data: LeadFormData) => void;
  onError?: (error: Error) => void;
}

interface UseFormSubmitReturn {
  isSubmitting: boolean;
  isSuccess: boolean;
  error: string | null;
  submit: (data: Record<string, string>) => Promise<void>;
  reset: () => void;
}

export function useFormSubmit({
  tenantId,
  pageId,
  webhookUrl,
  onSuccess,
  onError,
}: UseFormSubmitOptions): UseFormSubmitReturn {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = useCallback(
    async (data: Record<string, string>) => {
      setIsSubmitting(true);
      setError(null);

      try {
        // Validate with Zod
        const validatedData = leadSchema.parse({
          ...data,
          tenant_id: tenantId,
          page_id: pageId,
        });

        // Get UTM params
        const utmParams = getUTMFromCookie();
        const payload: LeadFormData = {
          ...validatedData,
          ...utmParams,
        };

        // Submit to API
        const response = await fetch('/api/leads', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });

        if (!response.ok) {
          throw new Error('Failed to submit form');
        }

        // Submit to webhook if provided
        if (webhookUrl) {
          await fetch(webhookUrl, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
          }).catch((err) => {
            console.warn('Webhook submission failed:', err);
          });
        }

        setIsSuccess(true);
        onSuccess?.(payload);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Có lỗi xảy ra';
        setError(errorMessage);
        onError?.(new Error(errorMessage));
      } finally {
        setIsSubmitting(false);
      }
    },
    [tenantId, pageId, webhookUrl, onSuccess, onError]
  );

  const reset = useCallback(() => {
    setIsSubmitting(false);
    setIsSuccess(false);
    setError(null);
  }, []);

  return {
    isSubmitting,
    isSuccess,
    error,
    submit,
    reset,
  };
}

function getUTMFromCookie(): Partial<LeadFormData> {
  if (typeof document === 'undefined') return {};

  const getCookie = (name: string) => {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()?.split(';').shift() || '';
    return '';
  };

  return {
    utm_source: getCookie('_rinco_utm_source'),
    utm_medium: getCookie('_rinco_utm_medium'),
    utm_campaign: getCookie('_rinco_utm_campaign'),
    utm_content: getCookie('_rinco_utm_content'),
    utm_term: getCookie('_rinco_utm_term'),
    referrer: document.referrer || undefined,
  };
}
