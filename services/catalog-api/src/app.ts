import Fastify, { type FastifyInstance, type FastifyReply } from "fastify";

import {
  CATALOG_LIST_KEY,
  type CacheStatus,
  cacheAside,
  catalogProductKey,
  type ProductCache,
} from "./cache.js";
import type { Product } from "./products.js";
import type { ProductStore } from "./store.js";

export type BuildAppOptions = {
  serviceName?: string;
  logger?: boolean;
  store: ProductStore;
  cache?: ProductCache;
};

export type HealthResponse = {
  status: "ok";
  service: string;
};

export type ProductListResponse = {
  products: Product[];
};

type ErrorResponse = {
  error: string;
};

export function buildApp(options: BuildAppOptions): FastifyInstance {
  const serviceName =
    options.serviceName ?? process.env.SERVICE_NAME ?? "catalog-api";
  const app = Fastify({ logger: options.logger ?? true });

  app.get(
    "/health",
    async (): Promise<HealthResponse> => ({
      status: "ok",
      service: serviceName,
    }),
  );

  app.get("/ready", async (request, reply) => {
    try {
      await options.store.ping();
      const body: HealthResponse = { status: "ok", service: serviceName };
      return body;
    } catch (err) {
      request.log.error(err);
      const body: ErrorResponse = { error: "not ready" };
      return reply.code(503).send(body);
    }
  });

  app.get("/api/catalog/products", async (request, reply) => {
    try {
      const { value: products, status } = await cacheAside(
        options.cache,
        CATALOG_LIST_KEY,
        () => options.store.listProducts(),
        { onError: (err) => request.log.error(err) },
      );
      setCacheHeader(reply, status);
      const body: ProductListResponse = { products };
      return body;
    } catch (err) {
      request.log.error(err);
      const body: ErrorResponse = { error: "failed to list products" };
      return reply.code(503).send(body);
    }
  });

  app.get<{ Params: { id: string } }>(
    "/api/catalog/products/:id",
    async (request, reply) => {
      try {
        const { value: product, status } = await cacheAside(
          options.cache,
          catalogProductKey(request.params.id),
          () => options.store.getProduct(request.params.id),
          {
            onError: (err) => request.log.error(err),
            shouldCache: (value) => value !== undefined,
          },
        );
        setCacheHeader(reply, status);
        if (!product) {
          const body: ErrorResponse = { error: "product not found" };
          return reply.code(404).send(body);
        }
        return product;
      } catch (err) {
        request.log.error(err);
        const body: ErrorResponse = { error: "failed to get product" };
        return reply.code(503).send(body);
      }
    },
  );

  return app;
}

function setCacheHeader(reply: FastifyReply, status: CacheStatus): void {
  if (status !== "bypass") {
    void reply.header("x-cache", status);
  }
}
