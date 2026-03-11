package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

// ── View ───────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return lipgloss.NewStyle().Foreground(clrLoading).Render("loading…")
	}

	// ── dimensions ──────────────────────────────────────────────────────────
	leftW, rightW, bodyH := m.layoutDimensions()

	// ── top bar: breadcrumb path ─────────────────────────────────────────────
	topBar := m.renderTopBar(m.width)

	// ── left pane: file list ─────────────────────────────────────────────────
	leftPane := m.renderFileList(leftW, bodyH)

	// ── right pane: preview ───────────────────────────────────────────────────
	rightPane := m.renderPreviewPane(rightW, bodyH)

	// ── bottom bar ────────────────────────────────────────────────────────────
	bottomBar := m.renderBottomBar(m.width)

	sepStyle := lipgloss.NewStyle().Foreground(clrBorder)
	sepLine := sepStyle.Render("│")
	sepLines := make([]string, bodyH)
	for i := range sepLines {
		sepLines[i] = sepLine
	}
	sep := strings.Join(sepLines, "\n")
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, sep, rightPane)

	if m.confirmingDelete {
		dialog := m.renderDeleteDialog(m.width, bodyH)
		return topBar + "\n" + dialog + "\n" + bottomBar
	}

	return topBar + "\n" + body + "\n" + bottomBar
}

func (m model) renderDeleteDialog(width, height int) string {
	dialogWidth := min(72, max(42, width-8))
	fileName := filepath.Base(m.deleteTarget)
	fileLabel := trimVisual(fileName, dialogWidth-12)
	meta := "file"
	if info, err := os.Stat(m.deleteTarget); err == nil {
		if info.IsDir() {
			if children, err := os.ReadDir(m.deleteTarget); err == nil {
				switch len(children) {
				case 0:
					meta = "empty folder"
				case 1:
					meta = "folder  •  1 item"
				default:
					meta = fmt.Sprintf("folder  •  %d items", len(children))
				}
			} else {
				meta = "folder"
			}
		} else {
			meta = humanSize(info.Size())
		}
	}

	title := lipgloss.NewStyle().
		Foreground(clrDanger).
		Bold(true).
		Render("Move to Trash?")
	nameLine := lipgloss.NewStyle().
		Foreground(clrAccentFg).
		Bold(true).
		Render(fileLabel)
	metaLine := lipgloss.NewStyle().
		Foreground(clrMuted).
		Render("Selected with backspace  •  " + meta)
	hintLine := lipgloss.NewStyle().
		Foreground(clrHintText).
		Render("Enter or y confirms. Esc or n cancels.")

	actionPrimary := lipgloss.NewStyle().
		Foreground(clrAccentFg).
		Background(clrDanger).
		Padding(0, 1).
		Bold(true).
		Render(" enter / y move ")
	actionSecondary := lipgloss.NewStyle().
		Foreground(clrHintText).
		Background(clrSurfaceAlt).
		Padding(0, 1).
		Render(" esc / n cancel ")

	dialogBox := lipgloss.NewStyle().
		Width(dialogWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrDanger).
		Background(clrDangerSoft).
		Padding(1, 2).
		Render(strings.Join([]string{
			title,
			"",
			nameLine,
			metaLine,
			"",
			hintLine,
			"",
			actionPrimary + "  " + actionSecondary,
		}, "\n"))

	boxLines := strings.Split(dialogBox, "\n")
	boxHeight := len(boxLines)
	topPad := max(0, (height-boxHeight)/2)
	leftPad := max(0, (width-lipgloss.Width(boxLines[0]))/2)
	lines := make([]string, 0, height)
	for i := 0; i < topPad; i++ {
		lines = append(lines, "")
	}
	for _, line := range boxLines {
		lines = append(lines, strings.Repeat(" ", leftPad)+line)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}

// renderTopBar draws the full-width breadcrumb path bar.
func (m model) renderTopBar(width int) string {
	sepStyle := lipgloss.NewStyle().Foreground(clrPathSep)
	segStyle := lipgloss.NewStyle().Foreground(clrBreadcrumb)
	countStyle := lipgloss.NewStyle().Foreground(clrMuted)

	// Right side: entry count (rendered first so we know its width)
	count := fmt.Sprintf("%d items", len(m.entries))
	if m.showHidden {
		count += " (hidden shown)"
	}
	rawCount := countStyle.Render(count)
	countW := lipgloss.Width(rawCount)

	// Available width for breadcrumb: total - 1 left padding - 1 space before count - countW
	breadcrumbBudget := width - 1 - 1 - countW
	if breadcrumbBudget < 4 {
		breadcrumbBudget = 4
	}

	// Build breadcrumb segments, then truncate from the left if too long
	parts := strings.Split(m.cwd, string(filepath.Separator))
	var segments []string
	for i, p := range parts {
		if p == "" {
			if i == 0 {
				segments = append(segments, segStyle.Render("/"))
			}
			continue
		}
		if i > 0 {
			segments = append(segments, sepStyle.Render(" › "))
		}
		segments = append(segments, segStyle.Render(p))
	}
	breadcrumb := strings.Join(segments, "")

	// If breadcrumb is too wide, show only the last N path components that fit
	if lipgloss.Width(breadcrumb) > breadcrumbBudget {
		ellipsis := sepStyle.Render("…")
		ellipsisW := lipgloss.Width(ellipsis)
		// Walk from the end adding components until we run out of budget
		var kept []string
		budget := breadcrumbBudget - ellipsisW - lipgloss.Width(sepStyle.Render(" › "))
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] == "" {
				continue
			}
			seg := segStyle.Render(parts[i])
			if len(kept) > 0 {
				budget -= lipgloss.Width(sepStyle.Render(" › "))
			}
			budget -= lipgloss.Width(seg)
			if budget < 0 {
				break
			}
			kept = append([]string{seg}, kept...)
		}
		if len(kept) == 0 {
			kept = []string{segStyle.Render(parts[len(parts)-1])}
		}
		breadcrumb = ellipsis + sepStyle.Render(" › ") + strings.Join(kept, sepStyle.Render(" › "))
	}

	// Compose bar: breadcrumb left, count right
	breadcrumbW := lipgloss.Width(breadcrumb)
	gap := width - 1 - breadcrumbW - countW // 1 = left padding
	if gap < 1 {
		gap = 1
	}
	inner := breadcrumb + strings.Repeat(" ", gap) + rawCount

	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(inner)
}

