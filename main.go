package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxPreviewBytes = 256 * 1024
	maxDirPreview   = 40
)

// ── color palette ──────────────────────────────────────────────────────────────
// A cohesive dark theme built around deep indigo / slate tones.
var (
	clrAccent     = lipgloss.Color("105") // soft violet – selected bg accent
	clrAccentFg   = lipgloss.Color("231") // near-white text on accent bg
	clrDir        = lipgloss.Color("75")  // sky blue – directories
	clrExec       = lipgloss.Color("114") // sage green – executables / scripts
	clrMedia      = lipgloss.Color("215") // warm amber – images / media
	clrDoc        = lipgloss.Color("189") // light lavender – markdown / docs
	clrConfig     = lipgloss.Color("222") // pale gold – config files
	clrBinary     = lipgloss.Color("203") // coral – binary / unknown
	clrSize       = lipgloss.Color("244") // medium grey – file sizes
	clrMuted      = lipgloss.Color("240") // dark grey – decorative / dividers
	clrDim        = lipgloss.Color("238") // very dark grey – subtle bg hints
	clrBreadcrumb = lipgloss.Color("147") // periwinkle – path text
	clrPathSep    = lipgloss.Color("238") // dimmer – path separators
	clrHintKey    = lipgloss.Color("105") // violet – keybind keys
	clrHintText   = lipgloss.Color("244") // grey – keybind descriptions
	clrStatus     = lipgloss.Color("189") // lavender – status messages
	clrBorder     = lipgloss.Color("237") // subtle – separator line
	clrTitle      = lipgloss.Color("147") // periwinkle – panel titles
	clrLoading    = lipgloss.Color("214") // orange – loading indicator
	clrScrollbar  = lipgloss.Color("99")  // muted violet – scroll indicator
)

var imageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
	".svg":  true,
}

// fileCategory returns a broad category for an entry used to pick colour/icon.
type fileCategory int

const (
	catDir fileCategory = iota
	catImage
	catDoc
	catCode
	catConfig
	catExec
	catBinary
	catOther
)

func categorise(e entry) fileCategory {
	if e.isDir {
		return catDir
	}
	ext := strings.ToLower(filepath.Ext(e.name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tiff", ".svg":
		return catImage
	case ".md", ".markdown", ".mdx", ".rst", ".txt":
		return catDoc
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rb", ".rs", ".c", ".cpp",
		".h", ".java", ".cs", ".php", ".swift", ".kt", ".sh", ".bash", ".zsh",
		".fish", ".lua", ".ex", ".exs", ".hs", ".ml", ".mli", ".clj", ".scala",
		".vim", ".mmd", ".mermaid":
		return catCode
	case ".json", ".yaml", ".yml", ".toml", ".ini", ".env", ".conf", ".config",
		".xml", ".dockerignore", ".gitignore", ".editorconfig", ".eslintrc",
		".prettierrc", ".babelrc", ".nvmrc":
		return catConfig
	}
	return catOther
}

func fileIcon(cat fileCategory) string {
	switch cat {
	case catDir:
		return "▸ "
	case catImage:
		return "⬡ "
	case catDoc:
		return "≡ "
	case catCode:
		return "⟨⟩ "
	case catConfig:
		return "⚙ "
	case catExec:
		return "⚡ "
	case catBinary:
		return "⬟ "
	default:
		return "· "
	}
}

func fileColor(cat fileCategory) lipgloss.Style {
	switch cat {
	case catDir:
		return lipgloss.NewStyle().Foreground(clrDir).Bold(true)
	case catImage:
		return lipgloss.NewStyle().Foreground(clrMedia)
	case catDoc:
		return lipgloss.NewStyle().Foreground(clrDoc)
	case catCode:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("231"))
	case catConfig:
		return lipgloss.NewStyle().Foreground(clrConfig)
	case catExec:
		return lipgloss.NewStyle().Foreground(clrExec)
	case catBinary:
		return lipgloss.NewStyle().Foreground(clrBinary)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	}
}

type entry struct {
	name    string
	path    string
	isDir   bool
	size    int64
	modTime time.Time
}

type previewLoadedMsg struct {
	requestID int
	cacheKey  string
	content   string
	err       error
}

