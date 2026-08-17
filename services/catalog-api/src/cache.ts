export const CACHE_TTL_SECONDS = 60;

export const CATALOG_LIST_KEY = "catalog:products";

export function catalogProductKey(id: string): string {
  return `catalog:product:${id}`;
}

export type ProductCache = {
  get(key: string): Promise<string | undefined>;
  set(key: string, value: string, ttlSeconds: number): Promise<void>;
};

export type CacheStatus = "hit" | "miss" | "bypass";

export type CacheAsideOptions<T> = {
  ttlSeconds?: number;
  shouldCache?: (value: T) => boolean;
  onError?: (err: unknown) => void;
};

export async function cacheAside<T>(
  cache: ProductCache | undefined,
  key: string,
  load: () => Promise<T>,
  options: CacheAsideOptions<T> = {},
): Promise<{ value: T; status: CacheStatus }> {
  if (!cache) {
    return { value: await load(), status: "bypass" };
  }

  const ttlSeconds = options.ttlSeconds ?? CACHE_TTL_SECONDS;
  const shouldCache = options.shouldCache ?? alwaysCache;
  const onError = options.onError;

  try {
    const raw = await cache.get(key);
    if (raw !== undefined) {
      return { value: JSON.parse(raw) as T, status: "hit" };
    }
  } catch (err) {
    onError?.(err);
  }

  const value = await load();

  if (shouldCache(value)) {
    try {
      await cache.set(key, JSON.stringify(value), ttlSeconds);
    } catch (err) {
      onError?.(err);
    }
  }

  return { value, status: "miss" };
}

function alwaysCache(): boolean {
  return true;
}
