'use client';

import { cn } from '@/lib/utils';
import { BlockData } from '@/lib/store';
import { Hero } from './Hero';
import { FeatureGrid } from './FeatureGrid';
import { Testimonial } from './Testimonial';
import { FAQ } from './FAQ';
import { FormBlock } from './FormBlock';
import { CTA } from './CTA';
import { PricingTable } from './PricingTable';
import { Stats } from './Stats';
import { Skeleton } from '@repo/ui/components/ui/skeleton';

interface BlockRendererProps {
  blocks: BlockData[];
  isLoading?: boolean;
  className?: string;
  tenantId?: string;
  pageId?: string;
}

export function BlockRenderer({
  blocks,
  isLoading = false,
  className,
  tenantId = '',
  pageId = '',
}: BlockRendererProps) {
  if (isLoading) {
    return (
      <div className={cn('space-y-4', className)}>
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-64 w-full rounded-xl" />
        ))}
      </div>
    );
  }

  if (!blocks || blocks.length === 0) {
    return null;
  }

  const sortedBlocks = [...blocks].sort((a, b) => a.order - b.order);

  return (
    <div className={cn('space-y-0', className)}>
      {sortedBlocks.map((block) => (
        <Block key={block.id} block={block} tenantId={tenantId} pageId={pageId} />
      ))}
    </div>
  );
}

interface BlockProps {
  block: BlockData;
  tenantId?: string;
  pageId?: string;
}

function Block({ block, tenantId, pageId }: BlockProps) {
  const { type, data } = block;

  switch (type) {
    case 'hero':
      return <Hero data={data as Parameters<typeof Hero>[0]['data']} />;

    case 'feature_grid':
      return <FeatureGrid data={data as Parameters<typeof FeatureGrid>[0]['data']} />;

    case 'testimonial':
      return <Testimonial data={data as Parameters<typeof Testimonial>[0]['data']} />;

    case 'faq':
      return <FAQ data={data as Parameters<typeof FAQ>[0]['data']} />;

    case 'form':
      return (
        <FormBlock
          data={data as Parameters<typeof FormBlock>[0]['data']}
          tenantId={tenantId}
          pageId={pageId}
        />
      );

    case 'cta':
      return <CTA data={data as Parameters<typeof CTA>[0]['data']} />;

    case 'pricing':
      return <PricingTable data={data as Parameters<typeof PricingTable>[0]['data']} />;

    case 'stats':
      return <Stats data={data as Parameters<typeof Stats>[0]['data']} />;

    default:
      console.warn(`Unknown block type: ${type}`);
      return null;
  }
}

export { Hero } from './Hero';
export { FeatureGrid } from './FeatureGrid';
export { Testimonial } from './Testimonial';
export { FAQ } from './FAQ';
export { FormBlock } from './FormBlock';
export { CTA } from './CTA';
export { PricingTable } from './PricingTable';
export { Stats } from './Stats';
