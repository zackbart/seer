# Agent Instructions

## Project Overview

Seer is a TypeScript/React TUI file browser with a two-pane layout (directory listing + live file preview), inspired by Yazi. Version 1.0.22.

## Tech Stack

- **Bun** — runtime and bundler
- **React 18 + Ink 5** — TUI framework (React components rendered to terminal)
- **Shiki (`shiki/core` + JS regex engine)** — syntax highlighting (nord theme). Uses `createHighlighterCore` with explicit per-grammar dynamic imports from `@shikijs/langs/*` rather than the full `shiki` package, so only the languages a user actually opens ever get materialized.
- **Chalk** — terminal color support
- **Marked + marked-terminal** — Markdown rendering (lazy-loaded)
- **turndown (+ turndown-plugin-gfm)** — HTML→Markdown conversion, reused by the HTML previewer (lazy-loaded)
- **mammoth** — `.docx` text extraction (lazy-loaded)
- **exceljs** — `.xlsx` parsing (lazy-loaded)
- **unpdf** — PDF text extraction, pdfjs-dist wrapper, Bun-friendly (lazy-loaded)
- **terminal-image + jimp** — image half-block rasterization (fallback path, lazy-loaded)
- **supports-terminal-graphics** — Kitty/iTerm2 graphics protocol detection
- **Text utilities** — `string-width` (display width), `cli-truncate` (width-aware truncation), `image-size` (image dimensions for the graphics path)

## Build & Run

```bash
bun run start             # Run directly
bun run build             # Compile to standalone binary (./seer)
bun run typecheck         # Type-check (tsc --noEmit)
bun test                  # Run *.test.ts (Bun auto-discovers; no script entry needed)
```

### Dev binary on PATH

`~/.local/bin/seer-dev` is a symlink → `./seer` in this repo. After any `bun run build`, calling `seer-dev` anywhere on the system runs the freshly built dev binary. The released production binary lives at `/opt/homebrew/bin/seer` and is unaffected by dev builds.

**Workflow:** after code changes, run `bun run build` to refresh `./seer`; the symlink picks it up automatically. No re-linking needed unless the repo moves.

**Setup (one-time):** `ln -sf /Users/zackbart/Dev/projects/tooling/seer/seer ~/.local/bin/seer-dev`

## Architecture

All source code lives in `src/`, organized as:

| File/Dir | Contents |
|---|---|
| `src/index.tsx` | Entry point, CLI flags, alt-screen setup, exit cleanup |
| `src/App.tsx` | Main component, useReducer state, key/mouse handlers, layout, preview orchestration |
| `src/types.ts` | Enums, `Entry`, `AppState` interface, all tuning constants |
| `src/theme.ts` | 9 color themes, persistence, icons, file categorization |
| `src/marked-terminal.d.ts` | Local type shim for `marked-terminal` (no shipped types) |
| `src/components/TopBar.tsx` | Breadcrumb path + badges |
| `src/components/FileList.tsx` | Directory listing with git badges |
| `src/components/Preview.tsx` | File preview with selection highlighting |
| `src/components/BottomBar.tsx` | Status line + key hints |
| `src/components/DeleteDialog.tsx` | Trash confirmation dialog |
| `src/previews/index.ts` | Preview dispatcher — dynamic-imports each previewer on first use, threads `AbortSignal` through the pipeline, applies `SEER_FAST_MODE` guards. Also exports `buildPlainPreview` (fast-path stage 1) and `isExpensivePreview` classifier. |
| `src/previews/code.ts` | Shiki syntax highlighting |
| `src/previews/markdown.ts` | Markdown rendering |
| `src/previews/html.ts` | HTML rendering — pre-strips `<script>/<style>/<noscript>` bodies (paired + open-only regex) BEFORE turndown to skip walking large inline JS/CSS, then turndown → markdown.ts pipeline. Accepts `AbortSignal`. |
| `src/previews/json.ts` | Colorized JSON |
| `src/previews/csv.ts` | Table-formatted CSV/TSV (exports shared `renderRowsAsTable`) |
| `src/previews/directory.ts` | Directory listing preview |
| `src/previews/hex.ts` | Hex dump for binary files |
| `src/previews/archive.ts` | Archive contents |
| `src/previews/mermaid.ts` | ASCII mermaid diagrams |
| `src/previews/docx.ts` | `.docx` text extraction via mammoth |
| `src/previews/xlsx.ts` | `.xlsx` table rendering via exceljs |
| `src/previews/pdf.ts` | PDF text extraction via unpdf |
| `src/previews/image.ts` | Image preview: Kitty-placeholder path (pixel-perfect) or half-block fallback via terminal-image |
| `src/utils/termGraphics.ts` | Kitty graphics protocol helpers: protocol detection, APC transmit chunker, unicode-placeholder grid builder (U+10EEEE + diacritic coords + 256-color-fg id), delete escapes |
| `src/utils/layout.ts` | `layoutDimensions()` — single source of truth for pane sizes, used by App.tsx and Preview.tsx |
| `src/utils/fs.ts` | Directory listing, sorting, trash, git status, cache-key construction |
| `src/utils/ansiText.ts` | ANSI-aware text utils: stripAnsi, visualWidth, ansiSlice, wrap, computeWrappedBody, truncateByWidth, sanitizeTerminalText |
| `src/utils/ansiText.test.ts` | `bun:test` coverage for the ANSI text utils (the only test file) |
| `src/utils/clipboard.ts` | OS clipboard integration |
| `src/utils/openInTerminal.ts` | Detect host terminal and open files in a new tab (nano) |
| `src/utils/humanSize.ts` | File size formatting |
| `src/hooks/useImageRegistry.ts` | Registry of live terminal-graphics image ids (1..255); lifetime bound to the preview cache so eviction frees terminal-side storage |
| `src/hooks/useMouse.ts` | Mouse event parsing (SGR protocol) |
| `src/hooks/useKeyBindings.ts` | Keyboard input handler |
| `src/hooks/usePreviewCache.ts` | LRU preview cache |

