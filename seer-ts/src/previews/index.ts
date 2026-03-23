import fsp from "fs/promises";
import path from "path";
import { MAX_PREVIEW_BYTES } from "../types.js";
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

export async function buildPreview(
  filePath: string,
  width: number,
  _height: number,
): Promise<string> {
  const stat = await fsp.stat(filePath);

  if (stat.isDirectory()) {
    return buildDirPreview(filePath);
  }

  const ext = path.extname(filePath).toLowerCase();

  // Image files — dropped for v1
  if (imageExts.has(ext)) {
    return `image file: ${path.basename(filePath)}\nsize: ${humanSize(stat.size)}\n\n(image preview not yet available in TS version)`;
  }

  // Archive files
  if (archiveExts.has(ext)) {
    return buildArchivePreview(filePath);
  }

  // Read file content
  const fd = await fsp.open(filePath, "r");
  try {
    const buf = Buffer.alloc(MAX_PREVIEW_BYTES);
    const { bytesRead } = await fd.read(buf, 0, MAX_PREVIEW_BYTES, 0);
    const data = buf.subarray(0, bytesRead);

    // Binary detection
    if (isLikelyBinary(data)) {
      return buildHexPreview(data, path.basename(filePath), stat.size, stat.mtime);
    }

    let text = data.toString("utf-8");
    // Check for invalid UTF-8 (replacement character indicates issues)
    if (text.includes("\uFFFD") && bytesRead > 0) {
      return buildHexPreview(data, path.basename(filePath), stat.size, stat.mtime);
    }

    // Normalize line endings
    text = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n");

    const truncated = bytesRead === MAX_PREVIEW_BYTES;

    switch (ext) {
      case ".md":
      case ".markdown":
      case ".mdx": {
        const rendered = renderMarkdown(text, width, truncated);
        if (rendered) return rendered;
        const hl = await highlightCode(filePath, text);
        if (hl !== text) {
          return truncated ? hl + "\n\n... preview truncated ..." : hl;
        }
        return text;
      }
      case ".mmd":
      case ".mermaid":
        return renderMermaid(text);
      case ".json":
        return renderJSONPreview(text, truncated);
      case ".csv":
        return renderCSVPreview(text, ",", width, truncated);
      case ".tsv":
        return renderCSVPreview(text, "\t", width, truncated);
    }

    // Syntax highlighting for everything else
    const highlighted = await highlightCode(filePath, text);
    if (highlighted !== text) {
      return truncated ? highlighted + "\n\n... preview truncated ..." : highlighted;
    }

    return truncated ? text + "\n\n... preview truncated ..." : text;
  } finally {
    await fd.close();
  }
}