// renderFileList draws the left pane with icons, names, sizes, and mod times.
func (m model) renderFileList(w, h int) string {
	paneStyle := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrBorder)
	innerW := max(8, w-2)
	innerH := max(3, h-2)

	// Column layout within the left pane:
	//   [icon+name ............ size  ]
	// Size column is 9 chars wide ("1023.9 KB" = 9 chars max), separated by a space.
	sizeW := 9
	nameW := max(8, innerW-sizeW-3)

	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)

	lines := make([]string, 0, innerH)

	// Panel title
	titleStyle := lipgloss.NewStyle().Foreground(clrTitle).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(clrMuted)
	title := titleStyle.Render("Explorer")
	count := countStyle.Render(fmt.Sprintf("%d", len(m.entries)))
	titleGap := innerW - lipgloss.Width(title) - lipgloss.Width(count)
	if titleGap < 1 {
		titleGap = 1
	}
	titleLine := lipgloss.NewStyle().
		Width(innerW).
		Render(title + strings.Repeat(" ", titleGap) + count)
	lines = append(lines, titleLine)
	lines = append(lines, lipgloss.NewStyle().Foreground(clrDim).Render(strings.Repeat("─", innerW)))

	if len(m.entries) == 0 {
		lines = append(lines, mutedStyle.Render("  (empty directory)"))
	} else {
		scrollStyle := lipgloss.NewStyle().Foreground(clrScrollbar)

		// Total rows available for file rows + scroll indicators below the header.
		listH := innerH - 2
		if listH < 1 {
			listH = 1
		}

		// First pass: compute window assuming no indicators
		start, end := visibleWindow(m.selected, len(m.entries), listH)
		needTop := start > 0
		needBot := end < len(m.entries)

		// If indicators are needed, shrink the window to make room for them.
		// We may need to do this iteratively (showing top indicator can reveal bottom need).
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
			start, end = visibleWindow(m.selected, len(m.entries), capacity)
			newNeedTop := start > 0
			newNeedBot := end < len(m.entries)
			if newNeedTop == needTop && newNeedBot == needBot {
				break
			}
			needTop = newNeedTop
			needBot = newNeedBot
		}

		if needTop {
			lines = append(lines, scrollStyle.Render(fmt.Sprintf("  ↑ %d more", start)))
		}

		for i := start; i < end; i++ {
			e := m.entries[i]
			cat := categorise(e)
			icon := fileIconExt(cat, filepath.Ext(e.name))
			colStyle := entryNameStyle(e)

			displayName := e.name
			if e.isDir {
				displayName = e.name + "/"
			}
			rawEntry := icon + displayName

			// Size field – right-aligned in sizeW columns
			sizeStr := ""
			if !e.isDir {
				sizeStr = humanSize(e.size)
			}
			sizeField := fmt.Sprintf("%*s", sizeW, sizeStr)

			if i == m.selected {
				// Selected row: full-width highlight using visual width.
				selBg := lipgloss.NewStyle().
					Foreground(clrAccentFg).
					Background(clrAccent).
					Bold(true).
					Padding(0, 1)
				// Measure the raw visual width of icon+name, pad to fill name column
				entryVisW := lipgloss.Width(rawEntry)
				nameColW := innerW - sizeW - 2
				padding := ""
				if entryVisW < nameColW {
					padding = strings.Repeat(" ", nameColW-entryVisW)
				}
				namepart := trimVisual(rawEntry, nameColW)
				row := selBg.Render(namepart + padding + sizeField)
				lines = append(lines, row)
			} else {
				nameField := trimVisual(rawEntry, nameW)
				namePart := lipgloss.NewStyle().PaddingLeft(1).Inherit(colStyle).Render(nameField)
				sizePart := lipgloss.NewStyle().Foreground(clrSize).Render(sizeField)
				lines = append(lines, namePart+sizePart)
			}
		}

		if needBot {
			lines = append(lines, scrollStyle.Render(fmt.Sprintf("  ↓ %d more", len(m.entries)-end)))
		}
	}

	return paneStyle.Render(strings.Join(lines, "\n"))
}

