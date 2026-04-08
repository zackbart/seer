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

export const MAX_PREVIEW_BYTES = 256 * 1024;
export const MAX_DIR_PREVIEW = 40;
export const PREVIEW_CACHE_MAX = 50;
