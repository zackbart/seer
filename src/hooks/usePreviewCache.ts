import { PREVIEW_CACHE_MAX, PreviewPayload } from "../types.js";
import { releaseId } from "./useImageRegistry.js";
import { buildKittyDelete } from "../utils/termGraphics.js";

// Native LRU via Map insertion order: delete-then-set moves a key to the end.
// Evict the oldest (first inserted) key when we exceed the max.
const cache = new Map<string, PreviewPayload>();

// When a cached payload carries a live terminal-graphics image id, evicting
// it must also free the id in the registry and tell the terminal to drop the
// pixel data. Called from all three eviction paths (overwrite, LRU-shift,
// clear).
function evictPayload(payload: PreviewPayload): void {
  const id = payload.image?.id;
  if (id === undefined) return;
  releaseId(id);
  try {
    process.stdout.write(buildKittyDelete(id));
  } catch {
    // stdout may be closed during shutdown — swallow.
  }
}

export function cacheGet(key: string): PreviewPayload | undefined {
  const value = cache.get(key);
  if (value === undefined) return undefined;
  // Promote on hit — move key to end so it's freshest.
  cache.delete(key);
  cache.set(key, value);
  return value;
}

export function cacheSet(key: string, value: PreviewPayload): void {
  if (cache.has(key)) {
    // Overwriting a prior entry: evict its image if present. Skip when the
    // replacement payload carries the SAME id (re-caching the same image).
    const prior = cache.get(key);
    if (prior && prior.image?.id !== value.image?.id) {
      evictPayload(prior);
    }
    cache.delete(key);
  }
  cache.set(key, value);
  while (cache.size > PREVIEW_CACHE_MAX) {
    const oldestKey = cache.keys().next().value;
    if (oldestKey === undefined) break;
    const oldestVal = cache.get(oldestKey);
    cache.delete(oldestKey);
    if (oldestVal) evictPayload(oldestVal);
  }
}

export function cacheClear(): void {
  for (const payload of cache.values()) evictPayload(payload);
  cache.clear();
}
