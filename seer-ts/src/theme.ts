import { Entry, FileCategory } from "./types.js";
import path from "path";

// ── color palette ────────────────────────────────────────────────────────────
// True-color hex palette — a refined dark theme with deep indigo/slate tones.
// Using hex instead of 256-color indices for much richer color expression.

export const colors = {
  // Surfaces — layered depth
  bg: "#1a1b26",              // deep navy background
  surface: "#1e2030",         // main panel fill
  surfaceAlt: "#242637",      // raised chrome, headers
  surfaceElevated: "#2a2d42", // selected row, modal bg
  surfaceHover: "#323550",    // hover emphasis

  // Accent — electric blue
  accent: "#7aa2f7",          // primary accent
  accentDim: "#5a7dd4",       // softer accent for borders
  accentFg: "#e0e6ff",        // text on accent backgrounds

  // File type colors — distinct but harmonious
  dir: "#7aa2f7",             // blue for directories
  dirHidden: "#545c8c",       // muted blue for hidden dirs
  file: "#c0caf5",            // bright lavender for regular files
  fileHidden: "#565f89",      // muted for hidden files
  exec: "#9ece6a",            // green for executables
  media: "#e0af68",           // warm gold for media
  doc: "#bb9af7",             // purple for docs
  config: "#e0af68",          // gold for config files
  binary: "#f7768e",          // rose for binary/unknown
  symlink: "#2ac3de",         // cyan for symlinks

  // UI chrome
  size: "#737aa2",            // steel for metadata
  muted: "#565f89",           // secondary text
  dim: "#3b3f5c",             // dividers, low contrast
  breadcrumb: "#a9b1d6",      // path text
  pathSep: "#565f89",         // breadcrumb separators
  hintKey: "#7dcfff",         // key caps — bright cyan
  hintText: "#737aa2",        // hint descriptions
  status: "#a9b1d6",          // status copy
  border: "#3b3f5c",          // panel borders
  borderStrong: "#7aa2f7",    // active panel border
  title: "#c0caf5",           // bright titles
  loading: "#e0af68",         // loading indicator
  scrollbar: "#7aa2f7",       // scroll indicator
  danger: "#f7768e",          // destructive accent
  dangerSoft: "#3b2040",      // danger surface
  success: "#9ece6a",         // confirmation
} as const;

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
