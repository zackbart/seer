# Agent Instructions

## Project Overview

Seer is a TypeScript/React TUI file browser with a two-pane layout (directory listing + live file preview), inspired by Yazi. Version 1.0.12.

## Tech Stack

- **Bun** — runtime and bundler
- **React 18 + Ink 5** — TUI framework (React components rendered to terminal)
- **Shiki (`shiki/core` + JS regex engine)** — syntax highlighting (nord theme). Uses `createHighlighterCore` with explicit per-grammar dynamic imports from `@shikijs/langs/*` rather than the full `shiki` package, so only the languages a user actually opens ever get materialized.
- **Chalk** — terminal color support
- **Marked + marked-terminal** — Markdown rendering (lazy-loaded)
- **mammoth** — `.docx` text extraction (lazy-loaded)
- **exceljs** — `.xlsx` parsing (lazy-loaded)
- **unpdf** — PDF text extraction, pdfjs-dist wrapper, Bun-friendly (lazy-loaded)

## Build & Run

```bash
bun run start             # Run directly
bun run build             # Compile to standalone binary
bun run typecheck         # Type-check (tsc --noEmit)
```

## Architecture

All source code lives in `src/`, organized as:

| File/Dir | Contents |
|---|---|
| `src/index.tsx` | Entry point, CLI flags, alt-screen setup |
| `src/App.tsx` | Main component, useReducer state, key/mouse handlers, layout |
| `src/types.ts` | Enums, `Entry`, `AppState` interface, constants |
| `src/theme.ts` | 9 color themes, persistence, icons, file categorization |
| `src/components/TopBar.tsx` | Breadcrumb path + badges |
| `src/components/FileList.tsx` | Directory listing with git badges |
| `src/components/Preview.tsx` | File preview with selection highlighting |
| `src/components/BottomBar.tsx` | Status line + key hints |
| `src/components/DeleteDialog.tsx` | Trash confirmation dialog |
| `src/previews/index.ts` | Preview dispatcher — dynamic-imports each previewer on first use, threads `AbortSignal` through the pipeline, applies `SEER_FAST_MODE` guards. Also exports `buildPlainPreview` (fast-path stage 1) and `isExpensivePreview` classifier. |
| `src/previews/code.ts` | Shiki syntax highlighting |
| `src/previews/markdown.ts` | Markdown rendering |
| `src/previews/json.ts` | Colorized JSON |
| `src/previews/csv.ts` | Table-formatted CSV/TSV (exports shared `renderRowsAsTable`) |
| `src/previews/directory.ts` | Directory listing preview |
| `src/previews/hex.ts` | Hex dump for binary files |
| `src/previews/archive.ts` | Archive contents |
| `src/previews/mermaid.ts` | ASCII mermaid diagrams |
| `src/previews/docx.ts` | `.docx` text extraction via mammoth |
| `src/previews/xlsx.ts` | `.xlsx` table rendering via exceljs |
| `src/previews/pdf.ts` | PDF text extraction via unpdf |
| `src/hooks/useMouse.ts` | Mouse event parsing (SGR protocol) |
| `src/hooks/useKeyBindings.ts` | Keyboard input handler |
| `src/hooks/usePreviewCache.ts` | LRU preview cache |
| `src/utils/fs.ts` | Directory listing, sorting, trash, git status |
| `src/utils/clipboard.ts` | OS clipboard integration |
| `src/utils/openInTerminal.ts` | Detect host terminal and open files in a new tab (nano) |
| `src/utils/humanSize.ts` | File size formatting |
| `src/utils/ansiText.ts` | ANSI-aware text utils: stripAnsi, visualWidth, ansiSlice, wrap, computeWrappedBody, truncateByWidth |

### Key Patterns

