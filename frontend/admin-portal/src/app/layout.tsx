import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'RINCO Admin Portal',
  description: 'Super Admin Portal cho RINCO Platform',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="vi">
      <body className="bg-slate-900 text-white min-h-screen">{children}</body>
    </html>
  );
}
