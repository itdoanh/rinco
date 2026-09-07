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
import FAQSection from "./FAQ";
import { CTA } from "./CTA";
import TestimonialSection from "./Testimonial";
import { RiskWarning } from "./RiskWarning";
import { FormBlock } from "@/components/form/FormBlock";

interface BlockProps {
  id: string;
  type: string;
  data: any; // eslint-disable-line @typescript-eslint/no-explicit-any -- heterogeneous block payloads
  tenantSlug?: string;
  pageSlug?: string;
  tenantId?: string;
  pageId?: string;
}

type AnyBlockProps = {
  id?: string;
  type?: string;
  data?: any; // eslint-disable-line @typescript-eslint/no-explicit-any
  tenantSlug?: string;
  pageSlug?: string;
  tenantId?: string;
  pageId?: string;
};

export function BlockRenderer({
  blocks,
  tenantSlug,
  pageSlug,
}: {
  blocks: AnyBlockProps[];
  tenantSlug?: string;
  pageSlug?: string;
}) {
  const [showModal, setShowModal] = useState(false)

  const handleCtaClick = () => {
    setShowModal(true)
    openLeadModal("button")
  }

  return (
    <>
      {blocks.map((block) => {
        const id = block.id ?? Math.random().toString(36).slice(2)
        const type = block.type ?? "unknown"
        const data = (block.data ?? {}) as Record<string, unknown>
        const onCtaClick = handleCtaClick
        const sharedProps = { tenantSlug, pageSlug }

        switch (type) {
          case "hero":
            return <Hero key={id} data={data} onCtaClick={onCtaClick} {...sharedProps} />
          case "feature_grid":
          case "features":
            return <FeatureGrid key={id} data={data} onCtaClick={onCtaClick} />
          case "logo_cloud":
          case "logos":
            return <LogoCloud key={id} data={data} onCtaClick={onCtaClick} />
          case "speaker":
          case "speakers":
            return <SpeakerSection key={id} data={data} onCtaClick={onCtaClick} />
          case "trust":
          case "trust_section":
            return <TrustSection key={id} data={data} onCtaClick={onCtaClick} />
          case "stats":
          case "statistics":
            return <Stats key={id} data={data} onCtaClick={onCtaClick} />
          case "faq":
          case "faqs":
            return <FAQSection key={id} data={data} />
          case "cta":
          case "call_to_action":
            return <CTA key={id} data={data} onCtaClick={onCtaClick} />
          case "testimonial":
          case "testimonials":
            return <TestimonialSection key={id} data={data} />
          case "risk_warning":
          case "risk":
            return <RiskWarning key={id} data={data} />
          case "form":
          case "registration_form":
            return (
              <FormBlock
                key={id}
                data={data}
                onCtaClick={onCtaClick}
                {...sharedProps}
              />
            )
          default:
            if (typeof window !== "undefined" && process.env.NODE_ENV === "development") {
              console.warn(`Unknown block type: ${type}`)
            }
            return null
        }
      })}
    </>
  )
}

// Helper to render a single block
export function renderBlock(
  type: string,
  data: Record<string, unknown>,
  callbacks?: { onCtaClick?: () => void }
) {
  const props = { data, ...callbacks }

  switch (type) {
    case "hero":
      return <Hero key={Math.random()} {...props} />
    case "feature_grid":
    case "features":
      return <FeatureGrid key={Math.random()} {...props} />
    case "logo_cloud":
    case "logos":
      return <LogoCloud key={Math.random()} {...props} />
    case "speaker":
    case "speakers":
      return <SpeakerSection key={Math.random()} {...props} />
    case "trust":
    case "trust_section":
      return <TrustSection key={Math.random()} {...props} />
    case "stats":
    case "statistics":
      return <Stats key={Math.random()} {...props} />
    case "faq":
    case "faqs":
      return <FAQSection key={Math.random()} {...props} />
    case "cta":
    case "call_to_action":
      return <CTA key={Math.random()} {...props} />
    case "testimonial":
    case "testimonials":
      return <TestimonialSection key={Math.random()} {...props} />
    case "risk_warning":
    case "risk":
      return <RiskWarning key={Math.random()} {...props} />
    case "form":
    case "registration_form":
      return <FormBlock key={Math.random()} {...props} />
    default:
      return null
  }
}

export default BlockRenderer
