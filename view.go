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

	leftW, rightW, bodyH := m.layoutDimensions()

	topBar := m.renderTopBar(m.width)
	leftPane := m.renderFileList(leftW, bodyH)
	rightPane := m.renderPreviewPane(rightW, bodyH)
	bottomBar := m.renderBottomBar(m.width)

	sepStyle := lipgloss.NewStyle().Foreground(clrBorder)
	sepLine := sepStyle.Render("│")
	sepLines := make([]string, bodyH)
	for i := range sepLines {
		sepLines[i] = sepLine
	}
	sep := strings.Join(sepLines, "\n")
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, sep, rightPane)

	// Overlays take over the body area.
	if m.inputMode != inputNone {
		overlay := m.renderInputDialog(m.width, bodyH)
		return topBar + "\n" + overlay + "\n" + bottomBar
	}
	if m.confirmingDelete {
		dialog := m.renderDeleteDialog(m.width, bodyH)
		return topBar + "\n" + dialog + "\n" + bottomBar
	}

	return topBar + "\n" + body + "\n" + bottomBar
}

// ── top bar ───────────────────────────────────────────────────────────────────

func (m model) renderTopBar(width int) string {
	sepStyle := lipgloss.NewStyle().Foreground(clrPathSep)
	segStyle := lipgloss.NewStyle().Foreground(clrBreadcrumb)
	countStyle := lipgloss.NewStyle().Foreground(clrMuted)

	// Right side: count info.
	count := fmt.Sprintf("%d items", len(m.entries))
	if m.showHidden {
		count += " (hidden)"
	}
	if m.sortBy != sortNameAsc {
		count += "  " + m.sortBy.Label()
	}
	if len(m.multiSelected) > 0 {
		count += fmt.Sprintf("  %d selected", len(m.multiSelected))
	}
	if len(m.yankPaths) > 0 && m.yankOp != yankNone {
		op := "copy"
		if m.yankOp == yankCut {
			op = "cut"
		}
		count += fmt.Sprintf("  [%s: %d]", op, len(m.yankPaths))
	}
	rawCount := countStyle.Render(count)
	countW := lipgloss.Width(rawCount)

	breadcrumbBudget := width - 1 - 1 - countW
	if breadcrumbBudget < 4 {
		breadcrumbBudget = 4
	}

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

	if lipgloss.Width(breadcrumb) > breadcrumbBudget {
		ellipsis := sepStyle.Render("…")
		ellipsisW := lipgloss.Width(ellipsis)
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

	breadcrumbW := lipgloss.Width(breadcrumb)
	gap := width - 2 - breadcrumbW - countW
	if gap < 1 {
		gap = 1
	}
	inner := breadcrumb + strings.Repeat(" ", gap) + rawCount

	return lipgloss.NewStyle().
		Width(width).
		Background(clrSurface).
		Padding(0, 1).
		Render(inner)
}

// ── file list ─────────────────────────────────────────────────────────────────

func (m model) renderFileList(w, h int) string {
	paneStyle := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrBorder)
	innerW := max(8, w-2)
	innerH := max(3, h-2)

	// Column layout:
	//   git(2) + icon+name (nameW) | size(sizeW)
	// Git column only shown when git status is loaded.
	hasGit := m.gitStatus != nil
	gitW := 0
	if hasGit {
		gitW = 2
	}
	sizeW := 9
	nameW := max(8, innerW-sizeW-gitW-3)

	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)
	lines := make([]string, 0, innerH)

	// Panel title
	titleStyle := lipgloss.NewStyle().Foreground(clrTitle).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(clrMuted)
	title := titleStyle.Render("Explorer")
	selCount := ""
	if len(m.multiSelected) > 0 {
		selCount = countStyle.Render(fmt.Sprintf(" ·  %d sel", len(m.multiSelected)))
	}
	count := countStyle.Render(fmt.Sprintf("%d", len(m.entries)))
	titleGap := innerW - lipgloss.Width(title) - lipgloss.Width(selCount) - lipgloss.Width(count)
	if titleGap < 1 {
		titleGap = 1
	}
	titleLine := lipgloss.NewStyle().
		Width(innerW).
		Background(clrSurfaceAlt).
		Render(title + selCount + strings.Repeat(" ", titleGap) + count)
	lines = append(lines, titleLine)
	lines = append(lines, lipgloss.NewStyle().Foreground(clrDim).Render(strings.Repeat("─", innerW)))

	if len(m.entries) == 0 {
		lines = append(lines, mutedStyle.Render("  (empty directory)"))
	} else {
		scrollStyle := lipgloss.NewStyle().Foreground(clrScrollbar)
		listH := innerH - 2
		if listH < 1 {
			listH = 1
		}

		start, end := visibleWindow(m.selected, len(m.entries), listH)
		needTop := start > 0
		needBot := end < len(m.entries)
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

			displayName := e.name
			if e.isDir {
				displayName = e.name + "/"
			}
			if e.isSymlink {
				icon = symlinkIcon()
				if !e.isDir {
					displayName = e.name + " →"
				}
			}

			colStyle := entryNameStyle(e)
			rawEntry := icon + displayName

			// Size field
			sizeStr := ""
			if !e.isDir {
				sizeStr = humanSize(e.size)
			}
			sizeField := fmt.Sprintf("%*s", sizeW, sizeStr)

			// Git status badge
			var gitBadge string
			if hasGit {
				code := m.gitStatus[e.name]
				gitBadge = renderGitBadge(code, gitW)
			}

			// Multi-select indicator
			isMultiSel := m.multiSelected[e.path]

			if i == m.selected {
				marker := lipgloss.NewStyle().Foreground(clrAccent).Render("▌")
				if isMultiSel {
					marker = lipgloss.NewStyle().Foreground(clrExec).Render("▌")
				}
				selBg := lipgloss.NewStyle().
					Foreground(clrAccentFg).
					Background(clrSurfaceElevated).
					Bold(true)
				entryVisW := lipgloss.Width(rawEntry)
				nameColW := innerW - sizeW - gitW - 2
				rowPadding := ""
				if entryVisW < nameColW {
					rowPadding = strings.Repeat(" ", nameColW-entryVisW)
				}
				namepart := trimVisual(rawEntry, nameColW)
				row := marker + selBg.Render(namepart+rowPadding+gitBadge+sizeField+" ")
				lines = append(lines, row)
			} else {
				nameField := trimVisual(rawEntry, nameW)
				prefix := " "
				if isMultiSel {
					prefix = lipgloss.NewStyle().Foreground(clrExec).Render("◆")
				}
				namePart := lipgloss.NewStyle().Inherit(colStyle).Render(prefix + nameField)
				gitPart := gitBadge
				sizePart := lipgloss.NewStyle().Foreground(clrSize).Render(sizeField)
				lines = append(lines, namePart+gitPart+sizePart)
			}
		}

		if needBot {
			lines = append(lines, scrollStyle.Render(fmt.Sprintf("  ↓ %d more", len(m.entries)-end)))
		}
	}

	return paneStyle.Render(strings.Join(lines, "\n"))
}