type model struct {
	cwd           string
	entries       []entry
	selected      int
	showHidden    bool
	preview       string
	status        string
	width         int
	height        int
	previewOffset int
	loading       bool
	requestID     int
	cache         map[string]string
}

func initialModel() model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	entries, listErr := listDir(cwd, false)
	status := "ready"
	if listErr != nil {
		status = listErr.Error()
	}

	return model{
		cwd:        cwd,
		entries:    entries,
		selected:   0,
		preview:    "",
		status:     status,
		cache:      make(map[string]string),
		showHidden: false,
	}
}

func (m model) Init() tea.Cmd {
	return m.requestPreview()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampPreviewOffset()
		return m, m.requestPreview()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.selected < len(m.entries)-1 {
				m.selected++
				m.previewOffset = 0
				return m, m.requestPreview()
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.previewOffset = 0
				return m, m.requestPreview()
			}
		case "g", "home":
			m.selected = 0
			m.previewOffset = 0
			return m, m.requestPreview()
		case "G", "end":
			if len(m.entries) > 0 {
				m.selected = len(m.entries) - 1
				m.previewOffset = 0
				return m, m.requestPreview()
			}
		case "l", "right", "enter":
			if len(m.entries) == 0 {
				break
			}
			picked := m.entries[m.selected]
			if picked.isDir {
				if err := m.changeDir(picked.path); err != nil {
					m.status = err.Error()
				}
				return m, m.requestPreview()
			}
			m.status = fmt.Sprintf("previewing %s", picked.name)
			return m, m.requestPreview()
		case "h", "left", "backspace":
			parent := filepath.Dir(m.cwd)
			if parent != m.cwd {
				if err := m.changeDir(parent); err != nil {
					m.status = err.Error()
				}
				return m, m.requestPreview()
			}
		case ".":
			m.showHidden = !m.showHidden
			entries, err := listDir(m.cwd, m.showHidden)
			if err != nil {
				m.status = err.Error()
			} else {
				m.entries = entries
				m.selected = 0
				m.previewOffset = 0
				if m.showHidden {
					m.status = "showing hidden files"
				} else {
					m.status = "hiding hidden files"
				}
			}
			return m, m.requestPreview()
		case "ctrl+d", "pagedown":
			m.previewOffset += previewPageSize(m.height)
			m.clampPreviewOffset()
		case "ctrl+u", "pageup":
			m.previewOffset -= previewPageSize(m.height)
			m.clampPreviewOffset()
		case "r":
			entries, err := listDir(m.cwd, m.showHidden)
			if err != nil {
				m.status = err.Error()
			} else {
				m.entries = entries
				if m.selected >= len(m.entries) {
					m.selected = max(0, len(m.entries)-1)
				}
				m.status = "reloaded"
			}
			return m, m.requestPreview()
		}

	case tea.MouseMsg:
		event := tea.MouseEvent(msg)
		if !m.isInPreviewPane(event.X, event.Y) {
			return m, nil
		}

		switch event.Button {
		case tea.MouseButtonWheelDown:
			m.previewOffset += 3
			m.clampPreviewOffset()
		case tea.MouseButtonWheelUp:
			m.previewOffset -= 3
			m.clampPreviewOffset()
		}

	case previewLoadedMsg:
		if msg.requestID != m.requestID {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.preview = "preview error: " + msg.err.Error()
			return m, nil
		}
		m.cache[msg.cacheKey] = msg.content
		m.preview = msg.content
		m.clampPreviewOffset()
	}

	return m, nil
}

// ── View ───────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return lipgloss.NewStyle().Foreground(clrLoading).Render("loading…")
	}

	// ── dimensions ──────────────────────────────────────────────────────────
	leftW := max(26, m.width/3)
	rightW := m.width - leftW - 1 // -1 for the vertical separator column
	// Reserve rows: 1 top-bar + 1 divider + 1 status + 1 keys = 4 chrome rows
	bodyH := max(4, m.height-4)

	// ── top bar: breadcrumb path ─────────────────────────────────────────────
	topBar := m.renderTopBar(m.width)

	// ── left pane: file list ─────────────────────────────────────────────────
	leftPane := m.renderFileList(leftW, bodyH)

	// ── vertical separator ────────────────────────────────────────────────────
	// Render each │ independently so ANSI reset codes don't span newlines.
	sepStyle := lipgloss.NewStyle().Foreground(clrBorder)
	sepLine := sepStyle.Render("│")
	sepLines := make([]string, bodyH)
	for i := range sepLines {
		sepLines[i] = sepLine
	}
	sep := strings.Join(sepLines, "\n")

	// ── right pane: preview ───────────────────────────────────────────────────
	rightPane := m.renderPreviewPane(rightW, bodyH)

	// ── bottom bar ────────────────────────────────────────────────────────────
	bottomBar := m.renderBottomBar(m.width)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, sep, rightPane)
	return topBar + "\n" + body + "\n" + bottomBar
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
		Background(clrDim).
		PaddingLeft(1).
		Render(inner)
}

