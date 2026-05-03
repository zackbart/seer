import fsp from "fs/promises";
import path from "path";
import { FAST_MODE, MAX_PREVIEW_BYTES, PreviewPayload } from "../types.js";
import { imageExts, archiveExts, officeExts, pdfExts } from "../theme.js";
import { isLikelyBinary } from "../utils/fs.js";
import { humanSize } from "../utils/humanSize.js";
import { sanitizeTerminalText } from "../utils/ansiText.js";

// ── lazy previewer loading ──────────────────────────────────────────────────
// Each previewer is dynamic-imported on first use and the promise is memoized,
// so repeat calls resolve instantly and untouched previewers never land in the
// startup image. This is the biggest startup win — mammoth, exceljs, unpdf,
// marked+marked-terminal, and shiki are all kept out of the critical path.

let directoryMod: Promise<typeof import("./directory.js")> | null = null;
let codeMod: Promise<typeof import("./code.js")> | null = null;
let markdownMod: Promise<typeof import("./markdown.js")> | null = null;
let jsonMod: Promise<typeof import("./json.js")> | null = null;
let csvMod: Promise<typeof import("./csv.js")> | null = null;
let hexMod: Promise<typeof import("./hex.js")> | null = null;
let archiveMod: Promise<typeof import("./archive.js")> | null = null;
let mermaidMod: Promise<typeof import("./mermaid.js")> | null = null;
let htmlMod: Promise<typeof import("./html.js")> | null = null;
let docxMod: Promise<typeof import("./docx.js")> | null = null;
let xlsxMod: Promise<typeof import("./xlsx.js")> | null = null;
let pdfMod: Promise<typeof import("./pdf.js")> | null = null;
let imageMod: Promise<typeof import("./image.js")> | null = null;

const loadDirectory = () => (directoryMod ??= import("./directory.js"));
const loadCode = () => (codeMod ??= import("./code.js"));
const loadMarkdown = () => (markdownMod ??= import("./markdown.js"));
const loadJson = () => (jsonMod ??= import("./json.js"));
const loadCsv = () => (csvMod ??= import("./csv.js"));
const loadHex = () => (hexMod ??= import("./hex.js"));
const loadArchive = () => (archiveMod ??= import("./archive.js"));
const loadMermaid = () => (mermaidMod ??= import("./mermaid.js"));
const loadHtml = () => (htmlMod ??= import("./html.js"));
const loadDocx = () => (docxMod ??= import("./docx.js"));
const loadXlsx = () => (xlsxMod ??= import("./xlsx.js"));
const loadPdf = () => (pdfMod ??= import("./pdf.js"));
const loadImage = () => (imageMod ??= import("./image.js"));

const MAX_OFFICE_BYTES = 10 * 1024 * 1024;
const MAX_IMAGE_BYTES = 10 * 1024 * 1024;

// ── payload helpers ─────────────────────────────────────────────────────────

function emptyMetrics(text: string): PreviewPayload {
  return { text: sanitizeTerminalText(text), lineCount: 0, tokenEstimate: 0, truncated: false };
}

function withMetrics(text: string, source: string, truncated: boolean): PreviewPayload {
  const lineCount = source.length === 0 ? 0 : source.split("\n").length;
  const tokenEstimate = Math.ceil(source.trim().length / 4);
  return { text: sanitizeTerminalText(text), lineCount, tokenEstimate, truncated };
}

function normalizePreviewSource(text: string, options: { preserveTabs?: boolean } = {}): string {
  return sanitizeTerminalText(text, options);
}

function aborted(signal?: AbortSignal): boolean {
  return signal?.aborted === true;
}

