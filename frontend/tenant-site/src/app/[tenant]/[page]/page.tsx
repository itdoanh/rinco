'use client';

import { use } from 'react';
import { notFound } from 'next/navigation';
import { PageRenderer } from '@/components/PageRenderer';
import { BrandingProvider } from '@/components/BrandingProvider';
import { Skeleton } from '@/components/ui/skeleton';

interface TenantPageProps {
  params: Promise<{
    tenant: string;
    page: string;
  }>;
}

export default function DynamicPage({ params }: TenantPageProps) {
  const { tenant, page } = use(params);

  return (
    <BrandingProvider tenantSlug={tenant}>
      <PageRenderer tenantSlug={tenant} pageSlug={page} />
    </BrandingProvider>
  );
}
