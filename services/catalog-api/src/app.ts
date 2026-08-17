import Fastify, { type FastifyInstance } from "fastify";

import type { Product } from "./products.js";
import type { ProductStore } from "./store.js";

export type BuildAppOptions = {
  serviceName?: string;
  logger?: boolean;
  store: ProductStore;
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

  app.get("/health", async (): Promise<HealthResponse> => ({
    status: "ok",
    service: serviceName,
  }));

  app.get("/api/catalog/products", async (request, reply) => {
    try {
      const products = await options.store.listProducts();
      const body: ProductListResponse = { products };
      return body;
    } catch (err) {
      request.log.error(err);
      const body: ErrorResponse = { error: "failed to list products" };
      return reply.code(503).send(body);
    }
  });

  return app;
}
