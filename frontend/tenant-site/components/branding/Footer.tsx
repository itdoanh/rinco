"use client";

import type { Branding } from "@/lib/schema";

interface FooterProps {
  companyName?: string;
  branding?: Branding;
  links?: Array<{ label: string; href: string }>;
}

export function Footer({
  companyName = "Company Name",
  branding,
  links,
}: FooterProps) {
  const defaultLinks = [
    { label: "Trang chủ", href: "/" },
    { label: "Giới thiệu", href: "/about" },
    { label: "Điều khoản", href: "/terms" },
    { label: "Chính sách bảo mật", href: "/privacy" },
  ];

  const footerLinks = links?.length ? links : defaultLinks;

  return (
    <footer
      className="py-8"
      style={{ backgroundColor: branding?.primary || "#0A192F" }}
    >
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex flex-col md:flex-row justify-between items-center gap-4 text-center md:text-left">
          <div className="text-white/60 text-sm">
            © {new Date().getFullYear()} {companyName}. All rights reserved.
          </div>
          <div className="flex flex-wrap justify-center gap-6 text-sm">
            {footerLinks.map((link, idx) => (
              <a
                key={idx}
                href={link.href}
                className="text-white/60 hover:text-white transition-colors"
              >
                {link.label}
              </a>
            ))}
          </div>
        </div>
      </div>
    </footer>
  );
}

export default Footer;
