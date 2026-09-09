"use client";

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
import { PricingTable } from "./PricingTable";

import {
  BLOCK_TYPE_ALIASES,
  resolveBlockType,
  asObject,
} from "./block-helpers";

/**
 * Payload for a single editable block on a landing page.
 *
 * The marketing backend stores blocks as opaque JSON objects keyed by
 * ``type``.  We use ``unknown`` (not ``any``) on purpose so that
 * per-block components can narrow ``data`` via runtime guards.
 */
export interface BlockData {
  id?: string;
  type?: string;
  data?: unknown;
  tenantSlug?: string;
  pageSlug?: string;
  tenantId?: string;
  pageId?: string;
}

export interface BlockRenderCallbacks {
  onCtaClick?: () => void;
}

/** Re-export for callers that imported the constants from
 * ``BlockRenderer.tsx``. */
export { BLOCK_TYPE_ALIASES, resolveBlockType, asObject };

export function BlockRenderer({
  blocks,
  tenantSlug,
  pageSlug,
}: {
  blocks: BlockData[];
  tenantSlug?: string;
  pageSlug?: string;
}) {
  const handleCtaClick = () => {
    openLeadModal("button");
  };

  return (
    <>
      {blocks.map((block) => {
        const id = block.id ?? Math.random().toString(36).slice(2);
        const canonical = resolveBlockType(block.type);
        if (!canonical) {
          if (
            typeof window !== "undefined" &&
            process.env.NODE_ENV === "development"
          ) {
            console.warn(`Unknown block type: ${block.type}`);
          }
          return null;
        }
        const data = asObject(block.data);
        const sharedProps = {
          tenantSlug,
          pageSlug,
          tenantId: block.tenantId,
          pageId: block.pageId,
        };

        switch (canonical) {
          case "hero":
            return (
              <Hero
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
                {...sharedProps}
              />
            );
          case "features":
            return (
              <FeatureGrid
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
              />
            );
          case "logos":
            return (
              <LogoCloud
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
              />
            );
          case "speakers":
            return (
              <SpeakerSection
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
              />
            );
          case "trust":
            return (
              <TrustSection
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
              />
            );
          case "stats":
            return (
              <Stats
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
              />
            );
          case "faq":
            return <FAQSection key={id} data={data} />;
          case "cta":
            return (
              <CTA
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
              />
            );
          case "testimonial":
            return <TestimonialSection key={id} data={data} />;
          case "risk_warning":
            return <RiskWarning key={id} data={data} />;
          case "form":
            return (
              <FormBlock
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
                {...sharedProps}
              />
            );
          case "pricing_table":
            return (
              <PricingTable
                key={id}
                data={data}
                onCtaClick={handleCtaClick}
              />
            );
          default:
            return null;
        }
      })}
    </>
  );
}

/** Render a single block by canonical type.  Useful for previews. */
export function renderBlock(
  type: string,
  data: unknown,
  callbacks?: BlockRenderCallbacks,
) {
  const canonical = resolveBlockType(type);
  if (!canonical) return null;
  const safeData = asObject(data);
  const props = { data: safeData, ...(callbacks ?? {}) };

  switch (canonical) {
    case "hero":
      return <Hero key={Math.random()} {...props} />;
    case "features":
      return <FeatureGrid key={Math.random()} {...props} />;
    case "logos":
      return <LogoCloud key={Math.random()} {...props} />;
    case "speakers":
      return <SpeakerSection key={Math.random()} {...props} />;
    case "trust":
      return <TrustSection key={Math.random()} {...props} />;
    case "stats":
      return <Stats key={Math.random()} {...props} />;
    case "faq":
      return <FAQSection key={Math.random()} {...props} />;
    case "cta":
      return <CTA key={Math.random()} {...props} />;
    case "testimonial":
      return <TestimonialSection key={Math.random()} {...props} />;
    case "risk_warning":
      return <RiskWarning key={Math.random()} {...props} />;
    case "form":
      return <FormBlock key={Math.random()} {...props} />;
    case "pricing_table":
      return <PricingTable key={Math.random()} {...props} />;
    default:
      return null;
  }
}

export default BlockRenderer;
