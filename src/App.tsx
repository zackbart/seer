import { useReducer, useEffect, useRef, useCallback } from "react";
import { Box, Text, useApp, useStdout } from "ink";
import path from "path";
import fsp from "fs/promises";

import {
  AppState, Entry, SortMode, PreviewPayload,
  SORT_MODE_COUNT, sortModeLabel,
  FAST_MODE, PREVIEW_DEBOUNCE_MS, SHIKI_FAST_PATH_THRESHOLD_MS,
} from "./types.js";
import { computeWrappedBody, stripAnsi } from "./utils/ansiText.js";
import { colors, cycleTheme } from "./theme.js";
import {
  listDir, applySort, moveToTrash,
  loadGitStatus, previewKey, fuzzyMatch,
} from "./utils/fs.js";
import { copyToClipboard } from "./utils/clipboard.js";
import { openInNewTab } from "./utils/openInTerminal.js";
import { buildPreview, buildPlainPreview, isExpensivePreview } from "./previews/index.js";
import { cacheGet, cacheSet } from "./hooks/usePreviewCache.js";
import { useKeyBindings } from "./hooks/useKeyBindings.js";
import { useMouse } from "./hooks/useMouse.js";
import { TopBar } from "./components/TopBar.js";
import { FileList } from "./components/FileList.js";
import { Preview } from "./components/Preview.js";
import { BottomBar } from "./components/BottomBar.js";
import { DeleteDialog } from "./components/DeleteDialog.js";

// ── action types ─────────────────────────────────────────────────────────────

