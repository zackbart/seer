// ── image preview ───────────────────────────────────────────────────────────
//
// Two rendering paths:
//
//   1. kitty-placeholder — pixel-perfect via Kitty graphics protocol in
//      unicode-placeholder mode. Transmits the PNG once, then renders a grid
//      of `\u{10EEEE}` cells that Ink lays out as regular <Text>. The
//      terminal substitutes pixels on draw. Works on Ghostty, Kitty ≥0.28,
//      WezTerm.
//   2. blocks — Unicode half-block rasterization via terminal-image + jimp.
//      Universal fallback. Looks surprisingly good; what you get on
//      Terminal.app, VS Code, iTerm2, and unknown terminals.
//
// Protocol selection lives in `src/utils/termGraphics.ts` (`detectImageProtocol`).
// `SEER_IMAGE_PROTOCOL` env overrides auto-detect. `SEER_FAST_MODE=1` forces
// a size-only placeholder.

import terminalImage from "terminal-image";
import { Jimp } from "jimp";
import { imageSize } from "image-size";
import path from "path";
import { PreviewPayload } from "../types.js";
import { humanSize } from "../utils/humanSize.js";
import {
  detectImageProtocol,
  buildKittyTransmit,
  buildKittyDelete,
  buildPlaceholderGrid,
  MAX_GRID_DIMENSION,
} from "../utils/termGraphics.js";
import { assignId } from "../hooks/useImageRegistry.js";

// ── transcoded-PNG byte cache ──────────────────────────────────────────────
//
// Non-PNG sources (JPEG/GIF/WebP) go through Jimp.read + getBuffer("image/png")
// — 100–500ms on the main thread. The preview cache is keyed on path+size+
// width+height, so a pane or terminal resize invalidates it and re-runs the
// whole pipeline. This module-level cache remembers the transcoded PNG bytes
// independently of pane dimensions, so resize skips the Jimp pass on known
// inputs. PNG sources already bypass Jimp (see below) and therefore never
// populate this cache.
//
// Bounded LRU via Map insertion order. 8 entries is enough for common
// workflows (screenshots folder, small batch of JPEGs).
const PNG_CACHE_MAX = 8;
interface PngCacheEntry { png: Buffer; width: number; height: number; }
const pngCache = new Map<string, PngCacheEntry>();

function pngCacheGet(key: string): PngCacheEntry | undefined {
  const v = pngCache.get(key);
  if (v === undefined) return undefined;
  // Promote on hit.
  pngCache.delete(key);
  pngCache.set(key, v);
  return v;
}

function pngCacheSet(key: string, entry: PngCacheEntry): void {
  if (pngCache.has(key)) pngCache.delete(key);
  pngCache.set(key, entry);
  while (pngCache.size > PNG_CACHE_MAX) {
    const oldest = pngCache.keys().next().value;
    if (oldest === undefined) break;
    pngCache.delete(oldest);
  }
}

// Reject absurdly large declared dimensions before decoding — a crafted small
// PNG can claim 50k×50k pixels and OOM jimp.
const MAX_PIXELS = 100 * 1_000_000;

// Preview.tsx reserves 4 rows inside its border (2 border + 1 header +
// 1 divider). The incoming `height` is the outer window body height.
const PREVIEW_CHROME_ROWS = 4;

function errorPayload(message: string): PreviewPayload {
  return { text: message, lineCount: 0, tokenEstimate: 0, truncated: false };
}

function okPayload(text: string): PreviewPayload {
  return { text, lineCount: 0, tokenEstimate: 0, truncated: false };
}

// ── sizing ─────────────────────────────────────────────────────────────────

// Compute an aspect-preserved cell rectangle that fits the preview pane.
// Cells are ~2:1 tall:wide, so an image with aspect W:H needs cells
// satisfying `cellCols / (cellRows * 2) ≈ W / H`.
function fitToCells(
  imageW: number,
  imageH: number,
  maxCols: number,
  maxRows: number,
): { cols: number; rows: number } {
  if (imageW <= 0 || imageH <= 0) return { cols: maxCols, rows: maxRows };
  // Target cells assuming width fits exactly.
  const rowsByWidth = Math.round((maxCols * imageH) / (imageW * 2));
  const colsByHeight = Math.round((maxRows * 2 * imageW) / imageH);

  let cols: number;
  let rows: number;
  if (rowsByWidth <= maxRows) {
    cols = maxCols;
    rows = Math.max(1, rowsByWidth);
  } else {
    cols = Math.max(1, colsByHeight);
    rows = maxRows;
  }
  return { cols, rows };
}

// ── kitty path ─────────────────────────────────────────────────────────────

