import { createClient } from "redis";

import type { ProductCache } from "../cache.js";

export type RedisCacheClient = {
  get(key: string): Promise<string | null>;
  set(
    key: string,
    value: string,
    options: { expiration: { type: "EX"; value: number } },
  ): Promise<unknown>;
  close(): Promise<void>;
};

export async function connectRedis(url: string): Promise<RedisCacheClient> {
  const client = createClient({ url });
  client.on("error", (err) => {
    console.error(err);
  });
  await client.connect();
  return client;
}

export class RedisProductCache implements ProductCache {
  readonly #client: RedisCacheClient;

  constructor(client: RedisCacheClient) {
    this.#client = client;
  }

  async get(key: string): Promise<string | undefined> {
    const value = await this.#client.get(key);
    return value === null ? undefined : value;
  }

  async set(key: string, value: string, ttlSeconds: number): Promise<void> {
    await this.#client.set(key, value, {
      expiration: { type: "EX", value: ttlSeconds },
    });
  }
}
