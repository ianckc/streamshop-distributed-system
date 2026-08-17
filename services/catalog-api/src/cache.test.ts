import assert from "node:assert/strict";
import { test } from "node:test";
import { MemoryProductCache } from "./cache/memory.js";
import { RedisProductCache } from "./cache/redis.js";
import {
  CACHE_TTL_SECONDS,
  CATALOG_LIST_KEY,
  cacheAside,
  catalogProductKey,
} from "./cache.js";
import type { Product } from "./products.js";

const sampleProducts: Product[] = [
  {
    id: "prod-001",
    name: "StreamShop Mug",
    price_pence: 1999,
    attributes: { colour: "navy", material: "ceramic" },
  },
];

test("cacheAside bypasses when no cache is configured", async () => {
  let loads = 0;
  const result = await cacheAside(undefined, CATALOG_LIST_KEY, async () => {
    loads += 1;
    return sampleProducts;
  });

  assert.equal(result.status, "bypass");
  assert.deepEqual(result.value, sampleProducts);
  assert.equal(loads, 1);
});

test("cacheAside misses then hits", async () => {
  const cache = new MemoryProductCache();
  let loads = 0;
  const load = async () => {
    loads += 1;
    return sampleProducts;
  };

  const miss = await cacheAside(cache, CATALOG_LIST_KEY, load);
  const hit = await cacheAside(cache, CATALOG_LIST_KEY, load);

  assert.equal(miss.status, "miss");
  assert.equal(hit.status, "hit");
  assert.deepEqual(hit.value, sampleProducts);
  assert.equal(loads, 1);
});

test("cacheAside skips set when shouldCache is false", async () => {
  const cache = new MemoryProductCache();
  let loads = 0;

  const first = await cacheAside(
    cache,
    catalogProductKey("missing"),
    async () => {
      loads += 1;
      return undefined;
    },
    { shouldCache: (value) => value !== undefined },
  );
  const second = await cacheAside(
    cache,
    catalogProductKey("missing"),
    async () => {
      loads += 1;
      return undefined;
    },
    { shouldCache: (value) => value !== undefined },
  );

  assert.equal(first.status, "miss");
  assert.equal(second.status, "miss");
  assert.equal(loads, 2);
});

test("cacheAside fail-open when get throws", async () => {
  const cache = {
    async get() {
      throw new Error("redis down");
    },
    async set() {},
  };
  let loads = 0;

  const result = await cacheAside(cache, CATALOG_LIST_KEY, async () => {
    loads += 1;
    return sampleProducts;
  });

  assert.equal(result.status, "miss");
  assert.deepEqual(result.value, sampleProducts);
  assert.equal(loads, 1);
});

test("cacheAside fail-open when set throws", async () => {
  const cache = {
    async get() {
      return undefined;
    },
    async set() {
      throw new Error("redis down");
    },
  };

  const result = await cacheAside(cache, CATALOG_LIST_KEY, async () => {
    return sampleProducts;
  });

  assert.equal(result.status, "miss");
  assert.deepEqual(result.value, sampleProducts);
});

test("cacheAside treats invalid JSON as a miss", async () => {
  const cache = new MemoryProductCache();
  await cache.set(CATALOG_LIST_KEY, "not-json", CACHE_TTL_SECONDS);
  let loads = 0;

  const result = await cacheAside(cache, CATALOG_LIST_KEY, async () => {
    loads += 1;
    return sampleProducts;
  });

  assert.equal(result.status, "miss");
  assert.equal(loads, 1);
  assert.deepEqual(result.value, sampleProducts);
});

test("RedisProductCache maps null get to undefined", async () => {
  const cache = new RedisProductCache({
    async get() {
      return null;
    },
    async set() {},
    async close() {},
  });

  assert.equal(await cache.get("catalog:products"), undefined);
});

test("RedisProductCache set uses EX ttl", async () => {
  let seen:
    | {
        key: string;
        value: string;
        options: { expiration: { type: "EX"; value: number } };
      }
    | undefined;
  const cache = new RedisProductCache({
    async get() {
      return "cached";
    },
    async set(key, value, options) {
      seen = { key, value, options };
    },
    async close() {},
  });

  assert.equal(await cache.get("catalog:products"), "cached");
  await cache.set("catalog:products", "[]", 60);
  assert.deepEqual(seen, {
    key: "catalog:products",
    value: "[]",
    options: { expiration: { type: "EX", value: 60 } },
  });
});
