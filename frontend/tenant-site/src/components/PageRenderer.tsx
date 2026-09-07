'use client';

import { useEffect, useState } from 'react';
import { pagesApi, type Page, type Block } from '@/lib/api';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

interface PageRendererProps {
  tenantSlug: string;
  pageSlug: string;
}

export function PageRenderer({ tenantSlug, pageSlug }: PageRendererProps) {
  const [page, setPage] = useState<Page | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchPage() {
      setIsLoading(true);
      setError(null);
      try {
        const data = await pagesApi.getPage(tenantSlug, pageSlug);
        setPage(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load page');
      } finally {
        setIsLoading(false);
      }
    }

    fetchPage();
  }, [tenantSlug, pageSlug]);

  if (isLoading) {
    return <PageSkeleton />;
  }

  if (error || !page) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-900 mb-4">404</h1>
          <p className="text-gray-600">Trang không tồn tại</p>
        </div>
      </div>
    );
  }

  const sortedBlocks = [...page.blocks].sort((a, b) => a.order - b.order);

  return (
    <main className="min-h-screen">
      {sortedBlocks.map((block) => (
        <BlockRenderer key={block.id} block={block} />
      ))}
    </main>
  );
}

function BlockRenderer({ block }: { block: Block }) {
  const { type, data } = block;

  switch (type) {
    case 'hero':
      return <HeroBlock data={data} />;
    case 'features':
      return <FeaturesBlock data={data} />;
    case 'pricing':
      return <PricingBlock data={data} />;
    case 'testimonials':
      return <TestimonialsBlock data={data} />;
    case 'faq':
      return <FAQBlock data={data} />;
    case 'cta':
      return <CTABlock data={data} />;
    case 'form':
      return <FormBlock data={data} />;
    default:
      return null;
  }
}

// Block Components
function HeroBlock({ data }: { data: Record<string, unknown> }) {
  return (
    <section className="relative bg-gradient-to-br from-primary-500 to-primary-700 text-white py-24 lg:py-32">
      <div className="max-w-4xl mx-auto px-4 text-center">
        <h1 className="text-4xl lg:text-6xl font-bold mb-6">{data.headline as string}</h1>
        {data.subheadline && (
          <p className="text-xl text-white/90 mb-8 max-w-2xl mx-auto">
            {data.subheadline as string}
          </p>
        )}
        {data.cta_text && (
          <a
            href={data.cta_link as string || '#'}
            className="inline-block bg-white text-primary-700 px-8 py-4 rounded-full font-semibold hover:bg-gray-100 transition-colors"
          >
            {data.cta_text as string}
          </a>
        )}
      </div>
    </section>
  );
}

