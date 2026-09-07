'use client';

import { cn } from '@/lib/utils';
import { PricingBlockData } from '@/lib/block-types';
import { Button } from '@rinco/ui';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@rinco/ui';
import { Check } from 'lucide-react';

interface PricingTableProps {
  data: PricingBlockData;
  className?: string;
}

export function PricingTable({ data, className }: PricingTableProps) {
  const {
    title,
    subtitle,
    plans = [],
  } = data;

  const formatPrice = (price: number, currency = 'VND', period?: string) => {
    const formatted = new Intl.NumberFormat('vi-VN', {
      style: 'currency',
      currency,
      minimumFractionDigits: 0,
    }).format(price);

    return period ? `${formatted}/${period}` : formatted;
  };

  return (
    <section className={cn('py-16 lg:py-24 bg-slate-50', className)}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="text-center mb-12">
          {title && (
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900 mb-4">
              {title}
            </h2>
          )}
          {subtitle && (
            <p className="text-lg text-gray-600 max-w-2xl mx-auto">
              {subtitle}
            </p>
          )}
        </div>

        {/* Pricing Cards */}
        <div className={cn(
          'grid gap-8',
          plans.length === 1 ? 'max-w-md mx-auto' : 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3'
        )}>
          {plans.map((plan, index) => (
            <Card
              key={index}
              className={cn(
                'relative overflow-hidden transition-transform hover:-translate-y-1',
                plan.highlighted
                  ? 'border-2 border-orange-500 shadow-xl scale-105'
                  : 'border-gray-200 shadow-lg'
              )}
            >
              {/* Badge */}
              {plan.badge && (
                <div className="absolute top-0 right-0 bg-orange-500 text-white text-xs font-bold px-3 py-1 rounded-bl-lg">
                  {plan.badge}
                </div>
              )}

              <CardHeader className="text-center pb-4">
                <CardTitle className="text-2xl font-bold">{plan.name}</CardTitle>
                {plan.description && (
                  <CardDescription className="mt-2">{plan.description}</CardDescription>
                )}
              </CardHeader>

              <CardContent className="text-center">
                <div className="mb-6">
                  <span className="text-4xl font-bold text-gray-900">
                    {formatPrice(plan.price, plan.currency, plan.period)}
                  </span>
                </div>

                <ul className="space-y-3 text-left">
                  {plan.features.map((feature, fIndex) => (
                    <li key={fIndex} className="flex items-start gap-2">
                      <Check className="w-5 h-5 text-green-500 flex-shrink-0 mt-0.5" />
                      <span className="text-gray-600 text-sm">{feature}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>

              <CardFooter className="pt-4">
                <Button
                  size="lg"
                  className="w-full"
                  variant={plan.highlighted ? 'default' : 'outline'}
                  asChild
                >
                  <a href={plan.cta_link || '#'}>
                    {plan.cta_text}
                  </a>
                </Button>
              </CardFooter>
            </Card>
          ))}
        </div>
      </div>
    </section>
  );
}
