import fs from "fs";
import fsp from "fs/promises";
import path from "path";
import { Entry, SortMode } from "../types.js";

// ── directory listing ────────────────────────────────────────────────────────

export async function listDir(dirPath: string, showHidden: boolean): Promise<Entry[]> {
  const items = await fsp.readdir(dirPath, { withFileTypes: true });
  const entries: Entry[] = [];

  for (const item of items) {
    const name = item.name;
    if (!showHidden && name.startsWith(".")) continue;

    const full = path.join(dirPath, name);

    let isSymlink = item.isSymbolicLink();
    let symlinkTarget = "";
    let isDir = item.isDirectory();
    let size = 0;
    let modTime = new Date();

    try {
      const lstat = await fsp.lstat(full);
      isSymlink = lstat.isSymbolicLink();
      size = lstat.size;
      modTime = lstat.mtime;
      isDir = lstat.isDirectory();

      if (isSymlink) {
        try {
          symlinkTarget = await fsp.readlink(full);
        } catch {}
        try {
          const stat = await fsp.stat(full);
          isDir = stat.isDirectory();
          size = stat.size;
          modTime = stat.mtime;
        } catch {}
      }
    } catch {
      continue;
    }

    entries.push({ name, path: full, isDir, size, modTime, isSymlink, symlinkTarget });
  }

  // Default sort: dirs first, then alphabetical
  entries.sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
  });

  return entries;
}

// ── sorting ──────────────────────────────────────────────────────────────────

export function applySort(entries: Entry[], mode: SortMode): Entry[] {
  const result = [...entries];
  result.sort((a, b) => {
    // Dirs always first
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;

    switch (mode) {
      case SortMode.NameAsc:
        return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
      case SortMode.NameDesc:
        return b.name.toLowerCase().localeCompare(a.name.toLowerCase());
      case SortMode.SizeDesc:
        if (a.size !== b.size) return b.size - a.size;
        break;
      case SortMode.SizeAsc:
        if (a.size !== b.size) return a.size - b.size;
        break;
      case SortMode.ModifiedDesc:
        if (a.modTime.getTime() !== b.modTime.getTime())
          return b.modTime.getTime() - a.modTime.getTime();
        break;
      case SortMode.ModifiedAsc:
        if (a.modTime.getTime() !== b.modTime.getTime())
          return a.modTime.getTime() - b.modTime.getTime();
        break;
    }
    return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
  });
  return result;
}

// ── trash ────────────────────────────────────────────────────────────────────

function trashDir(): string {
  const home = process.env.HOME ?? "";
  let dir: string;
  if (process.platform === "linux") {
    dir = path.join(home, ".local", "share", "Trash", "files");
  } else {
    dir = path.join(home, ".Trash");
  }
  fs.mkdirSync(dir, { recursive: true });
  return dir;
}

export async function moveToTrash(filePath: string): Promise<void> {
  const trash = trashDir();
  await fsp.access(filePath);

  const baseName = path.basename(filePath);
  const ext = path.extname(baseName);
  const stem = baseName.slice(0, baseName.length - ext.length);

  let destPath = path.join(trash, baseName);
  for (let i = 1; ; i++) {
    try {
      await fsp.access(destPath);
      destPath = path.join(trash, `${stem} ${i}${ext}`);
    } catch {
      break; // doesn't exist, use it
    }
  }
  await fsp.rename(filePath, destPath);
}

// ── git status ───────────────────────────────────────────────────────────────

export async function loadGitStatus(
  cwd: string,
  procRef?: { current: ReturnType<typeof Bun.spawn> | null },
): Promise<Map<string, string> | null> {
  try {
    const proc = Bun.spawn(["git", "-C", cwd, "status", "--porcelain"], {
      stdout: "pipe",
      stderr: "ignore",
    });
    if (procRef) procRef.current = proc;
    const text = await new Response(proc.stdout).text();
    const exitCode = await proc.exited;
    if (procRef) procRef.current = null;
    if (exitCode !== 0) return null;

    const status = new Map<string, string>();
    for (const line of text.split("\n")) {
      if (line.length < 4) continue;
      const xy = line.slice(0, 2).trim();
      let file = line.slice(3).trim();

      // Renames: "old -> new"
      const arrowIdx = file.indexOf(" -> ");
      if (arrowIdx >= 0) file = file.slice(arrowIdx + 4);
      file = file.replace(/^"|"$/g, "");

      const dir = path.dirname(file);
      if (dir === "." || dir === "") {
        // Top-level file
        const base = path.basename(file);
        if (xy && base) status.set(base, xy);
      } else {
        // Nested file — bubble status up to the top-level directory
        const topDir = file.split(path.sep)[0];
        if (xy && topDir) {
          const existing = status.get(topDir) ?? "";
          if (!existing.includes(xy)) {
            status.set(topDir, existing + xy);
          }
        }
      }
    }
    return status;
  } catch {
    return null;
  }
}

// ── preview cache key ────────────────────────────────────────────────────────

export function previewKey(
  filePath: string,
  modTime: Date,
  size: number,
  width: number,
  height: number,
): string {
  return `${filePath}|${modTime.getTime()}|${size}|${width}|${height}`;
}

// ── binary detection ─────────────────────────────────────────────────────────

export function isLikelyBinary(data: Buffer): boolean {
  if (data.length === 0) return false;
  const limit = Math.min(data.length, 8192);
  for (let i = 0; i < limit; i++) {
    if (data[i] === 0) return true;
  }
  return false;
}

// ── fuzzy match ──────────────────────────────────────────────────────────────

export function fuzzyMatch(name: string, query: string): boolean {
  const nameLower = name.toLowerCase();
  const queryLower = query.toLowerCase();
  let qi = 0;
  for (let i = 0; i < nameLower.length && qi < queryLower.length; i++) {
    if (nameLower[i] === queryLower[qi]) qi++;
  }
  return qi === queryLower.length;
}
