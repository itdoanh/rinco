'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { cn } from '@/lib/utils';
import { FormBlockData, LeadFormData } from '@/lib/block-types';
import { leadSchema } from '@/lib/schemas';
import { Button } from '@rinco/ui';
import { Input } from '@rinco/ui';
import { Label } from '@rinco/ui';
import { Textarea } from '@rinco/ui';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@rinco/ui';
import { Loader2, CheckCircle } from 'lucide-react';

interface FormBlockProps {
  data: FormBlockData;
  className?: string;
  tenantId?: string;
  pageId?: string;
  onSubmit?: (data: LeadFormData) => Promise<void>;
}

export function FormBlock({
  data,
  className,
  tenantId = '',
  pageId = '',
  onSubmit,
}: FormBlockProps) {
  const {
    title,
    description,
    fields = [],
    submit_text = 'Gửi',
    success_message = 'Cảm ơn bạn! Chúng tôi sẽ liên hệ sớm.',
    redirect_url,
  } = data;

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const defaultValues: Record<string, string> = {};
  fields.forEach((field) => {
    defaultValues[field.name] = '';
  });

  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
    reset,
  } = useForm<Record<string, string>>({
    defaultValues,
    resolver: zodResolver(
      leadSchema.extend(
        fields.reduce((acc, field) => {
          if (field.type === 'email') {
            acc[field.name] = field.required
              ? leadSchema.shape.email
              : leadSchema.shape.email.optional();
          } else if (field.type === 'phone') {
            acc[field.name] = field.required
              ? leadSchema.shape.phone
              : leadSchema.shape.phone.optional();
          } else {
            acc[field.name] = field.required
              ? leadSchema.shape.name.min(1)
              : leadSchema.shape.name.min(0).optional();
          }
          return acc;
        }, {} as Record<string, unknown>)
      )
    ),
  });

  const handleFormSubmit = async (formData: Record<string, string>) => {
    setIsSubmitting(true);
    setError(null);

    try {
      const payload: LeadFormData = {
        ...formData,
        tenant_id: tenantId,
        page_id: pageId,
      };

      if (onSubmit) {
        await onSubmit(payload);
      } else {
        // Default: submit to API
        const response = await fetch('/api/leads', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });

        if (!response.ok) {
          throw new Error('Failed to submit form');
        }
      }

      setIsSuccess(true);
      reset();

      if (redirect_url) {
        setTimeout(() => {
          window.location.href = redirect_url;
        }, 2000);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Có lỗi xảy ra. Vui lòng thử lại.');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isSuccess) {
    return (
      <section className={cn('py-16 lg:py-24 bg-slate-50', className)}>
        <div className="max-w-md mx-auto px-4 text-center">
          <div className="w-16 h-16 rounded-full bg-green-100 flex items-center justify-center mx-auto mb-4">
            <CheckCircle className="w-8 h-8 text-green-500" />
          </div>
          <h3 className="text-2xl font-bold text-gray-900 mb-2">Thành công!</h3>
          <p className="text-gray-600">{success_message}</p>
        </div>
      </section>
    );
  }

  return (
    <section className={cn('py-16 lg:py-24 bg-slate-50', className)}>
      <div className="max-w-md mx-auto px-4 sm:px-6">
        {/* Header */}
        {(title || description) && (
          <div className="text-center mb-8">
            {title && (
              <h2 className="text-3xl font-bold text-gray-900 mb-4">
                {title}
              </h2>
            )}
            {description && (
              <p className="text-gray-600">
                {description}
              </p>
            )}
          </div>
        )}

        {/* Form */}
        <form
          onSubmit={handleSubmit(handleFormSubmit)}
          className="bg-white rounded-2xl shadow-xl p-6 sm:p-8 space-y-5"
        >
          {fields.map((field) => (
            <div key={field.name}>
              <Label htmlFor={field.name} className="mb-1.5 block">
                {field.label}
                {field.required && <span className="text-red-500 ml-1">*</span>}
              </Label>

              {field.type === 'textarea' ? (
                <Textarea
                  id={field.name}
                  placeholder={field.placeholder}
                  {...register(field.name)}
                  className={cn(
                    errors[field.name] && 'border-red-500 focus-visible:ring-red-500'
                  )}
                  rows={4}
                />
              ) : field.type === 'select' && field.options ? (
                <Select
                  onValueChange={(value) => setValue(field.name, value)}
                  defaultValue=""
                >
                  <SelectTrigger
                    id={field.name}
                    className={cn(
                      errors[field.name] && 'border-red-500 focus-visible:ring-red-500'
                    )}
                  >
                    <SelectValue placeholder={field.placeholder || 'Chọn...'} />
                  </SelectTrigger>
                  <SelectContent>
                    {field.options.map((option) => (
                      <SelectItem key={option} value={option}>
                        {option}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  id={field.name}
                  type={field.type === 'email' ? 'email' : field.type === 'phone' ? 'tel' : 'text'}
                  placeholder={field.placeholder}
                  {...register(field.name)}
                  className={cn(
                    errors[field.name] && 'border-red-500 focus-visible:ring-red-500'
                  )}
                />
              )}

              {errors[field.name] && (
                <p className="text-red-500 text-sm mt-1">
                  {errors[field.name]?.message as string}
                </p>
              )}
            </div>
          ))}

          {error && (
            <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
              {error}
            </div>
          )}

          <Button
            type="submit"
            size="lg"
            className="w-full"
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Đang gửi...
              </>
            ) : (
              submit_text
            )}
          </Button>

          <p className="text-xs text-gray-500 text-center">
            Bằng cách gửi form, bạn đồng ý với{' '}
            <a href="/privacy" className="underline hover:text-gray-700">
              Chính sách bảo mật
            </a>{' '}
            của chúng tôi.
          </p>
        </form>
      </div>
    </section>
  );
}
