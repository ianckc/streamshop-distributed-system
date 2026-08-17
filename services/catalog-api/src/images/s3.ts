import {
  GetObjectCommand,
  HeadObjectCommand,
  S3Client,
} from "@aws-sdk/client-s3";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";

import {
  IMAGE_URL_EXPIRES_SECONDS,
  type ProductImages,
  productImageKey,
  rewritePresignedHost,
} from "../images.js";

export type S3ImagesConfig = {
  endpoint: string;
  region?: string;
  accessKeyId: string;
  secretAccessKey: string;
  bucket: string;
  publicEndpoint?: string;
  expiresSeconds?: number;
};

export function connectS3(config: S3ImagesConfig): S3Client {
  return new S3Client({
    endpoint: config.endpoint,
    region: config.region ?? "us-east-1",
    credentials: {
      accessKeyId: config.accessKeyId,
      secretAccessKey: config.secretAccessKey,
    },
    forcePathStyle: true,
  });
}

export class S3ProductImages implements ProductImages {
  readonly #client: S3Client;
  readonly #bucket: string;
  readonly #publicEndpoint: string | undefined;
  readonly #expiresSeconds: number;

  constructor(client: S3Client, config: S3ImagesConfig) {
    this.#client = client;
    this.#bucket = config.bucket;
    this.#publicEndpoint = config.publicEndpoint;
    this.#expiresSeconds = config.expiresSeconds ?? IMAGE_URL_EXPIRES_SECONDS;
  }

  async getImageUrl(productId: string): Promise<string | undefined> {
    const key = productImageKey(productId);
    try {
      await this.#client.send(
        new HeadObjectCommand({ Bucket: this.#bucket, Key: key }),
      );
    } catch (err) {
      if (isMissingObject(err)) {
        return undefined;
      }
      throw err;
    }

    const signed = await getSignedUrl(
      this.#client,
      new GetObjectCommand({ Bucket: this.#bucket, Key: key }),
      { expiresIn: this.#expiresSeconds },
    );
    return rewritePresignedHost(signed, this.#publicEndpoint);
  }
}

function isMissingObject(err: unknown): boolean {
  if (!err || typeof err !== "object") {
    return false;
  }
  const name = "name" in err ? String(err.name) : "";
  const metadata =
    "$metadata" in err && err.$metadata && typeof err.$metadata === "object"
      ? err.$metadata
      : undefined;
  const status =
    metadata && "httpStatusCode" in metadata
      ? Number(metadata.httpStatusCode)
      : undefined;
  return name === "NotFound" || name === "NoSuchKey" || status === 404;
}
