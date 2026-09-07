"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Branding } from "@/lib/schema";

interface ContactFormProps {
  tenantSlug: string;
  branding?: Branding;
  onSuccess?: () => void;
}

export function ContactForm({
  tenantSlug,
  branding,
  onSuccess,
}: ContactFormProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    const formData = new FormData(e.currentTarget);
    const data = {
      name: formData.get("name"),
      phone: formData.get("phone"),
      email: formData.get("email"),
      message: formData.get("message"),
    };

    try {
      const response = await fetch("/api/leads", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Tenant-Slug": tenantSlug,
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) throw new Error("Failed to submit");

      setIsSuccess(true);
      onSuccess?.();
      (e.target as HTMLFormElement).reset();
    } catch {
      setError("Có lỗi xảy ra, vui lòng thử lại.");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isSuccess) {
    return (
      <div className="text-center p-8 bg-emerald-50 rounded-2xl">
        <div className="text-4xl mb-4">✓</div>
        <h3 className="text-xl font-bold text-emerald-600">Gửi thành công!</h3>
        <p className="text-gray-600 mt-2">
          Chúng tôi sẽ liên hệ với bạn trong thời gian sớm nhất.
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <Label htmlFor="name">Họ và tên</Label>
        <Input id="name" name="name" required className="mt-1" />
      </div>
      <div>
        <Label htmlFor="phone">Số điện thoại</Label>
        <Input id="phone" name="phone" type="tel" required className="mt-1" />
      </div>
      <div>
        <Label htmlFor="email">Email</Label>
        <Input id="email" name="email" type="email" className="mt-1" />
      </div>
      <div>
        <Label htmlFor="message">Nội dung</Label>
        <textarea
          id="message"
          name="message"
          rows={4}
          className="mt-1 w-full rounded-lg border-2 border-input bg-background px-4 py-3 text-sm"
          required
        />
      </div>
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-red-600 text-sm">
          {error}
        </div>
      )}
      <Button
        type="submit"
        disabled={isSubmitting}
        className="w-full"
        style={{
          backgroundColor: branding?.primary || "#0A192F",
        }}
      >
        {isSubmitting ? "Đang gửi..." : "Gửi liên hệ"}
      </Button>
    </form>
  );
}

export default ContactForm;
