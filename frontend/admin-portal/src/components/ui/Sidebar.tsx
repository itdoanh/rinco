'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import {
  LayoutDashboard,
  Building2,
  Users,
  Target,
  BarChart3,
  Settings,
  Shield,
  Activity,
  FileText,
  Globe,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { useState } from 'react';

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Tenants', href: '/tenants', icon: Building2 },
  { name: 'Users', href: '/users', icon: Users },
  { name: 'Leads', href: '/leads', icon: Target },
  { name: 'Analytics', href: '/analytics', icon: BarChart3 },
  { name: 'Landing Pages', href: '/pages', icon: Globe },
  { name: 'Reports', href: '/reports', icon: FileText },
];

const systemNavigation = [
  { name: 'System Health', href: '/system/health', icon: Activity },
  { name: 'Settings', href: '/settings', icon: Settings },
  { name: 'Roles & Permissions', href: '/roles', icon: Shield },
];

export default function Sidebar() {
  const pathname = usePathname();
  const [isCollapsed, setIsCollapsed] = useState(false);

  return (
    <aside
      className={cn(
        'fixed left-0 top-0 h-screen bg-navy-800 border-r border-gray-700 transition-all duration-300 z-50',
        isCollapsed ? 'w-16' : 'w-64'
      )}
    >
      <div className="flex flex-col h-full">
        {/* Logo */}
        <div className="h-16 flex items-center justify-between px-4 border-b border-gray-700">
          {!isCollapsed && (
            <Link href="/dashboard" className="text-2xl font-bold text-gold">
              RINCO
            </Link>
          )}
          {isCollapsed && (
            <Link href="/dashboard" className="mx-auto text-2xl font-bold text-gold">
              R
            </Link>
          )}
          <button
            onClick={() => setIsCollapsed(!isCollapsed)}
            className="p-1 rounded hover:bg-navy-700 text-gray-400 hover:text-white transition-colors"
          >
            {isCollapsed ? (
              <ChevronRight className="w-5 h-5" />
            ) : (
              <ChevronLeft className="w-5 h-5" />
            )}
          </button>
        </div>

        {/* Main Navigation */}
        <nav className="flex-1 py-4 px-2 space-y-1 overflow-y-auto">
          {navigation.map((item) => {
            const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
            return (
              <Link
                key={item.name}
                href={item.href}
                className={cn(
                  'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-gold/20 text-gold'
                    : 'text-gray-400 hover:bg-navy-700 hover:text-white'
                )}
                title={isCollapsed ? item.name : undefined}
              >
                <item.icon className="w-5 h-5 flex-shrink-0" />
                {!isCollapsed && <span>{item.name}</span>}
              </Link>
            );
          })}

          {/* Divider */}
          <div className="my-4 border-t border-gray-700" />

          {/* System Navigation */}
          <div className="space-y-1">
            {!isCollapsed && (
              <div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
                System
              </div>
            )}
            {systemNavigation.map((item) => {
              const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
              return (
                <Link
                  key={item.name}
                  href={item.href}
                  className={cn(
                    'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-gold/20 text-gold'
                      : 'text-gray-400 hover:bg-navy-700 hover:text-white'
                  )}
                  title={isCollapsed ? item.name : undefined}
                >
                  <item.icon className="w-5 h-5 flex-shrink-0" />
                  {!isCollapsed && <span>{item.name}</span>}
                </Link>
              );
            })}
          </div>
        </nav>

        {/* User Profile */}
        <div className="p-4 border-t border-gray-700">
          <div className={cn('flex items-center gap-3', isCollapsed && 'justify-center')}>
            <div className="w-10 h-10 rounded-full bg-gold/20 flex items-center justify-center">
              <span className="text-gold font-semibold">A</span>
            </div>
            {!isCollapsed && (
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium text-white truncate">Admin</div>
                <div className="text-xs text-gray-400 truncate">admin@rinco.vn</div>
              </div>
            )}
          </div>
        </div>
      </div>
    </aside>
  );
}
