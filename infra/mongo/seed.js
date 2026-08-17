// Seed runs only on first start of an empty Mongo volume.
// Reset with: docker compose down -v
// _id is the public product id (matches order-api product_id).

db.products.insertMany([
  {
    _id: "prod-001",
    name: "StreamShop Mug",
    price_pence: 1999,
    attributes: { colour: "navy", material: "ceramic" },
  },
  {
    _id: "prod-002",
    name: "StreamShop T-shirt",
    price_pence: 2499,
    attributes: { colour: "black", size: "M" },
  },
  {
    _id: "prod-003",
    name: "Distributed Systems Notebook",
    price_pence: 1299,
    attributes: { pages: "192", binding: "paperback" },
  },
]);
