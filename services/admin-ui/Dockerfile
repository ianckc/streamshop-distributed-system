# Build
FROM node:22-bookworm-slim AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
ARG ADMIN_UI_BASE=/admin/
ENV ADMIN_UI_BASE=$ADMIN_UI_BASE
RUN npm run build

# Runtime — nginx (assets + /health) fronts TanStack Start (srvx) for SSR
FROM node:22-bookworm-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends nginx wget ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && rm -f /etc/nginx/sites-enabled/default

WORKDIR /app
ENV NODE_ENV=production

COPY package.json package-lock.json ./
RUN npm ci --omit=dev

COPY --from=builder /app/dist ./dist
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --retries=5 --start-period=25s \
  CMD wget -qO- http://127.0.0.1/health || exit 1

CMD ["/docker-entrypoint.sh"]