// renderPreviewPane draws the right pane with header and preview content.
func (m model) renderPreviewPane(w, h int) string {
	paneStyle := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrBorderStrong)
	innerW := max(12, w-2)
	innerH := max(3, h-2)

	dimStyle := lipgloss.NewStyle().Foreground(clrDim)
	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)

	// ── header row ──────────────────────────────────────────────────────────
	var headerLeft, headerRight string
	if len(m.entries) > 0 {
		e := m.entries[m.selected]
		cat := categorise(e)
		icon := fileIconExt(cat, filepath.Ext(e.name))
		col := entryNameStyle(e)

		name := icon + e.name
		if e.isDir {
			name = icon + e.name + "/"
		}
		headerLeft = col.Bold(true).Render(trimToWidth(name, w/2))

		// Right side metadata
		meta := ""
		if !e.isDir {
			meta = humanSize(e.size) + "  " + e.modTime.Format("Jan 02 15:04")
		} else {
			meta = e.modTime.Format("Jan 02 15:04")
		}
		if m.loading {
			meta = lipgloss.NewStyle().Foreground(clrLoading).Render("loading…")
		}
		headerRight = mutedStyle.Render(meta)
	} else {
		headerLeft = mutedStyle.Render("no selection")
	}

	// Compose header line
	headerLineStyle := lipgloss.NewStyle().Width(innerW)
	gap := innerW - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if gap < 1 {
		gap = 1
	}
	headerLine := headerLineStyle.Render(
		headerLeft + strings.Repeat(" ", gap) + headerRight,
	)

	// ── divider ──────────────────────────────────────────────────────────────
	divider := dimStyle.Render(strings.Repeat("─", max(1, innerW)))

	// ── preview body ─────────────────────────────────────────────────────────
	previewH := innerH - 2 // subtract header + divider
	if previewH < 1 {
		previewH = 1
	}

	previewBody := m.preview
	if previewBody == "" && !m.loading {
		previewBody = mutedStyle.Render("  (no preview available)")
	}
	if m.loading {
		previewBody = lipgloss.NewStyle().Foreground(clrLoading).Render("  loading preview…")
	}

	// Reserve one row for the scroll indicator when scrolled
	contentH := previewH
	var scrollIndicator string
	if m.previewOffset > 0 {
		contentH--
		scrollIndicator = lipgloss.NewStyle().Foreground(clrScrollbar).Render(
			fmt.Sprintf("  ↑ line %d", m.previewOffset+1),
		)
	}
	if contentH < 1 {
		contentH = 1
	}

	sliced := m.slicePreview(previewBody, contentH)
	if scrollIndicator != "" {
		sliced = scrollIndicator + "\n" + sliced
	}

	// Truncate each line to the pane width so no line can wrap in the terminal
	// and push the top/bottom chrome off screen.
	if w > 0 {
		rawLines := strings.Split(sliced, "\n")
		for i, line := range rawLines {
			if lipgloss.Width(line) > innerW {
				rawLines[i] = truncate.String(line, uint(innerW))
			}
		}
		sliced = strings.Join(rawLines, "\n")
	}

	body := lipgloss.NewStyle().Width(innerW).Height(previewH).Render(sliced)

	return paneStyle.Render(headerLine + "\n" + divider + "\n" + body)
}

