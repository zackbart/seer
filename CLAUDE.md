# Agent Instructions

## Project Overview

Seer is a Go TUI file browser with a two-pane layout (directory listing + live file preview), inspired by Yazi.

## Tech Stack

- **Go 1.25.0** — `package main`, no sub-packages
- **charmbracelet/bubbletea** — TUI framework (Elm Architecture: Model/Update/View)
- **charmbracelet/lipgloss** — terminal styling
- **charmbracelet/glamour** — Markdown rendering (tokyo-night style)
- **alecthomas/chroma** — syntax highlighting (nord theme)
- **golang.org/x/image** — BMP, TIFF, WebP image support

## Build & Run

```bash
go build -o seer .     # Build
go run .               # Run directly
go mod tidy            # Fetch/clean dependencies
```

## Testing & Quality

```bash
go test ./...          # Run tests (none exist yet)
go fmt ./...           # Format code
go vet ./...           # Static analysis
```

No linter config, no CI pipeline, no Makefile.

## Architecture

The codebase is split across focused files, all in `package main`, following the Bubble Tea Elm Architecture:

| File | Contents |
|---|---|
| `main.go` | Entry point, `--version`/`--help` flags |
| `types.go` | Constants, `entry`, `model` struct, message types |
| `theme.go` | Color palette, icons, file categorisation, styling helpers |
| `model.go` | `initialModel()`, `Init()`, `Update()` — core Elm loop |
| `view.go` | `View()` and all `render*` methods |
| `layout.go` | Layout math, text truncation helpers, preview selection |
| `model_ops.go` | Model mutations: navigation, cache, search, preview request |
| `clipboard.go` | OS clipboard integration |
| `fs.go` | `listDir()`, `moveToTrash()`, file utilities |
| `util.go` | `humanSize()`, `min()`, `max()`, `previewPageSize()` |
| `preview.go` | `buildPreview()` dispatch, dir/image/markdown/code/highlight |
| `preview_image.go` | Truecolor and ASCII image rendering |
| `preview_json.go` | Colorised JSON pretty-printer |
| `preview_mermaid.go` | Mermaid flowchart + sequence diagram ASCII renderers |

### Key Patterns

- **Async previews**: Preview generation runs via `tea.Cmd`; a `requestID` field prevents stale results from overwriting fresh ones
- **LRU cache**: 50-entry preview cache keyed by `path|modTime|size|width|height`
- **Layout math**: `layoutDimensions()` is the single source of truth — left pane is `max(26, width/3)`, right pane fills the rest minus a 1-char separator
- **File categorization**: `categorise()` maps extensions to categories (`catDir`, `catImage`, `catCode`, etc.) which drive icons and colors
- **Method receivers**: View/render methods use value receivers; mutating methods use pointer receivers

### Preview Pipeline

`buildPreview()` dispatches by file type to: directory listing, image (truecolor half-blocks or ASCII), Markdown (glamour), JSON (custom colorizer), Mermaid (native ASCII), syntax-highlighted code, or plain text fallback.

## Coding Conventions

- All code in `package main`, no sub-packages
- Section separators: `// ── section name ────────────────`
- Dark indigo/slate color palette using 256-color terminal indices
- No named return values (except `layoutDimensions()`)
- Errors set `m.status` for display; no panics
- Preview size cap: 256KB (`maxPreviewBytes`), directory cap: 40 items

## Environment Variables

| Variable | Effect |
|---|---|
| `SEER_NO_NERD_FONT=1` | Use plain Unicode instead of Nerd Font glyphs |
| `COLORTERM=truecolor` | Enable truecolor image preview |
| `NO_COLOR` | Disable truecolor image rendering |

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **Run quality gates** (if code changed) — `go fmt ./...`, `go vet ./...`, `go build .`
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
