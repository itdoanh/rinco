FROM oven/bun:1.1 AS builder
WORKDIR /app
COPY frontend/landing/package.json ./
RUN bun install --frozen-lockfile
COPY frontend/landing/ ./
RUN bun run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3001
ENV PORT 3001
ENV HOSTNAME 0.0.0.0
CMD ["node", "server.js"]
