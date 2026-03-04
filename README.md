# seer

A fast, minimal TUI file browser with live previews — inspired by [Yazi](https://github.com/sxyazi/yazi).

Two panes: directory listing on the left, instant file preview on the right. That's it.

## Features

- Two-pane layout with live file preview
- Syntax-highlighted code previews (Chroma, nord theme)
- Styled Markdown rendering (Glamour, tokyo-night)
- Native Mermaid diagram preview (sequence, flowchart, etc.)
- Image preview — truecolor half-blocks or ASCII fallback
- JSON pretty-printing with color
- Directory summaries and binary file info
- Fast fuzzy search (`/` to filter)
- Mouse support (scroll, click, select-to-copy in preview)
- Nerd Font icons (with plain Unicode fallback)
- Async preview pipeline with LRU cache

## Install

### Homebrew

```bash
brew install zackbart/tap/seer
```

### Go

```bash
go install github.com/zackbart/seer@latest
```

### From source

```bash
git clone https://github.com/zackbart/seer.git
cd seer
go build -o seer .
```

## Usage

```bash
seer              # Browse current directory
seer /some/path   # Browse a specific directory
seer --version    # Print version
seer --help       # Show help
```

## Controls

| Key | Action |
|---|---|
| `j` / `k` / arrows | Move selection |
| `enter` / `l` | Open directory or refresh preview |
| `h` / `backspace` | Parent directory |
| `g` / `G` | Jump to top / bottom |
| `.` | Toggle hidden files |
| `/` | Search / filter |
| `ctrl+d` / `ctrl+u` | Scroll preview down / up |
| `delete` | Delete file (with confirmation) |
| `r` | Reload directory |
| `q` / `ctrl+c` | Quit |

Mouse: click to select, scroll to navigate, select text in preview to copy.

## Environment Variables

| Variable | Effect |
|---|---|
| `SEER_NO_NERD_FONT=1` | Use plain Unicode instead of Nerd Font glyphs |
| `COLORTERM=truecolor` | Enable truecolor image preview |
| `NO_COLOR` | Disable color image rendering |

## Supported Formats

- **Code**: Go, JS/TS, Python, Rust, C/C++, Ruby, Java, and many more (via Chroma)
- **Markup**: Markdown, MDX, RST
- **Data**: JSON, YAML, TOML, INI, ENV
- **Images**: PNG, JPEG, GIF, WebP, BMP, TIFF
- **Diagrams**: Mermaid (`.mmd`)
- **Directories**: file count, item listing
- **Binary**: size and type info

## License

MIT
