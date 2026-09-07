"use client";

import { useState } from "react";
import { openLeadModal } from "@/components/layout/LeadModal";

// Block components
import { Hero } from "./Hero";
import { FeatureGrid } from "./FeatureGrid";
import { LogoCloud } from "./LogoCloud";
import { SpeakerSection } from "./SpeakerSection";
import { TrustSection } from "./TrustSection";
import { Stats } from "./Stats";
import { FAQ } from "./FAQ";
import { CTA } from "./CTA";
import { TestimonialSection } from "./Testimonial";
import { RiskWarning } from "./RiskWarning";
import { FormBlock } from "@/components/form/FormBlock";

interface BlockProps {
  id: string;
  type: string;
  data: Record<string, unknown>;
  tenantSlug?: string;
  pageSlug?: string;
}

export function BlockRenderer({ blocks, tenantSlug, pageSlug }: {
  blocks: BlockProps[];
  tenantSlug?: string;
  pageSlug?: string;
}) {
  const [showModal, setShowModal] = useState(false);

  const handleCtaClick = () => {
    setShowModal(true);
    openLeadModal("button");
  };

  return (
    <>
      {blocks.map((block) => {
        const { id, type, data } = block;
        const props = { data: data as Parameters<typeof renderBlock>[1]["data"], onCtaClick: handleCtaClick };
        const sharedProps = { tenantSlug, pageSlug };

        switch (type) {
          case "hero":
            return <Hero key={id} {...props} {...sharedProps} />;
          case "feature_grid":
          case "features":
            return <FeatureGrid key={id} {...props} />;
          case "logo_cloud":
          case "logos":
            return <LogoCloud key={id} {...props} />;
          case "speaker":
          case "speakers":
            return <SpeakerSection key={id} {...props} />;
          case "trust":
          case "trust_section":
            return <TrustSection key={id} {...props} />;
          case "stats":
          case "statistics":
            return <Stats key={id} {...props} />;
          case "faq":
          case "faqs":
            return <FAQSection key={id} {...props} />;
          case "cta":
          case "call_to_action":
            return <CTA key={id} {...props} />;
          case "testimonial":
          case "testimonials":
            return <TestimonialSection key={id} {...props} />;
          case "risk_warning":
          case "risk":
            return <RiskWarning key={id} {...props} />;
          case "form":
          case "registration_form":
            return <FormBlock key={id} {...props} {...sharedProps} />;
          default:
            if (process.env.NODE_ENV === "development") {
              console.warn(`Unknown block type: ${type}`);
            }
            return null;
        }
      })}
    </>
  );
}

// Helper to render a single block
function renderBlock(type: string, data: Record<string, unknown>, callbacks?: { onCtaClick?: () => void }) {
  const props = { data, ...callbacks };
  
  switch (type) {
    case "hero":
      return <Hero key={data.id as string} {...props} />;
    case "feature_grid":
    case "features":
      return <FeatureGrid key={data.id as string} {...props} />;
    case "logo_cloud":
      return <LogoCloud key={data.id as string} {...props} />;
    case "speaker":
    case "speakers":
      return <SpeakerSection key={data.id as string} {...props} />;
    case "trust":
    case "trust_section":
      return <TrustSection key={data.id as string} {...props} />;
    case "stats":
      return <Stats key={data.id as string} {...props} />;
    case "faq":
      return <FAQSection key={data.id as string} {...props} />;
    case "cta":
      return <CTA key={data.id as string} {...props} />;
    case "testimonial":
    case "testimonials":
      return <TestimonialSection key={data.id as string} {...props} />;
    case "risk_warning":
      return <RiskWarning key={data.id as string} {...props} />;
    case "form":
      return <FormBlock key={data.id as string} {...props} />;
    default:
      return null;
  }
}

export { renderBlock };
export default BlockRenderer;
