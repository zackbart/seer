import { PREVIEW_CACHE_MAX } from "../types.js";

const cache = new Map<string, string>();
const cacheOrder: string[] = [];

export function cacheGet(key: string): string | undefined {
  return cache.get(key);
}

export function cacheSet(key: string, value: string): void {
  if (!cache.has(key)) {
    cacheOrder.push(key);
  }
  cache.set(key, value);
  while (cacheOrder.length > PREVIEW_CACHE_MAX) {
    const oldest = cacheOrder.shift()!;
    cache.delete(oldest);
  }
}

export function cacheClear(): void {
  cache.clear();
  cacheOrder.length = 0;
}
