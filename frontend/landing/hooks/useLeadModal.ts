"use client";

import { useEffect, useState, useCallback } from "react";
import { openLeadModal } from "@/components/layout/LeadModal";
import { Button } from "@/components/ui/button";

interface LeadFormModalProps {
  tenantSlug?: string;
  pageSlug?: string;
}

export function useLeadModal() {
  const [isOpen, setIsOpen] = useState(false);

  const openModal = useCallback(() => {
    setIsOpen(true);
    openLeadModal("button");
  }, []);

  const closeModal = useCallback(() => {
    setIsOpen(false);
  }, []);

  return { isOpen, openModal, closeModal };
}

// Hook for tracking
export function useTracking() {
  useEffect(() => {
    // Client-side tracking is handled by Tracker and ClickTracker components
  }, []);
}

// Hook for page view
export function usePageView() {
  useEffect(() => {
    // Page view tracking is handled by Tracker component
  }, []);
}

export default useLeadModal;
