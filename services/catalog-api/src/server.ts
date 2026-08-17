import { buildApp } from "./app.js";
import { connectRedis, RedisProductCache } from "./cache/redis.js";
import { connectS3, S3ProductImages } from "./images/s3.js";
import { connect, MongoProductStore } from "./store/mongo.js";

const port = Number(process.env.PORT ?? 3001);
const uri = process.env.MONGODB_URI;
if (!uri) {
  console.error("MONGODB_URI is required");
  process.exit(1);
}

const redisUrl = process.env.REDIS_URL;
if (!redisUrl) {
  console.error("REDIS_URL is required");
  process.exit(1);
}

const s3Endpoint = process.env.S3_ENDPOINT;
const s3AccessKey = process.env.S3_ACCESS_KEY;
const s3SecretKey = process.env.S3_SECRET_KEY;
const s3Bucket = process.env.S3_BUCKET;
if (!s3Endpoint || !s3AccessKey || !s3SecretKey || !s3Bucket) {
  console.error(
    "S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, and S3_BUCKET are required",
  );
  process.exit(1);
}

const client = await connect(uri);
const redisClient = await connectRedis(redisUrl);
const s3Config = {
  endpoint: s3Endpoint,
  region: process.env.S3_REGION,
  accessKeyId: s3AccessKey,
  secretAccessKey: s3SecretKey,
  bucket: s3Bucket,
  publicEndpoint: process.env.S3_PUBLIC_ENDPOINT,
};
const s3Client = connectS3(s3Config);
const store = new MongoProductStore(client.db().collection("products"));
const cache = new RedisProductCache(redisClient);
const images = new S3ProductImages(s3Client, s3Config);
const app = buildApp({ store, cache, images });

const shutdown = async () => {
  await app.close();
  await redisClient.close();
  s3Client.destroy();
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
  await redisClient.close();
  s3Client.destroy();
  await client.close();
  process.exit(1);
}
