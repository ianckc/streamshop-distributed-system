import assert from "node:assert/strict";
import { test } from "node:test";
import { MemoryProductImages } from "./images/memory.js";
import {
  attachImageUrl,
  attachImageUrls,
  productImageKey,
  rewritePresignedHost,
} from "./images.js";
import type { Product } from "./products.js";

const sample: Product = {
  id: "prod-001",
  name: "StreamShop Mug",
  price_pence: 1999,
  attributes: { colour: "navy" },
};

test("productImageKey uses png object names", () => {
  assert.equal(productImageKey("prod-001"), "prod-001.png");
});

test("rewritePresignedHost swaps host for browser-facing MinIO", () => {
  const signed =
    "http://minio:9000/product-images/prod-001.png?X-Amz-Signature=abc";
  assert.equal(
    rewritePresignedHost(signed, "http://localhost:9000"),
    "http://localhost:9000/product-images/prod-001.png?X-Amz-Signature=abc",
  );
  assert.equal(rewritePresignedHost(signed, undefined), signed);
});

test("attachImageUrl adds image_url when present", async () => {
  const images = new MemoryProductImages({
    "prod-001": "http://localhost:9000/product-images/prod-001.png",
  });
  const got = await attachImageUrl(sample, images);
  assert.equal(
    got.image_url,
    "http://localhost:9000/product-images/prod-001.png",
  );
});

test("attachImageUrl omits image_url when missing", async () => {
  const got = await attachImageUrl(sample, new MemoryProductImages());
  assert.equal(got.image_url, undefined);
});

test("attachImageUrl fail-open when images throws", async () => {
  const images = {
    async getImageUrl() {
      throw new Error("minio down");
    },
  };
  const got = await attachImageUrl(sample, images);
  assert.deepEqual(got, sample);
});

test("attachImageUrls maps a list", async () => {
  const images = new MemoryProductImages({
    "prod-001": "http://example/prod-001.png",
  });
  const [got] = await attachImageUrls([sample], images);
  assert.equal(got?.image_url, "http://example/prod-001.png");
});
