export type Product = {
  id: string;
  name: string;
  price_pence: number;
  attributes: Record<string, string>;
};