// renderFileList draws the left pane with icons, names, sizes, and mod times.
func (m model) renderFileList(w, h int) string {
	// Column layout within the left pane:
	//   [icon+name ............ size  ]
	// Size column is 7 chars wide, separated by a space.
	sizeW := 7
	nameW := max(8, w-sizeW-1)

	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)
	dimStyle := lipgloss.NewStyle().Foreground(clrDim)

	lines := make([]string, 0, h)

	// Panel title
	titleStyle := lipgloss.NewStyle().Foreground(clrTitle).Bold(true)
	title := titleStyle.Render("files")
	titleLine := lipgloss.NewStyle().Width(w).Background(clrDim).PaddingLeft(1).Render(title)
	lines = append(lines, titleLine)

	// Divider
	divider := dimStyle.Render(strings.Repeat("─", max(1, w)))
	lines = append(lines, divider)

	if len(m.entries) == 0 {
		lines = append(lines, mutedStyle.Render("  (empty directory)"))
	} else {
		scrollStyle := lipgloss.NewStyle().Foreground(clrScrollbar)

		// Total rows available for file rows + scroll indicators (title+divider already added)
		listH := h - 2
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
			icon := fileIcon(cat)
			colStyle := fileColor(cat)

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
				// Selected row: full-width highlight using visual width
				selBg := lipgloss.NewStyle().
					Foreground(clrAccentFg).
					Background(clrAccent).
					Bold(true)
				// Measure the raw visual width of icon+name, pad to fill name column
				entryVisW := lipgloss.Width(rawEntry)
				nameColW := w - sizeW
				padding := ""
				if entryVisW < nameColW {
					padding = strings.Repeat(" ", nameColW-entryVisW)
				}
				namepart := trimVisual(rawEntry, nameColW)
				row := selBg.Render(namepart + padding + sizeField)
				lines = append(lines, row)
			} else {
				nameField := trimVisual(rawEntry, nameW)
				namePart := colStyle.Render(nameField)
				sizePart := lipgloss.NewStyle().Foreground(clrSize).Render(sizeField)
				lines = append(lines, namePart+sizePart)
			}
		}

		if needBot {
			lines = append(lines, scrollStyle.Render(fmt.Sprintf("  ↓ %d more", len(m.entries)-end)))
		}
	}

	pane := lipgloss.NewStyle().Width(w).Height(h).Render(strings.Join(lines, "\n"))
	return pane
}

// renderPreviewPane draws the right pane with header and preview content.
func (m model) renderPreviewPane(w, h int) string {
	dimStyle := lipgloss.NewStyle().Foreground(clrDim)
	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)

	// ── header row ──────────────────────────────────────────────────────────
	var headerLeft, headerRight string
	if len(m.entries) > 0 {
		e := m.entries[m.selected]
		cat := categorise(e)
		icon := fileIcon(cat)
		col := fileColor(cat)

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
	headerLineStyle := lipgloss.NewStyle().Width(w).Background(clrDim).PaddingLeft(1)
	gap := w - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight) - 2 // 2 for padding
	if gap < 1 {
		gap = 1
	}
	headerLine := headerLineStyle.Render(
		headerLeft + strings.Repeat(" ", gap) + headerRight,
	)

	// ── divider ──────────────────────────────────────────────────────────────
	divider := dimStyle.Render(strings.Repeat("─", max(1, w)))

	// ── preview body ─────────────────────────────────────────────────────────
	previewH := h - 2 // subtract header + divider
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

	body := lipgloss.NewStyle().Width(w).Height(previewH).Render(sliced)

	return headerLine + "\n" + divider + "\n" + body
}

