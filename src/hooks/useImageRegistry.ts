// ── image id registry ──────────────────────────────────────────────────────
//
// Tracks which terminal-graphics image ids are currently live. Lifetime is
// bound to the preview cache: when a cached PreviewPayload carrying an image
// is evicted, the cache calls releaseId() and emits the Kitty delete escape.
// Not a React hook despite the `hooks/` folder — kept here for locality with
// usePreviewCache.

// Key format: "path|modTime|size" (no width/height — the same image data is
// shared regardless of placement dimensions).
const keyToId = new Map<string, number>();
const idToKey = new Map<number, string>();

// Kitty placeholder ids use the 256-color foreground space; ids 1..255.
// Id 0 is reserved (no image).
const MIN_ID = 1;
const MAX_ID = 255;
let nextId = MIN_ID;

export interface AssignResult {
  id: number;
  isNew: boolean;
}

export function assignId(key: string): AssignResult {
  const existing = keyToId.get(key);
  if (existing !== undefined) return { id: existing, isNew: false };

  // Find the next unused id, wrapping through 1..255. If all ids are taken,
  // forcibly reclaim the oldest one — the caller's cache eviction path will
  // have released ids in the normal case, so this is a safety net.
  let id = nextId;
  for (let i = 0; i < MAX_ID; i++) {
    if (!idToKey.has(id)) break;
    id = id >= MAX_ID ? MIN_ID : id + 1;
  }
  if (idToKey.has(id)) {
    // Exhausted — evict whichever key was first.
    const victim = idToKey.get(id)!;
    keyToId.delete(victim);
  }
  nextId = id >= MAX_ID ? MIN_ID : id + 1;

  keyToId.set(key, id);
  idToKey.set(id, key);
  return { id, isNew: true };
}

// Called by the preview cache when it evicts a payload carrying image.id.
// Safe to call with an id not in the registry (returns without work).
export function releaseId(id: number): void {
  const key = idToKey.get(id);
  if (key === undefined) return;
  idToKey.delete(id);
  keyToId.delete(key);
}

export function registrySize(): number {
  return idToKey.size;
}
