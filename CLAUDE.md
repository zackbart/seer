# Agent Instructions

## Project Overview

Seer is a Go TUI file browser with a two-pane layout (directory listing + live file preview), inspired by Yazi. The entire application lives in a single file: `main.go`.

## Tech Stack

- **Go 1.25.0** — single `package main`, no sub-packages
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

Everything is in `main.go` (~2700 lines), following the Bubble Tea Elm Architecture:

- **`model` struct** — all application state (cwd, entries, preview, cache, search, etc.)
- **`Init()`** — fires initial preview request
- **`Update(msg)`** — handles keyboard, mouse, resize, and async preview messages
- **`View()`** — renders the full terminal frame

### Key Patterns

- **Async previews**: Preview generation runs via `tea.Cmd`; a `requestID` field prevents stale results from overwriting fresh ones
- **LRU cache**: 50-entry preview cache keyed by `path|modTime|size|width|height`
- **Layout math**: `layoutDimensions()` is the single source of truth — left pane is `max(26, width/3)`, right pane fills the rest minus a 1-char separator
- **File categorization**: `categorise()` maps extensions to categories (`catDir`, `catImage`, `catCode`, etc.) which drive icons and colors
- **Method receivers**: View/render methods use value receivers; mutating methods use pointer receivers

### Preview Pipeline

`buildPreview()` dispatches by file type to: directory listing, image (truecolor half-blocks or ASCII), Markdown (glamour), JSON (custom colorizer), Mermaid (native ASCII), syntax-highlighted code, or plain text fallback.

## Coding Conventions

- Single-file architecture — all code in `main.go`, `package main`
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

## Issue Tracking

This project uses `bd` (beads) for issue tracking.

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
