import type { Product } from "./products.js";

export type ProductStore = {
  listProducts(): Promise<Product[]>;
};
