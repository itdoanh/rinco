"use client";

interface FooterProps {
  companyName?: string;
  copyright?: string;
  links?: Array<{ label: string; href: string }>;
}

export function Footer({
  companyName = "hanghoaphaisinh.net",
  links,
}: FooterProps) {
  const defaultLinks = [
    { label: "Điều khoản dịch vụ", href: "#" },
    { label: "Chính sách bảo mật", href: "#" },
    { label: "Cảnh báo rủi ro", href: "#boicanh" },
  ];

  const footerLinks = links?.length ? links : defaultLinks;

  return (
    <footer className="bg-navy-900 border-t border-white/5 py-8">
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex flex-col md:flex-row justify-between items-center gap-4 text-center md:text-left">
          <div className="text-white/40 text-sm">
            © 2026 {companyName}. All rights reserved.
          </div>
          <div className="flex flex-wrap justify-center gap-6 text-sm">
            {footerLinks.map((link, idx) => (
              <a
                key={idx}
                href={link.href}
                className="text-white/40 hover:text-white transition-colors"
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
