import type { ProductImages } from "../images.js";

export class MemoryProductImages implements ProductImages {
  readonly #urls: Map<string, string>;

  constructor(urls: Record<string, string> = {}) {
    this.#urls = new Map(Object.entries(urls));
  }

  async getImageUrl(productId: string): Promise<string | undefined> {
    return this.#urls.get(productId);
  }
}
