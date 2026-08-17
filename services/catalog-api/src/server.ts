import { buildApp } from "./app.js";
import { connect, MongoProductStore } from "./store/mongo.js";

const port = Number(process.env.PORT ?? 3001);
const uri = process.env.MONGODB_URI;
if (!uri) {
  console.error("MONGODB_URI is required");
  process.exit(1);
}

const client = await connect(uri);
const store = new MongoProductStore(client.db().collection("products"));
const app = buildApp({ store });

const shutdown = async () => {
  await app.close();
  await client.close();
};

process.on("SIGINT", () => {
  void shutdown().then(() => process.exit(0));
});
process.on("SIGTERM", () => {
  void shutdown().then(() => process.exit(0));
});

try {
  await app.listen({ port, host: "0.0.0.0" });
} catch (err) {
  app.log.error(err);
  await client.close();
  process.exit(1);
}
