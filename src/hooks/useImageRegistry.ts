// ── image id registry ──────────────────────────────────────────────────────
//
// Tracks which terminal-graphics image ids are currently live. Lifetime is
// bound to the preview cache: when a cached PreviewPayload carrying an image
// is evicted, the cache calls releaseId() and emits the Kitty delete escape.
// Not a React hook despite the `hooks/` folder — kept here for locality with
// usePreviewCache.
//
// Also tracks the last-transmitted (cols, rows) per id. The Kitty virtual
// placement is fixed at transmit time — if the grid geometry changes (terminal
// resize, pane resize), we must re-transmit so placeholders beyond the old
// c×r don't render as raw codepoints.

// Key format: "path|modTime|size" (no width/height — the same image data is
// shared regardless of placement dimensions).
const keyToId = new Map<string, number>();
const idToKey = new Map<number, string>();
const idToCR = new Map<number, { cols: number; rows: number }>();

// Kitty placeholder ids use the 256-color foreground space; ids 1..255.
// Id 0 is reserved (no image).
const MIN_ID = 1;
const MAX_ID = 255;
let nextId = MIN_ID;

export interface AssignResult {
  id: number;
  // true when the image data must be (re)transmitted to the terminal — either
  // a brand-new id or the previously-stored (cols, rows) differ from the
  // requested geometry.
  needsTransmit: boolean;
  // true only on first assignment for this key. When false and needsTransmit
  // is true, the caller should emit a Kitty delete before the transmit so
  // the virtual placement is reset cleanly (terminal behavior of `a=T` reusing
  // an existing id is not spec-mandated; delete-first is belt-and-suspenders).
  wasNewId: boolean;
}

// Clears all registry maps for a given id. Used by releaseId (triggered by
// cache eviction) and by the victim branch of assignId (id exhaustion).
function forgetId(id: number): void {
  const key = idToKey.get(id);
  if (key !== undefined) keyToId.delete(key);
  idToKey.delete(id);
  idToCR.delete(id);
}

export function assignId(key: string, cols: number, rows: number): AssignResult {
  const existing = keyToId.get(key);
  if (existing !== undefined) {
    const prevCR = idToCR.get(existing);
    const needsTransmit = !prevCR || prevCR.cols !== cols || prevCR.rows !== rows;
    if (needsTransmit) idToCR.set(existing, { cols, rows });
    return { id: existing, needsTransmit, wasNewId: false };
  }

  // Find the next unused id, wrapping through 1..255. If all ids are taken,
  // forcibly reclaim the oldest one — the caller's cache eviction path will
  // have released ids in the normal case, so this is a safety net.
  let id = nextId;
  for (let i = 0; i < MAX_ID; i++) {
    if (!idToKey.has(id)) break;
    id = id >= MAX_ID ? MIN_ID : id + 1;
  }
  if (idToKey.has(id)) {
    // Exhausted — evict whichever key was first. Clear all three maps via
    // forgetId so idToCR doesn't leak stale geometry onto a reused id.
    forgetId(id);
  }
  nextId = id >= MAX_ID ? MIN_ID : id + 1;

  keyToId.set(key, id);
  idToKey.set(id, key);
  idToCR.set(id, { cols, rows });
  return { id, needsTransmit: true, wasNewId: true };
}

// Called by the preview cache when it evicts a payload carrying image.id.
// Safe to call with an id not in the registry (returns without work).
export function releaseId(id: number): void {
  if (!idToKey.has(id)) return;
  forgetId(id);
}

export function registrySize(): number {
  return idToKey.size;
}
