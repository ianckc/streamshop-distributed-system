import assert from "node:assert/strict";
import { test } from "node:test";

import { buildApp } from "./app.js";
import { products } from "./products.js";

test("GET /health returns ok", async () => {
  const app = buildApp({ logger: false, serviceName: "catalog-api" });

  const res = await app.inject({ method: "GET", url: "/health" });

  assert.equal(res.statusCode, 200);
  assert.equal(res.headers["content-type"], "application/json; charset=utf-8");
  assert.deepEqual(res.json(), { status: "ok", service: "catalog-api" });

  await app.close();
});

test("GET /api/catalog/products returns the in-memory list", async () => {
  const app = buildApp({ logger: false });

  const res = await app.inject({ method: "GET", url: "/api/catalog/products" });

  assert.equal(res.statusCode, 200);
  assert.equal(res.headers["content-type"], "application/json; charset=utf-8");

  const body = res.json() as { products: { id: string }[] };
  assert.deepEqual(body.products, products);
  assert.ok(body.products.some((product) => product.id === "prod-001"));

  await app.close();
});
