FROM node:22-alpine AS builder

WORKDIR /app/docs

COPY docs/package.json docs/package-lock.json ./
RUN npm ci

COPY docs/ .

ARG DOCUSAURUS_URL=http://localhost:8080
ENV DOCUSAURUS_URL=$DOCUSAURUS_URL

RUN npm run build

FROM nginx:1.27-alpine

RUN apk add --no-cache wget

COPY --from=builder /app/docs/build /usr/share/nginx/html
COPY infra/nginx/docs.conf /etc/nginx/conf.d/default.conf

HEALTHCHECK \
  CMD wget -qO- http://127.0.0.1/health || exit 1

EXPOSE 80
