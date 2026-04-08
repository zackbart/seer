// ── ANSI-aware text utilities ──────────────────────────────────────────────
//
// Shared helpers for working with terminal text that contains ANSI SGR
// (color/style) escape sequences. Used by the preview pipeline to measure,
// slice, wrap, and compute scroll state in a way that matches what the
// terminal actually renders — not what Ink's built-in width measurement
// thinks. The wrap/slice helpers maintain *cumulative* SGR state so that
// stacked attributes (e.g. bold + color) survive across segment boundaries.

// Matches a single ANSI escape sequence (CSI ... final byte).
const ANSI_RE = /\x1b\[[0-9;]*[a-zA-Z]/g;

// ── stripAnsi ───────────────────────────────────────────────────────────────

export function stripAnsi(s: string): string {
  return s.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, "");
}

// ── visualWidth ─────────────────────────────────────────────────────────────
//
// Approximates the terminal column width of a string. Strips ANSI, then walks
// code points and counts East-Asian Wide / Fullwidth / astral-plane emoji as
// width 2. Everything else counts as 1 (control chars collapse to 0). This
// is a wcwidth-lite — not perfect, but good enough for layout budgeting and
// for the wrap fast-path.

function codePointWidth(cp: number): number {
  if (cp === 0) return 0;
  if (cp < 0x20) return 0;              // C0 controls
  if (cp >= 0x7f && cp < 0xa0) return 0; // DEL + C1
  // Combining marks (very narrow subset — most common ranges)
  if (cp >= 0x0300 && cp <= 0x036f) return 0;
  // Astral plane (emoji, supplementary symbols) — treat as width 2
  if (cp >= 0x1f000) return 2;
  // CJK Unified Ideographs
  if (cp >= 0x4e00 && cp <= 0x9fff) return 2;
  // CJK Extension A
  if (cp >= 0x3400 && cp <= 0x4dbf) return 2;
  // Hangul Syllables
  if (cp >= 0xac00 && cp <= 0xd7a3) return 2;
  // Hiragana / Katakana
  if (cp >= 0x3040 && cp <= 0x30ff) return 2;
  // CJK Symbols & Punctuation
  if (cp >= 0x3000 && cp <= 0x303e) return 2;
  // Fullwidth forms
  if (cp >= 0xff00 && cp <= 0xff60) return 2;
  if (cp >= 0xffe0 && cp <= 0xffe6) return 2;
  // Misc symbols & pictographs fallback (BMP emoji block)
  if (cp >= 0x2600 && cp <= 0x27bf) return 2;
  return 1;
}

export function visualWidth(s: string): number {
  const plain = stripAnsi(s);
  let w = 0;
  for (const ch of plain) {
    w += codePointWidth(ch.codePointAt(0) ?? 0);
  }
  return w;
}

// ── SGR state accumulator ──────────────────────────────────────────────────
//
// SGR sequences stack (bold + color + background + ...). To replay state at
// an arbitrary cut point we keep an array of active sequences and clear it
// on \x1b[0m or bare \x1b[m. This is intentionally simple — we don't parse
// the numeric parameters to dedupe overlapping attributes; in practice the
// stream from marked-terminal/shiki/chalk doesn't accumulate duplicates.

function isReset(seq: string): boolean {
  return seq === "\x1b[0m" || seq === "\x1b[m";
}

function stateToString(state: string[]): string {
  return state.join("");
}

// ── ansiSlice ──────────────────────────────────────────────────────────────
//
// Visual-column slice [startCol, endCol) of an ANSI-decorated string.
// Preserves *cumulative* SGR state across the slice — the result is prefixed
// with every active SGR sequence that was in effect at startCol, and closed
// with \x1b[0m if state is non-empty.

