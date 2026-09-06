FROM oven/bun:1.1 AS builder
WORKDIR /app
COPY frontend/admin-portal/package.json ./
RUN bun install --frozen-lockfile
COPY frontend/admin-portal/ ./
RUN bun run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
EXPOSE 3002
ENV PORT 3002
ENV HOSTNAME 0.0.0.0
CMD ["node", "server.js"]
