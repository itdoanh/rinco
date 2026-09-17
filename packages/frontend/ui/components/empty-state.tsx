"use client";

import { cn } from "@/lib/utils";
import { Inbox, Search, FileX, AlertCircle, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

interface EmptyStateProps {
  /** Icon hiển thị (mặc định: Inbox). */
  icon?: LucideIcon;
  /** Tiêu đề ngắn. */
  title: string;
  /** Mô tả (optional). */
  description?: string;
  /** CTA element (button/link) — optional. */
  action?: ReactNode;
  /** Style variant. */
  variant?: "default" | "search" | "error" | "forbidden";
  className?: string;
}

const VARIANT_ICONS: Record<NonNullable<EmptyStateProps["variant"]>, LucideIcon> = {
  default: Inbox,
  search: Search,
  error: AlertCircle,
  forbidden: FileX,
};

const VARIANT_TONES: Record<NonNullable<EmptyStateProps["variant"]>, string> = {
  default: "text-muted-foreground",
  search: "text-muted-foreground",
  error: "text-destructive",
  forbidden: "text-muted-foreground",
};

/**
 * EmptyState — hiển thị khi list/grid không có data.
 *
 * Variants:
 *   - default: generic empty (Inbox icon)
 *   - search:  không có kết quả tìm kiếm (Search icon)
 *   - error:   fetch failed (AlertCircle icon)
 *   - forbidden: 403 / no permission (FileX icon)
 */
export function EmptyState({
  icon,
  title,
  description,
  action,
  variant = "default",
  className,
}: EmptyStateProps) {
  const Icon = icon ?? VARIANT_ICONS[variant];
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center py-12 px-6 border border-dashed rounded-xl bg-card/40",
        className,
      )}
      role="status"
      aria-label={title}
    >
      <div
        className={cn(
          "w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-4",
          VARIANT_TONES[variant],
        )}
      >
        <Icon className="w-6 h-6" aria-hidden />
      </div>
      <h3 className="text-base font-semibold mb-1">{title}</h3>
      {description && (
        <p className="text-sm text-muted-foreground max-w-md">{description}</p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
