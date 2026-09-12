# Build
FROM node:22-bookworm-slim AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
ARG ADMIN_UI_BASE=/admin/
ENV ADMIN_UI_BASE=$ADMIN_UI_BASE
RUN npm run build

# Runtime — TanStack Start (srvx) + static client assets
FROM node:22-bookworm-slim

WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3020

COPY package.json package-lock.json ./
RUN npm ci --omit=dev

COPY --from=builder /app/dist ./dist

EXPOSE 3020

HEALTHCHECK --interval=10s --timeout=3s --retries=5 --start-period=15s \
  CMD node -e "fetch('http://127.0.0.1:3020/admin/').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"

CMD ["npx", "srvx", "serve", "--prod", "--host", "0.0.0.0", "--port", "3020", "--static", "./dist/client", "--entry", "./dist/server/server.js"]
