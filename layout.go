package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// visibleWindow returns [start, end) range of entries to show given height.
func visibleWindow(selected, total, height int) (int, int) {
	if total <= height {
		return 0, total
	}
	// Keep selected roughly centred
	half := height / 2
	start := selected - half
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > total {
		end = total
		start = max(0, end-height)
	}
	return start, end
}

// trimVisual truncates s to at most n visible terminal columns, appending "…"
// if truncated. Uses lipgloss.Width for accurate multi-byte / ANSI measurement.
func trimVisual(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	// Walk runes, accumulating visual width until we exceed budget
	runes := []rune(s)
	var sb strings.Builder
	used := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if used+rw > n-1 { // leave 1 cell for the ellipsis
			sb.WriteRune('…')
			break
		}
		sb.WriteRune(r)
		used += rw
	}
	return sb.String()
}

// padRight pads or truncates s to exactly n visible terminal columns.
func padRight(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return trimVisual(s, n)
	}
	return s + strings.Repeat(" ", n-w)
}

// trimToWidth truncates s to fit within width columns, appending "…" if needed.
func trimToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	// Walk runes accumulating visual width, same approach as trimVisual.
	runes := []rune(s)
	var out []rune
	w := 0
	ellipsisW := lipgloss.Width("…")
	budget := width - ellipsisW
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > budget {
			break
		}
		out = append(out, r)
		w += rw
	}
	return string(out) + "…"
}

// layoutDimensions returns the canonical pane widths and body height derived
// from the current terminal size. Centralises the layout math used by View,
// isInPreviewPane, and requestPreview.
func (m model) layoutDimensions() (leftW, rightW, bodyH int) {
	leftW = max(26, m.width/3)
	rightW = m.width - leftW - 1
	bodyH = max(4, m.height-4)
	return
}

func (m model) isInPreviewPane(x, y int) bool {
	leftW, rightW, bodyH := m.layoutDimensions()
	previewStartX := leftW + 1
	previewEndX := previewStartX + rightW - 1
	previewStartY := 1 // top bar
	previewEndY := previewStartY + bodyH

	return x >= previewStartX && x <= previewEndX && y >= previewStartY && y <= previewEndY
}

func (m model) previewBodyRect() (startX, startY, width, height int) {
	leftW, rightW, bodyH := m.layoutDimensions()
	startX = leftW + 2
	// y=0: top bar, y=1: pane top border, y=2: header, y=3: divider, y=4: body start
	startY = 4
	width = max(1, rightW-2)
	height = max(1, bodyH-4)
	return
}

func (m model) isInPreviewBody(x, y int) bool {
	startX, startY, width, height := m.previewBodyRect()
	endX := startX + width - 1
	endY := startY + height - 1
	return x >= startX && x <= endX && y >= startY && y <= endY
}

func (m model) previewBodyPoint(x, y int) selectionPoint {
	startX, startY, width, height := m.previewBodyRect()
	col := x - startX
	row := y - startY
	col = max(0, min(col, width))
	row = max(0, min(row, height-1))
	return selectionPoint{x: col, y: row}
}

func (m model) selectedPreviewText() string {
	start := m.previewSelStart
	end := m.previewSelEnd
	if start.y > end.y || (start.y == end.y && start.x > end.x) {
		start, end = end, start
	}
	if start == end {
		return ""
	}

	_, _, width, height := m.previewBodyRect()
	lines := m.visiblePreviewLinesForCopy(width, height)
	if len(lines) == 0 {
		return ""
	}

	var out []string
	for row := start.y; row <= end.y; row++ {
		line := ""
		if row >= 0 && row < len(lines) {
			line = lines[row]
		}
		partStart := 0
		partEnd := width
		if row == start.y {
			partStart = start.x
		}
		if row == end.y {
			partEnd = end.x
		}
		if partEnd < partStart {
			partEnd = partStart
		}
		out = append(out, sliceByColumns(line, partStart, partEnd))
	}
	return strings.Join(out, "\n")
}

func (m model) visiblePreviewLinesForCopy(width, height int) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	previewBody := m.preview
	if previewBody == "" && !m.loading {
		previewBody = "  (no preview available)"
	}
	if m.loading {
		previewBody = "  loading preview..."
	}

	contentH := height
	lines := make([]string, 0, height)
	if m.previewOffset > 0 {
		contentH--
		lines = append(lines, fmt.Sprintf("  ↑ line %d", m.previewOffset+1))
	}
	if contentH < 1 {
		contentH = 1
	}

	tmp := m
	sliced := tmp.slicePreview(previewBody, contentH)
	bodyLines := strings.Split(sliced, "\n")
	lines = append(lines, bodyLines...)

	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}

	for i, line := range lines {
		plain := ansi.Strip(line)
		lines[i] = sliceByColumns(plain, 0, width)
	}
	return lines
}

func sliceByColumns(s string, start, end int) string {
	if end <= start {
		return ""
	}
	if start < 0 {
		start = 0
	}
	startIdx := byteIndexForColumn(s, start)
	endIdx := byteIndexForColumn(s, end)
	if endIdx < startIdx {
		endIdx = startIdx
	}
	return s[startIdx:endIdx]
}

func byteIndexForColumn(s string, col int) int {
	if col <= 0 {
		return 0
	}
	width := 0
	for idx, r := range s {
		rw := lipgloss.Width(string(r))
		if rw < 1 {
			rw = 1
		}
		if width+rw > col {
			return idx
		}
		width += rw
	}
	return len(s)
}
