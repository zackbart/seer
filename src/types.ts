// ── enums ────────────────────────────────────────────────────────────────────

export enum FileCategory {
  Dir = "dir",
  Image = "image",
  Doc = "doc",
  Code = "code",
  Config = "config",
  Exec = "exec",
  Binary = "binary",
  Other = "other",
}

export enum SortMode {
  NameAsc = 0,
  NameDesc = 1,
  SizeDesc = 2,
  SizeAsc = 3,
  ModifiedDesc = 4,
  ModifiedAsc = 5,
}

export const SORT_MODE_COUNT = 6;

export const sortModeLabel: Record<SortMode, string> = {
  [SortMode.NameAsc]: "name ↑",
  [SortMode.NameDesc]: "name ↓",
  [SortMode.SizeDesc]: "size ↓",
  [SortMode.SizeAsc]: "size ↑",
  [SortMode.ModifiedDesc]: "date ↓",
  [SortMode.ModifiedAsc]: "date ↑",
};

// ── entry ────────────────────────────────────────────────────────────────────

export interface Entry {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  modTime: Date;
  isSymlink: boolean;
  symlinkTarget: string;
}

// ── app state ────────────────────────────────────────────────────────────────

export interface AppState {
  cwd: string;
  allEntries: Entry[];
  entries: Entry[];
  selected: number;
  showHidden: boolean;
  preview: string;
  status: string;
  width: number;
  height: number;

  previewOffset: number;
  previewLineCount: number;
  previewTokenEstimate: number;
  previewTruncated: boolean;
  loading: boolean;
  requestId: number;

  searching: boolean;
  searchQuery: string;

  confirmingDelete: boolean;
  deleteTarget: string;

  sortBy: SortMode;

  paneOffset: number;

  gitStatus: Map<string, string> | null;
  gitLoadCwd: string;

  // Preview text selection (click-drag to copy)
  previewSelecting: boolean;
  previewSelStart: { x: number; y: number };
  previewSelEnd: { x: number; y: number };

  // Set when the current preview is a terminal-graphics image rather than
  // text content. Preview.tsx renders gridRows directly and skips the wrap /
  // scroll / selection paths.
  previewImage: ImagePayload | null;
}

// ── preview payload ──────────────────────────────────────────────────────────

export interface PreviewPayload {
  text: string;
  lineCount: number;
  tokenEstimate: number;
  truncated: boolean;
  // Present when the preview is a pixel-perfect terminal-graphics image.
  // The `text` field is empty in this case — the grid is in `image.gridRows`
  // and the preview body renders those rows directly, bypassing the wrap
  // pipeline. `transmitEscape` is the one-shot APC payload that uploads the
  // image data to the terminal; it's consumed once and stripped before the
  // payload is cached.
  image?: ImagePayload;
  transmitEscape?: string;
}

export interface ImagePayload {
  protocol: "kitty-placeholder";
  id: number;
  cols: number;
  rows: number;
  gridRows: string[];
}

// ── constants ────────────────────────────────────────────────────────────────

export const FAST_MODE = process.env.SEER_FAST_MODE === "1";

// Image rendering protocol override. `auto` (default) detects Kitty graphics
// support via supports-terminal-graphics and falls back to half-blocks.
// `blocks` forces half-blocks, `kitty` forces Kitty-placeholder (will render
// garbage on terminals that don't support it), `off` renders a size-only
// placeholder, `iterm` is reserved but not implemented (see CLAUDE.md).
export type ImageProtocol = "auto" | "kitty" | "blocks" | "iterm" | "off";
const rawImageProtocol = (process.env.SEER_IMAGE_PROTOCOL ?? "auto").toLowerCase();
export const SEER_IMAGE_PROTOCOL: ImageProtocol =
  rawImageProtocol === "kitty" ? "kitty" :
  rawImageProtocol === "blocks" ? "blocks" :
  rawImageProtocol === "iterm" ? "iterm" :
  rawImageProtocol === "off" ? "off" :
  "auto";

export const MAX_PREVIEW_BYTES = FAST_MODE ? 64 * 1024 : 256 * 1024;
export const MAX_RENDER_PREVIEW_CHARS = FAST_MODE ? 32 * 1024 : 80 * 1024;
export const MAX_RICH_RENDER_CHARS = FAST_MODE ? 32 * 1024 : 64 * 1024;
export const MAX_DIR_PREVIEW = 40;
export const PREVIEW_CACHE_MAX = 50;
export const MAX_DIR_STAT_CONCURRENCY = 128;

// Debounce applied to expensive previewers (code/markdown/office/pdf) while
// j/k-spam navigating. Cheap previewers (json/csv/hex/directory/plain) bypass.
export const PREVIEW_DEBOUNCE_MS = FAST_MODE ? 150 : 60;

// Fast-path gate: split a code preview into a plain-text dispatch followed by
// the highlighted dispatch only when the rolling median Shiki time has exceeded
// this threshold. On fast hardware the gate stays dormant so there's no extra
// render commit per file.
export const SHIKI_FAST_PATH_THRESHOLD_MS = 40;
