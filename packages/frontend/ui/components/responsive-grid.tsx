import { cn } from "../lib/utils";

/**
 * Responsive grid helper.
 *
 * Wraps children in a grid that adjusts columns based on Tailwind
 * breakpoints.  Default values target common admin layouts:
 *
 *  - mobile (<640px):   1 col
 *  - tablet (≥768px):  2 cols
 *  - desktop (≥1024px): 3 cols
 *  - wide (≥1280px):    4 cols
 */
export interface ResponsiveGridProps {
  children: React.ReactNode;
  className?: string;
  cols?: {
    default?: number;
    sm?: number;
    md?: number;
    lg?: number;
    xl?: number;
  };
  gap?: number;
}

const colsToClass: Record<number, string> = {
  1: "grid-cols-1",
  2: "grid-cols-2",
  3: "grid-cols-3",
  4: "grid-cols-4",
  5: "grid-cols-5",
  6: "grid-cols-6",
};

const gapToClass: Record<number, string> = {
  0: "gap-0",
  1: "gap-1",
  2: "gap-2",
  3: "gap-3",
  4: "gap-4",
  5: "gap-5",
  6: "gap-6",
  8: "gap-8",
};

export function ResponsiveGrid({
  children,
  className,
  cols = { default: 1, md: 2, lg: 3, xl: 4 },
  gap = 4,
}: ResponsiveGridProps) {
  const c =
    [
      cols.default && colsToClass[cols.default],
      cols.sm && `sm:${colsToClass[cols.sm]}`,
      cols.md && `md:${colsToClass[cols.md]}`,
      cols.lg && `lg:${colsToClass[cols.lg]}`,
      cols.xl && `xl:${colsToClass[cols.xl]}`,
    ]
      .filter(Boolean)
      .join(" ");

  return <div className={cn("grid", c, gapToClass[gap] ?? "gap-4", className)}>{children}</div>;
}

/**
 * Show children only at specified breakpoints.
 *
 * Usage:
 *  - `<MobileOnly>` only renders on mobile
 *  - `<DesktopOnly>` only renders on md+
 */
export function MobileOnly({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn("block md:hidden", className)}>{children}</div>;
}

export function DesktopOnly({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn("hidden md:block", className)}>{children}</div>;
}

export function TabletOnly({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn("hidden md:block lg:hidden", className)}>{children}</div>;
}