### Key Patterns

- **useReducer**: All state in `AppState`, mutations via dispatch. Cache writes happen in `App.tsx` (post-dispatch side effect), NOT inside the reducer — the reducer is pure.
- **Lazy preview pipeline**: `src/previews/index.ts` is the only previewer module imported eagerly; every concrete previewer (`code`, `markdown`, `json`, `docx`, `xlsx`, `pdf`, etc.) is dynamic-imported on first use and memoized. Bun's `--compile` defers top-level execution of dynamically-imported modules until the `import()` runs — verified in both `bun run start` and the release binary.
- **Debounce + AbortController**: `App.tsx:requestPreview` classifies the target via `isExpensivePreview`. Cache hits dispatch synchronously (zero-flicker). Cheap previewers (json/csv/hex/directory/plain text) also dispatch immediately — no debounce — so search-as-you-type stays responsive. Expensive previewers (code/markdown/office/pdf/archive) debounce via `PREVIEW_DEBOUNCE_MS` (60ms; 150ms in fast mode) held in `debounceTimerRef`; each new request aborts the previous via `abortRef` (`AbortController`). The stale body stays visible during the debounce window to avoid blank flash. `requestId` gates reducer acceptance as a final check.
- **Measurement-gated fast-path**: `shikiMedianRef` tracks a rolling EMA of recent `buildPreview` times for expensive files. When the median exceeds `SHIKI_FAST_PATH_THRESHOLD_MS` (40ms — only trips on slow hardware), the expensive path splits into two dispatches: `SET_PREVIEW_STAGED` (raw plain text) then `SET_PREVIEW` (highlighted). Fast hardware stays single-dispatch, no flicker.
- **Preview payload**: `buildPreview` returns `{ text, lineCount, tokenEstimate, truncated }`; metrics computed on raw source before rendering, shown in the preview header as `size · lines · ~tokens · date` with width-aware graceful degradation.
- **ANSI-aware wrap**: All preview lines wrapped to innerW via `wrapAnsiText` with cumulative SGR state carried across breaks; `computeWrappedBody` is the single source of truth for scroll math (Preview render, mouse wheel, click-drag selection all use it).
- **Flat-Text bars**: TopBar/BottomBar use explicit plain-string segment lists + `visualWidth` measurement + explicit `backgroundColor` on every Text, since Ink's `<Box>` does not support backgroundColor.
- **Native-LRU cache**: `usePreviewCache.ts` uses Map insertion-order with delete-then-set on hit and overwrite. 50-entry cap (`PREVIEW_CACHE_MAX`), keyed by `path|modTime|size|width|height` (built in `fs.ts`), stores full `PreviewPayload`.
- **Git status cache**: Keyed on `.git/index` mtime, not a wall-clock TTL. Revisiting a cwd whose index hasn't changed skips the subprocess. `loadGitStatus` + `loadGit` take `options.force` — used by `R` reload and after trash to invalidate.
- **Parallel `listDir`**: Per-entry lstat runs via `Promise.all`; symlinks fire `readlink` + `stat` in parallel via `Promise.allSettled`. Large directories don't serialize on I/O.
- **Position-aware mouse**: Scroll targets the pane under the cursor (3 rows per wheel tick); wheel nav on the file list routes through `navigate()` → `requestPreview()` so it honors the debounce; click-drag in preview copies text.
- **Themes**: 9 built-in (7 dark, 2 light), persisted to `~/.config/seer/theme`. Cycling theme (`t`) aborts any in-flight preview, flushes the preview cache (cached payloads carry old-theme ANSI), and re-requests the active selection so the visible body re-paints. Previewers that bake chalk styles (markdown's `markedTerminal` config, json/hex token tables) read live `colors.*` and bust their internal caches on theme change.
- **Kitty image rendering (out-of-band)**: Placeholder cells are `U+10EEEE` + two combining diacritics (row/col coords). Ink's `@alcalzone/ansi-tokenize` splits every codepoint into its own cell, so routing the grid through `<Text>` breaks coordinate binding — instead, Preview.tsx reserves vertical space with empty boxes and a `useEffect` writes the placeholder grid straight to `process.stdout.write` via CUP positioning, with follow-up emits at 50ms/150ms to survive Ink's 32ms throttled overwrite. Transmit APC (base64 PNG, 4096-char chunks) fires once per unique image id from `runBuild` in App.tsx. `useImageRegistry.ts` assigns ids 1..255 keyed by `path|modTime|size`; `usePreviewCache.ts` releases the id + emits a delete escape on eviction so terminal-side pixel storage frees in lockstep; `index.tsx` emits delete-all on exit when `hasTransmittedAny()`.

### Keybindings

- `j/k` or `↑/↓` — navigate files
- `g/G` — top/bottom
- `l/enter` or `→` — open directory
- `h` or `←` — parent directory
- `/` — fuzzy search (`esc` exits search mode)
- `.` — toggle hidden files
- `s` — cycle sort mode
- `t` — cycle theme
- `p` — copy path to clipboard
- `e` — edit file in nano (opens in a new terminal tab; supports Ghostty, iTerm2, Terminal.app, WezTerm, kitty, VS Code)
- `</> ` — resize panes
- `R` — reload
- `backspace` — trash (with confirmation)
- `q` or `ctrl+c` — quit

## Testing

- Test runner is `bun:test`; run with `bun test` (Bun auto-discovers `*.test.ts` — there is no `test` script in `package.json`).
- Coverage today is `src/utils/ansiText.test.ts` only. The ANSI text utils are the highest-leverage place to test since scroll math, wrapping, and selection all depend on them — extend this file when touching `ansiText.ts`.

## Coding Conventions

- Section separators: `// ── section name ────────────────`
- Theme colors via mutable `colors` export from `theme.ts`
- Errors set `status` field for display
- All tuning constants live in `src/types.ts` (caps, debounce, thresholds). Touch them there, not at call sites.
- Preview size cap: `MAX_PREVIEW_BYTES` = 256KB (64KB in fast mode), directory cap `MAX_DIR_PREVIEW` = 40 items
- Rich-render cap: `MAX_RICH_RENDER_CHARS` = 64KB chars (32KB in fast mode) — applies to code/markdown/office text/json after extraction; CSV/XLSX tables are additionally capped at 200 rows
- Office/PDF preview cap: 10MB whole-file read (`MAX_OFFICE_BYTES`); entirely disabled in `SEER_FAST_MODE`

## Environment Variables

| Variable | Effect |
|---|---|
| `SEER_NO_NERD_FONT=1` | Use plain Unicode instead of Nerd Font glyphs |
| `SEER_FAST_MODE=1` | Low-power mode: disables Shiki, markdown rendering, and office/PDF parsers (replaces with size-only placeholder); shrinks `MAX_PREVIEW_BYTES` to 64KB and `MAX_RICH_RENDER_CHARS` to 32KB; raises debounce to 150ms; shows `[fast]` badge in the status line. Opt-in for slow CPUs / SSH over slow links. |
| `SEER_IMAGE_PROTOCOL` | Image rendering protocol. `auto` (default) uses Kitty graphics on Ghostty/Kitty/WezTerm (detected via `supports-terminal-graphics`), falls back to half-blocks. `kitty` forces Kitty-placeholder. `blocks` forces half-blocks. `iterm` is reserved and currently degrades to blocks — iTerm2 inline images can't re-emit on Ink rerenders, so they'd get overdrawn on every state change; half-blocks on iTerm2 already look great. `off` renders a size-only placeholder. TMUX disables Kitty auto-detection (no passthrough). Kitty path shows a `[kitty]` badge in the status line. |

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **Run quality gates** (if code changed) — `bun run typecheck` (and `bun test` if `ansiText.ts` was touched)
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
</content>
</invoke>