// renderGitBadge returns a colored git status badge of width w.
func renderGitBadge(code string, w int) string {
	if w <= 0 || code == "" {
		return strings.Repeat(" ", w)
	}
	// Collapse two-char codes to single display char.
	display := " "
	var color lipgloss.Color
	switch {
	case strings.Contains(code, "M"):
		display = "M"
		color = lipgloss.Color("221") // amber
	case strings.Contains(code, "A"):
		display = "A"
		color = lipgloss.Color("114") // green
	case strings.Contains(code, "D"):
		display = "D"
		color = lipgloss.Color("203") // red
	case strings.Contains(code, "R"):
		display = "R"
		color = lipgloss.Color("111") // blue
	case code == "??" || strings.Contains(code, "?"):
		display = "?"
		color = lipgloss.Color("244") // gray
	default:
		return strings.Repeat(" ", w)
	}
	badge := lipgloss.NewStyle().Foreground(color).Bold(true).Render(display)
	// Pad to w total visible cols.
	pad := strings.Repeat(" ", max(0, w-1))
	return badge + pad
}

// ── preview pane ──────────────────────────────────────────────────────────────

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

	// Header
	var headerLeft, headerRight string
	if len(m.entries) > 0 {
		e := m.entries[m.selected]
		cat := categorise(e)
		ext := filepath.Ext(e.name)
		icon := fileIconExt(cat, ext)
		if e.isSymlink {
			icon = symlinkIcon()
		}
		col := entryNameStyle(e)

		name := icon + e.name
		if e.isDir {
			name = icon + e.name + "/"
		}
		if e.isSymlink && e.symlinkTarget != "" {
			name += " → " + e.symlinkTarget
		}
		headerLeft = col.Bold(true).Render(trimToWidth(name, w*2/3))

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

	headerLineStyle := lipgloss.NewStyle().Width(innerW).Background(clrSurfaceAlt)
	gap := innerW - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	var headerLine string
	if gap >= 1 {
		headerLine = headerLineStyle.Render(headerLeft + strings.Repeat(" ", gap) + headerRight)
	} else {
		headerLine = headerLineStyle.Render(headerLeft)
	}

	divider := dimStyle.Render(strings.Repeat("─", max(1, innerW)))

	previewH := innerH - 2
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

// ── input dialog (rename / new file / new dir) ────────────────────────────────

func (m model) renderInputDialog(width, height int) string {
	dialogWidth := min(72, max(42, width-8))
	title := lipgloss.NewStyle().
		Foreground(clrAccent).
		Bold(true).
		Render(m.inputMode.Title())

	// Input field with cursor.
	inputStyle := lipgloss.NewStyle().Foreground(clrAccentFg)
	cursor := lipgloss.NewStyle().Foreground(clrAccent).Render("█")
	inputLine := lipgloss.NewStyle().
		Foreground(clrMuted).
		Render("> ") +
		inputStyle.Render(m.inputValue) + cursor

	hintLine := lipgloss.NewStyle().
		Foreground(clrHintText).
		Render("Enter to confirm  ·  Esc to cancel")

	dialogBox := lipgloss.NewStyle().
		Width(dialogWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrAccent).
		Background(clrSurfaceAlt).
		Padding(1, 2).
		Render(strings.Join([]string{
			title,
			"",
			inputLine,
			"",
			hintLine,
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

// ── delete dialog ─────────────────────────────────────────────────────────────

func (m model) renderDeleteDialog(width, height int) string {
	dialogWidth := min(72, max(42, width-8))

	// Build the target description.
	targets := m.deleteTargets()
	var fileName, meta string
	if len(targets) == 1 {
		fileName = filepath.Base(targets[0])
		if info, err := os.Stat(targets[0]); err == nil {
			if info.IsDir() {
				if children, err := os.ReadDir(targets[0]); err == nil {
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
		} else {
			meta = "file"
		}
	} else {
		fileName = fmt.Sprintf("%d items", len(targets))
		meta = "multiple files / folders"
	}

	fileLabel := trimVisual(fileName, dialogWidth-12)

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
		Render(meta)
	hintLine := lipgloss.NewStyle().
		Foreground(clrHintText).
		Render("Enter or y confirms. Esc or n cancels.")

	actionPrimary := lipgloss.NewStyle().
		Foreground(clrAccentFg).
		Background(clrDanger).
		Padding(0, 1).
		Bold(true).
		Render(" enter / y  move ")
	actionSecondary := lipgloss.NewStyle().
		Foreground(clrHintText).
		Background(clrSurfaceAlt).
		Padding(0, 1).
		Render(" esc / n  cancel ")

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

// ── bottom bar ────────────────────────────────────────────────────────────────

func (m model) renderBottomBar(width int) string {
	// Status / search line
	var statusLine string
	if m.searching {
		searchStyle := lipgloss.NewStyle().Foreground(clrAccent).Bold(true)
		queryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		cursor := lipgloss.NewStyle().Foreground(clrAccent).Render("▌")
		modeLabel := "fuzzy"
		prompt := searchStyle.Render("/ ") + queryStyle.Render(m.searchQuery) + cursor +
			lipgloss.NewStyle().Foreground(clrDim).Render("  "+modeLabel)
		statusLine = lipgloss.NewStyle().
			Width(width).
			Background(clrSurface).
			Padding(0, 1).
			Render(prompt)
	} else if m.inputMode != inputNone {
		modeStyle := lipgloss.NewStyle().Foreground(clrAccent).Bold(true)
		statusLine = lipgloss.NewStyle().
			Width(width).
			Background(clrSurface).
			Padding(0, 1).
			Render(modeStyle.Render(m.inputMode.Title()))
	} else {
		statusIcon := "●"
		statusStyle := lipgloss.NewStyle().Foreground(clrStatus)
		statusText := m.status
		if statusText == "ready" {
			statusIcon = "◆"
			statusStyle = lipgloss.NewStyle().Foreground(clrExec)
		}
		maxStatusW := width - 4
		if maxStatusW < 1 {
			maxStatusW = 1
		}
		statusText = trimVisual(statusText, maxStatusW)
		statusLine = lipgloss.NewStyle().
			Width(width).
			Background(clrSurface).
			Padding(0, 1).
			Render(statusStyle.Render(statusIcon + " " + statusText))
	}

	// Key hints
	type hint struct{ key, desc string }
	var hints []hint

	switch {
	case m.searching:
		hints = []hint{
			{"esc", "cancel"},
			{"backspace", "delete"},
			{"enter/l", "open"},
		}
	case m.inputMode != inputNone:
		hints = []hint{
			{"enter", "confirm"},
			{"esc", "cancel"},
			{"backspace", "delete char"},
		}
	default:
		hints = []hint{
			{"j/k", "move"},
			{"enter/l", "open"},
			{"h", "up"},
			{"r", "rename"},
			{"n/N", "new file/dir"},
			{"e", "edit"},
			{"y/x", "yank/cut"},
			{"P", "paste"},
			{"space", "select"},
			{"s", "sort"},
			{"b", "bookmark"},
			{"1-9", "jump"},
			{"backspace", "trash"},
			{"p", "copy path"},
			{"/", "search"},
			{".", "hidden"},
			{"</> ", "pane"},
			{"R", "reload"},
			{"^d/u", "scroll"},
			{"q", "quit"},
		}
	}

	keyStyle := lipgloss.NewStyle().Foreground(clrHintKey).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(clrHintText)
	sepStyle := lipgloss.NewStyle().Foreground(clrDim)

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
		Background(clrSurface).
		Padding(0, 1).
		Render(strings.Join(parts, ""))

	return statusLine + "\n" + keysLine
}
