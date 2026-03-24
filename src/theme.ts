import { Entry, FileCategory } from "./types.js";
import path from "path";
import fs from "fs";

// ── theme type ──────────────────────────────────────────────────────────────

export interface ThemePalette {
  name: string;

  // Surfaces — layered depth
  bg: string;
  surface: string;
  surfaceAlt: string;
  surfaceElevated: string;
  surfaceHover: string;

  // Accent
  accent: string;
  accentDim: string;
  accentFg: string;

  // File type colors
  dir: string;
  dirHidden: string;
  file: string;
  fileHidden: string;
  exec: string;
  media: string;
  doc: string;
  config: string;
  binary: string;
  symlink: string;

  // UI chrome
  size: string;
  muted: string;
  dim: string;
  breadcrumb: string;
  pathSep: string;
  hintKey: string;
  hintText: string;
  status: string;
  border: string;
  borderStrong: string;
  title: string;
  loading: string;
  scrollbar: string;
  danger: string;
  dangerSoft: string;
  success: string;
}

// ── themes ──────────────────────────────────────────────────────────────────

const tokyoNight: ThemePalette = {
  name: "Tokyo Night",
  bg: "#1a1b26",
  surface: "#1e2030",
  surfaceAlt: "#242637",
  surfaceElevated: "#2a2d42",
  surfaceHover: "#323550",
  accent: "#7aa2f7",
  accentDim: "#5a7dd4",
  accentFg: "#e0e6ff",
  dir: "#7aa2f7",
  dirHidden: "#545c8c",
  file: "#c0caf5",
  fileHidden: "#565f89",
  exec: "#9ece6a",
  media: "#e0af68",
  doc: "#bb9af7",
  config: "#e0af68",
  binary: "#f7768e",
  symlink: "#2ac3de",
  size: "#737aa2",
  muted: "#565f89",
  dim: "#3b3f5c",
  breadcrumb: "#a9b1d6",
  pathSep: "#565f89",
  hintKey: "#7dcfff",
  hintText: "#737aa2",
  status: "#a9b1d6",
  border: "#3b3f5c",
  borderStrong: "#7aa2f7",
  title: "#c0caf5",
  loading: "#e0af68",
  scrollbar: "#7aa2f7",
  danger: "#f7768e",
  dangerSoft: "#3b2040",
  success: "#9ece6a",
};

const catppuccinMocha: ThemePalette = {
  name: "Catppuccin",
  bg: "#1e1e2e",
  surface: "#181825",
  surfaceAlt: "#313244",
  surfaceElevated: "#45475a",
  surfaceHover: "#585b70",
  accent: "#89b4fa",
  accentDim: "#74c7ec",
  accentFg: "#cdd6f4",
  dir: "#89b4fa",
  dirHidden: "#585b70",
  file: "#cdd6f4",
  fileHidden: "#6c7086",
  exec: "#a6e3a1",
  media: "#f9e2af",
  doc: "#cba6f7",
  config: "#fab387",
  binary: "#f38ba8",
  symlink: "#94e2d5",
  size: "#7f849c",
  muted: "#6c7086",
  dim: "#45475a",
  breadcrumb: "#bac2de",
  pathSep: "#6c7086",
  hintKey: "#89dceb",
  hintText: "#7f849c",
  status: "#bac2de",
  border: "#45475a",
  borderStrong: "#89b4fa",
  title: "#cdd6f4",
  loading: "#f9e2af",
  scrollbar: "#89b4fa",
  danger: "#f38ba8",
  dangerSoft: "#3b2040",
  success: "#a6e3a1",
};

const rosePine: ThemePalette = {
  name: "Rosé Pine",
  bg: "#191724",
  surface: "#1f1d2e",
  surfaceAlt: "#26233a",
  surfaceElevated: "#2a283e",
  surfaceHover: "#393552",
  accent: "#c4a7e7",
  accentDim: "#907aa9",
  accentFg: "#e0def4",
  dir: "#c4a7e7",
  dirHidden: "#6e6a86",
  file: "#e0def4",
  fileHidden: "#6e6a86",
  exec: "#31748f",
  media: "#f6c177",
  doc: "#9ccfd8",
  config: "#f6c177",
  binary: "#eb6f92",
  symlink: "#9ccfd8",
  size: "#908caa",
  muted: "#6e6a86",
  dim: "#403d52",
  breadcrumb: "#e0def4",
  pathSep: "#6e6a86",
  hintKey: "#9ccfd8",
  hintText: "#908caa",
  status: "#e0def4",
  border: "#403d52",
  borderStrong: "#c4a7e7",
  title: "#e0def4",
  loading: "#f6c177",
  scrollbar: "#c4a7e7",
  danger: "#eb6f92",
  dangerSoft: "#3a2244",
  success: "#31748f",
};

