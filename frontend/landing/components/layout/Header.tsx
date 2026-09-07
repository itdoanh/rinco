"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";

interface HeaderProps {
  logo?: string;
  logoAlt?: string;
  companyName?: string;
  navItems?: Array<{ label: string; href: string }>;
  ctaText?: string;
  onCtaClick?: () => void;
}

export function Header({
  logo = "/assets/img/logo_APEX_K_NEN.webp",
  logoAlt = "APEX",
  companyName = "FINTECH",
  navItems,
  ctaText = "GIỮ VÉ ZOOM MIỄN PHÍ",
  onCtaClick,
}: HeaderProps) {
  const defaultNavItems = [
    { label: "Bối cảnh", href: "#boicanh" },
    { label: "Lợi thế", href: "#loi-the" },
    { label: "Diễn giả", href: "#dien-gia" },
    { label: "Uy tín", href: "#uy-tin" },
    { label: "Đăng ký", href: "#dang-ky" },
  ];

  const items = navItems?.length ? navItems : defaultNavItems;

  return (
    <>
      {/* Top Bar */}
      <div className="bg-navy-900 text-white py-2.5 px-4 text-xs md:text-sm border-b-2 border-orange topbar">
        <div className="max-w-7xl mx-auto flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-2 flex-wrap justify-center md:justify-start flex-1 min-w-0">
            <svg
              className="w-4 h-4 text-orange flex-shrink-0"
              fill="currentColor"
              viewBox="0 0 24 24"
            >
              <path d="M12 2L4 6v6c0 5.5 3.8 10.7 8 12 4.2-1.3 8-6.5 8-12V6l-8-4z" />
            </svg>
            <span className="truncate">
              <strong className="text-gold-light">APEX FINTECH</strong>{" "}
              <span className="mx-1 text-white/30">|</span> Thành viên Kinh
              doanh Số <strong className="text-gold-light">080</strong> - MXV{" "}
              <span className="mx-1 text-white/30">|</span> Hotline:{" "}
              <strong className="text-gold-light">0984386538</strong>
            </span>
          </div>
          <Button
            variant="cta"
            size="sm"
            onClick={onCtaClick}
            className="flex-shrink-0"
          >
            🚀 {ctaText}
          </Button>
        </div>
      </div>

      {/* Main Header */}
      <header className="sticky top-0 z-40 bg-white/95 backdrop-blur-md border-b border-gray-100 shadow-sm">
        <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between gap-4">
          {/* Logo */}
          <a href="#hero" className="flex items-center gap-3">
            <img
              src={logo}
              alt={logoAlt}
              className="w-10 h-10 md:w-11 md:h-11 rounded-xl"
            />
            <div className="flex flex-col">
              <span className="text-lg md:text-xl font-heading font-extrabold bg-gradient-to-r from-orange to-amber bg-clip-text text-transparent leading-tight">
                {logoAlt}
              </span>
              <span className="text-[10px] md:text-xs font-bold text-gray-500 tracking-widest -mt-0.5">
                {companyName}
              </span>
            </div>
          </a>

          {/* Desktop Nav */}
          <nav className="hidden lg:flex items-center gap-7">
            {items.map((item, idx) => (
              <a
                key={idx}
                href={item.href}
                className="text-sm font-semibold text-gray-600 hover:text-orange transition-colors"
              >
                {item.label}
              </a>
            ))}
          </nav>

          {/* CTA Button */}
          <Button
            variant="cta"
            size="sm"
            onClick={onCtaClick}
            className="hidden lg:inline-flex"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              viewBox="0 0 24 24"
            >
              <path d="M5 13l4 4L19 7" />
            </svg>
            ĐĂNG KÝ NGAY
          </Button>

          {/* Mobile Menu Toggle */}
          <button
            type="button"
            className="lg:hidden p-2 rounded-lg hover:bg-gray-100"
            aria-label="Menu"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                d="M4 6h16M4 12h16M4 18h16"
              />
            </svg>
          </button>
        </div>
      </header>
    </>
  );
}

export default Header;