// renderBottomBar draws the two-line footer: status + keybindings.
func (m model) renderBottomBar(width int) string {
	// ── status line ──────────────────────────────────────────────────────────
	statusIcon := "●"
	statusStyle := lipgloss.NewStyle().Foreground(clrStatus)
	statusText := m.status
	if statusText == "ready" {
		statusIcon = "◆"
		statusStyle = lipgloss.NewStyle().Foreground(clrExec)
	}
	// Clamp status text so it never forces a line wrap
	maxStatusW := width - 3 // 1 pad + 1 icon + 1 space
	if maxStatusW < 1 {
		maxStatusW = 1
	}
	statusText = trimVisual(statusText, maxStatusW)
	statusLine := lipgloss.NewStyle().
		Width(width).
		Background(clrDim).
		PaddingLeft(1).
		Render(statusStyle.Render(statusIcon + " " + statusText))

	// ── key hints ────────────────────────────────────────────────────────────
	type hint struct{ key, desc string }
	hints := []hint{
		{"j/k", "move"},
		{"g/G", "top/end"},
		{"enter/l", "open"},
		{"h", "up"},
		{".", "hidden"},
		{"^d/u", "scroll"},
		{"r", "reload"},
		{"q", "quit"},
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
		Background(clrDim).
		PaddingLeft(1).
		Render(strings.Join(parts, ""))

	return statusLine + "\n" + keysLine
}

// ── helpers ────────────────────────────────────────────────────────────────────

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

func (m model) isInPreviewPane(x, y int) bool {
	leftW := max(26, m.width/3)
	bodyH := max(4, m.height-4)
	previewStartX := leftW + 1
	previewStartY := 3 // top bar + body header
	previewEndY := previewStartY + bodyH - 1

	return x >= previewStartX && y >= previewStartY && y <= previewEndY
}

func (m *model) changeDir(path string) error {
	entries, err := listDir(path, m.showHidden)
	if err != nil {
		return err
	}
	m.cwd = path
	m.entries = entries
	m.selected = 0
	m.previewOffset = 0
	m.status = path
	return nil
}

func (m *model) requestPreview() tea.Cmd {
	if len(m.entries) == 0 {
		m.preview = ""
		m.loading = false
		return nil
	}

	picked := m.entries[m.selected]
	cacheKey := previewKey(picked.path, picked.modTime, picked.size, m.width, m.height)
	if val, ok := m.cache[cacheKey]; ok {
		m.preview = val
		m.loading = false
		return nil
	}

	m.requestID++
	requestID := m.requestID
	m.loading = true
	path := picked.path
	width := max(40, m.width-m.width/3-2)
	height := max(8, m.height-4)

	return func() tea.Msg {
		content, err := buildPreview(path, width, height)
		return previewLoadedMsg{
			requestID: requestID,
			cacheKey:  cacheKey,
			content:   content,
			err:       err,
		}
	}
}

func (m *model) slicePreview(in string, h int) string {
	if h <= 0 {
		return ""
	}
	lines := strings.Split(in, "\n")
	maxStart := max(0, len(lines)-h)
	if m.previewOffset > maxStart {
		m.previewOffset = maxStart
	}
	if m.previewOffset < 0 {
		m.previewOffset = 0
	}
	start := m.previewOffset
	end := min(len(lines), start+h)
	return strings.Join(lines[start:end], "\n")
}

func (m *model) clampPreviewOffset() {
	if m.previewOffset < 0 {
		m.previewOffset = 0
	}
	if m.preview == "" {
		m.previewOffset = 0
		return
	}
	lines := strings.Split(m.preview, "\n")
	viewport := m.previewViewportHeight()
	maxStart := max(0, len(lines)-viewport)
	if m.previewOffset > maxStart {
		m.previewOffset = maxStart
	}
}

func (m model) previewViewportHeight() int {
	bodyH := max(4, m.height-4)
	return max(1, bodyH-2)
}

// ── preview builders ──────────────────────────────────────────────────────────

func buildPreview(path string, width, height int) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return buildDirPreview(path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if imageExts[ext] {
		if img, ok := imagePreview(path, width, height); ok {
			return img, nil
		}
		return fmt.Sprintf("image file: %s\nsize: %s\n\npreview unavailable for this format", filepath.Base(path), humanSize(info.Size())), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, maxPreviewBytes)
	n, readErr := f.Read(buf)
	if readErr != nil && readErr != io.EOF {
		return "", readErr
	}
	buf = buf[:n]

	if isLikelyBinary(buf) {
		return fmt.Sprintf("binary file: %s\nsize: %s\nmodified: %s", filepath.Base(path), humanSize(info.Size()), info.ModTime().Format(time.RFC822)), nil
	}

	text := string(buf)
	if !utf8.ValidString(text) {
		return fmt.Sprintf("non-utf8 text file: %s\nsize: %s", filepath.Base(path), humanSize(info.Size())), nil
	}

	switch ext {
	case ".md", ".markdown", ".mdx":
		return renderMarkdownPreview(text, width, n == maxPreviewBytes), nil
	case ".mmd", ".mermaid":
		return renderMermaidNative(text), nil
	case ".json":
		return renderJSONPreview(text, n == maxPreviewBytes), nil
	}

	if highlighted := highlight(path, text); highlighted != "" {
		if n == maxPreviewBytes {
			highlighted += "\n\n... preview truncated ..."
		}
		return highlighted, nil
	}

	if n == maxPreviewBytes {
		text += "\n\n... preview truncated ..."
	}
	return text, nil
}

func buildDirPreview(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	// Styled directory preview
	dirStyle := lipgloss.NewStyle().Foreground(clrDir).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)
	dimStyle := lipgloss.NewStyle().Foreground(clrDim)

	var sb strings.Builder
	sb.WriteString(dirStyle.Render("▸ "+filepath.Base(path)+"/") + "\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("  %d items", len(entries))) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", 30)) + "\n\n")

	limit := min(len(entries), maxDirPreview)
	for i := 0; i < limit; i++ {
		e := entries[i]
		name := e.Name()
		var line string
		if e.IsDir() {
			line = lipgloss.NewStyle().Foreground(clrDir).Render("  ▸ " + name + "/")
		} else {
			// categorise by name only (no stat for speed)
			fakeEntry := entry{name: name, isDir: false}
			cat := categorise(fakeEntry)
			col := fileColor(cat)
			line = col.Render("  " + fileIcon(cat) + name)
		}
		sb.WriteString(line + "\n")
	}
	if len(entries) > limit {
		sb.WriteString(mutedStyle.Render(fmt.Sprintf("\n  … and %d more", len(entries)-limit)) + "\n")
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}

