export type Product = {
  id: string;
  name: string;
  price_pence: number;
  attributes: Record<string, string>;
};

export const products: Product[] = [
  {
    id: "prod-001",
    name: "StreamShop Mug",
    price_pence: 1999,
    attributes: { colour: "navy", material: "ceramic" },
  },
  {
    id: "prod-002",
    name: "StreamShop T-shirt",
    price_pence: 2499,
    attributes: { colour: "black", size: "M" },
  },
  {
    id: "prod-003",
    name: "Distributed Systems Notebook",
    price_pence: 1299,
    attributes: { pages: "192", binding: "paperback" },
  },
];
