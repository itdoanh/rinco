import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'Tenant Site',
  description: 'Dynamic landing pages for tenants',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="vi">
      <body className="min-h-screen bg-white antialiased">
        {children}
      </body>
    </html>
  );
}
