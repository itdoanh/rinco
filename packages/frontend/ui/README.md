# @rinco/ui

Shared component library for all RINCO Next.js frontends. Built on **shadcn/ui** (new-york style), powered by **Radix UI** primitives, **CVA** variants, and **Tailwind CSS 3** (Tailwind 4–ready via CSS variables).

## 30+ components

### Form primitives
- `Button`, `Input`, `Textarea`, `Label`
- `Checkbox`, `RadioGroup`, `Switch`, `Slider`
- `Select`, `Calendar`
- `Form` (react-hook-form integration with `FormField`, `FormItem`, `FormLabel`, `FormControl`, `FormDescription`, `FormMessage`)

### Layout
- `Card` (+ Header / Footer / Title / Description / Content)
- `Separator`, `AspectRatio`, `ScrollArea`
- `ResizablePanelGroup` + `ResizablePanel` + `ResizableHandle`

### Overlays
- `Dialog`, `AlertDialog`
- `Sheet`, `Drawer`
- `Popover`, `HoverCard`, `Tooltip`

### Menus
- `DropdownMenu` (full Radix sub-API)
- `ContextMenu`
- `Menubar`
- `NavigationMenu`

### Tabs / Disclosure
- `Tabs`, `Accordion`, `Collapsible`

### Data display
- `Badge`, `Table`, `Avatar`, `Progress`, `Skeleton`
- `DataTable` (typed wrapper around TanStack Table)
- `StatCard`
- `Pagination` (+ Previous / Next / Ellipsis)
- `Carousel` (Embla-powered)

### Feedback
- `Toast` + `Toaster` (Radix toast)
- `Sonner` (modern toast variant)
- `Alert` (+ Title / Description, semantic variants)

### Command palette
- `Command`, `CommandDialog`, `CommandInput`, `CommandList`, `CommandEmpty`, `CommandGroup`, `CommandItem`, `CommandShortcut`, `CommandSeparator`

### Utility
- `cn(...)` – tailwind-merge + clsx

## Conventions

- All components are **forward-ref** typed (`React.forwardRef<RefType, Props>`).
- Variants declared with **`class-variance-authority`** (cva).
- All Radix primitives wrapped as `"use client"` components.
- Classnames merged with `cn(...)`.
- Tailwind 4 compatible (`@tailwind base/components/utilities`).

## Usage

```ts
import { Button, Card, Input } from "@rinco/ui"

<Card>
  <Input placeholder="Email" />
  <Button variant="cta" size="lg">Submit</Button>
</Card>
```

Apps consume `@rinco/ui` either by:
1. Path alias in tsconfig:
   ```json
   "paths": { "@rinco/ui": ["../../packages/frontend/ui"] }
   ```
2. Next transpilePackages:
   ```js
   transpilePackages: ["@rinco/ui"]
   ```

## Build / typecheck

```bash
pnpm --filter @rinco/ui typecheck
```