func imagePreview(path string, width, height int) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", false
	}

	rendered := renderImageASCII(img, width, height)
	if rendered == "" {
		return "", false
	}
	return rendered, true
}

func renderMarkdownPreview(markdown string, width int, truncated bool) string {
	prepared := replaceMermaidFences(markdown)
	rendered := prepared
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(24, width-3)),
		glamour.WithTableWrap(true),
		glamour.WithPreservedNewLines(),
		glamour.WithEmoji(),
	)
	if err == nil {
		if out, renderErr := r.Render(prepared); renderErr == nil {
			rendered = out
		}
	}

	if truncated {
		rendered += "\n\n... preview truncated ..."
	}

	return rendered
}

// ── JSON renderer ─────────────────────────────────────────────────────────────

// JSON color tokens
var (
	jsonKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("147"))            // periwinkle – keys
	jsonStr     = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))            // sage green – string values
	jsonNum     = lipgloss.NewStyle().Foreground(lipgloss.Color("222"))            // pale gold – numbers
	jsonBool    = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true) // amber – booleans
	jsonNull    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true) // dim – null
	jsonBracket = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))            // grey – brackets
	jsonMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))            // dim – punctuation / ellipsis
)

func renderJSONPreview(text string, truncated bool) string {
	// Parse into a generic value
	var v interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &v); err != nil {
		// Not valid JSON — show the error and fall back to raw text
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return errStyle.Render("  invalid JSON: "+err.Error()) + "\n\n" + text
	}

	var sb strings.Builder
	writeJSON(&sb, v, 0)
	out := sb.String()

	if truncated {
		out += "\n" + jsonMuted.Render("  … file truncated, showing partial parse")
	}
	return out
}

