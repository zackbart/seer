import fsp from "fs/promises";
import path from "path";
import { MAX_PREVIEW_BYTES, PreviewPayload } from "../types.js";
import { imageExts, archiveExts } from "../theme.js";
import { isLikelyBinary } from "../utils/fs.js";
import { humanSize } from "../utils/humanSize.js";
import { buildDirPreview } from "./directory.js";
import { highlightCode } from "./code.js";
import { renderMarkdown } from "./markdown.js";
import { renderJSONPreview } from "./json.js";
import { renderCSVPreview } from "./csv.js";
import { buildHexPreview } from "./hex.js";
import { buildArchivePreview } from "./archive.js";
import { renderMermaid } from "./mermaid.js";

// ── payload helpers ─────────────────────────────────────────────────────────

function emptyMetrics(text: string): PreviewPayload {
  return { text, lineCount: 0, tokenEstimate: 0, truncated: false };
}

function withMetrics(text: string, source: string, truncated: boolean): PreviewPayload {
  const lineCount = source.length === 0 ? 0 : source.split("\n").length;
  const tokenEstimate = Math.ceil(source.trim().length / 4);
  return { text, lineCount, tokenEstimate, truncated };
}

// ── main entry ──────────────────────────────────────────────────────────────

export async function buildPreview(
  filePath: string,
  width: number,
  _height: number,
): Promise<PreviewPayload> {
  const stat = await fsp.stat(filePath);

  if (stat.isDirectory()) {
    return emptyMetrics(await buildDirPreview(filePath));
  }

  const ext = path.extname(filePath).toLowerCase();

  // Image files — dropped for v1
  if (imageExts.has(ext)) {
    return emptyMetrics(
      `image file: ${path.basename(filePath)}\nsize: ${humanSize(stat.size)}\n\n(image preview not yet available in TS version)`,
    );
  }

  // Archive files
  if (archiveExts.has(ext)) {
    return emptyMetrics(await buildArchivePreview(filePath));
  }

  // Read file content
  const fd = await fsp.open(filePath, "r");
  try {
    const buf = Buffer.alloc(MAX_PREVIEW_BYTES);
    const { bytesRead } = await fd.read(buf, 0, MAX_PREVIEW_BYTES, 0);
    const data = buf.subarray(0, bytesRead);

    // Binary detection
    if (isLikelyBinary(data)) {
      return emptyMetrics(buildHexPreview(data, path.basename(filePath), stat.size, stat.mtime));
    }

    let text = data.toString("utf-8");
    // Check for invalid UTF-8 (replacement character indicates issues)
    if (text.includes("\uFFFD") && bytesRead > 0) {
      return emptyMetrics(buildHexPreview(data, path.basename(filePath), stat.size, stat.mtime));
    }

    // Normalize line endings and expand tabs to spaces.
    // Tabs must be expanded because Ink's width measurement treats \t as 0-width
    // while the terminal renders them at 8-column tab stops, causing lines to
    // overflow the preview pane.
    text = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
    text = text.replace(/\t/g, "    ");

    const truncated = bytesRead === MAX_PREVIEW_BYTES;
    // Snapshot metrics from the normalized source *before* rendering — so
    // "lines" matches the user's file, not the rendered markdown/highlighted
    // output. Tab-expansion inflates token estimates slightly; that's fine
    // for a char/4 approximation.
    const metricSource = text;

    switch (ext) {
      case ".md":
      case ".markdown":
      case ".mdx": {
        const rendered = renderMarkdown(text, width, truncated);
        if (rendered) return withMetrics(rendered, metricSource, truncated);
        const hl = await highlightCode(filePath, text);
        if (hl !== text) {
          const out = truncated ? hl + "\n\n... preview truncated ..." : hl;
          return withMetrics(out, metricSource, truncated);
        }
        return withMetrics(text, metricSource, truncated);
      }
      case ".mmd":
      case ".mermaid":
        return withMetrics(await renderMermaid(text), metricSource, truncated);
      case ".json":
        return withMetrics(renderJSONPreview(text, truncated), metricSource, truncated);
      case ".csv":
        return withMetrics(renderCSVPreview(text, ",", width, truncated), metricSource, truncated);
      case ".tsv":
        return withMetrics(renderCSVPreview(text, "\t", width, truncated), metricSource, truncated);
    }

    // Syntax highlighting for everything else
    const highlighted = await highlightCode(filePath, text);
    if (highlighted !== text) {
      const out = truncated ? highlighted + "\n\n... preview truncated ..." : highlighted;
      return withMetrics(out, metricSource, truncated);
    }

    const out = truncated ? text + "\n\n... preview truncated ..." : text;
    return withMetrics(out, metricSource, truncated);
  } finally {
    await fd.close();
  }
}
