// ── terminal graphics (Kitty protocol, unicode-placeholder mode) ────────────
//
// Implements the subset of the Kitty graphics protocol that integrates with
// Ink's React-to-terminal render model: "unicode placeholders". An image is
// transmitted once (APC-chunked PNG with `f=100,a=T,U=1,q=2`); subsequent
// draws are a grid of `\u{10EEEE}` characters carrying the image id in the
// 256-color foreground and cell coordinates in combining diacritics. The
// terminal replaces placeholders with pixels on draw, so Ink can lay them
// out as regular <Text> content without any cursor-positioning gymnastics.
//
// iTerm2 inline is intentionally absent — React's useEffect can't re-fire on
// every Ink frame, so any inline-image write gets overwritten on the next
// render and stays gone. See CLAUDE.md for the full rationale.

import supportsTerminalGraphics from "supports-terminal-graphics";
import { SEER_IMAGE_PROTOCOL, FAST_MODE } from "../types.js";
import { detectTerminal } from "./openInTerminal.js";

export type ResolvedProtocol = "kitty-placeholder" | "blocks" | "off";

// ── detection ──────────────────────────────────────────────────────────────

export function detectImageProtocol(): ResolvedProtocol {
  if (FAST_MODE) return "blocks";
  if (SEER_IMAGE_PROTOCOL === "off") return "off";
  if (SEER_IMAGE_PROTOCOL === "blocks") return "blocks";
  if (SEER_IMAGE_PROTOCOL === "iterm") return "blocks"; // not implemented — degrade
  if (SEER_IMAGE_PROTOCOL === "kitty") return "kitty-placeholder";

  // auto: inside tmux we can't rely on Kitty passthrough by default.
  if (process.env.TMUX) return "blocks";

  if (supportsTerminalGraphics.stdout.kitty) return "kitty-placeholder";

  // Belt-and-suspenders: detectTerminal() knows about Ghostty and friends
  // even if supports-terminal-graphics missed the env.
  const term = detectTerminal();
  if (term === "ghostty" || term === "kitty" || term === "wezterm") {
    return "kitty-placeholder";
  }
  return "blocks";
}

// ── transmit ───────────────────────────────────────────────────────────────

// Build the APC chunks that upload a PNG payload to the terminal under `id`,
// simultaneously reserving the virtual placement (U=1). Must be followed by
// the placeholder grid (from buildPlaceholderGrid) to actually render.
export function buildKittyTransmit(
  pngBuffer: Buffer,
  id: number,
  cols: number,
  rows: number,
): string {
  const base64 = pngBuffer.toString("base64");
  const CHUNK = 4096; // Kitty spec: ≤4096 base64 chars per chunk
  const out: string[] = [];
  const controlBase = `i=${id},q=2,U=1,f=100,c=${cols},r=${rows}`;

  if (base64.length <= CHUNK) {
    // Single-chunk: combine transmit + place with m=0.
    out.push(`\x1b_G${controlBase},a=T,m=0;${base64}\x1b\\`);
    return out.join("");
  }

  // Multi-chunk: first chunk carries a=T + m=1, subsequent chunks m=1 (or
  // m=0 for the final). Only the first chunk carries the full control set;
  // subsequent chunks only carry `m=` and the id for continuation.
  let offset = 0;
  let first = true;
  while (offset < base64.length) {
    const end = Math.min(offset + CHUNK, base64.length);
    const isLast = end === base64.length;
    const payload = base64.slice(offset, end);
    if (first) {
      out.push(`\x1b_G${controlBase},a=T,m=${isLast ? 0 : 1};${payload}\x1b\\`);
      first = false;
    } else {
      out.push(`\x1b_Gm=${isLast ? 0 : 1};${payload}\x1b\\`);
    }
    offset = end;
  }
  return out.join("");
}

// ── delete ─────────────────────────────────────────────────────────────────

export function buildKittyDelete(id: number): string {
  return `\x1b_Ga=d,d=i,i=${id},q=2\x1b\\`;
}

export function buildKittyDeleteAll(): string {
  return `\x1b_Ga=d,d=A,q=2\x1b\\`;
}

// ── placeholder grid ───────────────────────────────────────────────────────
//
// Each cell emits: SGR-set-foreground-256(id) + U+10EEEE + rowDiacritic +
// colDiacritic. Every row ends with SGR reset so subsequent text doesn't
// inherit the color-encoded id.

const PLACEHOLDER = "\u{10EEEE}";

