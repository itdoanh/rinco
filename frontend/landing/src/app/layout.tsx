import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Tối Ưu Dòng Tiền 2026 Với Hàng Hóa Phái Sinh | Webinar Miễn Phí APEX Fintech',
  description:
    'Giải mã cơ chế T+0 & sinh lời 2 chiều. Giữ vé Webinar Zoom độc quyền cùng Chuyên gia MXV + Nhận ngay Bộ 10 Ebook Thực Chiến.',
  keywords:
    'webinar hàng hóa 2026, đầu tư hàng hóa phái sinh, T+0, MXV, APEX Fintech',
  authors: [{ name: 'APEX Fintech' }],
  themeColor: '#0A192F',
  openGraph: {
    type: 'website',
    url: 'https://hanghoaphaisinh.net/chiase/',
    title: 'Tối Ưu Dòng Tiền 2026 Với Hàng Hóa Phái Sinh',
    description:
      'Webinar Zoom miễn phí - Giải mã T+0 & sinh lời 2 chiều. Tặng Bộ 10 Ebook Thực Chiến cho 50 NĐT đăng ký sớm nhất.',
    images: [
      'https://hanghoaphaisinh.net/chiase/assets/img/apex_top1_thiphan_quy2_2026.webp',
    ],
    siteName: 'APEX Fintech',
    locale: 'vi_VN',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'Tối Ưu Dòng Tiền 2026 Với Hàng Hóa Phái Sinh',
    description: 'Webinar Zoom miễn phí - Giải mã T+0 & sinh lời 2 chiều.',
    images: [
      'https://hanghoaphaisinh.net/chiase/assets/img/apex_top1_thiphan_quy2_2026.webp',
    ],
  },
  robots: { index: true, follow: true },
  alternates: {
    canonical: 'https://hanghoaphaisinh.net/chiase/',
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="vi">
      <head>
        <link rel="icon" type="image/webp" href="/assets/img/logo_APEX_K_NEN.webp" />
        <link rel="apple-touch-icon" href="/assets/img/logo_APEX_K_NEN.webp" />
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=Plus+Jakarta+Sans:wght@600;700;800&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="antialiased">{children}</body>
    </html>
  );
}
