import assert from "node:assert/strict";
import { test } from "node:test";

import { buildApp } from "./app.js";
import { MemoryProductCache } from "./cache/memory.js";
import { MemoryProductImages } from "./images/memory.js";
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
  assert.equal(res.headers["x-replica-id"], "catalog-api");
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

test("GET /api/catalog/products sets x-cache miss then hit", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
    cache: new MemoryProductCache(),
  });

  const miss = await app.inject({
    method: "GET",
    url: "/api/catalog/products",
  });
  const hit = await app.inject({
    method: "GET",
    url: "/api/catalog/products",
  });

  assert.equal(miss.statusCode, 200);
  assert.equal(miss.headers["x-cache"], "miss");
  assert.equal(hit.statusCode, 200);
  assert.equal(hit.headers["x-cache"], "hit");
  assert.deepEqual(hit.json(), { products: sampleProducts });

  await app.close();
});

test("GET /api/catalog/products/:id sets x-cache miss then hit", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
    cache: new MemoryProductCache(),
  });

  const miss = await app.inject({
    method: "GET",
    url: "/api/catalog/products/prod-001",
  });
  const hit = await app.inject({
    method: "GET",
    url: "/api/catalog/products/prod-001",
  });

  assert.equal(miss.headers["x-cache"], "miss");
  assert.equal(hit.headers["x-cache"], "hit");
  assert.deepEqual(hit.json(), sampleProducts[0]);

  await app.close();
});

test("GET /api/catalog/products/:id does not cache 404s", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
    cache: new MemoryProductCache(),
  });

  const first = await app.inject({
    method: "GET",
    url: "/api/catalog/products/prod-missing",
  });
  const second = await app.inject({
    method: "GET",
    url: "/api/catalog/products/prod-missing",
  });

  assert.equal(first.statusCode, 404);
  assert.equal(first.headers["x-cache"], "miss");
  assert.equal(second.statusCode, 404);
  assert.equal(second.headers["x-cache"], "miss");

  await app.close();
});

test("GET /api/catalog/products fail-open when cache get throws", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
    cache: {
      async get() {
        throw new Error("redis down");
      },
      async set() {},
    },
  });

  const res = await app.inject({
    method: "GET",
    url: "/api/catalog/products",
  });

  assert.equal(res.statusCode, 200);
  assert.equal(res.headers["x-cache"], "miss");
  assert.deepEqual(res.json(), { products: sampleProducts });

  await app.close();
});

test("GET /api/catalog/products includes image_url from the image store", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
    images: new MemoryProductImages({
      "prod-001": "http://localhost:9000/product-images/prod-001.png",
    }),
  });

  const res = await app.inject({
    method: "GET",
    url: "/api/catalog/products",
  });

  assert.equal(res.statusCode, 200);
  assert.deepEqual(res.json(), {
    products: [
      {
        ...sampleProducts[0],
        image_url: "http://localhost:9000/product-images/prod-001.png",
      },
    ],
  });

  await app.close();
});

test("GET /api/catalog/products/:id includes image_url from the image store", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
    images: new MemoryProductImages({
      "prod-001": "http://localhost:9000/product-images/prod-001.png",
    }),
  });

  const res = await app.inject({
    method: "GET",
    url: "/api/catalog/products/prod-001",
  });

  assert.equal(res.statusCode, 200);
  assert.deepEqual(res.json(), {
    ...sampleProducts[0],
    image_url: "http://localhost:9000/product-images/prod-001.png",
  });

  await app.close();
});

test("GET /api/catalog/products fail-open when image store throws", async () => {
  const app = buildApp({
    logger: false,
    store: new MemoryProductStore(sampleProducts),
    images: {
      async getImageUrl() {
        throw new Error("minio down");
      },
    },
  });

  const res = await app.inject({
    method: "GET",
    url: "/api/catalog/products",
  });

  assert.equal(res.statusCode, 200);
  assert.deepEqual(res.json(), { products: sampleProducts });

  await app.close();
});
