# seer

`seer` is a dead-simple Go TUI for browsing directories and previewing files.

It is inspired by the fast, preview-first feel of Yazi, but intentionally keeps a small surface area.

## Features

- Two-pane interface: directory list + live preview
- Fast keyboard navigation
- Syntax-highlighted text/code previews via Chroma
- Styled Markdown previews in-terminal
- Native Mermaid preview (`.mmd` or Markdown mermaid fences)
- Directory and binary previews
- Native grayscale image preview for common formats

## Controls

- `j` / `k` or arrow keys: move selection
- `enter` / `l`: open selected directory (or refresh file preview)
- `h` / `backspace`: go to parent directory
- `.`: toggle hidden files
- `ctrl+d` / `ctrl+u`: scroll preview
- `r`: reload current directory
- `q`: quit

## Install

```bash
go mod tidy
go build -o seer .
```

No external CLI preview tools are required.

## Run

```bash
./seer
```
