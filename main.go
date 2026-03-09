package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("seer " + version)
			return
		case "--help", "-h":
			fmt.Println("seer " + version)
			fmt.Println()
			fmt.Println("A dead-simple TUI for browsing directories and previewing files.")
			fmt.Println()
			fmt.Println("Usage: seer [directory]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  -h, --help      Show this help message")
			fmt.Println("  -v, --version   Show version")
			return
		}
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