// writeJSON recursively pretty-prints a JSON value with colour.
func writeJSON(sb *strings.Builder, v interface{}, depth int) {
	indent := strings.Repeat("  ", depth)
	childIndent := strings.Repeat("  ", depth+1)

	switch val := v.(type) {
	case map[string]interface{}:
		if len(val) == 0 {
			sb.WriteString(jsonBracket.Render("{}"))
			return
		}
		sb.WriteString(jsonBracket.Render("{") + "\n")
		// Sort keys for deterministic output
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			sb.WriteString(childIndent)
			sb.WriteString(jsonKey.Render(`"` + k + `"`))
			sb.WriteString(jsonMuted.Render(": "))
			writeJSON(sb, val[k], depth+1)
			if i < len(keys)-1 {
				sb.WriteString(jsonMuted.Render(","))
			}
			sb.WriteString("\n")
		}
		sb.WriteString(indent + jsonBracket.Render("}"))

	case []interface{}:
		if len(val) == 0 {
			sb.WriteString(jsonBracket.Render("[]"))
			return
		}
		sb.WriteString(jsonBracket.Render("[") + "\n")
		// Cap array preview at 100 items to avoid enormous output
		limit := len(val)
		capped := false
		if limit > 100 {
			limit = 100
			capped = true
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(childIndent)
			writeJSON(sb, val[i], depth+1)
			if i < len(val)-1 {
				sb.WriteString(jsonMuted.Render(","))
			}
			sb.WriteString("\n")
		}
		if capped {
			sb.WriteString(childIndent + jsonMuted.Render(fmt.Sprintf("… %d more items", len(val)-limit)) + "\n")
		}
		sb.WriteString(indent + jsonBracket.Render("]"))

	case string:
		// Escape double quotes inside the string for display
		escaped := strings.ReplaceAll(val, `"`, `\"`)
		sb.WriteString(jsonStr.Render(`"` + escaped + `"`))

	case float64:
		// Render as integer when there's no fractional part
		if val == float64(int64(val)) {
			sb.WriteString(jsonNum.Render(fmt.Sprintf("%d", int64(val))))
		} else {
			sb.WriteString(jsonNum.Render(fmt.Sprintf("%g", val)))
		}

	case bool:
		if val {
			sb.WriteString(jsonBool.Render("true"))
		} else {
			sb.WriteString(jsonBool.Render("false"))
		}

	case nil:
		sb.WriteString(jsonNull.Render("null"))

	default:
		sb.WriteString(fmt.Sprintf("%v", val))
	}
}

func renderImageASCII(img image.Image, width, height int) string {
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return ""
	}

	chars := []rune(" .:-=+*#%@")
	outW := max(16, width-2)
	outH := max(8, height-3)

	var sb strings.Builder
	for y := 0; y < outH; y++ {
		sy := b.Min.Y + (y*(b.Dy()-1))/max(1, outH-1)
		for x := 0; x < outW; x++ {
			sx := b.Min.X + (x*(b.Dx()-1))/max(1, outW-1)
			lum := luminance(img.At(sx, sy))
			idx := int(lum * float64(len(chars)-1) / 255.0)
			if idx < 0 {
				idx = 0
			}
			if idx >= len(chars) {
				idx = len(chars) - 1
			}
			sb.WriteRune(chars[idx])
		}
		if y < outH-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	rf := float64(r>>8) * 0.299
	gf := float64(g>>8) * 0.587
	bf := float64(b>>8) * 0.114
	return rf + gf + bf
}

func replaceMermaidFences(markdown string) string {
	lines := strings.Split(markdown, "\n")
	inMermaidFence := false
	var block strings.Builder
	transformed := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inMermaidFence {
			if strings.HasPrefix(trimmed, "```") {
				lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				if strings.EqualFold(lang, "mermaid") {
					inMermaidFence = true
					block.Reset()
					continue
				}
			}
			transformed = append(transformed, line)
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			content := strings.TrimSpace(block.String())
			if content != "" {
				transformed = append(transformed, "")
				transformed = append(transformed, renderMermaidMarkdownPreview(content))
				transformed = append(transformed, "")
			}
			inMermaidFence = false
			continue
		}
		block.WriteString(line)
		block.WriteString("\n")
	}

	if inMermaidFence {
		transformed = append(transformed, "```mermaid")
		transformed = append(transformed, strings.TrimRight(block.String(), "\n"))
	}

	return strings.Join(transformed, "\n")
}

type mermaidNode struct {
	id    string
	label string
}

type mermaidEdge struct {
	from      mermaidNode
	to        mermaidNode
	edgeLabel string
}

