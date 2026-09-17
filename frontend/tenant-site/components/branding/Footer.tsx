"use client";

import type { Branding } from "@/lib/schema";
import { Facebook, Twitter, Instagram, Linkedin, Youtube } from "lucide-react";

interface FooterProps {
  companyName?: string;
  branding?: Branding;
  links?: Array<{ label: string; href: string }>;
  social?: { facebook?: string; twitter?: string; instagram?: string; linkedin?: string; youtube?: string };
  address?: string;
  phone?: string;
  email?: string;
}

export function Footer({
  companyName = "Company Name",
  branding,
  links,
  social,
  address = "Tầng 10, Toà nhà ABC, Quận 1, TP. HCM",
  phone = "1900-6868",
  email = "contact@company.vn",
}: FooterProps) {
  const defaultLinks = [
    { label: "Trang chủ", href: "/" },
    { label: "Giới thiệu", href: "/about" },
    { label: "Dịch vụ", href: "/services" },
    { label: "Liên hệ", href: "/contact" },
    { label: "Điều khoản", href: "/terms" },
    { label: "Chính sách bảo mật", href: "/privacy" },
  ];

  const footerLinks = links?.length ? links : defaultLinks;
  const socialLinks = [
    { Icon: Facebook, href: social?.facebook, label: "Facebook" },
    { Icon: Twitter, href: social?.twitter, label: "Twitter" },
    { Icon: Instagram, href: social?.instagram, label: "Instagram" },
    { Icon: Linkedin, href: social?.linkedin, label: "LinkedIn" },
    { Icon: Youtube, href: social?.youtube, label: "YouTube" },
  ].filter((s) => s.href);

  return (
    <footer
      className="text-white"
      style={{ backgroundColor: branding?.primary || "#0A192F" }}
    >
      <div className="max-w-7xl mx-auto px-4 py-12">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
          <div className="md:col-span-2">
            <div className="flex items-center gap-3 mb-4">
              <div
                className="w-10 h-10 rounded-xl flex items-center justify-center text-white font-bold"
                style={{ backgroundColor: branding?.secondary || "#F5A623" }}
              >
                {companyName[0]}
              </div>
              <span className="text-xl font-bold">{companyName}</span>
            </div>
            <p className="text-white/70 text-sm leading-relaxed mb-4 max-w-md">
              Giải pháp toàn diện cho doanh nghiệp hiện đại. Chúng tôi cam kết mang đến
              giá trị tốt nhất cho khách hàng.
            </p>
            <div className="space-y-1 text-sm text-white/70">
              <div>📍 {address}</div>
              <div>📞 {phone}</div>
              <div>✉️ {email}</div>
            </div>
          </div>

          <div>
            <h3 className="font-semibold mb-4 uppercase text-sm tracking-wider">
              Liên kết
            </h3>
            <ul className="space-y-2">
              {footerLinks.map((link, idx) => (
                <li key={idx}>
                  <a
                    href={link.href}
                    className="text-white/70 hover:text-white transition-colors text-sm"
                  >
                    {link.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h3 className="font-semibold mb-4 uppercase text-sm tracking-wider">
              Theo dõi chúng tôi
            </h3>
            <div className="flex flex-wrap gap-2 mb-6">
              {socialLinks.length === 0 ? (
                <>
                  <SocialChip Icon={Facebook} label="Facebook" />
                  <SocialChip Icon={Twitter} label="Twitter" />
                  <SocialChip Icon={Instagram} label="Instagram" />
                  <SocialChip Icon={Linkedin} label="LinkedIn" />
                </>
              ) : (
                socialLinks.map(({ Icon, href, label }, idx) => (
                  <a
                    key={idx}
                    href={href}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label={label}
                    className="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors"
                  >
                    <Icon className="w-4 h-4" />
                  </a>
                ))
              )}
            </div>
            <p className="text-xs text-white/50">
              Đăng ký nhận tin tức mới nhất từ chúng tôi.
            </p>
          </div>
        </div>

        <div className="mt-10 pt-6 border-t border-white/10 flex flex-col md:flex-row justify-between items-center gap-3 text-center md:text-left">
          <div className="text-white/60 text-sm">
            © {new Date().getFullYear()} {companyName}. All rights reserved.
          </div>
          <div className="text-white/40 text-xs">
            Built with RINCO Platform
          </div>
        </div>
      </div>
    </footer>
  );
}

function SocialChip({ Icon, label }: { Icon: typeof Facebook; label: string }) {
  return (
    <a
      href="#"
      aria-label={label}
      className="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors"
    >
      <Icon className="w-4 h-4" />
    </a>
  );
}

export default Footer;
