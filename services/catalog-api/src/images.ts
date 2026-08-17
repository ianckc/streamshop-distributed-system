import type { Product } from "./products.js";

export const IMAGE_URL_EXPIRES_SECONDS = 3600;

export type ProductImages = {
  getImageUrl(productId: string): Promise<string | undefined>;
};

export function productImageKey(productId: string): string {
  return `${productId}.png`;
}

export async function attachImageUrl(
  product: Product,
  images: ProductImages | undefined,
  onError?: (err: unknown) => void,
): Promise<Product> {
  if (!images) {
    return product;
  }
  try {
    const image_url = await images.getImageUrl(product.id);
    return image_url ? { ...product, image_url } : product;
  } catch (err) {
    onError?.(err);
    return product;
  }
}

export async function attachImageUrls(
  products: Product[],
  images: ProductImages | undefined,
  onError?: (err: unknown) => void,
): Promise<Product[]> {
  return Promise.all(
    products.map((product) => attachImageUrl(product, images, onError)),
  );
}

export function rewritePresignedHost(
  signedUrl: string,
  publicEndpoint: string | undefined,
): string {
  if (!publicEndpoint) {
    return signedUrl;
  }
  const signed = new URL(signedUrl);
  const pub = new URL(publicEndpoint);
  signed.protocol = pub.protocol;
  signed.host = pub.host;
  return signed.toString();
}