export function ansiSlice(str: string, startCol: number, endCol: number): string {
  if (startCol >= endCol || !str) return "";

  // Fast path: no ANSI at all.
  if (!str.includes("\x1b[")) {
    // Still needs visual-width-aware slicing, but for ASCII the naive slice
    // is correct. For wide chars fall through to the walker.
    let hasWide = false;
    for (const ch of str) {
      if (codePointWidth(ch.codePointAt(0) ?? 0) > 1) { hasWide = true; break; }
    }
    if (!hasWide) return str.slice(startCol, endCol);
  }

  const activeState: string[] = [];
  let result = "";
  let col = 0;
  let i = 0;
  let emitting = false;

  while (i < str.length && col < endCol) {
    ANSI_RE.lastIndex = i;
    const match = ANSI_RE.exec(str);

    if (match && match.index === i) {
      const seq = match[0];
      if (isReset(seq)) {
        activeState.length = 0;
        if (emitting) result += seq;
      } else {
        activeState.push(seq);
        if (emitting) result += seq;
      }
      i += seq.length;
      continue;
    }

    // Visible code point
    const cp = str.codePointAt(i) ?? 0;
    const w = codePointWidth(cp);
    const chLen = cp > 0xffff ? 2 : 1; // surrogate pair width in UTF-16 units

    if (col >= startCol) {
      if (!emitting) {
        if (activeState.length > 0) result += stateToString(activeState);
        emitting = true;
      }
      // Only include the char if it fits entirely in the remaining range.
      if (col + w <= endCol) {
        result += str.substr(i, chLen);
      }
    }

    col += w;
    i += chLen;
  }

  if (emitting && activeState.length > 0) {
    result += "\x1b[0m";
  }

  return result;
}

// ── wrapAnsiLine ───────────────────────────────────────────────────────────
//
// Word-wrap a single logical line (no internal \n) into visual chunks that
// each fit within `width` columns. Prefers breaking at the last space; falls
// back to a hard char-break when a single token exceeds `width`. Each wrapped
// chunk carries forward the cumulative SGR state from previous chunks.

export function wrapAnsiLine(line: string, width: number): string[] {
  if (width <= 0) return [line];
  if (line === "") return [""];

  // Fast path: already fits.
  if (visualWidth(line) <= width) return [line];

  const out: string[] = [];
  const activeState: string[] = [];

  // Per-chunk working buffer.
  let buf = "";           // accumulated text (with ANSI) for current chunk
  let bufVisual = 0;      // visible column count of current chunk
  let lastSpaceBuf = -1;  // buf length at last space (split point)
  let lastSpaceVisual = 0; // bufVisual at last space
  let lastSpaceStateLen = 0; // activeState length snapshot at last space

  const flushChunk = (chunk: string) => {
    if (activeState.length > 0) {
      out.push(stateToString(activeState) + chunk + "\x1b[0m");
    } else {
      out.push(chunk);
    }
  };

  let i = 0;
  while (i < line.length) {
    ANSI_RE.lastIndex = i;
    const match = ANSI_RE.exec(line);

    if (match && match.index === i) {
      const seq = match[0];
      if (isReset(seq)) {
        activeState.length = 0;
      } else {
        activeState.push(seq);
      }
      buf += seq;
      i += seq.length;
      continue;
    }

    const cp = line.codePointAt(i) ?? 0;
    const w = codePointWidth(cp);
    const chLen = cp > 0xffff ? 2 : 1;
    const ch = line.substr(i, chLen);

    // If this char would overflow, cut the chunk.
    if (bufVisual + w > width) {
      if (lastSpaceBuf >= 0) {
        // Break at last space — drop the space itself.
        const head = buf.slice(0, lastSpaceBuf);
        const tail = buf.slice(lastSpaceBuf + 1); // +1 skips the space
        flushChunk(head);
        // Carry the tail into a fresh buffer; its visual width is what
        // followed the break point.
        buf = tail;
        bufVisual = bufVisual - lastSpaceVisual - 1; // -1 for dropped space
        // State at break was activeState truncated to lastSpaceStateLen, but
        // any SGR sequences in `tail` were appended after; our activeState
        // already reflects the *current* state, which is correct for the
        // next chunk's prefix.
        lastSpaceBuf = -1;
        lastSpaceVisual = 0;
        lastSpaceStateLen = 0;
      } else {
        // Hard break — no space to break on.
        flushChunk(buf);
        buf = "";
        bufVisual = 0;
        lastSpaceBuf = -1;
        lastSpaceVisual = 0;
        lastSpaceStateLen = 0;
      }
    }

    // Remember space positions as split points.
    if (ch === " ") {
      lastSpaceBuf = buf.length;
      lastSpaceVisual = bufVisual;
      lastSpaceStateLen = activeState.length;
    }

    buf += ch;
    bufVisual += w;
    i += chLen;
  }

  if (buf.length > 0 || out.length === 0) {
    flushChunk(buf);
  }

  // Silence unused-var warning for lastSpaceStateLen (reserved for future
  // optimization where we truncate activeState at break points).
  void lastSpaceStateLen;

  return out;
}

