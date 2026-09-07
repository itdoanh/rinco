'use client';

import { cn } from '@/lib/utils';
import { Button } from '@repo/ui/components/ui/button';
import { ArrowRight, Play, CheckCircle } from 'lucide-react';
import { HeroBlockData } from '@/lib/block-types';

interface HeroProps {
  data: HeroBlockData;
  className?: string;
}

export function Hero({ data, className }: HeroProps) {
  const {
    headline,
    subheadline,
    cta_text = 'Bắt đầu ngay',
    cta_link = '#',
    background_image,
    alignment = 'center',
    theme = 'light',
  } = data;

  const isDark = theme === 'dark';
  const textAlignClass = {
    left: 'text-left items-start',
    center: 'text-center items-center',
    right: 'text-right items-end',
  }[alignment];

  return (
    <section
      className={cn(
        'relative min-h-[600px] flex items-center justify-center py-20 lg:py-32',
        isDark ? 'bg-navy-900 text-white' : 'bg-gradient-to-br from-slate-50 to-white text-gray-900',
        className
      )}
      style={
        background_image
          ? { backgroundImage: `url(${background_image})`, backgroundSize: 'cover', backgroundPosition: 'center' }
          : undefined
      }
    >
      {/* Overlay for background images */}
      {background_image && (
        <div className="absolute inset-0 bg-black/50" />
      )}

      {/* Content */}
      <div className={cn('relative z-10 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col gap-6', textAlignClass)}>
        <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold leading-tight">
          {headline}
        </h1>

        {subheadline && (
          <p className={cn(
            'text-lg sm:text-xl max-w-2xl',
            isDark ? 'text-gray-300' : 'text-gray-600'
          )}>
            {subheadline}
          </p>
        )}

        <div className={cn('flex flex-col sm:flex-row gap-4 mt-4', alignment === 'center' ? 'justify-center' : '')}>
          <Button size="xl" asChild className="group">
            <a href={cta_link}>
              {cta_text}
              <ArrowRight className="ml-2 w-5 h-5 group-hover:translate-x-1 transition-transform" />
            </a>
          </Button>

          {data.background_video && (
            <Button variant="outline" size="xl" className={cn(
              !isDark && 'border-gray-300',
            )} asChild>
              <a href={data.background_video} target="_blank" rel="noopener noreferrer">
                <Play className="mr-2 w-5 h-5" />
                Xem Demo
              </a>
            </Button>
          )}
        </div>

        {/* Trust badges */}
        <div className={cn(
          'flex flex-wrap gap-4 mt-8 text-sm',
          alignment === 'center' ? 'justify-center' : '',
          isDark ? 'text-gray-400' : 'text-gray-500'
        )}>
          <span className="flex items-center gap-2">
            <CheckCircle className="w-4 h-4 text-green-500" />
            Miễn phí 14 ngày
          </span>
          <span className="flex items-center gap-2">
            <CheckCircle className="w-4 h-4 text-green-500" />
            Không cần thẻ tín dụng
          </span>
          <span className="flex items-center gap-2">
            <CheckCircle className="w-4 h-4 text-green-500" />
            Hỗ trợ 24/7
          </span>
        </div>
      </div>
    </section>
  );
}