// Extensions that get special "heavy" treatment: code highlighting, markdown
// rendering, or office/pdf parsing. Cheap previewers (json, csv, hex, dir,
// plain text) bypass debounce in App.tsx.
const CODE_HIGHLIGHT_EXTS = new Set([
  "js", "jsx", "ts", "tsx", "mjs", "cjs", "mts", "cts",
  "py", "rb", "rs", "go", "c", "cc", "cpp", "cxx", "h", "hpp",
  "java", "cs", "php", "swift", "kt", "kts", "lua", "hs",
  "ex", "exs", "ml", "mli", "clj", "cljs", "scala",
  "sh", "bash", "zsh", "fish", "ps1", "psm1",
  "yaml", "yml", "toml", "xml", "svg", "plist", "ini", "conf", "cfg",
  "sql", "graphql", "gql", "css", "scss", "sass", "less",
  "svelte", "vue", "dockerfile",
  "r", "dart", "zig", "nix", "tf", "tfvars", "proto",
]);

export function isExpensivePreview(filePath: string): boolean {
  const ext = path.extname(filePath).toLowerCase();
  if (!ext) return false;
  if (officeExts.has(ext) || pdfExts.has(ext)) return true;
  if (imageExts.has(ext)) return true;
  if (ext === ".md" || ext === ".markdown" || ext === ".mdx") return true;
  if (ext === ".mmd" || ext === ".mermaid") return true;
  if (ext === ".html" || ext === ".htm" || ext === ".xhtml") return true;
  if (archiveExts.has(ext)) return true;
  const bare = ext.startsWith(".") ? ext.slice(1) : ext;
  return CODE_HIGHLIGHT_EXTS.has(bare);
}

// ── main entry ──────────────────────────────────────────────────────────────

