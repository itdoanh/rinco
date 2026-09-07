import { Metadata } from "next";

interface SEOHeadProps {
  title?: string;
  description?: string;
  image?: string;
  url?: string;
  type?: "website" | "article";
  keywords?: string[];
  author?: string;
  publishedTime?: string;
  modifiedTime?: string;
}

export function SEOHead({
  title,
  description = "Giải mã cơ chế T+0 & sinh lời 2 chiều. Giữ vé Webinar Zoom độc quyền cùng Chuyên gia MXV + Nhận ngay Bộ 10 Ebook Thực Chiến.",
  image = "/assets/img/apex_top1_thiphan_quy2_2026.webp",
  url,
  type = "website",
  keywords = ["webinar hàng hóa 2026", "đầu tư hàng hóa phái sinh", "T+0", "MXV", "APEX Fintech"],
  author = "APEX Fintech",
  publishedTime,
  modifiedTime,
}: SEOHeadProps) {
  const fullUrl = url
    ? `${process.env.NEXT_PUBLIC_SITE_URL || ""}${url}`
    : process.env.NEXT_PUBLIC_SITE_URL;

  const metadata: Metadata = {
    title: title || "Tối Ưu Dòng Tiền 2026 Với Hàng Hóa Phái Sinh | APEX Fintech",
    description,
    keywords,
    authors: [{ name: author }],
    openGraph: {
      title,
      description,
      url: fullUrl,
      type,
      siteName: "APEX Fintech",
      locale: "vi_VN",
      images: [
        {
          url: image,
          width: 1200,
          height: 630,
          alt: title,
        },
      ],
      ...(publishedTime && { publishedTime }),
      ...(modifiedTime && { modifiedTime }),
    },
    twitter: {
      card: "summary_large_image",
      title,
      description,
      images: [image],
    },
    robots: {
      index: true,
      follow: true,
      googleBot: {
        index: true,
        follow: true,
        "max-video-preview": -1,
        "max-image-preview": "large",
        "max-snippet": -1,
      },
    },
  };

  return metadata as Metadata;
}

export default SEOHead;
