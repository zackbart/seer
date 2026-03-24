import { useReducer, useEffect, useRef, useCallback } from "react";
import { Box, Text, useApp, useStdout } from "ink";
import path from "path";
import fsp from "fs/promises";

import {
  AppState, Entry, InputMode, YankMode, SortMode,
  SORT_MODE_COUNT, sortModeLabel,
} from "./types.js";
import { colors } from "./theme.js";
import chalk from "chalk";
import {
  listDir, applySort, moveToTrash, pasteEntries,
  loadGitStatus, previewKey, fuzzyMatch,
} from "./utils/fs.js";
import { copyToClipboard } from "./utils/clipboard.js";
import { previewPageSize } from "./utils/humanSize.js";
import { buildPreview } from "./previews/index.js";
import { cacheGet, cacheSet } from "./hooks/usePreviewCache.js";
import { useKeyBindings } from "./hooks/useKeyBindings.js";
import { TopBar } from "./components/TopBar.js";
import { FileList } from "./components/FileList.js";
import { Preview } from "./components/Preview.js";
import { BottomBar } from "./components/BottomBar.js";
import { InputDialog } from "./components/InputDialog.js";
import { DeleteDialog } from "./components/DeleteDialog.js";

// ── action types ─────────────────────────────────────────────────────────────

type Action =
  | { type: "SET_STATE"; payload: Partial<AppState> }
  | { type: "NAVIGATE"; idx: number }
  | { type: "SET_ENTRIES"; all: Entry[]; filtered: Entry[] }
  | { type: "SET_PREVIEW"; content: string; requestId: number; cacheKey: string }
  | { type: "SET_GIT_STATUS"; cwd: string; status: Map<string, string> | null }
  | { type: "RESIZE"; width: number; height: number };

function reducer(state: AppState, action: Action): AppState {
  switch (action.type) {
    case "SET_STATE":
      return { ...state, ...action.payload };
    case "NAVIGATE":
      return { ...state, selected: action.idx, previewOffset: 0 };
    case "SET_ENTRIES":
      return { ...state, allEntries: action.all, entries: action.filtered };
    case "SET_PREVIEW":
      if (action.requestId !== state.requestId) return state;
      cacheSet(action.cacheKey, action.content);
      return { ...state, preview: action.content, loading: false };
    case "SET_GIT_STATUS":
      if (action.cwd !== state.cwd) return state;
      return { ...state, gitStatus: action.status, gitLoadCwd: action.cwd };
    case "RESIZE":
      return { ...state, width: action.width, height: action.height };
    default:
      return state;
  }
}

// ── layout ───────────────────────────────────────────────────────────────────

export function layoutDimensions(width: number, height: number, paneOffset: number) {
  const base = Math.max(26, Math.floor(width / 3));
  let leftW = Math.max(16, Math.min(Math.floor(width / 2), base + paneOffset));
  let rightW = width - leftW - 1;
  if (rightW < 20) {
    rightW = 20;
    leftW = width - rightW - 1;
  }
  const bodyH = Math.max(4, height - 4);
  return { leftW, rightW, bodyH };
}

// ── initial state factory ────────────────────────────────────────────────────

function createInitialState(cwd: string): AppState {
  return {
    cwd,
    allEntries: [],
    entries: [],
    selected: 0,
    showHidden: false,
    preview: "",
    status: "loading…",
    width: 0,
    height: 0,
    previewOffset: 0,
    loading: false,
    requestId: 0,
    searching: false,
    searchQuery: "",
    confirmingDelete: false,
    deleteTarget: "",
    inputMode: InputMode.None,
    inputValue: "",
    yankPaths: [],
    yankOp: YankMode.None,
    multiSelected: new Set(),
    sortBy: SortMode.NameAsc,
    dirHistory: [cwd],
    historyPos: 0,
    bookmarks: [],
    paneOffset: 0,
    gitStatus: null,
    gitLoadCwd: "",
  };
}

// ── App component ────────────────────────────────────────────────────────────

interface AppProps {
  startDir: string;
  cwdFile?: string;
}

