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

export enum InputMode {
  None = 0,
  Rename = 1,
  NewFile = 2,
  NewDir = 3,
}

export const inputModeTitle: Record<InputMode, string> = {
  [InputMode.None]: "",
  [InputMode.Rename]: "Rename",
  [InputMode.NewFile]: "New File",
  [InputMode.NewDir]: "New Directory",
};

export enum YankMode {
  None = 0,
  Copy = 1,
  Cut = 2,
}

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
  loading: boolean;
  requestId: number;

  searching: boolean;
  searchQuery: string;

  confirmingDelete: boolean;
  deleteTarget: string;

  inputMode: InputMode;
  inputValue: string;

  yankPaths: string[];
  yankOp: YankMode;

  multiSelected: Set<string>;

  sortBy: SortMode;

  dirHistory: string[];
  historyPos: number;

  bookmarks: string[];

  paneOffset: number;

  gitStatus: Map<string, string> | null;
  gitLoadCwd: string;
}

// ── constants ────────────────────────────────────────────────────────────────

export const MAX_PREVIEW_BYTES = 256 * 1024;
export const MAX_DIR_PREVIEW = 40;
export const PREVIEW_CACHE_MAX = 50;