type Action =
  | { type: "SET_STATE"; payload: Partial<AppState> }
  | { type: "NAVIGATE"; idx: number }
  | { type: "SET_ENTRIES"; all: Entry[]; filtered: Entry[] }
  | { type: "SET_PREVIEW"; payload: PreviewPayload; requestId: number }
  | { type: "SET_PREVIEW_STAGED"; payload: PreviewPayload; requestId: number }
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
      return {
        ...state,
        preview: action.payload.text,
        previewLineCount: action.payload.lineCount,
        previewTokenEstimate: action.payload.tokenEstimate,
        previewTruncated: action.payload.truncated,
        loading: false,
      };
    case "SET_PREVIEW_STAGED":
      // Fast-path: plain text dispatched before the highlighted version is
      // ready. Writes body + metrics but does NOT touch the cache — only the
      // final (highlighted) dispatch writes the cache entry. Clears loading so
      // the user can read the file while Shiki catches up.
      if (action.requestId !== state.requestId) return state;
      return {
        ...state,
        preview: action.payload.text,
        previewLineCount: action.payload.lineCount,
        previewTokenEstimate: action.payload.tokenEstimate,
        previewTruncated: action.payload.truncated,
        loading: false,
      };
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
  const base = Math.max(26, Math.floor(width / 4));
  let leftW = Math.max(16, Math.min(Math.floor(width / 2), base + paneOffset));
  let rightW = width - leftW - 1;
  if (rightW < 20) {
    rightW = 20;
    leftW = width - rightW - 1;
  }
  // Guard against tiny terminals: the narrow-fallback branch can drive leftW
  // or rightW negative if width is small enough that 20 cols don't fit. Clamp
  // both to at least 1 so Ink doesn't get confused.
  leftW = Math.max(1, leftW);
  rightW = Math.max(1, rightW);
  // TopBar=1 row + BottomBar=2 rows = 3 rows of chrome
  const bodyH = Math.max(4, height - 3);
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
    previewLineCount: 0,
    previewTokenEstimate: 0,
    previewTruncated: false,
    loading: false,
    requestId: 0,
    searching: false,
    searchQuery: "",
    confirmingDelete: false,
    deleteTarget: "",
    sortBy: SortMode.NameAsc,
    paneOffset: 0,
    gitStatus: null,
    gitLoadCwd: "",
    previewSelecting: false,
    previewSelStart: { x: 0, y: 0 },
    previewSelEnd: { x: 0, y: 0 },
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

  const gitProcRef = useRef<ReturnType<typeof Bun.spawn> | null>(null);
  const loadGit = useCallback(async (cwd: string, options?: { force?: boolean }) => {
    // Kill any in-flight git process before spawning a new one
    if (gitProcRef.current) {
      gitProcRef.current.kill();
      gitProcRef.current = null;
    }
    const status = await loadGitStatus(cwd, gitProcRef, options);
    dispatch({ type: "SET_GIT_STATUS", cwd, status });
  }, []);

  // Refs for the debounced + cancelable preview pipeline.
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  // Rolling EMA of recent Shiki run times. Gates the fast-path so we only
  // split renders in two when highlighting is actually slow — keeps fast
  // hardware flicker-free.
  const shikiMedianRef = useRef<number>(0);
  const updateShikiMedian = (ms: number) => {
    const prev = shikiMedianRef.current;
    shikiMedianRef.current = prev === 0 ? ms : prev * 0.7 + ms * 0.3;
  };

  const cancelPendingPreview = () => {
    if (debounceTimerRef.current !== null) {
      clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = null;
    }
    if (abortRef.current !== null) {
      abortRef.current.abort();
      abortRef.current = null;
    }
  };

  // Actually runs the buildPreview pipeline and dispatches the result.
  // Writes to cache on success (moved out of the reducer). Respects
  // AbortSignal — dispatches are gated on requestId equality in the reducer.
  const runBuild = useCallback(async (
    entry: Entry,
    width: number,
    height: number,
    cacheKey: string,
    rid: number,
    signal: AbortSignal,
    useFastPath: boolean,
  ) => {
    try {
      if (useFastPath) {
        // Stage 1: plain text, dispatched immediately for perceptibility on
        // slow hardware. No cache write.
        const plain = await buildPlainPreview(entry.path, signal);
        if (signal.aborted) return;
        if (plain) {
          dispatch({ type: "SET_PREVIEW_STAGED", payload: plain, requestId: rid });
        }
      }

      const t0 = performance.now();
      const payload = await buildPreview(entry.path, width, height, signal);
      const elapsed = performance.now() - t0;
      if (signal.aborted) return;

      // Track Shiki time approximation — for code files the bulk of the
      // elapsed work is the highlight call. Good enough to gate the fast-path.
      if (isExpensivePreview(entry.path)) {
        updateShikiMedian(elapsed);
      }

      dispatch({ type: "SET_PREVIEW", payload, requestId: rid });
      // Cache write lives here now rather than inside the reducer — the
      // reducer's stale-check above still applies, so stale dispatches don't
      // write. Only freshly-accepted payloads get cached.
      if (rid === requestIdRef.current) {
        cacheSet(cacheKey, payload);
      }
    } catch (e) {
      if (signal.aborted) return;
      dispatch({
        type: "SET_PREVIEW",
        payload: {
          text: `preview error: ${(e as Error).message}`,
          lineCount: 0,
          tokenEstimate: 0,
          truncated: false,
        },
        requestId: rid,
      });
    }
  }, []);

  const requestPreview = useCallback(async (
    entries: Entry[], selected: number, forceWidth?: number, forceHeight?: number,
  ) => {
    const s = stateRef.current;
    if (entries.length === 0 || selected >= entries.length) {
      cancelPendingPreview();
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

    // Cache hit — dispatch synchronously. No debounce, no abort, no flicker.
    // Rapid j/k through already-visited files stays instant.
    const cached = cacheGet(key);
    if (cached !== undefined) {
      cancelPendingPreview();
      dispatch({
        type: "SET_STATE",
        payload: {
          preview: cached.text,
          previewLineCount: cached.lineCount,
          previewTokenEstimate: cached.tokenEstimate,
          previewTruncated: cached.truncated,
          loading: false,
        },
      });
      return;
    }

    // Cache miss — abort anything in flight and start the new request.
    cancelPendingPreview();
    requestIdRef.current++;
    const rid = requestIdRef.current;
    const controller = new AbortController();
    abortRef.current = controller;

    const expensive = isExpensivePreview(entry.path);

    // Cheap previewers (json/csv/hex/directory/plain text) skip debounce
    // entirely — they're already fast and search-as-you-type would lag
    // otherwise. Clear stale body + show loading so the UI reflects the new
    // file immediately.
    if (!expensive) {
      dispatch({
        type: "SET_STATE",
        payload: {
          preview: "",
          previewLineCount: 0,
          previewTokenEstimate: 0,
          previewTruncated: false,
          loading: true,
          requestId: rid,
        },
      });
      await runBuild(entry, width, height, key, rid, controller.signal, false);
      return;
    }

    // Expensive previewer: hold the stale preview body visible for the
    // debounce window (no blank flash), then fire the real pipeline. The
    // shared requestId lets rapid navigation cancel in-flight work.
    dispatch({
      type: "SET_STATE",
      payload: {
        loading: true,
        requestId: rid,
      },
    });

    const useFastPath = !FAST_MODE && shikiMedianRef.current > SHIKI_FAST_PATH_THRESHOLD_MS;

    debounceTimerRef.current = setTimeout(async () => {
      debounceTimerRef.current = null;
      if (controller.signal.aborted) return;
      // Clear stale body now that the debounce has committed.
      dispatch({
        type: "SET_STATE",
        payload: {
          preview: "",
          previewLineCount: 0,
          previewTokenEstimate: 0,
          previewTruncated: false,
          loading: true,
          requestId: rid,
        },
      });
      await runBuild(entry, width, height, key, rid, controller.signal, useFastPath);
    }, PREVIEW_DEBOUNCE_MS);
  }, [runBuild]);

  const navigate = useCallback((idx: number) => {
    dispatch({ type: "NAVIGATE", idx });
    const s = stateRef.current;
    requestPreview(s.entries, idx);
  }, [requestPreview]);

  const changeDir = useCallback(async (newPath: string) => {
    const s = stateRef.current;
    try {
      const raw = await listDir(newPath, s.showHidden);
      const sorted = applySort(raw, s.sortBy);
      const filtered = applySearch(sorted, s.searchQuery);

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

      // Navigation — keyboard always controls file list
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
        // Preserve selection by name
        const prevName = s.selected < s.entries.length ? s.entries[s.selected].name : "";
        let newSel = 0;
        for (let i = 0; i < filtered.length; i++) {
          if (filtered[i].name === prevName) { newSel = i; break; }
        }
        if (newSel >= filtered.length) newSel = Math.max(0, filtered.length - 1);
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

      // Theme
      case "t": {
        const name = cycleTheme();
        dispatch({ type: "SET_STATE", payload: { status: `theme: ${name}` } });
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

      // Reload
      case "R": {
        const filtered = await reloadDir();
        dispatch({ type: "SET_STATE", payload: { status: "reloaded" } });
        if (filtered) requestPreview(filtered, stateRef.current.selected);
        loadGit(s.cwd, { force: true });
        return;
      }

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

      // Edit in new terminal tab (nano)
      case "e":
        if (s.entries.length > 0 && s.selected < s.entries.length) {
          const entry = s.entries[s.selected];
          if (entry.isDir) {
            dispatch({ type: "SET_STATE", payload: { status: "cannot edit directories" } });
            return;
          }
          try {
            await openInNewTab(entry.path, "nano");
            dispatch({ type: "SET_STATE", payload: { status: `opened in nano: ${entry.name}` } });
          } catch (err) {
            dispatch({ type: "SET_STATE", payload: { status: `edit failed: ${(err as Error).message}` } });
          }
        }
        return;
    }
  }, [exit, navigate, changeDir, reloadDir, requestPreview, loadGit, cwdFile]);

  // ── delete helper ────────────────────────────────────────────────────

  const handleDelete = useCallback(async () => {
    const s = stateRef.current;
    const target = s.deleteTarget;
    if (!target) return;

    try {
      await moveToTrash(target);
      dispatch({
        type: "SET_STATE",
        payload: {
          confirmingDelete: false,
          deleteTarget: "",
          preview: "",
          status: `moved to trash: ${path.basename(target)}`,
        },
      });
    } catch (e) {
      dispatch({
        type: "SET_STATE",
        payload: {
          confirmingDelete: false,
          deleteTarget: "",
          status: `trash failed: ${(e as Error).message}`,
        },
      });
    }

    const filtered = await reloadDir();
    if (filtered) requestPreview(filtered, stateRef.current.selected);
    // Tree changed on disk — force-refresh git status so badges on the
    // surrounding files don't show the pre-delete state.
    loadGit(stateRef.current.cwd, { force: true });
  }, [reloadDir, requestPreview, loadGit]);

  // Register key handler
  useKeyBindings(handleKey);

  // Mouse handler — scroll + preview text selection
  const handleMouse = useCallback((event: import("./hooks/useMouse.js").MouseEvent) => {
    const s = stateRef.current;

    // Scroll wheel — scroll whichever pane the mouse is over
    if (event.button === 64 || event.button === 65) {
      const direction = event.button === 64 ? "up" : "down";
      const { leftW, rightW, bodyH } = layoutDimensions(s.width, s.height, s.paneOffset);
      const overPreview = event.x > leftW;

      if (overPreview) {
        // Use the same wrap/layout math as Preview.tsx so scroll clamps to
        // the real bottom of the wrapped content. innerW = rightW - border(2);
        // contentBodyH = bodyH - border(2) - header(1) - divider(1).
        const innerW = Math.max(1, rightW - 2);
        const contentBodyH = Math.max(1, bodyH - 4);
        const { maxOffset } = computeWrappedBody(s.preview, innerW, contentBodyH, s.previewOffset);
        // Scroll by 3 rows per wheel tick — single-row steps feel sluggish
        // on long files.
        const step = 3;
        const newOff = direction === "down"
          ? Math.min(s.previewOffset + step, maxOffset)
          : Math.max(s.previewOffset - step, 0);
        dispatch({ type: "SET_STATE", payload: { previewOffset: newOff } });
      } else {
        const delta = direction === "down" ? 1 : -1;
        const newSel = Math.max(0, Math.min(s.selected + delta, s.entries.length - 1));
        if (newSel !== s.selected) {
          // Route through navigate() so wheel scrolling honors the same
          // debounce + abort machinery as keyboard nav. Without this, holding
          // the scroll wheel would fire unbounded buildPreview calls.
          navigate(newSel);
        }
      }
      return;
    }

    // Preview text selection — left button only
    if (event.button !== 0) return;

    const { leftW, rightW, bodyH } = layoutDimensions(s.width, s.height, s.paneOffset);
    // Preview body rect — SGR mouse coords are 1-based
    // X: leftPane(leftW) + separator(1) + previewBorder(1) + 1 for 1-based
    const pStartX = leftW + 3;
    // Y: topbar(1) + paneBorder(1) + header(1) + divider(1) + 1 for 1-based
    const pStartY = 5;
    const pWidth = Math.max(1, rightW - 2);
    const pHeight = Math.max(1, bodyH - 4);

    const toPoint = (mx: number, my: number) => ({
      x: Math.max(0, Math.min(mx - pStartX, pWidth - 1)),
      y: Math.max(0, Math.min(my - pStartY, pHeight - 1)),
    });

    const inPreviewBody = event.x >= pStartX && event.x < pStartX + pWidth
      && event.y >= pStartY && event.y < pStartY + pHeight;

    if (event.action === "press" && inPreviewBody) {
      const p = toPoint(event.x, event.y);
      dispatch({
        type: "SET_STATE",
        payload: { previewSelecting: true, previewSelStart: p, previewSelEnd: p },
      });
    } else if (event.action === "drag" && s.previewSelecting) {
      const p = toPoint(event.x, event.y);
      dispatch({ type: "SET_STATE", payload: { previewSelEnd: p } });
    } else if (event.action === "release" && s.previewSelecting) {
      const p = toPoint(event.x, event.y);
      const start = s.previewSelStart;
      const end = p;

      dispatch({
        type: "SET_STATE",
        payload: { previewSelecting: false, previewSelEnd: end },
      });

      // Normalize start/end
      let s0 = start, s1 = end;
      if (s0.y > s1.y || (s0.y === s1.y && s0.x > s1.x)) {
        [s0, s1] = [s1, s0];
      }
      if (s0.x === s1.x && s0.y === s1.y) return; // no selection

      // Extract visible preview lines (plain text)
      const lines = getVisiblePreviewLines(s, pWidth, pHeight);
      const selected: string[] = [];
      for (let row = s0.y; row <= s1.y; row++) {
        const line = row < lines.length ? lines[row] : "";
        const colStart = row === s0.y ? s0.x : 0;
        const colEnd = row === s1.y ? s1.x : pWidth;
        selected.push(line.slice(colStart, colEnd));
      }
      const text = selected.join("\n").trimEnd();
      if (!text) return;

      copyToClipboard(text)
        .then(() => dispatch({ type: "SET_STATE", payload: { status: `copied ${text.length} chars` } }))
        .catch((e) => dispatch({ type: "SET_STATE", payload: { status: `copy failed: ${(e as Error).message}` } }));
    }
  }, [navigate]);

  useMouse(handleMouse);

  // ── render ───────────────────────────────────────────────────────────

  if (state.width === 0 || state.height === 0) {
    return null;
  }

  const { leftW, rightW, bodyH } = layoutDimensions(state.width, state.height, state.paneOffset);

  // Separator: single string, stable child count
  const sepStr = Array.from({ length: bodyH }, () => "│").join("\n");

  // Overlays
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
          <Text color={colors.dim}>{sepStr}</Text>
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

// Get the plain-text visible preview lines (matching what Preview.tsx renders).
// Delegates to computeWrappedBody so row coordinates line up with the render
// path — click-drag selection uses the row indices this returns.
function getVisiblePreviewLines(state: AppState, width: number, height: number): string[] {
  let body = state.preview;
  if (!body && !state.loading) body = "  no preview";
  if (state.loading && !body) body = "  loading…";

  const { offset, scrollRow, visibleLines } = computeWrappedBody(body, width, height, state.previewOffset);

  const result: string[] = [];
  if (scrollRow > 0) {
    result.push(`  ↑ line ${offset + 1}`);
  }
  for (const line of visibleLines) {
    // Selection extraction wants plain text; strip ANSI here so downstream
    // slice(colStart, colEnd) measures in visible columns.
    result.push(stripAnsi(line));
  }
  while (result.length < height) result.push("");
  return result.slice(0, height);
}