type mermaidGraph struct {
	chartType string
	nodeOrder []string
	nodes     map[string]string
	edges     []mermaidEdge
}

func renderMermaidNative(code string) string {
	g := parseMermaidGraph(code)
	if len(g.edges) == 0 {
		return "Mermaid (native preview)\n\n" + code
	}

	var sb strings.Builder
	sb.WriteString("Mermaid (native preview)\n")
	sb.WriteString("type: ")
	sb.WriteString(g.chartType)
	sb.WriteString("\n")
	sb.WriteString("nodes: ")
	sb.WriteString(fmt.Sprintf("%d", len(g.nodeOrder)))
	sb.WriteString("  edges: ")
	sb.WriteString(fmt.Sprintf("%d", len(g.edges)))
	sb.WriteString("\n\n")

	maxEdges := min(len(g.edges), 30)
	for i := 0; i < maxEdges; i++ {
		e := g.edges[i]
		sb.WriteString(fmt.Sprintf("%2d. %s -> %s", i+1, fitMermaidLabel(e.from.label), fitMermaidLabel(e.to.label)))
		if e.edgeLabel != "" {
			sb.WriteString("  [")
			sb.WriteString(fitMermaidLabel(e.edgeLabel))
			sb.WriteString("]")
		}
		sb.WriteByte('\n')
	}
	if len(g.edges) > maxEdges {
		sb.WriteString("... and ")
		sb.WriteString(fmt.Sprintf("%d", len(g.edges)-maxEdges))
		sb.WriteString(" more edges")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func renderMermaidMarkdownPreview(code string) string {
	g := parseMermaidGraph(code)
	if len(g.edges) == 0 {
		return "_Mermaid block found, but no flow edges were parsed._"
	}

	var sb strings.Builder
	sb.WriteString("#### Mermaid Preview\n\n")
	sb.WriteString("**Type:** `")
	sb.WriteString(g.chartType)
	sb.WriteString("`  \\n")
	sb.WriteString("**Nodes:** `")
	sb.WriteString(fmt.Sprintf("%d", len(g.nodeOrder)))
	sb.WriteString("`  \\n")
	sb.WriteString("**Edges:** `")
	sb.WriteString(fmt.Sprintf("%d", len(g.edges)))
	sb.WriteString("`\n\n")

	sb.WriteString("| # | From | To | Label |\n")
	sb.WriteString("|---:|------|----|-------|\n")
	maxEdges := min(len(g.edges), 20)
	for i := 0; i < maxEdges; i++ {
		e := g.edges[i]
		label := e.edgeLabel
		if label == "" {
			label = "-"
		}
		sb.WriteString("| ")
		sb.WriteString(fmt.Sprintf("%d", i+1))
		sb.WriteString(" | ")
		sb.WriteString(escapeMarkdownTableCell(fitMermaidLabel(e.from.label)))
		sb.WriteString(" | ")
		sb.WriteString(escapeMarkdownTableCell(fitMermaidLabel(e.to.label)))
		sb.WriteString(" | ")
		sb.WriteString(escapeMarkdownTableCell(fitMermaidLabel(label)))
		sb.WriteString(" |\n")
	}
	if len(g.edges) > maxEdges {
		sb.WriteString("\n_...and ")
		sb.WriteString(fmt.Sprintf("%d", len(g.edges)-maxEdges))
		sb.WriteString(" more edges._\n")
	}

	return sb.String()
}

func parseMermaidGraph(code string) mermaidGraph {
	lines := strings.Split(code, "\n")
	edges := make([]mermaidEdge, 0)
	chartType := "diagram"

	nodeOrder := make([]string, 0)
	nodes := make(map[string]string)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		if chartType == "diagram" {
			parts := strings.Fields(trimmed)
			if len(parts) > 0 {
				chartType = parts[0]
			}
		}

		edge, ok := parseMermaidEdge(trimmed)
		if !ok {
			continue
		}
		edges = append(edges, edge)
		registerMermaidNode(nodes, &nodeOrder, edge.from)
		registerMermaidNode(nodes, &nodeOrder, edge.to)
	}

	return mermaidGraph{
		chartType: chartType,
		nodeOrder: nodeOrder,
		nodes:     nodes,
		edges:     edges,
	}
}

func parseMermaidEdge(line string) (mermaidEdge, bool) {
	operators := []string{"-->", "==>", "-.->"}
	op := ""
	idx := -1
	for _, candidate := range operators {
		i := strings.Index(line, candidate)
		if i >= 0 && (idx == -1 || i < idx) {
			idx = i
			op = candidate
		}
	}
	if idx < 0 {
		return mermaidEdge{}, false
	}

	left := strings.TrimSpace(line[:idx])
	right := strings.TrimSpace(line[idx+len(op):])
	edgeLabel := ""
	if strings.HasPrefix(right, "|") {
		if end := strings.Index(right[1:], "|"); end >= 0 {
			edgeLabel = strings.TrimSpace(right[1 : end+1])
			right = strings.TrimSpace(right[end+2:])
		}
	}

	from := parseMermaidNode(left)
	to := parseMermaidNode(right)
	if from.id == "" || to.id == "" {
		return mermaidEdge{}, false
	}
	return mermaidEdge{from: from, to: to, edgeLabel: edgeLabel}, true
}

func parseMermaidNode(raw string) mermaidNode {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, ";"))
	if raw == "" {
		return mermaidNode{}
	}
	raw = strings.Split(raw, ":::")[0]

	openToClose := map[byte]byte{'[': ']', '(': ')', '{': '}'}
	for open, close := range openToClose {
		if i := strings.IndexByte(raw, open); i > 0 {
			id := strings.TrimSpace(raw[:i])
			if j := strings.LastIndexByte(raw, close); j > i {
				label := cleanMermaidText(raw[i+1 : j])
				if label == "" {
					label = cleanMermaidText(id)
				}
				return mermaidNode{id: cleanMermaidID(id), label: label}
			}
		}
	}

	id := cleanMermaidID(raw)
	if id == "" {
		return mermaidNode{}
	}
	return mermaidNode{id: id, label: cleanMermaidText(id)}
}

