'use client';

import { cn } from '@/lib/utils';
import { Card, CardContent } from '@repo/ui/components/ui/card';
import { FeatureGridBlockData } from '@/lib/block-types';
import {
  CheckCircle,
  Zap,
  Shield,
  Users,
  BarChart3,
  Globe,
  Clock,
  Headphones,
  type LucideIcon,
} from 'lucide-react';

const iconMap: Record<string, LucideIcon> = {
  zap: Zap,
  shield: Shield,
  users: Users,
  'bar-chart': BarChart3,
  globe: Globe,
  clock: Clock,
  headphones: Headphones,
  check: CheckCircle,
};

interface FeatureGridProps {
  data: FeatureGridBlockData;
  className?: string;
}

export function FeatureGrid({ data, className }: FeatureGridProps) {
  const {
    title,
    subtitle,
    columns = 3,
    features = [],
  } = data;

  const gridCols = {
    2: 'grid-cols-1 sm:grid-cols-2',
    3: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4',
  }[columns];

  return (
    <section className={cn('py-16 lg:py-24 bg-white', className)}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header */}
        {(title || subtitle) && (
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
        )}

        {/* Features Grid */}
        <div className={cn('grid gap-8', gridCols)}>
          {features.map((feature, index) => {
            const IconComponent = feature.icon ? iconMap[feature.icon.toLowerCase()] || CheckCircle : CheckCircle;

            return (
              <Card
                key={index}
                className="border-none shadow-lg hover:shadow-xl transition-shadow duration-300"
              >
                <CardContent className="p-6">
                  <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-orange-500 to-amber-500 flex items-center justify-center mb-4">
                    <IconComponent className="w-6 h-6 text-white" />
                  </div>

                  <h3 className="text-lg font-semibold text-gray-900 mb-2">
                    {feature.title}
                  </h3>

                  {feature.description && (
                    <p className="text-gray-600 text-sm leading-relaxed">
                      {feature.description}
                    </p>
                  )}

                  {feature.link && (
                    <a
                      href={feature.link}
                      className="inline-flex items-center text-orange-500 hover:text-orange-600 text-sm font-medium mt-3"
                    >
                      Tìm hiểu thêm
                      <svg className="w-4 h-4 ml-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                      </svg>
                    </a>
                  )}
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>
    </section>
  );
}
