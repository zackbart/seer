# Agent Instructions

## Project Overview

Seer is a TypeScript/React TUI file browser with a two-pane layout (directory listing + live file preview), inspired by Yazi. Version 1.0.0.

## Tech Stack

- **Bun** — runtime and bundler
- **React 18 + Ink 5** — TUI framework (React components rendered to terminal)
- **Shiki** — syntax highlighting (nord theme)
- **Chalk** — terminal color support
- **Marked + marked-terminal** — Markdown rendering

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
| `src/theme.ts` | 7 color themes, persistence, icons, file categorization |
| `src/components/TopBar.tsx` | Breadcrumb path + badges |
| `src/components/FileList.tsx` | Directory listing with git badges |
| `src/components/Preview.tsx` | File preview with selection highlighting |
| `src/components/BottomBar.tsx` | Status line + key hints |
| `src/components/DeleteDialog.tsx` | Trash confirmation dialog |
| `src/previews/index.ts` | Preview dispatcher |
| `src/previews/code.ts` | Shiki syntax highlighting |
| `src/previews/markdown.ts` | Markdown rendering |
| `src/previews/json.ts` | Colorized JSON |
| `src/previews/csv.ts` | Table-formatted CSV/TSV |
| `src/previews/directory.ts` | Directory listing preview |
| `src/previews/hex.ts` | Hex dump for binary files |
| `src/previews/archive.ts` | Archive contents |
| `src/previews/mermaid.ts` | ASCII mermaid diagrams |
| `src/hooks/useMouse.ts` | Mouse event parsing (SGR protocol) |
| `src/hooks/useKeyBindings.ts` | Keyboard input handler |
| `src/hooks/usePreviewCache.ts` | LRU preview cache |
| `src/utils/fs.ts` | Directory listing, sorting, trash, git status |
| `src/utils/clipboard.ts` | OS clipboard integration |
| `src/utils/humanSize.ts` | File size formatting |

### Key Patterns

- **useReducer**: All state in `AppState`, mutations via dispatch
- **Async previews**: Preview generation is async; `requestId` prevents stale results
- **LRU cache**: 50-entry preview cache keyed by `path|modTime|size|width|height`
- **Layout math**: `layoutDimensions()` is the single source of truth
- **Position-aware mouse**: Scroll targets the pane under the cursor; click-drag in preview copies text
- **Themes**: 7 built-in themes (5 dark, 2 light), persisted to `~/.config/seer/theme`

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
- `</> ` — resize panes
- `R` — reload
- `backspace` — trash (with confirmation)
- `q` — quit

## Coding Conventions

- Section separators: `// ── section name ────────────────`
- Theme colors via mutable `colors` export from `theme.ts`
- Errors set `status` field for display
- Preview size cap: 256KB (`MAX_PREVIEW_BYTES`), directory cap: 40 items

## Environment Variables

| Variable | Effect |
|---|---|
| `SEER_NO_NERD_FONT=1` | Use plain Unicode instead of Nerd Font glyphs |

## Archive

The original Go implementation is preserved in `archive/go-v0/`.

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
