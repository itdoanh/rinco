"use client";

import { useState } from "react";
import { useBranding } from "@/lib/branding";
import type { Branding } from "@/lib/schema";
import { Menu, X } from "lucide-react";

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
  const [mobileOpen, setMobileOpen] = useState(false);
  useBranding(branding);

  const defaultNavItems = [
    { label: "Trang chủ", href: "/" },
    { label: "Giới thiệu", href: "/about" },
    { label: "Dịch vụ", href: "/services" },
    { label: "Liên hệ", href: "/contact" },
  ];

  const items = navItems?.length ? navItems : defaultNavItems;

  return (
    <header className="sticky top-0 z-40 bg-white/95 backdrop-blur-md border-b shadow-sm">
      <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between gap-4">
        {/* Logo */}
        <a href="/" className="flex items-center gap-3 shrink-0">
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
            <span className="font-bold text-lg hidden sm:inline">{companyName}</span>
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
        <button
          onClick={onCtaClick}
          className="hidden lg:inline-flex items-center px-5 py-2 rounded-full text-white font-semibold shadow-md hover:shadow-lg hover:-translate-y-0.5 transition-all"
          style={{ backgroundColor: branding?.primary || "#0A192F" }}
        >
          {ctaText}
        </button>

        {/* Mobile menu trigger */}
        <button
          className="lg:hidden p-2 rounded-lg hover:bg-gray-100"
          onClick={() => setMobileOpen((v) => !v)}
          aria-label="Toggle menu"
          aria-expanded={mobileOpen}
        >
          {mobileOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
        </button>
      </div>

      {/* Mobile nav */}
      {mobileOpen && (
        <nav className="lg:hidden border-t bg-white">
          <div className="max-w-7xl mx-auto px-4 py-3 flex flex-col gap-2">
            {items.map((item, idx) => (
              <a
                key={idx}
                href={item.href}
                className="py-2 px-3 rounded-lg text-sm font-semibold text-gray-700 hover:bg-gray-50"
                onClick={() => setMobileOpen(false)}
              >
                {item.label}
              </a>
            ))}
            <button
              onClick={onCtaClick}
              className="mt-2 py-2 rounded-lg text-white font-semibold"
              style={{ backgroundColor: branding?.primary || "#0A192F" }}
            >
              {ctaText}
            </button>
          </div>
        </nav>
      )}
    </header>
  );
}

export default Header;
