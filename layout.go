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
// if truncated.
func trimVisual(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	runes := []rune(s)
	var sb strings.Builder
	used := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if used+rw > n-1 {
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

// layoutDimensions returns the canonical pane widths and body height.
// paneOffset shifts the divider (positive = wider left pane).
func (m model) layoutDimensions() (leftW, rightW, bodyH int) {
	base := max(26, m.width/3)
	leftW = max(16, min(m.width/2, base+m.paneOffset))
	rightW = m.width - leftW - 1
	if rightW < 20 {
		rightW = 20
		leftW = m.width - rightW - 1
	}
	bodyH = max(4, m.height-4)
	return
}

// ── hit testing ──────────────────────────────────────────────────────────────

func (m model) isInPreviewPane(x, y int) bool {
	leftW, rightW, bodyH := m.layoutDimensions()
	previewStartX := leftW + 1
	previewEndX := previewStartX + rightW - 1
	previewStartY := 1
	previewEndY := previewStartY + bodyH

	return x >= previewStartX && x <= previewEndX && y >= previewStartY && y <= previewEndY
}

func (m model) previewBodyRect() (startX, startY, width, height int) {
	leftW, rightW, bodyH := m.layoutDimensions()
	startX = leftW + 2
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

// isInFileList returns true when (x,y) falls within the left pane body.
func (m model) isInFileList(x, y int) bool {
	leftW, _, bodyH := m.layoutDimensions()
	// Left pane occupies x: 0..leftW-1, y: 1..1+bodyH.
	return x >= 0 && x < leftW && y >= 1 && y < 1+bodyH
}

// fileListClickIndex converts a click position to an entry index.
// Returns (index, true) on success, (0, false) if the click is outside the
// entry list area.
func (m model) fileListClickIndex(x, y int) (int, bool) {
	_, _, bodyH := m.layoutDimensions()
	innerH := max(3, bodyH-2)
	// Within the pane: border(1) + title(1) + divider(1) = 3 fixed rows.
	// Entry rows start at pane-relative row 3 (absolute y = 1+1+3 = 4? no...)
	// Absolute y=1: top pane border, y=2: title row, y=3: divider, y=4+: entries.
	listH := innerH - 2
	if listH < 1 {
		listH = 1
	}

	needTop := false
	needBot := false
	start, end := visibleWindow(m.selected, len(m.entries), listH)
	needTop = start > 0
	needBot = end < len(m.entries)
	for {
		capacity := listH
		if needTop {
			capacity--
		}
		if needBot {
			capacity--
		}
		if capacity < 1 {
			capacity = 1
		}
		s2, e2 := visibleWindow(m.selected, len(m.entries), capacity)
		nt2 := s2 > 0
		nb2 := e2 < len(m.entries)
		if nt2 == needTop && nb2 == needBot {
			start, end = s2, e2
			break
		}
		needTop = nt2
		needBot = nb2
	}

	// Absolute Y for first entry row.
	// y=0: top bar, y=1: pane top border, y=2: title, y=3: divider, y=4: first entry.
	entryStartY := 4
	if needTop {
		entryStartY++ // scroll indicator row
	}

	relY := y - entryStartY
	if relY < 0 {
		return 0, false
	}
	idx := start + relY
	if idx >= end || idx >= len(m.entries) {
		return 0, false
	}
	return idx, true
}

// ── preview text selection helpers ───────────────────────────────────────────

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
