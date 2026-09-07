"use client";

import { useBranding } from "@/lib/branding";
import type { Branding } from "@/lib/schema";
import { Button } from "@/components/ui/button";

interface HeaderProps {
  logo?: string;
  logoAlt?: string;
  companyName?: string;
  branding?: Branding;
  navItems?: Array<{ label: string; href: string }>;
  ctaText?: string;
  onCtaClick?: () => void;
}

export function Header({
  logo,
  logoAlt = "Logo",
  companyName,
  branding,
  navItems,
  ctaText = "Liên hệ",
  onCtaClick,
}: HeaderProps) {
  // Apply branding
  useBranding(branding);

  const defaultNavItems = [
    { label: "Trang chủ", href: "/" },
    { label: "Giới thiệu", href: "/about" },
    { label: "Dịch vụ", href: "/services" },
    { label: "Liên hệ", href: "/contact" },
  ];

  const items = navItems?.length ? navItems : defaultNavItems;

  return (
    <>
      <header className="sticky top-0 z-40 bg-white/95 backdrop-blur-md border-b shadow-sm">
        <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between gap-4">
          {/* Logo */}
          <a href="/" className="flex items-center gap-3">
            {logo ? (
              <img src={logo} alt={logoAlt} className="h-10 w-auto" />
            ) : (
              <div
                className="w-10 h-10 rounded-xl flex items-center justify-center text-white font-bold"
                style={{ backgroundColor: branding?.primary || "#0A192F" }}
              >
                {logoAlt[0]}
              </div>
            )}
            {companyName && (
              <span className="font-bold text-lg">{companyName}</span>
            )}
          </a>

          {/* Desktop Nav */}
          <nav className="hidden lg:flex items-center gap-7">
            {items.map((item, idx) => (
              <a
                key={idx}
                href={item.href}
                className="text-sm font-semibold text-gray-600 hover:text-[var(--brand-primary)] transition-colors"
              >
                {item.label}
              </a>
            ))}
          </nav>

          {/* CTA */}
          <Button
            onClick={onCtaClick}
            className="hidden lg:inline-flex"
            style={{
              backgroundColor: branding?.primary || "#0A192F",
            }}
          >
            {ctaText}
          </Button>

          {/* Mobile menu */}
          <button className="lg:hidden p-2 rounded-lg hover:bg-gray-100">
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
        </div>
      </header>
    </>
  );
}

export default Header;
