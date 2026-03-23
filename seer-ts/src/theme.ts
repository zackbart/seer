import { Entry, FileCategory } from "./types.js";
import path from "path";

// ── color palette ────────────────────────────────────────────────────────────
// 256-color terminal indices matching the Go version's dark indigo/slate theme.

export const colors = {
  bg: "234",
  surface: "236",
  surfaceAlt: "237",
  surfaceElevated: "239",
  accent: "111",
  accentFg: "255",
  dir: "68",
  dirHidden: "60",
  file: "255",
  fileHidden: "245",
  exec: "150",
  media: "221",
  doc: "189",
  config: "223",
  binary: "210",
  size: "248",
  muted: "245",
  dim: "240",
  breadcrumb: "189",
  pathSep: "243",
  hintKey: "117",
  hintText: "250",
  status: "189",
  border: "241",
  borderStrong: "111",
  title: "255",
  loading: "221",
  scrollbar: "110",
  danger: "203",
  dangerSoft: "52",
  symlink: "80",
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
  // languages
  ".go": "\ue627 ",
  ".js": "\ue60c ",
  ".ts": "\ue628 ",
  ".jsx": "\ue60c ",
  ".tsx": "\ue60c ",
  ".py": "\ue606 ",
  ".rb": "\ue21e ",
  ".rs": "\ue7a8 ",
  ".c": "\ue61e ",
  ".cpp": "\ue61d ",
  ".h": "\uf0fd ",
  ".java": "\ue204 ",
  ".cs": "\uf031b ",
  ".php": "\ue60a ",
  ".swift": "\ue755 ",
  ".kt": "\ue634 ",
  ".lua": "\ue620 ",
  ".hs": "\ue61f ",
  ".vim": "\ue62b ",
  ".sh": "\uf489 ",
  ".bash": "\uf489 ",
  ".zsh": "\uf489 ",
  ".fish": "\uf489 ",
  ".ps1": "\uf489 ",
  ".bat": "\uf489 ",
  ".cmd": "\uf489 ",
  // docs
  ".md": "\ue609 ",
  ".markdown": "\ue609 ",
  ".mdx": "\ue609 ",
  ".rst": "\uf15c ",
  ".txt": "\uf15c ",
  // config
  ".json": "\ue60b ",
  ".yaml": "\uf481 ",
  ".yml": "\uf481 ",
  ".toml": "\uf481 ",
  ".xml": "\uf05c0 ",
  ".env": "\uf462 ",
  ".ini": "\uf17a ",
  ".conf": "\uf17a ",
  // images
  ".png": "\uf1c5 ",
  ".jpg": "\uf1c5 ",
  ".jpeg": "\uf1c5 ",
  ".gif": "\uf1c5 ",
  ".webp": "\uf1c5 ",
  ".svg": "\uf1c5 ",
  ".bmp": "\uf1c5 ",
  // misc
  ".mmd": "\ueb43 ",
  ".mermaid": "\ueb43 ",
  ".pdf": "\uf1c1 ",
  ".zip": "\uf410 ",
  ".tar": "\uf410 ",
  ".gz": "\uf410 ",
  ".gitignore": "\ue702 ",
  ".dockerignore": "\uf308 ",
};

const nerdIconByCategory: Partial<Record<FileCategory, string>> = {
  [FileCategory.Dir]: "\uf07b ",
  [FileCategory.Image]: "\uf1c5 ",
  [FileCategory.Doc]: "\uf15c ",
  [FileCategory.Code]: "\uf121 ",
  [FileCategory.Config]: "\uf462 ",
  [FileCategory.Exec]: "\uf489 ",
  [FileCategory.Binary]: "\uf471 ",
};

const plainIcon: Partial<Record<FileCategory, string>> = {
  [FileCategory.Dir]: "▸ ",
  [FileCategory.Image]: "⬡ ",
  [FileCategory.Doc]: "≡ ",
  [FileCategory.Code]: "⟨⟩ ",
  [FileCategory.Config]: "⚙ ",
  [FileCategory.Exec]: "⚡ ",
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
  if (!nerdFonts) {
    return plainIcon[cat] ?? "· ";
  }
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
    case FileCategory.Code: return "231";
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
