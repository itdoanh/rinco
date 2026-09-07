import type { Metadata } from "next";
import { Inter, Nunito } from "next/font/google";
import { cn } from "@/lib/utils";
import "./globals.css";
import { Tracker } from "@/components/tracking/Tracker";
import { UTMCapture } from "@/components/tracking/UTMCapture";
import { Providers } from "@/components/providers";

const inter = Inter({ 
  subsets: ["latin"],
  variable: "--font-inter",
});

const nunito = Nunito({
  subsets: ["latin"],
  variable: "--font-nunito",
});

export const metadata: Metadata = {
  title: {
    default: "APEX Fintech - Tối Ưu Dòng Tiền 2026 Với Hàng Hóa Phái Sinh",
    template: "%s | APEX Fintech",
  },
  description: "Giải mã cơ chế T+0 & sinh lời 2 chiều. Giữ vé Webinar Zoom độc quyền cùng Chuyên gia MXV + Nhận ngay Bộ 10 Ebook Thực Chiến.",
  keywords: ["webinar hàng hóa 2026", "đầu tư hàng hóa phái sinh", "T+0", "MXV", "APEX Fintech"],
  authors: [{ name: "APEX Fintech" }],
  openGraph: {
    type: "website",
    siteName: "APEX Fintech",
    locale: "vi_VN",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="vi" suppressHydrationWarning>
      <head>
        <link rel="icon" href="/assets/img/logo_APEX_K_NEN.webp" />
      </head>
      <body className={cn(
        inter.variable,
        nunito.variable,
        "font-sans antialiased"
      )}>
        <Providers>
          <UTMCapture />
          {children}
          <Tracker />
        </Providers>
      </body>
    </html>
  );
}
