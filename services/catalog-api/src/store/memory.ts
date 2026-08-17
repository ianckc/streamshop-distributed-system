import type { Product } from "../products.js";
import type { ProductStore } from "../store.js";

export class MemoryProductStore implements ProductStore {
  readonly #products: Product[];

  constructor(products: Product[]) {
    this.#products = products;
  }

  async listProducts(): Promise<Product[]> {
    return this.#products;
  }

  async getProduct(id: string): Promise<Product | undefined> {
    return this.#products.find((product) => product.id === id);
  }

  async ping(): Promise<void> {}
}
