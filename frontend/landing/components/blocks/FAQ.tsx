'use client';

import { useState } from 'react';
import { cn } from '@/lib/utils';
import { FAQBlockData } from '@/lib/block-types';
import { ChevronDown } from 'lucide-react';

interface FAQProps {
  data: FAQBlockData;
  className?: string;
}

export function FAQ({ data, className }: FAQProps) {
  const {
    title,
    subtitle,
    items = [],
    defaultOpen = 0,
  } = data;

  const [openIndex, setOpenIndex] = useState<number | null>(defaultOpen);

  const toggleItem = (index: number) => {
    setOpenIndex(openIndex === index ? null : index);
  };

  return (
    <section className={cn('py-16 lg:py-24 bg-white', className)}>
      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="text-center mb-12">
          {title && (
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900 mb-4">
              {title}
            </h2>
          )}
          {subtitle && (
            <p className="text-lg text-gray-600">
              {subtitle}
            </p>
          )}
        </div>

        {/* FAQ Items */}
        <div className="space-y-4">
          {items.map((item, index) => (
            <div
              key={index}
              className={cn(
                'rounded-xl border transition-all duration-200',
                openIndex === index
                  ? 'border-orange-500 shadow-md'
                  : 'border-gray-200 hover:border-gray-300'
              )}
            >
              <button
                type="button"
                onClick={() => toggleItem(index)}
                className="w-full flex items-center justify-between p-5 text-left"
                aria-expanded={openIndex === index}
              >
                <span className="font-semibold text-gray-900 pr-4">
                  {item.question}
                </span>
                <ChevronDown
                  className={cn(
                    'w-5 h-5 text-gray-500 flex-shrink-0 transition-transform duration-200',
                    openIndex === index && 'rotate-180'
                  )}
                />
              </button>

              <div
                className={cn(
                  'overflow-hidden transition-all duration-200',
                  openIndex === index ? 'max-h-96' : 'max-h-0'
                )}
              >
                <div className="p-5 pt-0 text-gray-600 leading-relaxed">
                  {item.answer}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