const gruvbox: ThemePalette = {
  name: "Gruvbox",
  bg: "#282828",
  surface: "#1d2021",
  surfaceAlt: "#3c3836",
  surfaceElevated: "#504945",
  surfaceHover: "#665c54",
  accent: "#83a598",
  accentDim: "#689d6a",
  accentFg: "#ebdbb2",
  dir: "#83a598",
  dirHidden: "#665c54",
  file: "#ebdbb2",
  fileHidden: "#928374",
  exec: "#b8bb26",
  media: "#fabd2f",
  doc: "#d3869b",
  config: "#fe8019",
  binary: "#fb4934",
  symlink: "#8ec07c",
  size: "#928374",
  muted: "#7c6f64",
  dim: "#504945",
  breadcrumb: "#d5c4a1",
  pathSep: "#7c6f64",
  hintKey: "#8ec07c",
  hintText: "#928374",
  status: "#d5c4a1",
  border: "#504945",
  borderStrong: "#83a598",
  title: "#ebdbb2",
  loading: "#fabd2f",
  scrollbar: "#83a598",
  danger: "#fb4934",
  dangerSoft: "#442020",
  success: "#b8bb26",
};

const nord: ThemePalette = {
  name: "Nord",
  bg: "#2e3440",
  surface: "#272c36",
  surfaceAlt: "#3b4252",
  surfaceElevated: "#434c5e",
  surfaceHover: "#4c566a",
  accent: "#88c0d0",
  accentDim: "#81a1c1",
  accentFg: "#eceff4",
  dir: "#88c0d0",
  dirHidden: "#4c566a",
  file: "#eceff4",
  fileHidden: "#616e88",
  exec: "#a3be8c",
  media: "#ebcb8b",
  doc: "#b48ead",
  config: "#d08770",
  binary: "#bf616a",
  symlink: "#8fbcbb",
  size: "#616e88",
  muted: "#4c566a",
  dim: "#434c5e",
  breadcrumb: "#d8dee9",
  pathSep: "#4c566a",
  hintKey: "#8fbcbb",
  hintText: "#616e88",
  status: "#d8dee9",
  border: "#434c5e",
  borderStrong: "#88c0d0",
  title: "#eceff4",
  loading: "#ebcb8b",
  scrollbar: "#88c0d0",
  danger: "#bf616a",
  dangerSoft: "#3b2030",
  success: "#a3be8c",
};

const githubLight: ThemePalette = {
  name: "GitHub Light",
  bg: "#ffffff",
  surface: "#f6f8fa",
  surfaceAlt: "#eaeef2",
  surfaceElevated: "#d0d7de",
  surfaceHover: "#c8d1da",
  accent: "#0969da",
  accentDim: "#218bff",
  accentFg: "#1f2328",
  dir: "#0969da",
  dirHidden: "#8c959f",
  file: "#1f2328",
  fileHidden: "#656d76",
  exec: "#1a7f37",
  media: "#9a6700",
  doc: "#8250df",
  config: "#bf8700",
  binary: "#cf222e",
  symlink: "#0550ae",
  size: "#656d76",
  muted: "#8c959f",
  dim: "#d0d7de",
  breadcrumb: "#1f2328",
  pathSep: "#8c959f",
  hintKey: "#0550ae",
  hintText: "#656d76",
  status: "#1f2328",
  border: "#d0d7de",
  borderStrong: "#0969da",
  title: "#1f2328",
  loading: "#9a6700",
  scrollbar: "#0969da",
  danger: "#cf222e",
  dangerSoft: "#ffebe9",
  success: "#1a7f37",
};

