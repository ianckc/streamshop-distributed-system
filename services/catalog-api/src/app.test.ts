import assert from "node:assert/strict";
import { test } from "node:test";

import { buildApp } from "./app.js";
import type { Product } from "./products.js";
import { MemoryProductStore } from "./store/memory.js";

const failingStore = {
  async listProducts() {
    throw new Error("mongo unavailable");
  },
  async getProduct() {
    throw new Error("mongo unavailable");
  },
  async ping() {
    throw new Error("mongo unavailable");
  },
};

const sampleProducts: Product[] = [
  {
    id: "prod-001",
    name: "StreamShop Mug",
    price_pence: 1999,
    attributes: { colour: "navy", material: "ceramic" },
  },
];

test("GET /health returns ok", async () => {
  const app = buildApp({
    logger: false,
    serviceName: "catalog-api",
    store: new MemoryProductStore([]),
  });

  const res = await app.inject({ method: "GET", url: "/health" });

  assert.equal(res.statusCode, 200);
  assert.equal(res.headers["content-type"], "application/json; charset=utf-8");
  assert.deepEqual(res.json(), { status: "ok", service: "catalog-api" });

  await app.close();
});

test("GET /api/catalog/products returns products from the store", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
  });

  const res = await app.inject({ method: "GET", url: "/api/catalog/products" });

  assert.equal(res.statusCode, 200);
  assert.equal(res.headers["content-type"], "application/json; charset=utf-8");
  assert.deepEqual(res.json(), { products: sampleProducts });

  await app.close();
});

test("GET /api/catalog/products returns 503 when the store fails", async () => {
  const app = buildApp({
    logger: false,
    store: failingStore,
  });

  const res = await app.inject({ method: "GET", url: "/api/catalog/products" });

  assert.equal(res.statusCode, 503);
  assert.deepEqual(res.json(), { error: "failed to list products" });

  await app.close();
});

test("GET /api/catalog/products/:id returns a product from the store", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
  });

  const res = await app.inject({
    method: "GET",
    url: "/api/catalog/products/prod-001",
  });

  assert.equal(res.statusCode, 200);
  assert.equal(res.headers["content-type"], "application/json; charset=utf-8");
  assert.deepEqual(res.json(), sampleProducts[0]);

  await app.close();
});

test("GET /api/catalog/products/:id returns 404 when missing", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
  });

  const res = await app.inject({
    method: "GET",
    url: "/api/catalog/products/prod-missing",
  });

  assert.equal(res.statusCode, 404);
  assert.deepEqual(res.json(), { error: "product not found" });

  await app.close();
});

test("GET /api/catalog/products/:id returns 503 when the store fails", async () => {
  const app = buildApp({
    logger: false,
    store: failingStore,
  });

  const res = await app.inject({
    method: "GET",
    url: "/api/catalog/products/prod-001",
  });

  assert.equal(res.statusCode, 503);
  assert.deepEqual(res.json(), { error: "failed to get product" });

  await app.close();
});

test("GET /ready returns ok when the store pings", async () => {
  const app = buildApp({
    logger: false,
    serviceName: "catalog-api",
    store: new MemoryProductStore([]),
  });

  const res = await app.inject({ method: "GET", url: "/ready" });

  assert.equal(res.statusCode, 200);
  assert.deepEqual(res.json(), { status: "ok", service: "catalog-api" });

  await app.close();
});

test("GET /ready returns 503 when the store ping fails", async () => {
  const app = buildApp({
    logger: false,
    store: failingStore,
  });

  const res = await app.inject({ method: "GET", url: "/ready" });

  assert.equal(res.statusCode, 503);
  assert.deepEqual(res.json(), { error: "not ready" });

  await app.close();
});
