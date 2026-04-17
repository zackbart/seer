// ── layout ──────────────────────────────────────────────────────────────────
//
// Single source of truth for pane dimensions. Used by App.tsx (render +
// mouse handlers) and Preview.tsx (out-of-band image grid positioning).

export function layoutDimensions(width: number, height: number, paneOffset: number) {
  const base = Math.max(26, Math.floor(width / 4));
  let leftW = Math.max(16, Math.min(Math.floor(width / 2), base + paneOffset));
  let rightW = width - leftW - 1;
  if (rightW < 20) {
    rightW = 20;
    leftW = width - rightW - 1;
  }
  // Guard against tiny terminals: the narrow-fallback branch can drive leftW
  // or rightW negative if width is small enough that 20 cols don't fit. Clamp
  // both to at least 1 so Ink doesn't get confused.
  leftW = Math.max(1, leftW);
  rightW = Math.max(1, rightW);
  // TopBar=1 row + BottomBar=2 rows = 3 rows of chrome
  const bodyH = Math.max(4, height - 3);
  return { leftW, rightW, bodyH };
}
