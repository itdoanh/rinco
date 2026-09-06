'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const leadSchema = z.object({
  full_name: z.string().min(2, 'Vui lòng nhập họ tên'),
  phone: z.string().regex(/^(0|\+84)[0-9]{9,10}$/, 'Số điện thoại không hợp lệ'),
  email: z.string().email('Email không hợp lệ').optional().or(z.literal('')),
});

type LeadFormData = z.infer<typeof leadSchema>;

interface LeadFormProps {
  formId?: string;
  tenantId?: string;
  ctaText?: string;
}

export default function LeadForm({
  formId = 'webinar-2026',
  tenantId = 'demo',
  ctaText = 'ĐĂNG KÝ NGAY - GIỮ VÉ MIỄN PHÍ',
}: LeadFormProps) {
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<LeadFormData>({
    resolver: zodResolver(leadSchema),
  });

  const onSubmit = async (data: LeadFormData) => {
    setStatus('loading');
    setErrorMessage(null);

    try {
      // Capture UTM params from URL
      const urlParams = new URLSearchParams(window.location.search);
      const utm = {
        utm_source: urlParams.get('utm_source') || '',
        utm_medium: urlParams.get('utm_medium') || '',
        utm_campaign: urlParams.get('utm_campaign') || '',
        utm_content: urlParams.get('utm_content') || '',
        utm_term: urlParams.get('utm_term') || '',
        fbclid: urlParams.get('fbclid') || '',
        gclid: urlParams.get('gclid') || '',
        ttclid: urlParams.get('ttclid') || '',
      };

      // Generate idempotency key
      const idempotencyKey = `${data.phone}-${Date.now()}`;

      // Browser fingerprint (simple)
      const userAgent = navigator.userAgent;

      // Submit via Tracking SDK
      const response = await fetch('/api/v1/leads/submit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          form_id: formId,
          tenant_id: tenantId,
          ...data,
          ...utm,
          user_agent: userAgent,
          idempotency_key: idempotencyKey,
          client_sent_at: Date.now(),
        }),
      });

      const result = await response.json();

      if (response.ok && result.accepted) {
        setStatus('success');
        reset();

        // Track conversion via browser pixel
        if (typeof window !== 'undefined' && (window as any).fbq) {
          (window as any).fbq('track', 'Lead', {
            content_name: formId,
            value: 0,
            currency: 'VND',
          });
        }

        // Track via GA4
        if (typeof window !== 'undefined' && (window as any).gtag) {
          (window as any).gtag('event', 'generate_lead', {
            form_id: formId,
            method: 'organic',
          });
        }

        // Show success message + redirect after 2s
        setTimeout(() => {
          window.location.href = '/thank-you';
        }, 2000);
      } else {
        setStatus('error');
        setErrorMessage(result.message || 'Đăng ký thất bại, vui lòng thử lại');
      }
    } catch (err) {
      setStatus('error');
      setErrorMessage('Lỗi kết nối, vui lòng kiểm tra mạng và thử lại');
      console.error('Lead submission error:', err);
    }
  };

  if (status === 'success') {
    return (
      <div className="bg-green-500/20 border-2 border-green-500 rounded-2xl p-8 text-center">
        <div className="text-6xl mb-4">🎉</div>
        <h3 className="text-2xl font-bold mb-2">Đăng ký thành công!</h3>
        <p className="text-gray-200">Chúng tôi sẽ gửi link Webinar Zoom qua SMS và Email trong ít phút.</p>
        <p className="text-sm text-gray-400 mt-4">Đang chuyển hướng...</p>
      </div>
    );
  }

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="bg-white/5 backdrop-blur-lg border-2 border-gold/30 rounded-2xl p-6 md:p-8 space-y-5"
    >
      <h2 className="text-2xl md:text-3xl font-bold text-center text-white">
        Giữ vé Webinar Zoom <span className="text-gold">miễn phí</span>
      </h2>
      <p className="text-center text-gray-300 text-sm md:text-base">
        50 NĐT đăng ký sớm nhất sẽ nhận Bộ 10 Ebook Thực Chiến từ chuyên gia MXV
      </p>

      <div>
        <input
          type="text"
          placeholder="Họ và tên *"
          autoComplete="name"
          className={`w-full bg-navy-800/80 border ${
            errors.full_name ? 'border-red-500' : 'border-gray-600'
          } text-white px-4 py-3 rounded-lg focus:outline-none focus:border-gold transition`}
          {...register('full_name')}
        />
        {errors.full_name && (
          <p className="text-red-400 text-sm mt-1">{errors.full_name.message}</p>
        )}
      </div>

      <div>
        <input
          type="tel"
          placeholder="Số điện thoại *"
          autoComplete="tel"
          className={`w-full bg-navy-800/80 border ${
            errors.phone ? 'border-red-500' : 'border-gray-600'
          } text-white px-4 py-3 rounded-lg focus:outline-none focus:border-gold transition`}
          {...register('phone')}
        />
        {errors.phone && (
          <p className="text-red-400 text-sm mt-1">{errors.phone.message}</p>
        )}
      </div>

      <div>
        <input
          type="email"
          placeholder="Email (tùy chọn)"
          autoComplete="email"
          className={`w-full bg-navy-800/80 border ${
            errors.email ? 'border-red-500' : 'border-gray-600'
          } text-white px-4 py-3 rounded-lg focus:outline-none focus:border-gold transition`}
          {...register('email')}
        />
        {errors.email && (
          <p className="text-red-400 text-sm mt-1">{errors.email.message}</p>
        )}
      </div>

      {errorMessage && (
        <div className="bg-red-500/20 border border-red-500 text-red-200 px-4 py-3 rounded-lg text-sm">
          {errorMessage}
        </div>
      )}

      <button
        type="submit"
        disabled={status === 'loading'}
        className="w-full bg-gradient-to-r from-gold to-orange text-navy-900 font-bold text-lg px-6 py-4 rounded-lg hover:scale-105 transition-transform shadow-2xl animate-pulse-glow disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100"
      >
        {status === 'loading' ? '⏳ Đang xử lý...' : `🚀 ${ctaText}`}
      </button>

      <p className="text-xs text-gray-400 text-center">
        Bằng việc đăng ký, bạn đồng ý nhận thông tin từ APEX Fintech. Chúng tôi cam kết bảo mật thông tin cá nhân.
      </p>
    </form>
  );
}
