'use client';

import { cn } from '@/lib/utils';
import { TestimonialBlockData } from '@/lib/block-types';
import { Avatar, AvatarFallback, AvatarImage } from '@repo/ui/components/ui/avatar';
import { Star } from 'lucide-react';

interface TestimonialProps {
  data: TestimonialBlockData;
  className?: string;
}

export function Testimonial({ data, className }: TestimonialProps) {
  const {
    title,
    testimonials = [],
    theme = 'light',
  } = data;

  const isDark = theme === 'dark';

  return (
    <section className={cn(
      'py-16 lg:py-24',
      isDark ? 'bg-navy-900 text-white' : 'bg-slate-50'
    )}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header */}
        {title && (
          <div className="text-center mb-12">
            <h2 className="text-3xl sm:text-4xl font-bold mb-4">
              {title}
            </h2>
          </div>
        )}

        {/* Testimonials Grid */}
        <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          {testimonials.map((testimonial, index) => (
            <div
              key={index}
              className={cn(
                'rounded-2xl p-6 shadow-lg',
                isDark ? 'bg-navy-800' : 'bg-white'
              )}
            >
              {/* Stars */}
              {testimonial.rating && (
                <div className="flex gap-1 mb-4">
                  {Array.from({ length: 5 }).map((_, i) => (
                    <Star
                      key={i}
                      className={cn(
                        'w-4 h-4',
                        i < (testimonial.rating || 0)
                          ? 'fill-yellow-400 text-yellow-400'
                          : isDark ? 'text-gray-600' : 'text-gray-300'
                      )}
                    />
                  ))}
                </div>
              )}

              {/* Quote */}
              <blockquote className={cn(
                'text-base leading-relaxed mb-6',
                isDark ? 'text-gray-300' : 'text-gray-600'
              )}>
                &ldquo;{testimonial.quote}&rdquo;
              </blockquote>

              {/* Author */}
              <div className="flex items-center gap-3">
                <Avatar className="w-12 h-12">
                  {testimonial.avatar && (
                    <AvatarImage src={testimonial.avatar} alt={testimonial.name} />
                  )}
                  <AvatarFallback className={isDark ? 'bg-navy-700' : 'bg-gray-200'}>
                    {testimonial.name.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <div className="font-semibold">{testimonial.name}</div>
                  {(testimonial.role || testimonial.company) && (
                    <div className={cn(
                      'text-sm',
                      isDark ? 'text-gray-400' : 'text-gray-500'
                    )}>
                      {[testimonial.role, testimonial.company].filter(Boolean).join(' tại ')}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