function FeaturesBlock({ data }: { data: Record<string, unknown> }) {
  const features = (data.features as Array<{ title: string; description?: string }>) || [];

  return (
    <section className="py-16 lg:py-24 bg-white">
      <div className="max-w-7xl mx-auto px-4">
        {data.title && (
          <h2 className="text-3xl lg:text-4xl font-bold text-center mb-12">{data.title as string}</h2>
        )}
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
          {features.map((feature, i) => (
            <div key={i} className="p-6 rounded-xl bg-gray-50">
              <h3 className="text-xl font-semibold mb-2">{feature.title}</h3>
              {feature.description && (
                <p className="text-gray-600">{feature.description}</p>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function PricingBlock({ data }: { data: Record<string, unknown> }) {
  const plans = (data.plans as Array<{
    name: string;
    price: number;
    features: string[];
    highlighted?: boolean;
  }>) || [];

  return (
    <section className="py-16 lg:py-24 bg-gray-50">
      <div className="max-w-7xl mx-auto px-4">
        {data.title && (
          <h2 className="text-3xl lg:text-4xl font-bold text-center mb-12">{data.title as string}</h2>
        )}
        <div className="grid md:grid-cols-3 gap-8">
          {plans.map((plan, i) => (
            <div
              key={i}
              className={cn(
                'p-8 rounded-2xl',
                plan.highlighted ? 'bg-primary-500 text-white' : 'bg-white'
              )}
            >
              <h3 className="text-2xl font-bold mb-4">{plan.name}</h3>
              <div className="text-4xl font-bold mb-6">
                {plan.price.toLocaleString()}đ
              </div>
              <ul className="space-y-3 mb-8">
                {plan.features.map((feature, j) => (
                  <li key={j} className="flex items-start gap-2">
                    <span className={plan.highlighted ? 'text-white' : 'text-primary-500'}>✓</span>
                    <span className={plan.highlighted ? 'text-white/90' : 'text-gray-600'}>{feature}</span>
                  </li>
                ))}
              </ul>
              <button
                className={cn(
                  'w-full py-3 rounded-lg font-semibold transition-colors',
                  plan.highlighted
                    ? 'bg-white text-primary-500 hover:bg-gray-100'
                    : 'bg-primary-500 text-white hover:bg-primary-600'
                )}
              >
                Get Started
              </button>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function TestimonialsBlock({ data }: { data: Record<string, unknown> }) {
  const testimonials = (data.testimonials as Array<{
    name: string;
    quote: string;
    role?: string;
  }>) || [];

  return (
    <section className="py-16 lg:py-24 bg-white">
      <div className="max-w-7xl mx-auto px-4">
        {data.title && (
          <h2 className="text-3xl lg:text-4xl font-bold text-center mb-12">{data.title as string}</h2>
        )}
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
          {testimonials.map((testimonial, i) => (
            <div key={i} className="p-6 rounded-xl bg-gray-50">
              <p className="text-gray-600 mb-4">&ldquo;{testimonial.quote}&rdquo;</p>
              <div>
                <div className="font-semibold">{testimonial.name}</div>
                {testimonial.role && (
                  <div className="text-sm text-gray-500">{testimonial.role}</div>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function FAQBlock({ data }: { data: Record<string, unknown> }) {
  const items = (data.items as Array<{ question: string; answer: string }>) || [];

  return (
    <section className="py-16 lg:py-24 bg-gray-50">
      <div className="max-w-3xl mx-auto px-4">
        {data.title && (
          <h2 className="text-3xl lg:text-4xl font-bold text-center mb-12">{data.title as string}</h2>
        )}
        <div className="space-y-4">
          {items.map((item, i) => (
            <div key={i} className="p-6 bg-white rounded-xl">
              <h3 className="text-lg font-semibold mb-2">{item.question}</h3>
              <p className="text-gray-600">{item.answer}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function CTABlock({ data }: { data: Record<string, unknown> }) {
  return (
    <section className="py-16 lg:py-24 bg-primary-500 text-white">
      <div className="max-w-4xl mx-auto px-4 text-center">
        <h2 className="text-3xl lg:text-4xl font-bold mb-4">{data.title as string}</h2>
        {data.description && (
          <p className="text-xl text-white/90 mb-8">{data.description as string}</p>
        )}
        {data.button_text && (
          <a
            href={data.button_link as string || '#'}
            className="inline-block bg-white text-primary-700 px-8 py-4 rounded-full font-semibold hover:bg-gray-100 transition-colors"
          >
            {data.button_text as string}
          </a>
        )}
      </div>
    </section>
  );
}

function FormBlock({ data }: { data: Record<string, unknown> }) {
  return (
    <section className="py-16 lg:py-24 bg-white">
      <div className="max-w-md mx-auto px-4">
        {data.title && (
          <h2 className="text-3xl font-bold text-center mb-8">{data.title as string}</h2>
        )}
        <form className="space-y-4" onSubmit={(e) => e.preventDefault()}>
          <input
            type="text"
            placeholder="Name"
            className="w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
          <input
            type="email"
            placeholder="Email"
            className="w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
          <input
            type="tel"
            placeholder="Phone"
            className="w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
          <button
            type="submit"
            className="w-full bg-primary-500 text-white py-3 rounded-lg font-semibold hover:bg-primary-600 transition-colors"
          >
            {data.submit_text as string || 'Submit'}
          </button>
        </form>
      </div>
    </section>
  );
}

function PageSkeleton() {
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