// renderBottomBar draws the two-line footer: status + keybindings.
func (m model) renderBottomBar(width int) string {
	// ── status / search line ─────────────────────────────────────────────────
	var statusLine string
	if m.searching {
		searchStyle := lipgloss.NewStyle().Foreground(clrAccent).Bold(true)
		queryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		cursor := lipgloss.NewStyle().Foreground(clrAccent).Render("▌")
		prompt := searchStyle.Render("/ ") + queryStyle.Render(m.searchQuery) + cursor
		statusLine = lipgloss.NewStyle().
			Width(width).
			Padding(0, 1).
			Render(prompt)
	} else {
		statusIcon := "●"
		statusStyle := lipgloss.NewStyle().Foreground(clrStatus)
		statusText := m.status
		if statusText == "ready" {
			statusIcon = "◆"
			statusStyle = lipgloss.NewStyle().Foreground(clrExec)
		}
		maxStatusW := width - 3
		if maxStatusW < 1 {
			maxStatusW = 1
		}
		statusText = trimVisual(statusText, maxStatusW)
		statusLine = lipgloss.NewStyle().
			Width(width).
			Padding(0, 1).
			Render(statusStyle.Render(statusIcon + " " + statusText))
	}

	// ── key hints ────────────────────────────────────────────────────────────
	type hint struct{ key, desc string }
	var hints []hint
	if m.searching {
		hints = []hint{
			{"esc", "cancel"},
			{"backspace", "delete"},
			{"enter/l", "open"},
		}
	} else {
		hints = []hint{
			{"j/k", "move"},
			{"g/G", "top/end"},
			{"enter/l", "open"},
			{"h", "up"},
			{"backspace", "trash"},
			{"/", "search"},
			{".", "hidden"},
			{"^d/u", "scroll"},
			{"r", "reload"},
			{"q", "quit"},
		}
	}

	keyStyle := lipgloss.NewStyle().Foreground(clrHintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(clrHintText)
	sepStyle := lipgloss.NewStyle().Foreground(clrDim)

	// Build hints left-to-right, stopping before we'd overflow the terminal width.
	// Budget: width - 1 (left padding) - 1 (safety margin)
	hintBudget := width - 2
	dotW := lipgloss.Width(sepStyle.Render("  ·  "))
	var parts []string
	used := 0
	for i, h := range hints {
		seg := keyStyle.Render(h.key) + descStyle.Render(" "+h.desc)
		segW := lipgloss.Width(seg)
		extra := 0
		if i > 0 {
			extra = dotW
		}
		if used+extra+segW > hintBudget {
			break
		}
		if i > 0 {
			parts = append(parts, sepStyle.Render("  ·  "))
			used += dotW
		}
		parts = append(parts, seg)
		used += segW
	}
	keysLine := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(strings.Join(parts, ""))

	return statusLine + "\n" + keysLine
}