func registerMermaidNode(nodes map[string]string, order *[]string, n mermaidNode) {
	if n.id == "" {
		return
	}
	if _, exists := nodes[n.id]; !exists {
		nodes[n.id] = n.label
		*order = append(*order, n.id)
	}
}

func cleanMermaidText(in string) string {
	in = strings.TrimSpace(in)
	replacer := strings.NewReplacer("\"", "", "'", "", "|", " ", "`", "")
	in = replacer.Replace(in)
	in = strings.Join(strings.Fields(in), " ")
	return in
}

func cleanMermaidID(in string) string {
	in = strings.TrimSpace(in)
	in = strings.TrimPrefix(in, "(")
	in = strings.TrimPrefix(in, "[")
	in = strings.TrimPrefix(in, "{")
	in = strings.TrimSuffix(in, ")")
	in = strings.TrimSuffix(in, "]")
	in = strings.TrimSuffix(in, "}")
	in = strings.TrimSuffix(in, ";")
	fields := strings.Fields(in)
	if len(fields) == 0 {
		return ""
	}
	in = fields[0]
	in = strings.TrimSpace(in)
	return strings.Trim(in, "\"")
}

func fitMermaidLabel(in string) string {
	in = cleanMermaidText(in)
	if in == "" {
		return "node"
	}
	if len(in) <= 24 {
		return in
	}
	return in[:23] + "~"
}

func escapeMarkdownTableCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.TrimSpace(s)
}

func listDir(path string, showHidden bool) ([]entry, error) {
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(items))
	for _, item := range items {
		name := item.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		info, err := item.Info()
		if err != nil {
			continue
		}
		entries = append(entries, entry{
			name:    name,
			path:    full,
			isDir:   item.IsDir(),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})

	return entries, nil
}

func previewKey(path string, modTime time.Time, size int64, width, height int) string {
	return fmt.Sprintf("%s|%d|%d|%d|%d", path, modTime.UnixNano(), size, width, height)
}

func highlight(path, text string) string {
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Analyse(text)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("nord")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return ""
	}
	return buf.String()
}

func isLikelyBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	for i := 0; i < len(data) && i < 8192; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

func trimToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "…"
}

func humanSize(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(n)
	idx := 0
	for v >= 1024 && idx < len(units)-1 {
		v /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%d %s", n, units[idx])
	}
	return fmt.Sprintf("%.1f %s", v, units[idx])
}

func previewPageSize(h int) int {
	return max(3, h/3)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
