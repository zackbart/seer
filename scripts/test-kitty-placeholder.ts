#!/usr/bin/env bun
// ── test-kitty-placeholder ──────────────────────────────────────────────────
//
// Manual smoke test for the Kitty graphics unicode-placeholder path. Run in
// Ghostty/Kitty/WezTerm:
//
//     bun run scripts/test-kitty-placeholder.ts path/to/image.png
//
// Emits the transmit APC, the placeholder grid (~60 cols wide, aspect-preserved),
// sleeps 3s, then writes the delete-all escape. Image should render for 3s
// then disappear.

import fs from "fs";
import path from "path";
import { Jimp } from "jimp";
import { imageSize } from "image-size";
import {
  buildKittyTransmit,
  buildPlaceholderGrid,
  buildKittyDeleteAll,
} from "../src/utils/termGraphics.js";

const filePath = process.argv[2];
if (!filePath) {
  console.error("usage: test-kitty-placeholder <image-path>");
  process.exit(1);
}

const buf = fs.readFileSync(filePath);
const ext = path.extname(filePath).toLowerCase();

let pngBuffer: Buffer;
let imageW = 0;
let imageH = 0;

try {
  const dims = imageSize(buf);
  imageW = dims.width ?? 0;
  imageH = dims.height ?? 0;
} catch {}

if (ext === ".png") {
  pngBuffer = buf;
} else {
  const img = await Jimp.read(buf);
  pngBuffer = await img.getBuffer("image/png");
  if (!imageW) { imageW = img.width; imageH = img.height; }
}

const MAX_COLS = 60;
const MAX_ROWS = 30;
function fit(w: number, h: number) {
  if (!w || !h) return { cols: MAX_COLS, rows: MAX_ROWS };
  const rowsByWidth = Math.round((MAX_COLS * h) / (w * 2));
  const colsByHeight = Math.round((MAX_ROWS * 2 * w) / h);
  if (rowsByWidth <= MAX_ROWS) return { cols: MAX_COLS, rows: Math.max(1, rowsByWidth) };
  return { cols: Math.max(1, colsByHeight), rows: MAX_ROWS };
}
const { cols, rows } = fit(imageW, imageH);

const id = 42;
process.stdout.write(buildKittyTransmit(pngBuffer, id, cols, rows));
for (const line of buildPlaceholderGrid(id, cols, rows)) {
  process.stdout.write(line + "\n");
}

await new Promise((r) => setTimeout(r, 3000));
process.stdout.write(buildKittyDeleteAll());
console.log("\n(deleted)");
