'use client';

import { cn } from '@/lib/utils';
import { CTABlockData } from '@/lib/block-types';
import { Button } from '@repo/ui/components/ui/button';
import { ArrowRight } from 'lucide-react';

interface CTAProps {
  data: CTABlockData;
  className?: string;
}

export function CTA({ data, className }: CTAProps) {
  const {
    title,
    description,
    button_text,
    button_link = '#',
    secondary_button_text,
    secondary_button_link = '#',
    theme = 'gradient',
  } = data;

  const themeClasses = {
    light: 'bg-white',
    dark: 'bg-navy-900 text-white',
    gradient: 'bg-gradient-to-r from-orange-500 to-amber-500',
  }[theme];

  const isDark = theme === 'dark';
  const isGradient = theme === 'gradient';

  return (
    <section className={cn('py-16 lg:py-24', themeClasses, className)}>
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <h2 className={cn(
          'text-3xl sm:text-4xl lg:text-5xl font-bold mb-6',
          isGradient ? 'text-white' : isDark ? '' : 'text-gray-900'
        )}>
          {title}
        </h2>

        {description && (
          <p className={cn(
            'text-lg mb-8 max-w-2xl mx-auto',
            isGradient ? 'text-white/90' : isDark ? 'text-gray-300' : 'text-gray-600'
          )}>
            {description}
          </p>
        )}

        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <Button
            size="xl"
            variant={isGradient ? 'secondary' : 'default'}
            asChild
            className={cn(
              'group',
              isGradient && 'bg-white text-orange-600 hover:bg-gray-100'
            )}
          >
            <a href={button_link}>
              {button_text}
              <ArrowRight className="ml-2 w-5 h-5 group-hover:translate-x-1 transition-transform" />
            </a>
          </Button>

          {secondary_button_text && (
            <Button
              size="xl"
              variant={isGradient ? 'outline' : 'outline'}
              asChild
              className={cn(
                'border-2',
                isGradient
                  ? 'border-white text-white hover:bg-white/10'
                  : isDark
                  ? 'border-gray-600 text-white hover:bg-gray-800'
                  : 'border-gray-300 text-gray-900 hover:bg-gray-50'
              )}
            >
              <a href={secondary_button_link}>
                {secondary_button_text}
              </a>
            </Button>
          )}
        </div>
      </div>
    </section>
  );
}
