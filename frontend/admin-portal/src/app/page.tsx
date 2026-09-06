'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const loginSchema = z.object({
  email: z.string().email('Email không hợp lệ'),
  password: z.string().min(8, 'Mật khẩu tối thiểu 8 ký tự'),
  tenant_slug: z.string().min(1, 'Tenant slug bắt buộc'),
});

type LoginFormData = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: '',
      password: '',
      tenant_slug: 'demo',
    },
  });

  const onSubmit = async (data: LoginFormData) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });

      if (!res.ok) {
        const body = await res.json();
        throw new Error(body.message || 'Đăng nhập thất bại');
      }

      const result = await res.json();
      localStorage.setItem('access_token', result.access_token);
      localStorage.setItem('user', JSON.stringify(result));
      window.location.href = '/dashboard';
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-navy-900 via-slate-900 to-navy-800 px-4">
      <div className="max-w-md w-full">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-gold mb-2">RINCO</h1>
          <p className="text-gray-400">Super Admin Portal</p>
        </div>

        <form
          onSubmit={handleSubmit(onSubmit)}
          className="bg-navy-800/80 backdrop-blur-lg border border-gray-700 rounded-2xl p-8 space-y-5"
        >
          <h2 className="text-2xl font-bold text-center">Đăng nhập</h2>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Email</label>
            <input
              type="email"
              autoComplete="email"
              className={`w-full bg-navy-900 border ${
                errors.email ? 'border-red-500' : 'border-gray-700'
              } rounded-lg px-4 py-2.5 focus:outline-none focus:border-gold`}
              {...register('email')}
            />
            {errors.email && <p className="text-red-400 text-xs mt-1">{errors.email.message}</p>}
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Mật khẩu</label>
            <input
              type="password"
              autoComplete="current-password"
              className={`w-full bg-navy-900 border ${
                errors.password ? 'border-red-500' : 'border-gray-700'
              } rounded-lg px-4 py-2.5 focus:outline-none focus:border-gold`}
              {...register('password')}
            />
            {errors.password && <p className="text-red-400 text-xs mt-1">{errors.password.message}</p>}
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Tenant Slug</label>
            <input
              type="text"
              className={`w-full bg-navy-900 border ${
                errors.tenant_slug ? 'border-red-500' : 'border-gray-700'
              } rounded-lg px-4 py-2.5 focus:outline-none focus:border-gold`}
              {...register('tenant_slug')}
            />
            {errors.tenant_slug && <p className="text-red-400 text-xs mt-1">{errors.tenant_slug.message}</p>}
          </div>

          {error && (
            <div className="bg-red-500/20 border border-red-500 text-red-200 px-4 py-2 rounded-lg text-sm">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-gradient-to-r from-gold to-orange text-navy-900 font-bold py-3 rounded-lg hover:scale-105 transition-transform disabled:opacity-50"
          >
            {loading ? 'Đang đăng nhập...' : 'Đăng nhập'}
          </button>

          <p className="text-center text-xs text-gray-500">
            Demo: demo@demo.com / rinco_dev_password
          </p>
        </form>
      </div>
    </div>
  );
}