async function renderKittyPlaceholder(
  buffer: Buffer,
  filePath: string,
  modTimeMs: number,
  size: number,
  usableCols: number,
  usableRows: number,
  signal?: AbortSignal,
): Promise<PreviewPayload | null> {
  const ext = path.extname(filePath).toLowerCase();

  let imageW = 0;
  let imageH = 0;
  try {
    const dims = imageSize(buffer);
    if (dims.width && dims.height) {
      imageW = dims.width;
      imageH = dims.height;
      if (imageW * imageH > MAX_PIXELS) return null;
    }
  } catch {
    // dimensions unknown — we'll still try jimp decode below
  }

  if (signal?.aborted) return okPayload("");

  // Transcode to PNG if the file isn't already PNG. Kitty `f=100` is PNG-only.
  // Cache transcoded bytes across pane/terminal resizes — the preview cache
  // key includes width/height so resize invalidates it, but the PNG bytes
  // themselves don't depend on placement geometry.
  const fileKey = `${filePath}|${modTimeMs}|${size}`;
  let pngBuffer: Buffer;
  if (ext === ".png") {
    pngBuffer = buffer;
  } else {
    const cached = pngCacheGet(fileKey);
    if (cached) {
      pngBuffer = cached.png;
      if (imageW === 0 || imageH === 0) {
        imageW = cached.width;
        imageH = cached.height;
      }
    } else {
      const img = await Jimp.read(buffer);
      if (signal?.aborted) return okPayload("");
      pngBuffer = await img.getBuffer("image/png");
      if (imageW === 0 || imageH === 0) {
        imageW = img.width;
        imageH = img.height;
        if (imageW * imageH > MAX_PIXELS) return null;
      }
      pngCacheSet(fileKey, { png: pngBuffer, width: img.width, height: img.height });
    }
  }

  if (signal?.aborted) return okPayload("");

  // Clamp to the diacritic table size — placeholder coords are encoded in a
  // 297-entry set, so images wider or taller than that simply can't place.
  // In practice preview panes never exceed this.
  const maxCols = Math.min(usableCols, MAX_GRID_DIMENSION);
  const maxRows = Math.min(usableRows, MAX_GRID_DIMENSION);
  const { cols, rows } = fitToCells(imageW, imageH, maxCols, maxRows);

  const { id, needsTransmit, wasNewId } = assignId(fileKey, cols, rows);

  const gridRows = buildPlaceholderGrid(id, cols, rows);
  // On geometry change for an existing id, delete-first then re-transmit.
  // Kitty's `a=T` reusing an id replaces the pixel data, but whether it also
  // updates the virtual placement (c, r) is terminal-defined. Delete-first
  // guarantees a clean reset and costs ~20 bytes on the wire.
  let transmitEscape: string | undefined;
  if (needsTransmit) {
    const transmit = buildKittyTransmit(pngBuffer, id, cols, rows);
    transmitEscape = wasNewId ? transmit : buildKittyDelete(id) + transmit;
  }

  return {
    text: "",
    lineCount: 0,
    tokenEstimate: 0,
    truncated: false,
    image: { protocol: "kitty-placeholder", id, cols, rows, gridRows },
    transmitEscape,
  };
}

// ── blocks path ────────────────────────────────────────────────────────────

async function renderBlocks(
  buffer: Buffer,
  filePath: string,
  width: number,
  height: number,
  signal?: AbortSignal,
): Promise<PreviewPayload> {
  if (signal?.aborted) return okPayload("");

  const base = path.basename(filePath);

  // Dimension sniff — same rationale as Kitty path.
  try {
    const dims = imageSize(buffer);
    if (dims.width && dims.height) {
      if (dims.width * dims.height > MAX_PIXELS) {
        return errorPayload(
          `${base}\nsize: ${humanSize(buffer.byteLength)}\ndimensions: ${dims.width}×${dims.height}\n\n(image too large to render — limit ${MAX_PIXELS / 1_000_000}MP)`,
        );
      }
    }
  } catch {
    // Let terminal-image try anyway.
  }

  if (signal?.aborted) return okPayload("");

  const usableHeight = Math.max(1, height - PREVIEW_CHROME_ROWS);
  const usableWidth = Math.max(1, width);

  try {
    const rendered = await terminalImage.buffer(buffer, {
      width: usableWidth,
      height: usableHeight,
      preserveAspectRatio: true,
      preferNativeRender: false,
    });
    if (signal?.aborted) return okPayload("");
    return okPayload(rendered.replace(/\n+$/, ""));
  } catch (err) {
    if (signal?.aborted) return okPayload("");
    const msg = err instanceof Error ? err.message : String(err);
    return errorPayload(
      `${base}\nsize: ${humanSize(buffer.byteLength)}\n\n(unable to decode image: ${msg})`,
    );
  }
}

// ── entry point ────────────────────────────────────────────────────────────

export async function renderImagePreview(
  buffer: Buffer,
  filePath: string,
  modTimeMs: number,
  size: number,
  width: number,
  height: number,
  signal?: AbortSignal,
): Promise<PreviewPayload> {
  if (signal?.aborted) return okPayload("");

  const protocol = detectImageProtocol();

  if (protocol === "off") {
    return errorPayload(
      `${path.basename(filePath)}\nsize: ${humanSize(size)}\n\n(image preview disabled — unset SEER_IMAGE_PROTOCOL to enable)`,
    );
  }

  if (protocol === "kitty-placeholder") {
    const usableCols = Math.max(1, width);
    const usableRows = Math.max(1, height - PREVIEW_CHROME_ROWS);
    const payload = await renderKittyPlaceholder(
      buffer,
      filePath,
      modTimeMs,
      size,
      usableCols,
      usableRows,
      signal,
    );
    if (payload) return payload;
    // Pixel-cap exceeded or jimp failure — fall through to blocks.
  }

  return renderBlocks(buffer, filePath, width, height, signal);
}