export async function buildPreview(
  filePath: string,
  width: number,
  height: number,
  signal?: AbortSignal,
): Promise<PreviewPayload> {
  if (aborted(signal)) return emptyMetrics("");
  const stat = await fsp.stat(filePath);
  if (aborted(signal)) return emptyMetrics("");

  if (stat.isDirectory()) {
    const { buildDirPreview } = await loadDirectory();
    if (aborted(signal)) return emptyMetrics("");
    return emptyMetrics(await buildDirPreview(filePath));
  }

  const ext = path.extname(filePath).toLowerCase();

  // Image files — rasterized to Unicode half-blocks via terminal-image.
  if (imageExts.has(ext)) {
    if (FAST_MODE) {
      return emptyMetrics(
        `${path.basename(filePath)}\nsize: ${humanSize(stat.size)}\n\n(image preview disabled in fast mode — unset SEER_FAST_MODE to enable)`,
      );
    }
    if (stat.size > MAX_IMAGE_BYTES) {
      return emptyMetrics(
        `${path.basename(filePath)}\nsize: ${humanSize(stat.size)}\n\n(image too large to preview — limit ${humanSize(MAX_IMAGE_BYTES)})`,
      );
    }
    try {
      const buffer = await fsp.readFile(filePath);
      if (aborted(signal)) return emptyMetrics("");
      const { renderImagePreview } = await loadImage();
      if (aborted(signal)) return emptyMetrics("");
      return await renderImagePreview(
        buffer,
        filePath,
        stat.mtimeMs,
        stat.size,
        width,
        height,
        signal,
      );
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      return emptyMetrics(
        `${path.basename(filePath)}\n\n(unable to read: ${msg})`,
      );
    }
  }

  // Archive files
  if (archiveExts.has(ext)) {
    const { buildArchivePreview } = await loadArchive();
    if (aborted(signal)) return emptyMetrics("");
    return emptyMetrics(await buildArchivePreview(filePath));
  }

  // Office / PDF files — ZIP or binary formats that require a full-file read
  // and a parser library. Must be intercepted before the UTF-8 read path.
  if (officeExts.has(ext) || pdfExts.has(ext)) {
    if (FAST_MODE) {
      return emptyMetrics(
        `${path.basename(filePath)}\nsize: ${humanSize(stat.size)}\n\n(office/pdf preview disabled in fast mode — unset SEER_FAST_MODE to enable)`,
      );
    }
    if (stat.size > MAX_OFFICE_BYTES) {
      return emptyMetrics(
        `${path.basename(filePath)}\nsize: ${humanSize(stat.size)}\n\n(file too large to preview — limit ${humanSize(MAX_OFFICE_BYTES)})`,
      );
    }
    try {
      const buffer = await fsp.readFile(filePath);
      if (aborted(signal)) return emptyMetrics("");
      if (ext === ".docx") {
        const { renderDocxPreview } = await loadDocx();
        if (aborted(signal)) return emptyMetrics("");
        const r = await renderDocxPreview(buffer);
        const extracted = normalizePreviewSource(r.extracted);
        return withMetrics(r.rendered, extracted, r.truncated);
      }
      if (ext === ".xlsx") {
        const { renderXlsxPreview } = await loadXlsx();
        if (aborted(signal)) return emptyMetrics("");
        const r = await renderXlsxPreview(buffer, width);
        const extracted = normalizePreviewSource(r.extracted, { preserveTabs: true });
        return withMetrics(r.rendered, extracted, r.truncated);
      }
      if (ext === ".pdf") {
        const { renderPdfPreview } = await loadPdf();
        if (aborted(signal)) return emptyMetrics("");
        const r = await renderPdfPreview(buffer);
        const extracted = normalizePreviewSource(r.extracted);
        return withMetrics(r.rendered, extracted, r.truncated);
      }
      return emptyMetrics(
        `${path.basename(filePath)}\n\n(no preview handler registered for ${ext})`,
      );
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      return emptyMetrics(
        `${path.basename(filePath)}\n\n(unable to read: ${msg})`,
      );
    }
  }

  // Read file content
  const fd = await fsp.open(filePath, "r");
  try {
    const buf = Buffer.alloc(MAX_PREVIEW_BYTES);
    const { bytesRead } = await fd.read(buf, 0, MAX_PREVIEW_BYTES, 0);
    if (aborted(signal)) return emptyMetrics("");
    const data = buf.subarray(0, bytesRead);

    // Binary detection
    if (isLikelyBinary(data)) {
      const { buildHexPreview } = await loadHex();
      return emptyMetrics(buildHexPreview(data, path.basename(filePath), stat.size, stat.mtime));
    }

    let text = data.toString("utf-8");
    // Check for invalid UTF-8 (replacement character indicates issues)
    if (text.includes("\uFFFD") && bytesRead > 0) {
      const { buildHexPreview } = await loadHex();
      return emptyMetrics(buildHexPreview(data, path.basename(filePath), stat.size, stat.mtime));
    }

    const truncated = bytesRead === MAX_PREVIEW_BYTES;
    const preserveTabs = ext === ".tsv";
    const normalizedText = normalizePreviewSource(text, { preserveTabs });
    const metricSource = preserveTabs
      ? normalizedText.replace(/\t/g, "    ")
      : normalizedText;

    switch (ext) {
      case ".md":
      case ".markdown":
      case ".mdx": {
        if (FAST_MODE) {
          // In fast mode, treat markdown as plain text — no marked, no Shiki.
          const out = truncated ? metricSource + "\n\n... preview truncated ..." : metricSource;
          return withMetrics(out, metricSource, truncated);
        }
        const { renderMarkdown } = await loadMarkdown();
        if (aborted(signal)) return emptyMetrics("");
        const rendered = renderMarkdown(metricSource, width, truncated);
        if (rendered) return withMetrics(rendered, metricSource, truncated);
        const { highlightCode } = await loadCode();
        if (aborted(signal)) return emptyMetrics("");
        const hl = await highlightCode(filePath, metricSource, signal);
        if (aborted(signal)) return emptyMetrics("");
        if (hl !== metricSource) {
          const out = truncated ? hl + "\n\n... preview truncated ..." : hl;
          return withMetrics(out, metricSource, truncated);
        }
        return withMetrics(metricSource, metricSource, truncated);
      }
      case ".mmd":
      case ".mermaid": {
        const { renderMermaid } = await loadMermaid();
        if (aborted(signal)) return emptyMetrics("");
        return withMetrics(await renderMermaid(metricSource), metricSource, truncated);
      }
      case ".html":
      case ".htm":
      case ".xhtml": {
        if (FAST_MODE) {
          return emptyMetrics(
            `${path.basename(filePath)}\nsize: ${humanSize(stat.size)}\n\n(html preview disabled in fast mode — unset SEER_FAST_MODE to enable)`,
          );
        }
        const { renderHtml } = await loadHtml();
        if (aborted(signal)) return emptyMetrics("");
        const htmlOut = renderHtml(metricSource, width, truncated, signal);
        if (aborted(signal)) return emptyMetrics("");
        return withMetrics(htmlOut, metricSource, truncated);
      }
      case ".json":
      case ".jsonc": {
        const { renderJSONPreview } = await loadJson();
        return withMetrics(renderJSONPreview(metricSource, truncated), metricSource, truncated);
      }
      case ".csv": {
        const { renderCSVPreview } = await loadCsv();
        return withMetrics(renderCSVPreview(metricSource, ",", width, truncated), metricSource, truncated);
      }
      case ".tsv": {
        const { renderCSVPreview } = await loadCsv();
        return withMetrics(renderCSVPreview(normalizedText, "\t", width, truncated), metricSource, truncated);
      }
    }

    // Syntax highlighting for everything else
    if (FAST_MODE) {
      const out = truncated ? metricSource + "\n\n... preview truncated ..." : metricSource;
      return withMetrics(out, metricSource, truncated);
    }
    const { highlightCode } = await loadCode();
    if (aborted(signal)) return emptyMetrics("");
    const highlighted = await highlightCode(filePath, metricSource, signal);
    if (aborted(signal)) return emptyMetrics("");
    if (highlighted !== metricSource) {
      const out = truncated ? highlighted + "\n\n... preview truncated ..." : highlighted;
      return withMetrics(out, metricSource, truncated);
    }

    const out = truncated ? metricSource + "\n\n... preview truncated ..." : metricSource;
    return withMetrics(out, metricSource, truncated);
  } finally {
    await fd.close();
  }
}

