FROM oven/bun:1.1-alpine AS base
WORKDIR /app
COPY package*.json ./
RUN bun install --frozen-lockfile
COPY ./frontend/meeting-ui/ .
RUN bun run build
EXPOSE 3003
ENV PORT=3003
CMD ["bun", "run", "start"]
