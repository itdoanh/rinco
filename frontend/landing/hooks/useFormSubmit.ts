"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { leadSchema, type LeadFormData } from "@/lib/schema";
import { useState, useCallback } from "react";
import { trackFormSubmit } from "@/lib/tracking";
import { trackLead } from "@/lib/pixel";

interface UseFormSubmitOptions {
  formName: string;
  tenantSlug?: string;
  pageSlug?: string;
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}

export function useFormSubmit({
  formName,
  tenantSlug,
  pageSlug,
  onSuccess,
  onError,
}: UseFormSubmitOptions) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<LeadFormData>({
    resolver: zodResolver(leadSchema),
    mode: "onBlur",
  });

  const submit = useCallback(
    async (data: LeadFormData) => {
      setIsSubmitting(true);
      setError(null);

      try {
        // Submit via API
        const response = await fetch("/api/leads", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-Slug": tenantSlug || "",
            "X-Page-Slug": pageSlug || "",
            "X-Form-Name": formName,
          },
          body: JSON.stringify({
            ...data,
            form_name: formName,
            tenant_slug: tenantSlug,
            page_slug: pageSlug,
          }),
        });

        if (!response.ok) {
          throw new Error("Submission failed");
        }

        // Track events
        trackLead(formName, "Webinar Registration", "VND", 0);
        await trackFormSubmit(
          formName,
          {
            name: data.name,
            phone: data.phone,
            email: data.email,
          },
          tenantSlug,
          pageSlug
        );

        setIsSuccess(true);
        form.reset();
        onSuccess?.();

        // Reset after 3s
        setTimeout(() => setIsSuccess(false), 3000);
      } catch (err) {
        const error = err instanceof Error ? err : new Error("Unknown error");
        setError(error.message);
        onError?.(error);
      } finally {
        setIsSubmitting(false);
      }
    },
    [formName, tenantSlug, pageSlug, form, onSuccess, onError]
  );

  return {
    form,
    submit,
    isSubmitting,
    isSuccess,
    error,
  };
}

export default useFormSubmit;