export function App({ startDir, cwdFile }: AppProps) {
  const { exit } = useApp();
  const { stdout } = useStdout();
  const [state, dispatch] = useReducer(reducer, startDir, createInitialState);
  const stateRef = useRef(state);
  stateRef.current = state;
  const requestIdRef = useRef(0);

  // Terminal size
  useEffect(() => {
    const update = () => {
      dispatch({
        type: "RESIZE",
        width: stdout.columns ?? 80,
        height: stdout.rows ?? 24,
      });
    };
    update();
    stdout.on("resize", update);
    return () => { stdout.off("resize", update); };
  }, [stdout]);

  // Initial load
  useEffect(() => {
    (async () => {
      try {
        const entries = await listDir(startDir, false);
        dispatch({ type: "SET_ENTRIES", all: entries, filtered: entries });
        dispatch({ type: "SET_STATE", payload: { status: "ready" } });
        requestPreview(entries, 0);
        loadGit(startDir);
      } catch (e) {
        dispatch({ type: "SET_STATE", payload: { status: (e as Error).message } });
      }
    })();
  }, []);

  // ── helpers ──────────────────────────────────────────────────────────────

  const loadGit = useCallback(async (cwd: string) => {
    const status = await loadGitStatus(cwd);
    dispatch({ type: "SET_GIT_STATUS", cwd, status });
  }, []);

  const requestPreview = useCallback(async (
    entries: Entry[], selected: number, forceWidth?: number, forceHeight?: number,
  ) => {
    const s = stateRef.current;
    if (entries.length === 0 || selected >= entries.length) {
      dispatch({ type: "SET_STATE", payload: { preview: "", loading: false } });
      return;
    }
    const entry = entries[selected];
    const w = forceWidth ?? s.width;
    const h = forceHeight ?? s.height;
    const { rightW, bodyH } = layoutDimensions(w, h, s.paneOffset);
    const width = Math.max(40, rightW - 2);
    const height = Math.max(8, bodyH);
    const key = previewKey(entry.path, entry.modTime, entry.size, width, height);

    const cached = cacheGet(key);
    if (cached !== undefined) {
      dispatch({ type: "SET_STATE", payload: { preview: cached, loading: false } });
      return;
    }

    requestIdRef.current++;
    const rid = requestIdRef.current;
    dispatch({ type: "SET_STATE", payload: { loading: true, requestId: rid } });

    try {
      const content = await buildPreview(entry.path, width, height);
      dispatch({ type: "SET_PREVIEW", content, requestId: rid, cacheKey: key });
    } catch (e) {
      dispatch({
        type: "SET_PREVIEW",
        content: `preview error: ${(e as Error).message}`,
        requestId: rid,
        cacheKey: key,
      });
    }
  }, []);

  const navigate = useCallback((idx: number) => {
    dispatch({ type: "NAVIGATE", idx });
    const s = stateRef.current;
    requestPreview(s.entries, idx);
  }, [requestPreview]);

  const changeDir = useCallback(async (newPath: string, pushHistory = true) => {
    const s = stateRef.current;
    try {
      const raw = await listDir(newPath, s.showHidden);
      const sorted = applySort(raw, s.sortBy);
      const filtered = applySearch(sorted, s.searchQuery);

      let history = [...s.dirHistory];
      let histPos = s.historyPos;
      if (pushHistory) {
        if (histPos < history.length - 1) history = history.slice(0, histPos + 1);
        if (history.length === 0 || history[history.length - 1] !== newPath) {
          history.push(newPath);
        }
        histPos = history.length - 1;
      }

      dispatch({
        type: "SET_STATE",
        payload: {
          cwd: newPath,
          allEntries: sorted,
          entries: filtered,
          selected: 0,
          previewOffset: 0,
          searchQuery: "",
          searching: false,
          status: newPath,
          multiSelected: new Set(),
          dirHistory: history,
          historyPos: histPos,
        },
      });

      requestPreview(filtered, 0);
      loadGit(newPath);
    } catch (e) {
      dispatch({ type: "SET_STATE", payload: { status: (e as Error).message } });
    }
  }, [requestPreview, loadGit]);

  const reloadDir = useCallback(async () => {
    const s = stateRef.current;
    try {
      const raw = await listDir(s.cwd, s.showHidden);
      const sorted = applySort(raw, s.sortBy);
      const filtered = applySearch(sorted, s.searchQuery);

      // Preserve selection by name
      let prevName = "";
      if (s.selected < s.entries.length) prevName = s.entries[s.selected].name;
      let newSel = 0;
      for (let i = 0; i < filtered.length; i++) {
        if (filtered[i].name === prevName) { newSel = i; break; }
      }
      if (newSel >= filtered.length) newSel = Math.max(0, filtered.length - 1);

      dispatch({
        type: "SET_STATE",
        payload: {
          allEntries: sorted,
          entries: filtered,
          selected: newSel,
        },
      });
      return filtered;
    } catch (e) {
      dispatch({ type: "SET_STATE", payload: { status: (e as Error).message } });
      return null;
    }
  }, []);

  // ── key handler ────────────────────────────────────────────────────────

  const handleKey = useCallback(async (key: string, raw: string) => {
    const s = stateRef.current;

    // ── input mode ─────────────────────────────────────────────────────
    if (s.inputMode !== InputMode.None) {
      if (key === "esc") {
        dispatch({ type: "SET_STATE", payload: { inputMode: InputMode.None, inputValue: "", status: "cancelled" } });
        return;
      }
      if (key === "enter") {
        await confirmInput();
        return;
      }
      if (key === "backspace") {
        const runes = [...s.inputValue];
        if (runes.length > 0) {
          dispatch({ type: "SET_STATE", payload: { inputValue: runes.slice(0, -1).join("") } });
        }
        return;
      }
      if (raw.length === 1) {
        dispatch({ type: "SET_STATE", payload: { inputValue: s.inputValue + raw } });
      }
      return;
    }

    // ── delete confirmation ────────────────────────────────────────────
    if (s.confirmingDelete) {
      if (key === "y" || key === "Y" || key === "enter") {
        await handleDelete();
        return;
      }
      if (key === "n" || key === "N" || key === "esc") {
        dispatch({ type: "SET_STATE", payload: { confirmingDelete: false, deleteTarget: "", status: "delete cancelled" } });
        return;
      }
      return;
    }

    // ── search mode typing ─────────────────────────────────────────────
    if (s.searching && raw.length === 1 && key !== "esc" && key !== "backspace" && key !== "enter") {
      const newQuery = s.searchQuery + raw;
      const filtered = applySearch(s.allEntries, newQuery);
      dispatch({
        type: "SET_STATE",
        payload: { searchQuery: newQuery, entries: filtered, selected: 0 },
      });
      requestPreview(filtered, 0);
      return;
    }

    // ── normal keys ────────────────────────────────────────────────────
    switch (key) {
      case "q":
      case "ctrl+c": {
        if (cwdFile) {
          try { await fsp.writeFile(cwdFile, s.cwd + "\n"); } catch {}
        }
        exit();
        return;
      }

      // Navigation
      case "j":
      case "down":
        if (s.selected < s.entries.length - 1) navigate(s.selected + 1);
        return;
      case "k":
      case "up":
        if (s.selected > 0) navigate(s.selected - 1);
        return;
      case "g":
        navigate(0);
        return;
      case "G":
        if (s.entries.length > 0) navigate(s.entries.length - 1);
        return;
      case "l":
      case "right":
      case "enter":
        if (s.entries.length > 0 && s.selected < s.entries.length) {
          const picked = s.entries[s.selected];
          if (picked.isDir) {
            await changeDir(picked.path);
          }
        }
        return;
      case "h":
      case "left":
        if (s.searching) return;
        {
          const parent = path.dirname(s.cwd);
          if (parent !== s.cwd) {
            const childName = path.basename(s.cwd);
            await changeDir(parent);
            // Re-select the directory we came from
            const curr = stateRef.current;
            for (let i = 0; i < curr.entries.length; i++) {
              if (curr.entries[i].name === childName) {
                navigate(i);
                break;
              }
            }
          }
        }
        return;

      // History
      case "alt+left":
        if (s.historyPos > 0) {
          const dest = s.dirHistory[s.historyPos - 1];
          dispatch({ type: "SET_STATE", payload: { historyPos: s.historyPos - 1 } });
          await changeDir(dest, false);
        }
        return;
      case "alt+right":
        if (s.historyPos < s.dirHistory.length - 1) {
          const dest = s.dirHistory[s.historyPos + 1];
          dispatch({ type: "SET_STATE", payload: { historyPos: s.historyPos + 1 } });
          await changeDir(dest, false);
        }
        return;

      // Preview scroll
      case "ctrl+d":
      case "pagedown": {
        const newOffset = Math.min(
          s.previewOffset + previewPageSize(s.height),
          Math.max(0, s.preview.split("\n").length - Math.max(1, s.height - 8)),
        );
        dispatch({ type: "SET_STATE", payload: { previewOffset: newOffset } });
        return;
      }
      case "ctrl+u":
      case "pageup": {
        const newOffset = Math.max(0, s.previewOffset - previewPageSize(s.height));
        dispatch({ type: "SET_STATE", payload: { previewOffset: newOffset } });
        return;
      }

      // Search
      case "/":
        dispatch({ type: "SET_STATE", payload: { searching: true, searchQuery: "" } });
        return;
      case "esc":
        if (s.searching) {
          dispatch({
            type: "SET_STATE",
            payload: { searching: false, searchQuery: "", entries: s.allEntries, selected: 0 },
          });
          requestPreview(s.allEntries, 0);
        }
        return;
      case "backspace":
        if (s.searching) {
          const runes = [...s.searchQuery];
          if (runes.length > 0) {
            const newQuery = runes.slice(0, -1).join("");
            const filtered = applySearch(s.allEntries, newQuery);
            dispatch({
              type: "SET_STATE",
              payload: { searchQuery: newQuery, entries: filtered, selected: 0 },
            });
            requestPreview(filtered, 0);
          }
          return;
        }
        // Fall through to delete
        if (s.entries.length > 0 && s.selected < s.entries.length) {
          dispatch({
            type: "SET_STATE",
            payload: {
              confirmingDelete: true,
              deleteTarget: s.entries[s.selected].path,
              status: "confirm move to trash",
            },
          });
        }
        return;

      // Hidden files
      case ".": {
        const newHidden = !s.showHidden;
        const raw = await listDir(s.cwd, newHidden);
        const sorted = applySort(raw, s.sortBy);
        const filtered = applySearch(sorted, s.searchQuery);
        let prevName = s.selected < s.entries.length ? s.entries[s.selected].name : "";
        let newSel = 0;
        for (let i = 0; i < filtered.length; i++) {
          if (filtered[i].name === prevName) { newSel = i; break; }
        }
        dispatch({
          type: "SET_STATE",
          payload: {
            showHidden: newHidden,
            allEntries: sorted,
            entries: filtered,
            selected: newSel,
            previewOffset: 0,
            status: newHidden ? "showing hidden files" : "hiding hidden files",
          },
        });
        requestPreview(filtered, newSel);
        return;
      }

      // Sort
      case "s": {
        const newSort = ((s.sortBy + 1) % SORT_MODE_COUNT) as SortMode;
        const sorted = applySort(s.allEntries, newSort);
        const filtered = applySearch(sorted, s.searchQuery);
        let newSel = Math.min(s.selected, Math.max(0, filtered.length - 1));
        dispatch({
          type: "SET_STATE",
          payload: {
            sortBy: newSort,
            allEntries: sorted,
            entries: filtered,
            selected: newSel,
            status: "sort: " + sortModeLabel[newSort],
          },
        });
        requestPreview(filtered, newSel);
        return;
      }

      // Pane resize
      case "<": {
        const minOffset = -(Math.floor(s.width / 3) - 16);
        const newOffset = Math.max(minOffset, s.paneOffset - 2);
        dispatch({ type: "SET_STATE", payload: { paneOffset: newOffset } });
        requestPreview(s.entries, s.selected);
        return;
      }
      case ">": {
        const maxOffset = Math.floor(s.width / 4);
        const newOffset = Math.min(maxOffset, s.paneOffset + 2);
        dispatch({ type: "SET_STATE", payload: { paneOffset: newOffset } });
        requestPreview(s.entries, s.selected);
        return;
      }

      // Multi-select
      case "space":
        if (s.entries.length > 0 && s.selected < s.entries.length) {
          const p = s.entries[s.selected].path;
          const newSet = new Set(s.multiSelected);
          if (newSet.has(p)) newSet.delete(p); else newSet.add(p);
          dispatch({ type: "SET_STATE", payload: { multiSelected: newSet } });
          if (s.selected < s.entries.length - 1) navigate(s.selected + 1);
        }
        return;

      // Yank / cut / paste
      case "y": {
        const paths = selectedPaths(s);
        if (paths.length > 0) {
          dispatch({
            type: "SET_STATE",
            payload: {
              yankPaths: paths,
              yankOp: YankMode.Copy,
              multiSelected: new Set(),
              status: paths.length === 1
                ? `yanked: ${path.basename(paths[0])}`
                : `yanked ${paths.length} files`,
            },
          });
        }
        return;
      }
      case "x": {
        const paths = selectedPaths(s);
        if (paths.length > 0) {
          dispatch({
            type: "SET_STATE",
            payload: {
              yankPaths: paths,
              yankOp: YankMode.Cut,
              multiSelected: new Set(),
              status: paths.length === 1
                ? `cut: ${path.basename(paths[0])}`
                : `cut ${paths.length} files`,
            },
          });
        }
        return;
      }
      case "P": {
        if (s.yankPaths.length === 0 || s.yankOp === YankMode.None) {
          dispatch({ type: "SET_STATE", payload: { status: "nothing to paste (yank with y or x)" } });
          return;
        }
        try {
          const n = await pasteEntries(s.yankPaths, s.cwd, s.yankOp);
          const payload: Partial<AppState> = { status: `pasted ${n} item(s)` };
          if (s.yankOp === YankMode.Cut) {
            payload.yankPaths = [];
            payload.yankOp = YankMode.None;
          }
          dispatch({ type: "SET_STATE", payload });
          await reloadDir();
          requestPreview(stateRef.current.entries, stateRef.current.selected);
        } catch (e) {
          dispatch({ type: "SET_STATE", payload: { status: `paste failed: ${(e as Error).message}` } });
        }
        return;
      }

      // Rename
      case "r":
        if (s.entries.length > 0 && s.selected < s.entries.length) {
          dispatch({
            type: "SET_STATE",
            payload: { inputMode: InputMode.Rename, inputValue: s.entries[s.selected].name },
          });
        }
        return;

      // Reload
      case "R": {
        const filtered = await reloadDir();
        dispatch({ type: "SET_STATE", payload: { status: "reloaded" } });
        if (filtered) requestPreview(filtered, stateRef.current.selected);
        loadGit(s.cwd);
        return;
      }

      // New file / dir
      case "n":
        dispatch({ type: "SET_STATE", payload: { inputMode: InputMode.NewFile, inputValue: "" } });
        return;
      case "N":
        dispatch({ type: "SET_STATE", payload: { inputMode: InputMode.NewDir, inputValue: "" } });
        return;

      // Open in editor
      case "e":
        if (s.entries.length > 0 && s.selected < s.entries.length) {
          const entry = s.entries[s.selected];
          if (entry.isDir) {
            dispatch({ type: "SET_STATE", payload: { status: "can't edit a directory" } });
            return;
          }
          const editor = process.env.EDITOR ?? process.env.VISUAL ?? "";
          if (!editor) {
            dispatch({ type: "SET_STATE", payload: { status: "$EDITOR not set" } });
            return;
          }
          // Spawn editor synchronously, inheriting stdio
          Bun.spawnSync([editor, entry.path], {
            stdin: "inherit",
            stdout: "inherit",
            stderr: "inherit",
          });
          dispatch({ type: "SET_STATE", payload: { status: "returned from editor" } });
          await reloadDir();
          requestPreview(stateRef.current.entries, stateRef.current.selected);
        }
        return;

      // Copy path
      case "p":
        if (s.entries.length > 0 && s.selected < s.entries.length) {
          const p = s.entries[s.selected].path;
          try {
            await copyToClipboard(p);
            dispatch({ type: "SET_STATE", payload: { status: `copied path: ${p}` } });
          } catch (e) {
            dispatch({ type: "SET_STATE", payload: { status: `copy failed: ${(e as Error).message}` } });
          }
        }
        return;

      // Bookmarks
      case "b": {
        const bookmarks = [...s.bookmarks];
        const idx = bookmarks.indexOf(s.cwd);
        if (idx >= 0) {
          bookmarks.splice(idx, 1);
          dispatch({ type: "SET_STATE", payload: { bookmarks, status: `bookmark #${idx + 1} removed` } });
        } else {
          if (bookmarks.length >= 9) bookmarks.shift();
          bookmarks.push(s.cwd);
          dispatch({ type: "SET_STATE", payload: { bookmarks, status: `bookmarked at #${bookmarks.length}` } });
        }
        return;
      }
      case "1": case "2": case "3": case "4": case "5":
      case "6": case "7": case "8": case "9": {
        const n = parseInt(key);
        if (n >= 1 && n <= s.bookmarks.length) {
          await changeDir(s.bookmarks[n - 1]);
        } else {
          dispatch({ type: "SET_STATE", payload: { status: `no bookmark #${n}` } });
        }
        return;
      }
    }
  }, [exit, navigate, changeDir, reloadDir, requestPreview, loadGit, cwdFile]);

  // ── confirm input helper ─────────────────────────────────────────────

  const confirmInput = useCallback(async () => {
    const s = stateRef.current;
    const name = s.inputValue.trim();
    if (!name) {
      dispatch({ type: "SET_STATE", payload: { inputMode: InputMode.None, inputValue: "", status: "cancelled" } });
      return;
    }

    const mode = s.inputMode;
    dispatch({ type: "SET_STATE", payload: { inputMode: InputMode.None, inputValue: "" } });

    try {
      switch (mode) {
        case InputMode.Rename:
          if (s.selected < s.entries.length) {
            const old = s.entries[s.selected].path;
            const newPath = path.join(s.cwd, name);
            await fsp.rename(old, newPath);
            dispatch({ type: "SET_STATE", payload: { status: `renamed → ${name}` } });
          }
          break;
        case InputMode.NewFile: {
          const newPath = path.join(s.cwd, name);
          await fsp.writeFile(newPath, "");
          dispatch({ type: "SET_STATE", payload: { status: `created ${name}` } });
          break;
        }
        case InputMode.NewDir: {
          const newPath = path.join(s.cwd, name);
          await fsp.mkdir(newPath, { recursive: true });
          dispatch({ type: "SET_STATE", payload: { status: `created ${name}/` } });
          break;
        }
      }
    } catch (e) {
      dispatch({ type: "SET_STATE", payload: { status: `failed: ${(e as Error).message}` } });
    }

    const filtered = await reloadDir();
    if (filtered) {
      // Try to select the named entry
      for (let i = 0; i < filtered.length; i++) {
        if (filtered[i].name === name) {
          dispatch({ type: "SET_STATE", payload: { selected: i } });
          break;
        }
      }
      requestPreview(filtered, stateRef.current.selected);
    }
  }, [reloadDir, requestPreview]);

  // ── delete helper ────────────────────────────────────────────────────

  const handleDelete = useCallback(async () => {
    const s = stateRef.current;
    const targets = s.multiSelected.size > 0
      ? [...s.multiSelected]
      : s.deleteTarget ? [s.deleteTarget] : [];

    let deleted = 0;
    let lastErr: Error | null = null;
    for (const t of targets) {
      try { await moveToTrash(t); deleted++; } catch (e) { lastErr = e as Error; }
    }

    dispatch({
      type: "SET_STATE",
      payload: {
        confirmingDelete: false,
        deleteTarget: "",
        multiSelected: new Set(),
        preview: "",
        status: lastErr ? `trash failed: ${lastErr.message}` : `moved ${deleted} to trash`,
      },
    });

    const filtered = await reloadDir();
    if (filtered) requestPreview(filtered, stateRef.current.selected);
  }, [reloadDir, requestPreview]);

  // Register key handler
  useKeyBindings(handleKey);

  // ── render ───────────────────────────────────────────────────────────

  if (state.width === 0 || state.height === 0) {
    return null;
  }

  const { leftW, rightW, bodyH } = layoutDimensions(state.width, state.height, state.paneOffset);

  // Separator: single chalk string, stable child count
  const sepChar = chalk.ansi256(Number(colors.dim))("│");
  const sepStr = Array.from({ length: bodyH }, () => sepChar).join("\n");

  // Overlays
  if (state.inputMode !== InputMode.None) {
    return (
      <Box flexDirection="column" width={state.width} height={state.height}>
        <TopBar state={state} width={state.width} />
        <InputDialog state={state} width={state.width} height={bodyH} />
        <BottomBar state={state} width={state.width} />
      </Box>
    );
  }

  if (state.confirmingDelete) {
    return (
      <Box flexDirection="column" width={state.width} height={state.height}>
        <TopBar state={state} width={state.width} />
        <DeleteDialog state={state} width={state.width} height={bodyH} />
        <BottomBar state={state} width={state.width} />
      </Box>
    );
  }

  return (
    <Box flexDirection="column" width={state.width} height={state.height}>
      <TopBar state={state} width={state.width} />
      <Box flexDirection="row" height={bodyH}>
        <FileList state={state} width={leftW} height={bodyH} />
        <Box width={1} height={bodyH}>
          <Text>{sepStr}</Text>
        </Box>
        <Preview state={state} width={rightW} height={bodyH} />
      </Box>
      <BottomBar state={state} width={state.width} />
    </Box>
  );
}

// ── helpers ──────────────────────────────────────────────────────────────────

function applySearch(entries: Entry[], query: string): Entry[] {
  if (!query) return entries;
  return entries.filter((e) => fuzzyMatch(e.name, query));
}

function selectedPaths(state: AppState): string[] {
  if (state.multiSelected.size > 0) return [...state.multiSelected];
  if (state.entries.length > 0 && state.selected < state.entries.length) {
    return [state.entries[state.selected].path];
  }
  return [];
}
