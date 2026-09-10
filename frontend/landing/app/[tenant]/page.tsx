'use client';

import { use, useEffect } from 'react';
import { notFound } from 'next/navigation';
import { BlockRenderer } from '@/components/blocks';
import { useLandingStore, type BlockData } from '@/lib/store';
import { Skeleton } from '@rinco/ui'

interface TenantPageProps {
  params: Promise<{
    tenant: string;
  }>;
}

export default function TenantPage({ params }: TenantPageProps) {
  const { tenant } = use(params);
  const { pageData, isLoading, setPageData, setLoading, setError } = useLandingStore();

  // Fetch page data
  useEffect(() => {
    async function fetchPage() {
      setLoading(true);
      try {
        const response = await fetch(`/api/pages/${tenant}/default`);
        if (!response.ok) {
          throw new Error('Page not found');
        }
        const data = await response.json();
        setPageData(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load page');
      }
    }

    if (tenant) {
      fetchPage();
    }
  }, [tenant, setLoading, setPageData, setError]);

  if (isLoading) {
    return (
      <div className="min-h-screen">
        <Skeleton className="h-96 w-full rounded-none" />
        <div className="max-w-7xl mx-auto px-4 py-8 space-y-4">
          <Skeleton className="h-64 w-full rounded-xl" />
          <Skeleton className="h-96 w-full rounded-xl" />
        </div>
      </div>
    );
  }

  if (!pageData) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-900 mb-4">404</h1>
          <p className="text-gray-600">Trang không tồn tại</p>
        </div>
      </div>
    );
  }

  return (
    <main className="min-h-screen">
      <BlockRenderer
        blocks={pageData.blocks}
        tenantId={pageData.tenant_id}
        pageId={pageData.id}
      />
    </main>
  );
}