// Kitty's official rowcolumn-diacritics.txt — 297 combining-mark codepoints
// used to encode placeholder (row, col) position.
const ROWCOL_DIACRITICS: number[] = [
  0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F, 0x0346, 0x034A,
  0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357, 0x035B, 0x0363, 0x0364, 0x0365,
  0x0366, 0x0367, 0x0368, 0x0369, 0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F,
  0x0483, 0x0484, 0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
  0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1, 0x05A8, 0x05A9,
  0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611, 0x0612, 0x0613, 0x0614, 0x0615,
  0x0616, 0x0617, 0x0657, 0x0658, 0x0659, 0x065A, 0x065B, 0x065D, 0x065E, 0x06D6,
  0x06D7, 0x06D8, 0x06D9, 0x06DA, 0x06DB, 0x06DC, 0x06DF, 0x06E0, 0x06E1, 0x06E2,
  0x06E4, 0x06E7, 0x06E8, 0x06EB, 0x06EC, 0x0730, 0x0732, 0x0733, 0x0735, 0x0736,
  0x073A, 0x073D, 0x073F, 0x0740, 0x0741, 0x0743, 0x0745, 0x0747, 0x0749, 0x074A,
  0x07EB, 0x07EC, 0x07ED, 0x07EE, 0x07EF, 0x07F0, 0x07F1, 0x07F3, 0x0816, 0x0817,
  0x0818, 0x0819, 0x081B, 0x081C, 0x081D, 0x081E, 0x081F, 0x0820, 0x0821, 0x0822,
  0x0823, 0x0825, 0x0826, 0x0827, 0x0829, 0x082A, 0x082B, 0x082C, 0x082D, 0x0951,
  0x0953, 0x0954, 0x0F82, 0x0F83, 0x0F86, 0x0F87, 0x135D, 0x135E, 0x135F, 0x17DD,
  0x193A, 0x1A17, 0x1A75, 0x1A76, 0x1A77, 0x1A78, 0x1A79, 0x1A7A, 0x1A7B, 0x1A7C,
  0x1B6B, 0x1B6D, 0x1B6E, 0x1B6F, 0x1B70, 0x1B71, 0x1B72, 0x1B73, 0x1CD0, 0x1CD1,
  0x1CD2, 0x1CDA, 0x1CDB, 0x1CE0, 0x1DC0, 0x1DC1, 0x1DC3, 0x1DC4, 0x1DC5, 0x1DC6,
  0x1DC7, 0x1DC8, 0x1DC9, 0x1DCB, 0x1DCC, 0x1DD1, 0x1DD2, 0x1DD3, 0x1DD4, 0x1DD5,
  0x1DD6, 0x1DD7, 0x1DD8, 0x1DD9, 0x1DDA, 0x1DDB, 0x1DDC, 0x1DDD, 0x1DDE, 0x1DDF,
  0x1DE0, 0x1DE1, 0x1DE2, 0x1DE3, 0x1DE4, 0x1DE5, 0x1DE6, 0x1DFE, 0x20D0, 0x20D1,
  0x20D4, 0x20D5, 0x20D6, 0x20D7, 0x20DB, 0x20DC, 0x20E1, 0x20E7, 0x20E9, 0x20F0,
  0x2CEF, 0x2CF0, 0x2CF1, 0x2DE0, 0x2DE1, 0x2DE2, 0x2DE3, 0x2DE4, 0x2DE5, 0x2DE6,
  0x2DE7, 0x2DE8, 0x2DE9, 0x2DEA, 0x2DEB, 0x2DEC, 0x2DED, 0x2DEE, 0x2DEF, 0x2DF0,
  0x2DF1, 0x2DF2, 0x2DF3, 0x2DF4, 0x2DF5, 0x2DF6, 0x2DF7, 0x2DF8, 0x2DF9, 0x2DFA,
  0x2DFB, 0x2DFC, 0x2DFD, 0x2DFE, 0x2DFF, 0xA66F, 0xA67C, 0xA67D, 0xA6F0, 0xA6F1,
  0xA8E0, 0xA8E1, 0xA8E2, 0xA8E3, 0xA8E4, 0xA8E5, 0xA8E6, 0xA8E7, 0xA8E8, 0xA8E9,
  0xA8EA, 0xA8EB, 0xA8EC, 0xA8ED, 0xA8EE, 0xA8EF, 0xA8F0, 0xA8F1, 0xAAB0, 0xAAB2,
  0xAAB3, 0xAAB7, 0xAAB8, 0xAABE, 0xAABF, 0xAAC1, 0xFE20, 0xFE21, 0xFE22, 0xFE23,
  0xFE24, 0xFE25, 0xFE26, 0x10A0F, 0x10A38, 0x1D185, 0x1D186, 0x1D187, 0x1D188,
  0x1D189, 0x1D1AA, 0x1D1AB, 0x1D1AC, 0x1D1AD, 0x1D242, 0x1D243, 0x1D244,
];

export const MAX_GRID_DIMENSION = ROWCOL_DIACRITICS.length; // 297

function diacritic(index: number): string {
  const cp = ROWCOL_DIACRITICS[Math.min(index, ROWCOL_DIACRITICS.length - 1)];
  return String.fromCodePoint(cp);
}

// Build per-row placeholder strings. Each string renders one logical row of
// the image; caller emits them via one <Text> per row in Preview.tsx, which
// keeps Ink's layout stable and avoids touching the ansi-wrap pipeline.
export function buildPlaceholderGrid(id: number, cols: number, rows: number): string[] {
  const fg = `\x1b[38;5;${id}m`;
  const reset = "\x1b[0m";
  const gridRows: string[] = [];
  for (let r = 0; r < rows; r++) {
    const rowDia = diacritic(r);
    let line = fg;
    for (let c = 0; c < cols; c++) {
      const colDia = diacritic(c);
      line += PLACEHOLDER + rowDia + colDia;
    }
    line += reset;
    gridRows.push(line);
  }
  return gridRows;
}

// ── transmit bookkeeping ───────────────────────────────────────────────────

let _hasTransmittedAny = false;
export function markTransmitted(): void { _hasTransmittedAny = true; }
export function hasTransmittedAny(): boolean { return _hasTransmittedAny; }
