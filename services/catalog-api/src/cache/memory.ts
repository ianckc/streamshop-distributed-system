import type { ProductCache } from "../cache.js";

export class MemoryProductCache implements ProductCache {
  readonly #values = new Map<string, string>();

  async get(key: string): Promise<string | undefined> {
    return this.#values.get(key);
  }

  async set(key: string, value: string, _ttlSeconds: number): Promise<void> {
    this.#values.set(key, value);
  }
}