const solarizedLight: ThemePalette = {
  name: "Solarized Light",
  bg: "#fdf6e3",
  surface: "#eee8d5",
  surfaceAlt: "#e4ddc8",
  surfaceElevated: "#d6ceb5",
  surfaceHover: "#ccc4a8",
  accent: "#268bd2",
  accentDim: "#2aa198",
  accentFg: "#073642",
  dir: "#268bd2",
  dirHidden: "#93a1a1",
  file: "#586e75",
  fileHidden: "#93a1a1",
  exec: "#859900",
  media: "#b58900",
  doc: "#6c71c4",
  config: "#cb4b16",
  binary: "#dc322f",
  symlink: "#2aa198",
  size: "#93a1a1",
  muted: "#93a1a1",
  dim: "#d6ceb5",
  breadcrumb: "#586e75",
  pathSep: "#93a1a1",
  hintKey: "#2aa198",
  hintText: "#93a1a1",
  status: "#586e75",
  border: "#d6ceb5",
  borderStrong: "#268bd2",
  title: "#073642",
  loading: "#b58900",
  scrollbar: "#268bd2",
  danger: "#dc322f",
  dangerSoft: "#fce8e6",
  success: "#859900",
};

export const themes: ThemePalette[] = [tokyoNight, catppuccinMocha, rosePine, gruvbox, nord, githubLight, solarizedLight];

// ── active theme ────────────────────────────────────────────────────────────

let activeIndex = 0;

// Mutable colors object — all components read from this.
export let colors: ThemePalette = { ...tokyoNight };

function applyTheme(idx: number) {
  activeIndex = idx;
  Object.assign(colors, themes[idx]);
}

export function currentThemeIndex(): number {
  return activeIndex;
}

export function cycleTheme(): string {
  const next = (activeIndex + 1) % themes.length;
  applyTheme(next);
  saveTheme(themes[next].name);
  return themes[next].name;
}

// ── persistence ─────────────────────────────────────────────────────────────

function configDir(): string {
  const home = process.env.HOME ?? "";
  return path.join(home, ".config", "seer");
}

function configPath(): string {
  return path.join(configDir(), "theme");
}

export function loadSavedTheme(): void {
  try {
    const name = fs.readFileSync(configPath(), "utf-8").trim();
    const idx = themes.findIndex((t) => t.name === name);
    if (idx >= 0) applyTheme(idx);
  } catch {
    // No saved theme — use default
  }
}

function saveTheme(name: string): void {
  try {
    const dir = configDir();
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(configPath(), name + "\n");
  } catch {
    // Best effort
  }
}

// Load saved theme on import
loadSavedTheme();

// ── image extensions ─────────────────────────────────────────────────────────

export const imageExts = new Set([
  ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tiff",
]);

// ── archive extensions ───────────────────────────────────────────────────────

export const archiveExts = new Set([
  ".zip", ".jar", ".war", ".tar", ".gz", ".tgz", ".bz2", ".tbz2",
  ".xz", ".txz", ".zst", ".7z", ".rar",
]);

// ── nerd font control ────────────────────────────────────────────────────────

export const nerdFonts = process.env.SEER_NO_NERD_FONT !== "1";

// ── nerd font icon maps ──────────────────────────────────────────────────────

const nerdIconByExt: Record<string, string> = {
  ".go": "\ue627 ", ".js": "\ue60c ", ".ts": "\ue628 ",
  ".jsx": "\ue60c ", ".tsx": "\ue60c ", ".py": "\ue606 ",
  ".rb": "\ue21e ", ".rs": "\ue7a8 ", ".c": "\ue61e ",
  ".cpp": "\ue61d ", ".h": "\uf0fd ", ".java": "\ue204 ",
  ".cs": "\uf031b ", ".php": "\ue60a ", ".swift": "\ue755 ",
  ".kt": "\ue634 ", ".lua": "\ue620 ", ".hs": "\ue61f ",
  ".vim": "\ue62b ", ".sh": "\uf489 ", ".bash": "\uf489 ",
  ".zsh": "\uf489 ", ".fish": "\uf489 ", ".ps1": "\uf489 ",
  ".bat": "\uf489 ", ".cmd": "\uf489 ",
  ".md": "\ue609 ", ".markdown": "\ue609 ", ".mdx": "\ue609 ",
  ".rst": "\uf15c ", ".txt": "\uf15c ",
  ".json": "\ue60b ", ".yaml": "\uf481 ", ".yml": "\uf481 ",
  ".toml": "\uf481 ", ".xml": "\uf05c0 ", ".env": "\uf462 ",
  ".ini": "\uf17a ", ".conf": "\uf17a ",
  ".png": "\uf1c5 ", ".jpg": "\uf1c5 ", ".jpeg": "\uf1c5 ",
  ".gif": "\uf1c5 ", ".webp": "\uf1c5 ", ".svg": "\uf1c5 ",
  ".bmp": "\uf1c5 ",
  ".mmd": "\ueb43 ", ".mermaid": "\ueb43 ",
  ".pdf": "\uf1c1 ", ".zip": "\uf410 ", ".tar": "\uf410 ",
  ".gz": "\uf410 ", ".gitignore": "\ue702 ", ".dockerignore": "\uf308 ",
};

