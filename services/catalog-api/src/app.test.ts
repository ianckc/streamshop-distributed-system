import assert from "node:assert/strict";
import { test } from "node:test";

import { buildApp } from "./app.js";

test("GET /health returns ok", async () => {
  const app = buildApp({ logger: false, serviceName: "catalog-api" });

  const res = await app.inject({ method: "GET", url: "/health" });

  assert.equal(res.statusCode, 200);
  assert.equal(res.headers["content-type"], "application/json; charset=utf-8");
  assert.deepEqual(res.json(), { status: "ok", service: "catalog-api" });

  await app.close();
});
