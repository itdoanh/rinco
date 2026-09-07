# Tenant Site

Dynamic tenant site renderer for RINCO multi-tenant platform.

## Features

- **Dynamic Branding**: CSS variables for per-tenant theming
- **Block Rendering**: Dynamic content blocks from API
- **SSR with ISR**: Server-side rendering with incremental static regeneration

## Getting Started

```bash
bun install
bun dev
```

Access at http://localhost:3002

## URL Structure

- `/[tenant]` - Tenant homepage
- `/[tenant]/[page]` - Tenant subpages
