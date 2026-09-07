'use client';

import { cn } from '@/lib/utils';
import { StatsBlockData } from '@/lib/block-types';

interface StatsProps {
  data: StatsBlockData;
  className?: string;
}

export function Stats({ data, className }: StatsProps) {
  const {
    title,
    stats = [],
    theme = 'light',
  } = data;

  const isDark = theme === 'dark';

  return (
    <section className={cn(
      'py-16 lg:py-20',
      isDark ? 'bg-navy-900' : 'bg-white'
    )}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header */}
        {title && (
          <div className="text-center mb-12">
            <h2 className={cn(
              'text-3xl sm:text-4xl font-bold',
              isDark ? 'text-white' : 'text-gray-900'
            )}>
              {title}
            </h2>
          </div>
        )}

        {/* Stats Grid */}
        <div className={cn(
          'grid gap-8',
          stats.length === 1
            ? 'max-w-sm mx-auto'
            : stats.length === 2
            ? 'grid-cols-1 sm:grid-cols-2 max-w-2xl mx-auto'
            : 'grid-cols-2 lg:grid-cols-4'
        )}>
          {stats.map((stat, index) => (
            <div key={index} className="text-center">
              <div className={cn(
                'text-4xl sm:text-5xl lg:text-6xl font-bold mb-2',
                isDark ? 'text-gold' : 'text-orange-500'
              )}>
                {stat.prefix}
                {stat.value}
                {stat.suffix}
              </div>
              <div className={cn(
                'text-sm sm:text-base font-medium',
                isDark ? 'text-gray-400' : 'text-gray-600'
              )}>
                {stat.label}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