- **useReducer**: All state in `AppState`, mutations via dispatch. Cache writes happen in `App.tsx` (post-dispatch side effect), NOT inside the reducer — the reducer is pure.
- **Lazy preview pipeline**: `src/previews/index.ts` is the only module imported eagerly; every concrete previewer (`code`, `markdown`, `json`, `docx`, `xlsx`, `pdf`, etc.) is dynamic-imported on first use and memoized. Bun's `--compile` mode bundles these dynamically-imported modules AND defers their top-level execution until the `import()` call runs — verified in practice, so lazy loading works in both `bun run start` and the release binary.
- **Debounce + AbortController**: `App.tsx:requestPreview` classifies the target via `isExpensivePreview`. Cache hits dispatch synchronously (zero-flicker instant). Cheap previewers (json/csv/hex/directory/plain text) also dispatch immediately — no debounce — so search-as-you-type stays responsive. Expensive previewers (code/markdown/office/pdf/archive) are debounced via `PREVIEW_DEBOUNCE_MS` held in `debounceTimerRef`; each new request aborts the previous via `abortRef` (`AbortController`). The stale preview body stays visible during the debounce window to avoid blank flash. `requestId` still gates reducer acceptance as a final belt-and-suspenders check.
- **Measurement-gated fast-path**: `shikiMedianRef` tracks a rolling EMA of recent `buildPreview` times for expensive files. When the median exceeds `SHIKI_FAST_PATH_THRESHOLD_MS` (~40ms — only trips on slow hardware), the expensive path splits into two dispatches: `SET_PREVIEW_STAGED` with raw plain text first, then `SET_PREVIEW` with the highlighted version. Fast hardware stays single-dispatch, so there's no flicker regression.
- **Preview payload**: `buildPreview` returns `{ text, lineCount, tokenEstimate, truncated }`; metrics computed on raw source before rendering and shown in the preview header as `size · lines · ~tokens · date` with width-aware graceful degradation
- **ANSI-aware wrap**: All preview lines wrapped to innerW via `wrapAnsiText` with cumulative SGR state carried across breaks; `computeWrappedBody` is the single source of truth for scroll math (Preview render, mouse wheel, click-drag selection all use it)
- **Flat-Text bars**: TopBar and BottomBar use explicit plain-string segment lists + `visualWidth` measurement + explicit `backgroundColor` on every Text, since Ink's `<Box>` does not support backgroundColor
- **Native-LRU cache**: `src/hooks/usePreviewCache.ts` uses Map insertion-order with delete-then-set on both hit and overwrite. 50-entry cap, keyed by `path|modTime|size|width|height`, stores full `PreviewPayload`.
- **Git status cache**: Keyed on `.git/index` mtime, not a wall-clock TTL. Revisiting a cwd whose index hasn't changed skips the subprocess. `loadGitStatus` + `loadGit` take an `options.force` flag — used by `R` reload and after trash to invalidate the cache.
- **Parallel `listDir`**: Per-entry lstat runs via `Promise.all`, and symlinks fire `readlink` + `stat` in parallel via `Promise.allSettled`. Large directories no longer serialize on I/O.
- **Layout math**: `layoutDimensions()` is the single source of truth
- **Position-aware mouse**: Scroll targets the pane under the cursor (3 rows per wheel tick); wheel nav on the file list routes through `navigate()` → `requestPreview()` so it honors the debounce; click-drag in preview copies text
- **Themes**: 9 built-in themes (7 dark, 2 light), persisted to `~/.config/seer/theme`

### Keybindings

- `j/k` — navigate files
- `g/G` — top/bottom
- `l/enter` — open directory
- `h` — parent directory
- `/` — fuzzy search
- `.` — toggle hidden files
- `s` — cycle sort mode
- `t` — cycle theme
- `p` — copy path to clipboard
- `e` — edit file in nano (opens in a new terminal tab; supports Ghostty, iTerm2, Terminal.app, WezTerm, kitty, VS Code)
- `</> ` — resize panes
- `R` — reload
- `backspace` — trash (with confirmation)
- `q` — quit

## Coding Conventions

- Section separators: `// ── section name ────────────────`
- Theme colors via mutable `colors` export from `theme.ts`
- Errors set `status` field for display
- Preview size cap: 256KB (`MAX_PREVIEW_BYTES`), 64KB in `SEER_FAST_MODE`, directory cap: 40 items
- Office/PDF preview cap: 10MB (docx/xlsx/pdf require full-file reads; content capped at 20k chars or 200 rows); entirely disabled in `SEER_FAST_MODE`

## Environment Variables

| Variable | Effect |
|---|---|
| `SEER_NO_NERD_FONT=1` | Use plain Unicode instead of Nerd Font glyphs |
| `SEER_FAST_MODE=1` | Low-power mode: disables Shiki, markdown rendering, and office/PDF parsers (replaces with size-only placeholder); shrinks `MAX_PREVIEW_BYTES` to 64KB; raises debounce to 150ms; shows `[fast]` badge in the status line. Opt-in for slow CPUs / SSH over slow links. |

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **Run quality gates** (if code changed) — `bun run typecheck`
2. **PUSH TO REMOTE** — This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
3. **Verify** — All changes committed AND pushed
4. **Hand off** — Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
