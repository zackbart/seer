package main

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderCSVPreview parses CSV/TSV text and renders it as an aligned table.
func renderCSVPreview(text string, sep rune, width int, truncated bool) string {
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = sep
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil || len(records) == 0 {
		// Fall back to plain text on parse error.
		if truncated {
			text += "\n\n... preview truncated ..."
		}
		return text
	}

	// Cap rows for preview performance.
	maxRows := 200
	showMore := false
	if len(records) > maxRows+1 {
		records = records[:maxRows+1]
		showMore = true
	}

	// Calculate max width per column.
	cols := 0
	for _, row := range records {
		if len(row) > cols {
			cols = len(row)
		}
	}
	colW := make([]int, cols)
	for _, row := range records {
		for c, cell := range row {
			if len(cell) > colW[c] {
				colW[c] = len(cell)
			}
		}
	}

	// Cap column widths so the table fits in the preview pane.
	available := width - 2
	for i := range colW {
		maxCellW := available / max(1, cols)
		if maxCellW < 6 {
			maxCellW = 6
		}
		if colW[i] > maxCellW {
			colW[i] = maxCellW
		}
	}

	headerStyle := lipgloss.NewStyle().Foreground(clrAccentFg).Bold(true)
	cellStyle := lipgloss.NewStyle().Foreground(clrFile)
	dimStyle := lipgloss.NewStyle().Foreground(clrDim)
	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)

	var sb strings.Builder

	for rowIdx, row := range records {
		var parts []string
		for c := 0; c < cols; c++ {
			cell := ""
			if c < len(row) {
				cell = row[c]
			}
			// Truncate cell to column width.
			runes := []rune(cell)
			if len(runes) > colW[c] {
				if colW[c] > 1 {
					cell = string(runes[:colW[c]-1]) + "…"
				} else {
					cell = string(runes[:colW[c]])
				}
			}
			padded := fmt.Sprintf("%-*s", colW[c], cell)
			if rowIdx == 0 {
				parts = append(parts, headerStyle.Render(padded))
			} else {
				parts = append(parts, cellStyle.Render(padded))
			}
		}
		sb.WriteString("  " + strings.Join(parts, dimStyle.Render("  │  ")) + "\n")

		// Divider after header row.
		if rowIdx == 0 {
			divParts := make([]string, cols)
			for c := range divParts {
				divParts[c] = strings.Repeat("─", colW[c])
			}
			sb.WriteString("  " + dimStyle.Render(strings.Join(divParts, "──┼──")) + "\n")
		}
	}

	if showMore {
		sb.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("  … and more rows (showing %d)", maxRows)))
	}
	if truncated {
		sb.WriteString("\n\n" + mutedStyle.Render("... preview truncated ..."))
	}

	return strings.TrimRight(sb.String(), "\n")
}
