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
}

// ── preview payload ──────────────────────────────────────────────────────────

export interface PreviewPayload {
  text: string;
  lineCount: number;
  tokenEstimate: number;
  truncated: boolean;
}

// ── constants ────────────────────────────────────────────────────────────────

export const FAST_MODE = process.env.SEER_FAST_MODE === "1";

export const MAX_PREVIEW_BYTES = FAST_MODE ? 64 * 1024 : 256 * 1024;
export const MAX_DIR_PREVIEW = 40;
export const PREVIEW_CACHE_MAX = 50;

// Debounce applied to expensive previewers (code/markdown/office/pdf) while
// j/k-spam navigating. Cheap previewers (json/csv/hex/directory/plain) bypass.
export const PREVIEW_DEBOUNCE_MS = FAST_MODE ? 150 : 60;

// Fast-path gate: split a code preview into a plain-text dispatch followed by
// the highlighted dispatch only when the rolling median Shiki time has exceeded
// this threshold. On fast hardware the gate stays dormant so there's no extra
// render commit per file.
export const SHIKI_FAST_PATH_THRESHOLD_MS = 40;
