import Fastify, { type FastifyInstance } from "fastify";

export type BuildAppOptions = {
  serviceName?: string;
  logger?: boolean;
};

export type HealthResponse = {
  status: "ok";
  service: string;
};

export function buildApp(options: BuildAppOptions = {}): FastifyInstance {
  const serviceName =
    options.serviceName ?? process.env.SERVICE_NAME ?? "catalog-api";
  const app = Fastify({ logger: options.logger ?? true });

  app.get("/health", async (): Promise<HealthResponse> => ({
    status: "ok",
    service: serviceName,
  }));

  return app;
}
