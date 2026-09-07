"use client";

import { useState, useEffect } from "react";
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DynamicForm } from "@/components/form/DynamicForm";
import { trackClick } from "@/lib/pixel";

interface ModalProps {
  trigger?: "button" | "sticky";
  formName?: string;
  tenantSlug?: string;
  pageSlug?: string;
  ctaLabel?: string;
}

export function LeadModal({
  trigger = "sticky",
  formName = "modal-form",
  tenantSlug,
  pageSlug,
  ctaLabel = "sticky_bottom",
}: ModalProps) {
  const [isOpen, setIsOpen] = useState(false);

  // Listen for modal triggers from other components
  useEffect(() => {
    const handleModalTrigger = (event: CustomEvent<{ trigger?: string }>) => {
      if (event.detail?.trigger === trigger) {
        setIsOpen(true);
        trackClick(ctaLabel, "modal", trigger);
      }
    };

    window.addEventListener(
      "openLeadModal",
      handleModalTrigger as EventListener
    );
    return () => {
      window.removeEventListener(
        "openLeadModal",
        handleModalTrigger as EventListener
      );
    };
  }, [trigger, ctaLabel]);

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent className="max-w-md p-0 overflow-hidden">
        <div className="modal-glow-wrapper">
          <div className="modal-glow-inner">
            <DialogHeader className="text-center px-6 pt-6 pb-4">
              <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-gradient-to-r from-orange/10 to-amber/10 border border-orange/30 text-orange text-xs font-bold uppercase tracking-wider mb-4">
                <span className="w-2 h-2 rounded-full bg-orange animate-pulse" />
                Số lượng vé miễn phí có giới hạn!
              </div>
              <DialogTitle className="text-2xl md:text-3xl font-heading font-black text-navy-900 mb-2">
                ĐĂNG KÝ GIỮ VÉ{" "}
                <span className="gradient-text">BUỔI CHIA SẺ</span>
              </DialogTitle>
              <p className="text-gray-500 text-sm md:text-base">
                Nhận ngay{" "}
                <strong className="text-orange">
                  Bộ 10 Ebook Thực Chiến
                </strong>{" "}
                khi đăng ký sớm!
              </p>
              {/* Event time chips */}
              <div className="flex flex-wrap justify-center gap-3 mt-4">
                <div className="modal-time-chip">
                  <span>📅</span>
                  <span>
                    <strong>20:00 | 07/09/2026</strong>
                  </span>
                </div>
                <div className="modal-time-chip">
                  <span>💻</span>
                  <span>Zoom Online</span>
                </div>
              </div>
            </DialogHeader>

            <div className="px-6 py-5">
              <DynamicForm
                formName={formName}
                tenantSlug={tenantSlug}
                pageSlug={pageSlug}
                onSuccess={() => setTimeout(() => setIsOpen(false), 3000)}
              />
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// Helper to open modal from anywhere
export function openLeadModal(trigger: string = "default") {
  if (typeof window !== "undefined") {
    window.dispatchEvent(
      new CustomEvent("openLeadModal", { detail: { trigger } })
    );
  }
}

export default LeadModal;
