// ── image preview ───────────────────────────────────────────────────────────
//
// Rasterizes an image to Unicode half-block characters (▄) with 24-bit
// truecolor via terminal-image + jimp (pure JS, no native deps — survives
// `bun build --compile`). Forces ANSI output (preferNativeRender: false) so
// the result is a plain string that flows through the existing Ink-based
// Preview render path. Kitty/iTerm2 inline protocols would be pixel-perfect
// but require out-of-band stdout writes and fight Ink's rerender cycle —
// deferred to a follow-up.

import terminalImage from "terminal-image";
import { imageSize } from "image-size";
import path from "path";
import { PreviewPayload } from "../types.js";
import { humanSize } from "../utils/humanSize.js";

// Reject absurdly large declared dimensions before handing bytes to jimp —
// a crafted small-file PNG can claim 50k×50k pixels and allocate gigabytes.
const MAX_PIXELS = 100 * 1_000_000; // 100 megapixels

// Preview.tsx reserves 4 rows inside its border (2 border + 1 header +
// 1 divider). The `height` we receive is the outer window body height.
const PREVIEW_CHROME_ROWS = 4;

function errorPayload(message: string): PreviewPayload {
  return { text: message, lineCount: 0, tokenEstimate: 0, truncated: false };
}

function okPayload(text: string): PreviewPayload {
  return { text, lineCount: 0, tokenEstimate: 0, truncated: false };
}

export async function renderImagePreview(
  buffer: Buffer,
  filePath: string,
  width: number,
  height: number,
  signal?: AbortSignal,
): Promise<PreviewPayload> {
  if (signal?.aborted) return okPayload("");

  const base = path.basename(filePath);

  // Header sniff: reject oversized dimensions without decoding pixels.
  try {
    const dims = imageSize(buffer);
    if (dims.width && dims.height) {
      const pixels = dims.width * dims.height;
      if (pixels > MAX_PIXELS) {
        return errorPayload(
          `${base}\nsize: ${humanSize(buffer.byteLength)}\ndimensions: ${dims.width}×${dims.height}\n\n(image too large to render — limit ${MAX_PIXELS / 1_000_000}MP)`,
        );
      }
    }
  } catch {
    // image-size didn't recognize the format; let jimp try anyway.
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
    // terminal-image sometimes appends a trailing newline; trim so the scroll
    // indicator math doesn't think there's an extra empty row at the bottom.
    return okPayload(rendered.replace(/\n+$/, ""));
  } catch (err) {
    if (signal?.aborted) return okPayload("");
    const msg = err instanceof Error ? err.message : String(err);
    return errorPayload(
      `${base}\nsize: ${humanSize(buffer.byteLength)}\n\n(unable to decode image: ${msg})`,
    );
  }
}