const nerdIconByCategory: Partial<Record<FileCategory, string>> = {
  [FileCategory.Dir]: "\uf07b ", [FileCategory.Image]: "\uf1c5 ",
  [FileCategory.Doc]: "\uf15c ", [FileCategory.Code]: "\uf121 ",
  [FileCategory.Config]: "\uf462 ", [FileCategory.Exec]: "\uf489 ",
  [FileCategory.Binary]: "\uf471 ",
};

const plainIcon: Partial<Record<FileCategory, string>> = {
  [FileCategory.Dir]: "▸ ", [FileCategory.Image]: "⬡ ",
  [FileCategory.Doc]: "≡ ", [FileCategory.Code]: "⟨⟩",
  [FileCategory.Config]: "⚙ ", [FileCategory.Exec]: "⚡",
  [FileCategory.Binary]: "⬟ ",
};

// ── categorise ───────────────────────────────────────────────────────────────

export function categorise(e: Entry): FileCategory {
  if (e.isDir) return FileCategory.Dir;
  const ext = path.extname(e.name).toLowerCase();
  switch (ext) {
    case ".png": case ".jpg": case ".jpeg": case ".webp":
    case ".gif": case ".bmp": case ".tiff":
      return FileCategory.Image;
    case ".md": case ".markdown": case ".mdx": case ".rst": case ".txt":
      return FileCategory.Doc;
    case ".sh": case ".bash": case ".zsh": case ".fish":
    case ".ps1": case ".bat": case ".cmd":
      return FileCategory.Exec;
    case ".go": case ".js": case ".ts": case ".jsx": case ".tsx":
    case ".py": case ".rb": case ".rs": case ".c": case ".cpp":
    case ".h": case ".java": case ".cs": case ".php": case ".swift":
    case ".kt": case ".lua": case ".ex": case ".exs": case ".hs":
    case ".ml": case ".mli": case ".clj": case ".scala":
    case ".vim": case ".mmd": case ".mermaid":
      return FileCategory.Code;
    case ".json": case ".yaml": case ".yml": case ".toml": case ".ini":
    case ".env": case ".conf": case ".config": case ".xml":
    case ".dockerignore": case ".gitignore": case ".editorconfig":
    case ".eslintrc": case ".prettierrc": case ".babelrc": case ".nvmrc":
      return FileCategory.Config;
    default:
      return FileCategory.Other;
  }
}

// ── icons ────────────────────────────────────────────────────────────────────

export function symlinkIcon(): string {
  return nerdFonts ? "\uf0c1 " : "⇒ ";
}

export function fileIcon(cat: FileCategory): string {
  return fileIconExt(cat, "");
}

export function fileIconExt(cat: FileCategory, ext: string): string {
  if (!nerdFonts) return plainIcon[cat] ?? "· ";
  if (ext) {
    const icon = nerdIconByExt[ext.toLowerCase()];
    if (icon) return icon;
  }
  return nerdIconByCategory[cat] ?? "\uf15b ";
}

// ── file colors ──────────────────────────────────────────────────────────────

export function fileColor(cat: FileCategory): string {
  switch (cat) {
    case FileCategory.Dir: return colors.dir;
    case FileCategory.Image: return colors.media;
    case FileCategory.Doc: return colors.doc;
    case FileCategory.Code: return colors.file;
    case FileCategory.Config: return colors.config;
    case FileCategory.Exec: return colors.exec;
    case FileCategory.Binary: return colors.binary;
    default: return colors.file;
  }
}

export function entryColor(e: Entry): string {
  if (e.isSymlink) return colors.symlink;
  if (e.isDir && isHidden(e.name)) return colors.dirHidden;
  if (e.isDir) return colors.dir;
  if (isHidden(e.name)) return colors.fileHidden;
  return fileColor(categorise(e));
}

export function entryBold(e: Entry): boolean {
  return e.isDir;
}

export function isHidden(name: string): boolean {
  return name.startsWith(".");
}
