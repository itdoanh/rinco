"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { leadSchema, type LeadFormData } from "@/lib/schema";
import { useState } from "react";
import { trackFormSubmit } from "@/lib/tracking";
import { trackLead } from "@/lib/pixel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface DynamicFormProps {
  formName: string;
  onSuccess?: () => void;
  tenantSlug?: string;
  pageSlug?: string;
}

export function DynamicForm({
  formName,
  onSuccess,
  tenantSlug,
  pageSlug,
}: DynamicFormProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<LeadFormData>({
    resolver: zodResolver(leadSchema),
    mode: "onBlur",
  });

  const onSubmit = async (data: LeadFormData) => {
    setIsSubmitting(true);
    setError(null);

    try {
      // Submit via our own API proxy
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

      // Track on client
      trackLead(formName, "Webinar Registration", "VND", 0);
      // Track via tracking library (also sends CAPI)
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
      reset();
      onSuccess?.();

      // Reset success state after 3s
      setTimeout(() => setIsSuccess(false), 5000);
    } catch (err) {
      console.error("Form submit error:", err);
      setError("Có lỗi xảy ra, vui lòng thử lại.");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isSuccess) {
    return (
      <div className="text-center p-8 bg-emerald-50 rounded-2xl">
        <div className="success-icon mb-4">✓</div>
        <h3 className="text-xl font-heading font-bold text-emerald-500">
          ĐĂNG KÝ THÀNH CÔNG!
        </h3>
        <p className="text-gray-600 text-sm mt-2">
          Bạn sẽ nhận được link Zoom & Ebook qua Zalo trong thời gian sớm nhất.
        </p>
        <p className="text-sm font-bold text-navy-900 mt-3">
          Hotline: <span className="text-orange">0984386538</span>
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
      {/* Honeypot - hidden field */}
      <input
        type="text"
        {...register("website")}
        tabIndex={-1}
        autoComplete="off"
        aria-hidden="true"
        style={{ position: "absolute", left: "-9999px" }}
      />

      <div>
        <Label htmlFor="name">Họ và tên</Label>
        <Input
          id="name"
          type="text"
          placeholder="Nhập họ và tên của bạn"
          autoComplete="name"
          {...register("name")}
          className="mt-2"
        />
        {errors.name && (
          <p className="text-red-500 text-xs mt-1">{errors.name.message}</p>
        )}
      </div>

      <div>
        <Label htmlFor="phone">
          Số điện thoại (Zalo) <span className="text-red-500">*</span>
        </Label>
        <Input
          id="phone"
          type="tel"
          required
          placeholder="Nhập số điện thoại Zalo"
          autoComplete="tel"
          inputMode="numeric"
          {...register("phone")}
          className="mt-2"
        />
        {errors.phone && (
          <p className="text-red-500 text-xs mt-1">{errors.phone.message}</p>
        )}
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-xl p-3 text-red-600 text-sm">
          {error}
        </div>
      )}

      <Button
        type="submit"
        variant="cta"
        size="xl"
        disabled={isSubmitting}
        className="mt-2"
      >
        {isSubmitting ? (
          <>
            <svg className="animate-spin w-5 h-5" viewBox="0 0 24 24">
              <circle
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
                fill="none"
                opacity="0.25"
              />
              <path
                fill="currentColor"
                d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
              />
            </svg>
            ĐANG XỬ LÝ...
          </>
        ) : (
          <>🚀 GIỮ VÉ ZOOM & NHẬN EBOOK MIỄN PHÍ</>
        )}
      </Button>

      <div className="flex items-center justify-center gap-2 text-gray-400 text-xs text-center">
        <svg
          className="w-4 h-4 text-emerald-500 flex-shrink-0"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.5"
          viewBox="0 0 24 24"
        >
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        </svg>
        <span>Cam kết bảo mật 100% thông tin cá nhân theo quy định pháp luật.</span>
      </div>
    </form>
  );
}
