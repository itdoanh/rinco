"use client";

import { cn } from "@/lib/utils";

/**
 * Generic loading skeleton. Có thể dùng inline (h-4 w-32) hoặc
 * composite (LoadingSkeleton.Card).
 */
export function LoadingSkeleton({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("animate-pulse rounded-md bg-muted", className)}
      aria-busy="true"
      aria-live="polite"
      {...props}
    />
  );
}

/** Preset composite skeletons. */
LoadingSkeleton.Card = function CardSkeleton({
  lines = 3,
  className,
}: {
  lines?: number;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "rounded-xl border bg-card p-6 space-y-3",
        className,
      )}
      role="status"
      aria-label="Loading card"
    >
      <LoadingSkeleton className="h-5 w-1/3" />
      {Array.from({ length: lines }).map((_, i) => (
        <LoadingSkeleton
          key={i}
          className={cn("h-3", i === lines - 1 ? "w-2/3" : "w-full")}
        />
      ))}
    </div>
  );
};

LoadingSkeleton.Table = function TableSkeleton({
  rows = 5,
  columns = 4,
}: {
  rows?: number;
  columns?: number;
}) {
  return (
    <div
      className="rounded-xl border bg-card overflow-hidden"
      role="status"
      aria-label="Loading table"
    >
      <div className="border-b bg-muted/30 px-4 py-3 flex gap-3">
        {Array.from({ length: columns }).map((_, i) => (
          <LoadingSkeleton key={i} className="h-3 flex-1" />
        ))}
      </div>
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="border-b last:border-0 px-4 py-3 flex gap-3">
          {Array.from({ length: columns }).map((_, c) => (
            <LoadingSkeleton
              key={c}
              className={cn("h-3 flex-1", c === 0 && "max-w-[120px]")}
            />
          ))}
        </div>
      ))}
    </div>
  );
};

LoadingSkeleton.Stats = function StatsSkeleton({
  count = 4,
}: {
  count?: number;
}) {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4" role="status" aria-label="Loading stats">
      {Array.from({ length: count }).map((_, i) => (
        <LoadingSkeleton.Card key={i} lines={2} />
      ))}
    </div>
  );
};
