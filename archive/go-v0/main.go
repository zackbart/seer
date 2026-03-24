package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "0.4.0"

func main() {
	var cwdFile string
	var startDir string

	for _, arg := range os.Args[1:] {
		switch {
		case arg == "--version" || arg == "-v":
			fmt.Println("seer " + version)
			return
		case arg == "--help" || arg == "-h":
			printHelp()
			return
		case strings.HasPrefix(arg, "--cwd-file="):
			cwdFile = strings.TrimPrefix(arg, "--cwd-file=")
		default:
			if startDir == "" {
				startDir = arg
			}
		}
	}

	p := tea.NewProgram(
		initialModel(startDir),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Shell cd integration: write final cwd to the requested file on exit.
	if cwdFile != "" {
		if m, ok := finalModel.(model); ok {
			_ = os.WriteFile(cwdFile, []byte(m.cwd+"\n"), 0600)
		}
	}
}

func printHelp() {
	fmt.Println("seer " + version)
	fmt.Println()
	fmt.Println("A TUI file browser with live preview.")
	fmt.Println()
	fmt.Println("Usage: seer [options] [directory]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help            Show this help message")
	fmt.Println("  -v, --version         Show version")
	fmt.Println("  --cwd-file=FILE       Write final directory to FILE on exit")
	fmt.Println("                        (for shell cd integration)")
	fmt.Println()
	fmt.Println("Shell cd integration — add to ~/.bashrc or ~/.zshrc:")
	fmt.Println()
	fmt.Println("  function s() {")
	fmt.Println("    local tmp=$(mktemp)")
	fmt.Println("    command seer --cwd-file=\"$tmp\" \"$@\"")
	fmt.Println("    local dir=$(cat \"$tmp\" 2>/dev/null)")
	fmt.Println("    rm -f \"$tmp\"")
	fmt.Println("    [[ -n \"$dir\" && -d \"$dir\" ]] && cd \"$dir\"")
	fmt.Println("  }")
	fmt.Println()
	fmt.Println("Keybindings:")
	fmt.Println("  j/k         Move up/down")
	fmt.Println("  l/enter     Open directory")
	fmt.Println("  h           Go to parent")
	fmt.Println("  g/G         Top/bottom")
	fmt.Println("  alt+←/→     History back/forward")
	fmt.Println("  r           Rename")
	fmt.Println("  R           Reload")
	fmt.Println("  n / N       New file / New directory")
	fmt.Println("  e           Open in $EDITOR")
	fmt.Println("  y / x       Yank (copy) / Cut")
	fmt.Println("  P           Paste")
	fmt.Println("  space       Multi-select toggle")
	fmt.Println("  s           Cycle sort mode")
	fmt.Println("  b           Bookmark current directory")
	fmt.Println("  1-9         Jump to bookmark slot")
	fmt.Println("  < / >       Resize left pane")
	fmt.Println("  /           Fuzzy search")
	fmt.Println("  .           Toggle hidden files")
	fmt.Println("  p           Copy path to clipboard")
	fmt.Println("  backspace   Move to trash")
	fmt.Println("  ctrl+d/u    Scroll preview")
	fmt.Println("  q           Quit")
}