// ── wrapAnsiText ───────────────────────────────────────────────────────────

export function wrapAnsiText(text: string, width: number): string[] {
  const rawLines = text.split("\n");
  const out: string[] = [];
  for (const line of rawLines) {
    const wrapped = wrapAnsiLine(line, width);
    for (const w of wrapped) out.push(w);
  }
  return out;
}

// ── computeWrappedBody ─────────────────────────────────────────────────────
//
// Single source of truth for preview-body scroll math. Given the preview
// text, the inner pane width, body height, and requested scroll offset,
// returns the wrapped-line array and all derived layout values. Used by
// Preview.tsx (render), App.tsx mouse-wheel handler (scroll clamping), and
// App.tsx click-drag selection helper (row coordinate mapping).

export interface WrappedBody {
  wrappedLines: string[];
  maxOffset: number;
  offset: number;
  scrollRow: number;      // 1 if a "↑ N more" top indicator is shown, else 0
  scrollRowBot: number;   // 1 if a "↓ N more" bottom indicator is shown, else 0
  contentH: number;       // rows available for wrapped content (bodyH - indicators)
  visibleLines: string[]; // the actual content slice, not including indicators
}

export function computeWrappedBody(
  previewText: string,
  innerW: number,
  bodyH: number,
  previewOffset: number,
): WrappedBody {
  const width = Math.max(1, innerW);
  const height = Math.max(1, bodyH);
  const wrappedLines = wrapAnsiText(previewText, width);
  const total = wrappedLines.length;

  // Short-circuit: everything fits, no indicators needed.
  if (total <= height) {
    const offset0 = 0;
    return {
      wrappedLines,
      maxOffset: 0,
      offset: offset0,
      scrollRow: 0,
      scrollRowBot: 0,
      contentH: height,
      visibleLines: wrappedLines.slice(0, height),
    };
  }

  // Bounded fixed-point: the system has 4 coupled variables (top, bot, offset,
  // maxOffset) and each iteration can flip at most one. Stabilizes in ≤ 4
  // iterations for every non-pathological input.
  let top = previewOffset > 0 ? 1 : 0;
  let bot = 0;
  let contentH = Math.max(1, height - top - bot);
  let maxOffset = Math.max(0, total - contentH);
  let offset = Math.min(Math.max(0, previewOffset), maxOffset);

  for (let iter = 0; iter < 4; iter++) {
    const newTop = offset > 0 ? 1 : 0;
    const newContentHGuess = Math.max(1, height - newTop - bot);
    const newBot = offset + newContentHGuess < total ? 1 : 0;
    const newContentH = Math.max(1, height - newTop - newBot);
    const newMaxOffset = Math.max(0, total - newContentH);
    const newOffset = Math.min(Math.max(0, offset), newMaxOffset);

    if (newTop === top && newBot === bot && newOffset === offset && newMaxOffset === maxOffset) {
      top = newTop;
      bot = newBot;
      offset = newOffset;
      maxOffset = newMaxOffset;
      contentH = newContentH;
      break;
    }
    top = newTop;
    bot = newBot;
    offset = newOffset;
    maxOffset = newMaxOffset;
    contentH = newContentH;
  }

  const visibleLines = wrappedLines.slice(offset, offset + contentH);

  return {
    wrappedLines,
    maxOffset,
    offset,
    scrollRow: top,
    scrollRowBot: bot,
    contentH,
    visibleLines,
  };
}

// ── truncateByWidth ─────────────────────────────────────────────────────────
//
// Truncate a string to at most `maxCols` visual columns. Walks code points
// (not UTF-16 units) so multi-byte characters aren't split mid-surrogate, and
// uses `visualWidth` semantics so wide characters count as 2.

export function truncateByWidth(str: string, maxCols: number): string {
  if (maxCols <= 0) return "";
  let col = 0;
  let out = "";
  for (const ch of str) {
    const w = codePointWidth(ch.codePointAt(0) ?? 0);
    if (col + w > maxCols) break;
    out += ch;
    col += w;
  }
  return out;
}
