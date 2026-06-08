import fs from "fs";
import fsp from "fs/promises";
import path from "path";
import { Entry, MAX_DIR_STAT_CONCURRENCY, SortMode } from "../types.js";
import { logPerf, perfNow } from "./perf.js";

// ── directory listing ────────────────────────────────────────────────────────

export async function listDir(dirPath: string, showHidden: boolean): Promise<Entry[]> {
  const t0 = perfNow();
  const items = await fsp.readdir(dirPath, { withFileTypes: true });

  const visibleItems = showHidden ? items : items.filter((item) => !item.name.startsWith("."));
  const concurrency = Math.max(1, Math.min(MAX_DIR_STAT_CONCURRENCY, visibleItems.length || 1));
  const resolved = new Array<Entry | null>(visibleItems.length).fill(null);
  let next = 0;

  const worker = async () => {
    while (next < visibleItems.length) {
      const idx = next++;
      resolved[idx] = await entryFromDirent(dirPath, visibleItems[idx]);
    }
  };

  await Promise.all(Array.from({ length: concurrency }, worker));

  const entries = resolved.filter((e): e is Entry => e !== null);

  // Default sort: dirs first, then alphabetical
  entries.sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
  });

  logPerf("listDir", {
    path: dirPath,
    items: items.length,
    visible: visibleItems.length,
    entries: entries.length,
    concurrency,
    ms: Math.round((performance.now() - t0) * 10) / 10,
  });

  return entries;
}

async function entryFromDirent(dirPath: string, item: fs.Dirent): Promise<Entry | null> {
  const name = item.name;
  const full = path.join(dirPath, name);

  try {
    const lstat = await fsp.lstat(full);
    const isSymlink = lstat.isSymbolicLink();
    let isDir = lstat.isDirectory();
    let size = lstat.size;
    let modTime = lstat.mtime;
    let symlinkTarget = "";

    if (isSymlink) {
      const [targetResult, statResult] = await Promise.allSettled([
        fsp.readlink(full),
        fsp.stat(full),
      ]);
      if (targetResult.status === "fulfilled") symlinkTarget = targetResult.value;
      if (statResult.status === "fulfilled") {
        isDir = statResult.value.isDirectory();
        size = statResult.value.size;
        modTime = statResult.value.mtime;
      }
    }

    return { name, path: full, isDir, size, modTime, isSymlink, symlinkTarget };
  } catch {
    return null;
  }
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

// Cache git status results keyed on the cwd + .git/index mtime. Revisiting a
// cwd whose index hasn't changed avoids spawning a fresh subprocess.
interface GitCacheEntry {
  indexMtimeMs: number;
  status: Map<string, string> | null;
}
const gitStatusCache = new Map<string, GitCacheEntry>();

export function invalidateGitStatusCache(cwd?: string): void {
  if (cwd === undefined) gitStatusCache.clear();
  else gitStatusCache.delete(cwd);
}

async function readGitIndexMtime(cwd: string): Promise<number | null> {
  // Walk up looking for a .git directory (or a .git file for worktrees).
  let dir = cwd;
  while (true) {
    const gitPath = path.join(dir, ".git");
    try {
      const st = await fsp.stat(gitPath);
      if (st.isDirectory()) {
        const idxPath = path.join(gitPath, "index");
        try {
          const idxStat = await fsp.stat(idxPath);
          return idxStat.mtimeMs;
        } catch {
          // No index yet (fresh repo) — fall back to HEAD mtime.
          try {
            const headStat = await fsp.stat(path.join(gitPath, "HEAD"));
            return headStat.mtimeMs;
          } catch {
            return null;
          }
        }
      }
      // .git is a file (worktree): read to find the real gitdir, then stat its index.
      const content = await fsp.readFile(gitPath, "utf-8");
      const match = content.match(/^gitdir:\s*(.+)$/m);
      if (!match) return null;
      const gitdir = path.isAbsolute(match[1]) ? match[1] : path.join(dir, match[1]);
      try {
        const idxStat = await fsp.stat(path.join(gitdir, "index"));
        return idxStat.mtimeMs;
      } catch {
        return null;
      }
    } catch {
      // Not here, walk up.
      const parent = path.dirname(dir);
      if (parent === dir) return null;
      dir = parent;
    }
  }
}

export async function loadGitStatus(
  cwd: string,
  procRef?: { current: ReturnType<typeof Bun.spawn> | null },
  options?: { force?: boolean },
): Promise<Map<string, string> | null> {
  const t0 = perfNow();
  const force = options?.force === true;
  const indexMtime = await readGitIndexMtime(cwd);
  if (!force && indexMtime !== null) {
    const cached = gitStatusCache.get(cwd);
    if (cached && cached.indexMtimeMs === indexMtime) {
      logPerf("gitStatus", {
        cwd,
        cached: true,
        entries: cached.status?.size ?? 0,
        ms: Math.round((performance.now() - t0) * 10) / 10,
      });
      return cached.status;
    }
  }

  try {
    const proc = Bun.spawn(["git", "-C", cwd, "status", "--porcelain"], {
      stdout: "pipe",
      stderr: "ignore",
    });
    if (procRef) procRef.current = proc;
    const text = await new Response(proc.stdout).text();
    const exitCode = await proc.exited;
    if (procRef) procRef.current = null;
    if (exitCode !== 0) {
      if (indexMtime !== null) {
        gitStatusCache.set(cwd, { indexMtimeMs: indexMtime, status: null });
      }
      logPerf("gitStatus", {
        cwd,
        cached: false,
        status: "not-git",
        ms: Math.round((performance.now() - t0) * 10) / 10,
      });
      return null;
    }

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
    if (indexMtime !== null) {
      gitStatusCache.set(cwd, { indexMtimeMs: indexMtime, status });
    }
    logPerf("gitStatus", {
      cwd,
      cached: false,
      entries: status.size,
      ms: Math.round((performance.now() - t0) * 10) / 10,
    });
    return status;
  } catch {
    logPerf("gitStatus", {
      cwd,
      cached: false,
      status: "error",
      ms: Math.round((performance.now() - t0) * 10) / 10,
    });
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
