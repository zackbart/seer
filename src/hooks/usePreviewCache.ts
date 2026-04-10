import { PREVIEW_CACHE_MAX, PreviewPayload } from "../types.js";

// Native LRU via Map insertion order: delete-then-set moves a key to the end.
// Evict the oldest (first inserted) key when we exceed the max.
const cache = new Map<string, PreviewPayload>();

export function cacheGet(key: string): PreviewPayload | undefined {
  const value = cache.get(key);
  if (value === undefined) return undefined;
  // Promote on hit — move key to end so it's freshest.
  cache.delete(key);
  cache.set(key, value);
  return value;
}

export function cacheSet(key: string, value: PreviewPayload): void {
  if (cache.has(key)) cache.delete(key);
  cache.set(key, value);
  while (cache.size > PREVIEW_CACHE_MAX) {
    const oldest = cache.keys().next().value;
    if (oldest === undefined) break;
    cache.delete(oldest);
  }
}

export function cacheClear(): void {
  cache.clear();
}
