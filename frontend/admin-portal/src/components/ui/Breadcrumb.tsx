'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ChevronRight, Home } from 'lucide-react';

interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface BreadcrumbProps {
  items?: BreadcrumbItem[];
}

export default function Breadcrumb({ items }: BreadcrumbProps) {
  const pathname = usePathname();

  // Generate breadcrumbs from pathname if not provided
  const breadcrumbs: BreadcrumbItem[] = items || generateBreadcrumbs(pathname);

  return (
    <nav className="flex items-center gap-1 text-sm">
      <Link
        href="/dashboard"
        className="p-1.5 rounded hover:bg-navy-700 text-gray-400 hover:text-white transition-colors"
      >
        <Home className="w-4 h-4" />
      </Link>

      {breadcrumbs.map((item, index) => (
        <div key={index} className="flex items-center gap-1">
          <ChevronRight className="w-4 h-4 text-gray-600" />
          {item.href ? (
            <Link
              href={item.href}
              className="px-2 py-1 rounded hover:bg-navy-700 text-gray-400 hover:text-white transition-colors capitalize"
            >
              {item.label}
            </Link>
          ) : (
            <span className="px-2 py-1 text-white capitalize">{item.label}</span>
          )}
        </div>
      ))}
    </nav>
  );
}

function generateBreadcrumbs(pathname: string): BreadcrumbItem[] {
  const segments = pathname.split('/').filter(Boolean);

  return segments.map((segment, index) => {
    // Check if segment is an ID (UUID or numeric)
    const isId = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(segment) ||
                 /^\d+$/.test(segment);

    const label = segment
      .replace(/-/g, ' ')
      .replace(/\b\w/g, (l) => l.toUpperCase());

    const href = '/' + segments.slice(0, index + 1).join('/');

    return {
      label: isId ? 'Detail' : label,
      href: isId ? undefined : href,
    };
  });
}
