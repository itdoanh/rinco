"use client";

import { useEffect, useState, useCallback } from "react";
import { cn } from "@/lib/utils";

/**
 * Minimal toast helper — single-file replacement for a heavyweight
 * notification library.  Toasts auto-dismiss after ``duration`` ms
 * (default 3000) and can be stacked.
 */
export type ToastVariant = "info" | "success" | "error";

export interface Toast {
  id: number;
  message: string;
  variant: ToastVariant;
}

let listeners: ((t: Toast) => void)[] = [];
let nextId = 1;

export function toast(message: string, variant: ToastVariant = "info") {
  const t = { id: nextId++, message, variant };
  for (const fn of listeners) fn(t);
}

export function ToastContainer() {
  const [items, setItems] = useState<Toast[]>([]);

  const remove = useCallback((id: number) => {
    setItems((current) => current.filter((t) => t.id !== id));
  }, []);

  useEffect(() => {
    const handler = (t: Toast) => {
      setItems((current) => [...current, t]);
      setTimeout(() => remove(t.id), 3000);
    };
    listeners.push(handler);
    return () => {
      listeners = listeners.filter((fn) => fn !== handler);
    };
  }, [remove]);

  return (
    <div
      data-testid="toast-container"
      className="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2"
    >
      {items.map((t) => (
        <div
          key={t.id}
          role="status"
          className={cn(
            "pointer-events-auto rounded-lg px-4 py-2 text-sm shadow-lg",
            t.variant === "success" && "bg-emerald-500 text-white",
            t.variant === "error" && "bg-red-500 text-white",
            t.variant === "info" && "bg-slate-800 text-white",
          )}
        >
          {t.message}
        </div>
      ))}
    </div>
  );
}

export default ToastContainer;
