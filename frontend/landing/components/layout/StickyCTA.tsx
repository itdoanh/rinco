"use client";

import { Button } from "@/components/ui/button";

interface StickyCTAProps {
  ctaText?: string;
  onCtaClick?: () => void;
}

export function StickyCTA({
  ctaText = "ĐĂNG KÝ NGAY – NHẬN EBOOK MIỄN PHÍ",
  onCtaClick,
}: StickyCTAProps) {
  return (
    <div
      id="stickyCta"
      className="fixed bottom-0 left-0 right-0 z-50 p-4 bg-gradient-to-t from-white via-white to-transparent"
    >
      <div style={{ maxWidth: "600px", margin: "0 auto" }}>
        <Button
          variant="cta"
          size="xl"
          onClick={onCtaClick}
          className="w-full sticky-btn"
        >
          <svg
            style={{ width: "20px", height: "20px", flexShrink: 0 }}
            fill="none"
            stroke="currentColor"
            strokeWidth="2.5"
            viewBox="0 0 24 24"
          >
            <path d="M5 13l4 4L19 7" />
          </svg>
          <span>{ctaText}</span>
        </Button>
      </div>
    </div>
  );
}

export default StickyCTA;
