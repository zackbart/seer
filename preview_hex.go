package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// buildHexPreview renders up to 512 bytes of data as a hex dump, similar to
// the output of `xxd`.
func buildHexPreview(data []byte, info os.FileInfo) string {
	addrStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
	hexStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("189"))
	asciiStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)
	labelStyle := lipgloss.NewStyle().Foreground(clrBinary).Bold(true)

	var sb strings.Builder

	// Header
	sb.WriteString(labelStyle.Render("binary file: "+info.Name()) + "\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("size: %s  modified: %s",
		humanSize(info.Size()), info.ModTime().Format("02 Jan 2006 15:04 MST"))) + "\n")
	sb.WriteString(dimStyle.Render(strings.Repeat("─", 50)) + "\n\n")

	// Hex dump – max 512 bytes, 16 bytes per row.
	limit := len(data)
	if limit > 512 {
		limit = 512
	}

	for offset := 0; offset < limit; offset += 16 {
		end := offset + 16
		if end > limit {
			end = limit
		}
		row := data[offset:end]

		// Address
		addr := addrStyle.Render(fmt.Sprintf("%08x", offset))

		// Hex bytes (two groups of 8)
		rowLen := len(row)
		var hexParts []string
		for i, b := range row {
			if i == 8 {
				hexParts = append(hexParts, " ")
			}
			hexParts = append(hexParts, fmt.Sprintf("%02x", b))
		}
		// Pad incomplete rows to 16 bytes for alignment.
		for i := rowLen; i < 16; i++ {
			if i == 8 {
				hexParts = append(hexParts, " ")
			}
			hexParts = append(hexParts, "  ")
		}
		hex := hexStyle.Render(strings.Join(hexParts, " "))

		// ASCII representation
		var ascii strings.Builder
		for _, b := range data[offset : offset+rowLen] {
			if b >= 0x20 && b < 0x7f {
				ascii.WriteByte(b)
			} else {
				ascii.WriteByte('.')
			}
		}

		pipe := dimStyle.Render("|")
		sb.WriteString(fmt.Sprintf("%s  %s  %s%s%s\n",
			addr, hex, pipe, asciiStyle.Render(ascii.String()), pipe))
	}

	if len(data) > 512 {
		sb.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("… showing 512 of %d bytes", len(data))))
	}

	return strings.TrimRight(sb.String(), "\n")
}