// ── fast-path helpers for App.tsx measurement-gated upgrade ─────────────────
// buildPlainPreview reads the file and returns an un-highlighted payload.
// It is used as the "first stage" when Shiki is slow; App.tsx then runs the
// full buildPreview to compute the highlighted version and dispatches an
// upgrade. Only code files route through here; everything else uses the
// normal buildPreview path.
export async function buildPlainPreview(
  filePath: string,
  signal?: AbortSignal,
): Promise<PreviewPayload | null> {
  if (aborted(signal)) return null;
  // HTML doesn't stage well — raw source is noisier than a brief delay.
  const ext = path.extname(filePath).toLowerCase();
  if (ext === ".html" || ext === ".htm" || ext === ".xhtml") return null;
  try {
    const stat = await fsp.stat(filePath);
    if (aborted(signal) || stat.isDirectory()) return null;

    const fd = await fsp.open(filePath, "r");
    try {
      const buf = Buffer.alloc(MAX_PREVIEW_BYTES);
      const { bytesRead } = await fd.read(buf, 0, MAX_PREVIEW_BYTES, 0);
      if (aborted(signal)) return null;
      const data = buf.subarray(0, bytesRead);
      if (isLikelyBinary(data)) return null;

      let text = data.toString("utf-8");
      if (text.includes("\uFFFD") && bytesRead > 0) return null;

      text = normalizePreviewSource(text);
      const truncated = bytesRead === MAX_PREVIEW_BYTES;
      const out = truncated ? text + "\n\n... preview truncated ..." : text;
      return withMetrics(out, text, truncated);
    } finally {
      await fd.close();
    }
  } catch {
    return null;
  }
}
