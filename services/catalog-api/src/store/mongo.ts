import { type Collection, MongoClient } from "mongodb";

import type { Product } from "../products.js";
import type { ProductStore } from "../store.js";

export type ProductDocument = {
  _id: string;
  name: string;
  price_pence: number;
  attributes: Record<string, string>;
};

export async function connect(uri: string): Promise<MongoClient> {
  const client = new MongoClient(uri);
  await client.connect();
  return client;
}

export class MongoProductStore implements ProductStore {
  readonly #collection: Collection<ProductDocument>;

  constructor(collection: Collection<ProductDocument>) {
    this.#collection = collection;
  }

  async listProducts(): Promise<Product[]> {
    const docs = await this.#collection.find().sort({ _id: 1 }).toArray();
    return docs.map(toProduct);
  }

  async getProduct(id: string): Promise<Product | undefined> {
    const doc = await this.#collection.findOne({ _id: id });
    return doc ? toProduct(doc) : undefined;
  }

  async ping(): Promise<void> {
    await this.#collection.db.command({ ping: 1 });
  }
}

function toProduct(doc: ProductDocument): Product {
  return {
    id: doc._id,
    name: doc.name,
    price_pence: doc.price_pence,
    attributes: doc.attributes,
  };
}
