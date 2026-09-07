'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Bell, Search, User, LogOut, Settings, ChevronDown } from 'lucide-react';
import { useState, useRef, useEffect } from 'react';
import { useAuthStore } from '@/store/auth';

interface HeaderProps {
  title?: string;
}

export default function Header({ title }: HeaderProps) {
  const pathname = usePathname();
  const { user, logout } = useAuthStore();
  const [isProfileOpen, setIsProfileOpen] = useState(false);
  const profileRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (profileRef.current && !profileRef.current.contains(event.target as Node)) {
        setIsProfileOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleLogout = () => {
    logout();
    window.location.href = '/(auth)/login';
  };

  return (
    <header className="h-16 bg-navy-800 border-b border-gray-700 px-6 flex items-center justify-between">
      {/* Left: Page Title or Breadcrumbs */}
      <div className="flex items-center">
        {title ? (
          <h1 className="text-xl font-semibold text-white">{title}</h1>
        ) : (
          <nav className="flex items-center gap-2 text-sm">
            <Link href="/dashboard" className="text-gray-400 hover:text-white">
              Dashboard
            </Link>
            {pathname !== '/dashboard' && (
              <>
                <span className="text-gray-600">/</span>
                <span className="text-white capitalize">
                  {pathname.split('/')[1].replace(/-/g, ' ')}
                </span>
              </>
            )}
          </nav>
        )}
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-4">
        {/* Search */}
        <div className="relative hidden md:block">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
          <input
            type="text"
            placeholder="Search..."
            className="w-64 pl-10 pr-4 py-2 bg-navy-700 border border-gray-600 rounded-lg text-sm text-white placeholder:text-gray-500 focus:outline-none focus:border-gold transition-colors"
          />
          <kbd className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-500 bg-navy-600 px-1.5 py-0.5 rounded">
            ⌘K
          </kbd>
        </div>

        {/* Notifications */}
        <button className="relative p-2 rounded-lg hover:bg-navy-700 text-gray-400 hover:text-white transition-colors">
          <Bell className="w-5 h-5" />
          <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
        </button>

        {/* Profile Dropdown */}
        <div className="relative" ref={profileRef}>
          <button
            onClick={() => setIsProfileOpen(!isProfileOpen)}
            className="flex items-center gap-2 p-2 rounded-lg hover:bg-navy-700 transition-colors"
          >
            <div className="w-8 h-8 rounded-full bg-gold/20 flex items-center justify-center">
              <span className="text-gold font-semibold text-sm">
                {user?.name?.charAt(0) || 'A'}
              </span>
            </div>
            <span className="hidden md:block text-sm text-white">
              {user?.name || 'Admin'}
            </span>
            <ChevronDown className="w-4 h-4 text-gray-400" />
          </button>

          {isProfileOpen && (
            <div className="absolute right-0 top-full mt-2 w-56 bg-navy-800 border border-gray-700 rounded-lg shadow-xl py-1 z-50">
              <div className="px-4 py-3 border-b border-gray-700">
                <div className="text-sm font-medium text-white">
                  {user?.name || 'Admin'}
                </div>
                <div className="text-xs text-gray-400">
                  {user?.email || 'admin@rinco.vn'}
                </div>
              </div>

              <Link
                href="/settings/profile"
                className="flex items-center gap-2 px-4 py-2 text-sm text-gray-400 hover:bg-navy-700 hover:text-white transition-colors"
                onClick={() => setIsProfileOpen(false)}
              >
                <User className="w-4 h-4" />
                Profile
              </Link>

              <Link
                href="/settings"
                className="flex items-center gap-2 px-4 py-2 text-sm text-gray-400 hover:bg-navy-700 hover:text-white transition-colors"
                onClick={() => setIsProfileOpen(false)}
              >
                <Settings className="w-4 h-4" />
                Settings
              </Link>

              <div className="border-t border-gray-700 mt-1 pt-1">
                <button
                  onClick={handleLogout}
                  className="flex items-center gap-2 w-full px-4 py-2 text-sm text-red-400 hover:bg-navy-700 transition-colors"
                >
                  <LogOut className="w-4 h-4" />
                  Logout
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
