package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/truncate"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
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
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tiff":
		return catImage
	case ".md", ".markdown", ".mdx", ".rst", ".txt":
		return catDoc
	case ".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd":
		return catExec
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rb", ".rs", ".c", ".cpp",
		".h", ".java", ".cs", ".php", ".swift", ".kt",
		".lua", ".ex", ".exs", ".hs", ".ml", ".mli", ".clj", ".scala",
		".vim", ".mmd", ".mermaid":
		return catCode
	case ".json", ".yaml", ".yml", ".toml", ".ini", ".env", ".conf", ".config",
		".xml", ".dockerignore", ".gitignore", ".editorconfig", ".eslintrc",
		".prettierrc", ".babelrc", ".nvmrc":
		return catConfig
	}
	return catOther
}

// nerdFonts controls whether Nerd Font glyphs are used.
// Set SEER_NO_NERD_FONT=1 to force plain Unicode fallback.
var nerdFonts = os.Getenv("SEER_NO_NERD_FONT") != "1"

// nerdIconByExt maps file extensions to specific Nerd Font glyphs.
var nerdIconByExt = map[string]string{
	// languages
	".go":    "\ue627 ",  //
	".js":    "\ue60c ",  //
	".ts":    "\ue628 ",  //
	".jsx":   "\ue60c ",  //
	".tsx":   "\ue60c ",  //
	".py":    "\ue606 ",  //
	".rb":    "\ue21e ",  //
	".rs":    "\ue7a8 ",  //
	".c":     "\ue61e ",  //
	".cpp":   "\ue61d ",  //
	".h":     "\uf0fd ",  //
	".java":  "\ue204 ",  //
	".cs":    "\uf031b ", // 󰌛
	".php":   "\ue60a ",  //
	".swift": "\ue755 ",  //
	".kt":    "\ue634 ",  //
	".lua":   "\ue620 ",  //
	".hs":    "\ue61f ",  //
	".vim":   "\ue62b ",  //
	".sh":    "\uf489 ",  //
	".bash":  "\uf489 ",  //
	".zsh":   "\uf489 ",  //
	".fish":  "\uf489 ",  //
	".ps1":   "\uf489 ",  //
	".bat":   "\uf489 ",  //
	".cmd":   "\uf489 ",  //
	// docs
	".md":       "\ue609 ", //
	".markdown": "\ue609 ", //
	".mdx":      "\ue609 ", //
	".rst":      "\uf15c ", //
	".txt":      "\uf15c ", //
	// config
	".json": "\ue60b ",  //
	".yaml": "\uf481 ",  //
	".yml":  "\uf481 ",  //
	".toml": "\uf481 ",  //
	".xml":  "\uf05c0 ", // 󰗀
	".env":  "\uf462 ",  //
	".ini":  "\uf17a ",  //
	".conf": "\uf17a ",  //
	// images
	".png":  "\uf1c5 ", //
	".jpg":  "\uf1c5 ", //
	".jpeg": "\uf1c5 ", //
	".gif":  "\uf1c5 ", //
	".webp": "\uf1c5 ", //
	".svg":  "\uf1c5 ", //
	".bmp":  "\uf1c5 ", //
	// misc
	".mmd":          "\ueb43 ", //
	".mermaid":      "\ueb43 ", //
	".pdf":          "\uf1c1 ", //
	".zip":          "\uf410 ", //
	".tar":          "\uf410 ", //
	".gz":           "\uf410 ", //
	".gitignore":    "\ue702 ", //
	".dockerignore": "\uf308 ", //
}

// nerdIconByCategory is the fallback Nerd Font icon per broad category.
var nerdIconByCategory = map[fileCategory]string{
	catDir:    "\uf74a ", //
	catImage:  "\uf1c5 ", //
	catDoc:    "\uf15c ", //
	catCode:   "\uf121 ", //
	catConfig: "\uf462 ", //
	catExec:   "\uf489 ", //
	catBinary: "\uf471 ", //
}

// plainIcon is the Unicode-only fallback per category.
var plainIcon = map[fileCategory]string{
	catDir:    "▸ ",
	catImage:  "⬡ ",
	catDoc:    "≡ ",
	catCode:   "⟨⟩ ",
	catConfig: "⚙ ",
	catExec:   "⚡ ",
	catBinary: "⬟ ",
}

func fileIcon(cat fileCategory) string {
	return fileIconExt(cat, "")
}

func fileIconExt(cat fileCategory, ext string) string {
	if !nerdFonts {
		if icon, ok := plainIcon[cat]; ok {
			return icon
		}
		return "· "
	}
	if ext != "" {
		if icon, ok := nerdIconByExt[strings.ToLower(ext)]; ok {
			return icon
		}
	}
	if icon, ok := nerdIconByCategory[cat]; ok {
		return icon
	}
	return "\uf15b " // generic file
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

type selectionPoint struct {
	x int
	y int
}

const previewCacheMax = 50

type model struct {
	cwd           string
	allEntries    []entry // full unfiltered listing
	entries       []entry // visible (filtered) listing
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
	cacheOrder    []string // LRU insertion order for cache eviction
	// Search / filter state
	searching   bool
	searchQuery string
	// Delete confirmation dialog
	confirmingDelete bool
	deleteTarget     string
	// Preview mouse selection state for auto-copy on release.
	previewSelecting bool
	previewSelStart  selectionPoint
	previewSelEnd    selectionPoint
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
		allEntries: entries,
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

// navigate sets the selected index, resets the preview scroll, and returns a
// requestPreview command. It is the single canonical way to change selection.
func (m *model) navigate(idx int) tea.Cmd {
	m.selected = idx
	m.previewOffset = 0
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
		// Handle delete confirmation at top level
		if m.confirmingDelete {
			key := msg.String()
			if key == "y" || key == "Y" || key == "enter" {
				if err := moveToTrash(m.deleteTarget); err != nil {
					m.status = "delete failed: " + err.Error()
				} else {
					m.status = "moved to trash"
					entries, err := listDir(m.cwd, m.showHidden)
					if err != nil {
						m.status = err.Error()
					} else {
						m.allEntries = entries
						m.entries = m.applySearch(entries)
						if m.selected >= len(m.entries) {
							m.selected = max(0, len(m.entries)-1)
						}
					}
				}
				m.confirmingDelete = false
				m.deleteTarget = ""
				return m, m.requestPreview()
			}
			if key == "n" || key == "N" || key == "esc" {
				m.confirmingDelete = false
				m.deleteTarget = ""
				m.status = "delete cancelled"
				return m, nil
			}
			return m, nil
		}

		// In search mode, printable characters extend the query.
		if m.searching && len(msg.Runes) == 1 {
			m.searchQuery += string(msg.Runes)
			m.entries = m.applySearch(m.allEntries)
			m.selected = 0
			return m, m.requestPreview()
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.selected < len(m.entries)-1 {
				return m, m.navigate(m.selected + 1)
			}
		case "k", "up":
			if m.selected > 0 {
				return m, m.navigate(m.selected - 1)
			}
		case "g", "home":
			return m, m.navigate(0)
		case "G", "end":
			if len(m.entries) > 0 {
				return m, m.navigate(len(m.entries) - 1)
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
			return m, m.requestPreview()
		case "h", "left", "backspace":
			if m.searching {
				if key := msg.String(); key == "backspace" && len(m.searchQuery) > 0 {
					runes := []rune(m.searchQuery)
					m.searchQuery = string(runes[:len(runes)-1])
					m.entries = m.applySearch(m.allEntries)
					m.selected = 0
					return m, m.requestPreview()
				}
				break
			}
			parent := filepath.Dir(m.cwd)
			if parent != m.cwd {
				if err := m.changeDir(parent); err != nil {
					m.status = err.Error()
				}
				return m, m.requestPreview()
			}
		case "delete":
			if len(m.entries) > 0 && m.selected < len(m.entries) {
				m.confirmingDelete = true
				m.deleteTarget = m.entries[m.selected].path
				m.status = "confirm delete: y/n"
				return m, nil
			}
		case ".":
			// Remember current filename so we can restore position after reload.
			var prevName string
			if m.selected < len(m.entries) {
				prevName = m.entries[m.selected].name
			}
			m.showHidden = !m.showHidden
			entries, err := listDir(m.cwd, m.showHidden)
			if err != nil {
				m.status = err.Error()
			} else {
				m.allEntries = entries
				m.entries = m.applySearch(entries)
				// Restore selection to the same file if still visible.
				m.selected = 0
				for i, e := range m.entries {
					if e.name == prevName {
						m.selected = i
						break
					}
				}
				m.previewOffset = 0
				if m.showHidden {
					m.status = "showing hidden files"
				} else {
					m.status = "hiding hidden files"
				}
			}
			return m, m.requestPreview()
		case "/":
			m.searching = true
			m.searchQuery = ""
			return m, nil
		case "esc":
			if m.searching {
				m.searching = false
				m.searchQuery = ""
				m.entries = m.allEntries
				m.selected = 0
				return m, m.requestPreview()
			}
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
				m.allEntries = entries
				m.entries = m.applySearch(entries)
				if m.selected >= len(m.entries) {
					m.selected = max(0, len(m.entries)-1)
				}
				m.status = "reloaded"
			}
			return m, m.requestPreview()
		}

	case tea.MouseMsg:
		event := tea.MouseEvent(msg)
		inPreviewPane := m.isInPreviewPane(event.X, event.Y)
		inPreviewBody := m.isInPreviewBody(event.X, event.Y)

		if event.IsWheel() {
			if !inPreviewPane {
				return m, nil
			}
			scroll := previewPageSize(m.height) / 3
			if scroll < 1 {
				scroll = 1
			}
			switch event.Button {
			case tea.MouseButtonWheelDown:
				m.previewOffset += scroll
				m.clampPreviewOffset()
			case tea.MouseButtonWheelUp:
				m.previewOffset -= scroll
				m.clampPreviewOffset()
			}
			return m, nil
		}

		// Track left-button drag in the preview body and auto-copy on release.
		switch event.Action {
		case tea.MouseActionPress:
			if event.Button == tea.MouseButtonLeft && inPreviewBody {
				m.previewSelecting = true
				p := m.previewBodyPoint(event.X, event.Y)
				m.previewSelStart = p
				m.previewSelEnd = p
			}
		case tea.MouseActionMotion:
			if m.previewSelecting {
				m.previewSelEnd = m.previewBodyPoint(event.X, event.Y)
			}
		case tea.MouseActionRelease:
			if m.previewSelecting && (event.Button == tea.MouseButtonLeft || event.Button == tea.MouseButtonNone) {
				m.previewSelEnd = m.previewBodyPoint(event.X, event.Y)
				selected := m.selectedPreviewText()
				m.previewSelecting = false
				if selected == "" {
					return m, nil
				}
				if err := copyToClipboard(selected); err != nil {
					m.status = "copy failed: " + err.Error()
					return m, nil
				}
				m.status = fmt.Sprintf("copied %d chars", utf8.RuneCountInString(selected))
			}
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
		m.cacheSet(msg.cacheKey, msg.content)
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
	leftW, rightW, bodyH := m.layoutDimensions()

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

	if m.confirmingDelete {
		dialog := m.renderDeleteDialog(leftW + rightW + 1)
		return topBar + "\n" + dialog + "\n" + bottomBar
	}

	return topBar + "\n" + body + "\n" + bottomBar
}

func (m model) renderDeleteDialog(width int) string {
	fileName := filepath.Base(m.deleteTarget)
	dialogWidth := min(60, width-4)
	dialogHeight := 5

	content := fmt.Sprintf("Delete \"%s\"? [y/n]", fileName)
	contentLen := lipgloss.Width(content)
	padding := (dialogWidth - contentLen - 2) / 2
	extra := (dialogWidth - contentLen - 2) % 2

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrAccent).
		Foreground(clrAccentFg)

	contentRow := borderStyle.Width(dialogWidth).Render(
		strings.Repeat(" ", padding) + content + strings.Repeat(" ", padding+extra),
	)
	bottomRow := borderStyle.Width(dialogWidth).Render(
		" y = confirm  ·  n / esc = cancel ",
	)

	dialogBox := lipgloss.JoinVertical(
		lipgloss.Center,
		contentRow,
		bottomRow,
	)

	dialogH := lipgloss.Height(dialogBox)
	topPad := (dialogHeight - dialogH) / 2
	bottomPad := (dialogHeight - dialogH) - topPad

	lines := make([]string, 0, dialogHeight)
	for i := 0; i < topPad; i++ {
		lines = append(lines, strings.Repeat(" ", dialogWidth))
	}
	lines = append(lines, dialogBox)
	for i := 0; i < bottomPad; i++ {
		lines = append(lines, strings.Repeat(" ", dialogWidth))
	}

	dialogStr := lipgloss.JoinVertical(lipgloss.Center, lines...)
	dialogStr = borderStyle.Padding(0, 1).Render(dialogStr)

	sidePad := (width - dialogWidth - 2) / 2

	allLines := strings.Split(dialogStr, "\n")
	result := make([]string, 0, dialogHeight)
	for _, line := range allLines {
		result = append(result, strings.Repeat(" ", sidePad)+line)
	}
	return lipgloss.JoinVertical(lipgloss.Top, result...)
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
	// Size column is 9 chars wide ("1023.9 KB" = 9 chars max), separated by a space.
	sizeW := 9
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
			icon := fileIconExt(cat, filepath.Ext(e.name))
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
		icon := fileIconExt(cat, filepath.Ext(e.name))
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

	// Truncate each line to the pane width so no line can wrap in the terminal
	// and push the top/bottom chrome off screen.
	if w > 0 {
		rawLines := strings.Split(sliced, "\n")
		for i, line := range rawLines {
			if lipgloss.Width(line) > w {
				rawLines[i] = truncate.String(line, uint(w))
			}
		}
		sliced = strings.Join(rawLines, "\n")
	}

	body := lipgloss.NewStyle().Width(w).Height(previewH).Render(sliced)

	return headerLine + "\n" + divider + "\n" + body
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
			Background(clrDim).
			PaddingLeft(1).
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
			Background(clrDim).
			PaddingLeft(1).
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
	previewStartY := 3 // top bar + body header
	previewEndY := previewStartY + bodyH - 1

	return x >= previewStartX && x <= previewEndX && y >= previewStartY && y <= previewEndY
}

func (m model) previewBodyRect() (startX, startY, width, height int) {
	leftW, rightW, bodyH := m.layoutDimensions()
	startX = leftW + 1
	startY = 3
	width = max(1, rightW)
	height = max(1, bodyH-2)
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

func copyToClipboard(text string) error {
	if text == "" {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		return runClipboardCommand(text, "pbcopy")
	case "windows":
		return runClipboardCommand(text, "cmd", "/c", "clip")
	default:
		candidates := [][]string{
			{"wl-copy"},
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
		}
		var lastErr error
		for _, c := range candidates {
			if _, err := exec.LookPath(c[0]); err != nil {
				continue
			}
			if err := runClipboardCommand(text, c[0], c[1:]...); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		if lastErr != nil {
			return lastErr
		}
		return errors.New("no clipboard utility found (tried wl-copy, xclip, xsel)")
	}
}

func runClipboardCommand(text, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func (m *model) changeDir(path string) error {
	entries, err := listDir(path, m.showHidden)
	if err != nil {
		return err
	}
	m.cwd = path
	m.allEntries = entries
	m.entries = entries
	m.selected = 0
	m.previewOffset = 0
	m.searchQuery = ""
	m.searching = false
	m.status = path
	return nil
}

// applySearch filters entries by the current searchQuery (case-insensitive substring).
// Returns all entries unchanged when the query is empty.
func (m model) applySearch(entries []entry) []entry {
	if m.searchQuery == "" {
		return entries
	}
	q := strings.ToLower(m.searchQuery)
	var out []entry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.name), q) {
			out = append(out, e)
		}
	}
	return out
}

// cacheSet stores a preview result and evicts the oldest entry when the cache
// exceeds previewCacheMax entries.
func (m *model) cacheSet(key, value string) {
	if _, exists := m.cache[key]; !exists {
		m.cacheOrder = append(m.cacheOrder, key)
	}
	m.cache[key] = value
	for len(m.cacheOrder) > previewCacheMax {
		oldest := m.cacheOrder[0]
		m.cacheOrder = m.cacheOrder[1:]
		delete(m.cache, oldest)
	}
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
	_, rightW, bodyH := m.layoutDimensions()
	width := max(40, rightW)
	height := max(8, bodyH)

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
	// Normalize Windows-style line endings so \r doesn't corrupt terminal rendering.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

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
	sb.WriteString(dirStyle.Render(fileIconExt(catDir, "")+filepath.Base(path)+"/") + "\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("  %d items", len(entries))) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", 30)) + "\n\n")

	limit := min(len(entries), maxDirPreview)
	for i := 0; i < limit; i++ {
		e := entries[i]
		name := e.Name()
		var line string
		if e.IsDir() {
			line = lipgloss.NewStyle().Foreground(clrDir).Render("  " + fileIconExt(catDir, "") + name + "/")
		} else {
			// categorise by name only (no stat for speed)
			fakeEntry := entry{name: name, isDir: false}
			cat := categorise(fakeEntry)
			col := fileColor(cat)
			line = col.Render("  " + fileIconExt(cat, filepath.Ext(name)) + name)
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
		glamour.WithStandardStyle("tokyo-night"),
		glamour.WithWordWrap(max(24, width-2)),
		glamour.WithTableWrap(true),
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

	outW := max(16, width-2)
	outH := max(8, height-3)

	if supportsTrueColor() {
		return renderImageTrueColor(img, outW, outH)
	}
	return renderImageGray(img, outW, outH)
}

func rgbValues(c color.Color) (int, int, int) {
	r, g, b, _ := c.RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8)
}

func renderImageTrueColor(img image.Image, outW, outH int) string {
	b := img.Bounds()
	scaledH := outH * 2

	var sb strings.Builder
	for row := 0; row < outH; row++ {
		upperY := b.Min.Y + ((row*2)*(b.Dy()-1))/max(1, scaledH-1)
		lowerY := b.Min.Y + ((row*2+1)*(b.Dy()-1))/max(1, scaledH-1)

		lastFgR, lastFgG, lastFgB := -1, -1, -1
		lastBgR, lastBgG, lastBgB := -1, -1, -1

		for x := 0; x < outW; x++ {
			sx := b.Min.X + (x*(b.Dx()-1))/max(1, outW-1)
			fgR, fgG, fgB := rgbValues(img.At(sx, upperY))
			bgR, bgG, bgB := rgbValues(img.At(sx, lowerY))

			if fgR != lastFgR || fgG != lastFgG || fgB != lastFgB || bgR != lastBgR || bgG != lastBgG || bgB != lastBgB {
				writeTrueColorANSI(&sb, fgR, fgG, fgB, bgR, bgG, bgB)
				lastFgR, lastFgG, lastFgB = fgR, fgG, fgB
				lastBgR, lastBgG, lastBgB = bgR, bgG, bgB
			}
			sb.WriteRune('▀')
		}

		sb.WriteString("\x1b[0m")
		if row < outH-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func renderImageGray(img image.Image, outW, outH int) string {
	b := img.Bounds()
	chars := []rune(" .:-=+*#%@")

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

func writeTrueColorANSI(sb *strings.Builder, fgR, fgG, fgB, bgR, bgG, bgB int) {
	fmt.Fprintf(sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm", fgR, fgG, fgB, bgR, bgG, bgB)
}

func supportsTrueColor() bool {
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		return false
	}
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))
	if strings.Contains(colorTerm, "truecolor") || strings.Contains(colorTerm, "24bit") {
		return true
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(term, "kitty") || strings.Contains(term, "wezterm") {
		return true
	}
	return false
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
	ct := mermaidChartType(code)
	switch ct {
	case "sequenceDiagram":
		parts, msgs := parseSequenceDiagram(code)
		if len(msgs) > 0 {
			return asciiSequenceDiagram(parts, msgs, 0)
		}
	default:
		g := parseMermaidGraph(code)
		if len(g.nodeOrder) > 0 {
			return asciiFlowchart(g, 0)
		}
	}
	return code
}

func renderMermaidMarkdownPreview(code string) string {
	ct := mermaidChartType(code)
	var art string
	switch ct {
	case "sequenceDiagram":
		parts, msgs := parseSequenceDiagram(code)
		if len(msgs) > 0 {
			art = asciiSequenceDiagram(parts, msgs, 80)
		}
	default:
		g := parseMermaidGraph(code)
		if len(g.nodeOrder) > 0 {
			art = asciiFlowchart(g, 80)
		}
	}
	if art == "" {
		return "_Mermaid block: no diagram content parsed._"
	}
	return "```text\n" + art + "\n```\n"
}

func mermaidChartType(code string) string {
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		if parts := strings.Fields(trimmed); len(parts) > 0 {
			return parts[0]
		}
	}
	return "diagram"
}

// ── ASCII flowchart renderer ──────────────────────────────────────────────────

// boxDrawMask maps box-drawing runes to a 4-bit NESW connectivity mask.
// N=8 E=4 S=2 W=1 — combining masks via OR produces the correct merged character.
var boxDrawMask = map[rune]int{
	'│': 8 + 2, '─': 4 + 1,
	'┌': 4 + 2, '┐': 1 + 2, '└': 8 + 4, '┘': 8 + 1,
	'├': 8 + 4 + 2, '┤': 8 + 1 + 2, '┬': 4 + 1 + 2, '┴': 8 + 4 + 1, '┼': 15,
}

// maskBoxDraw is the reverse of boxDrawMask.
var maskBoxDraw = map[int]rune{
	4 + 2: '┌', 1 + 2: '┐', 8 + 4: '└', 8 + 1: '┘',
	4 + 1: '─', 8 + 2: '│',
	8 + 4 + 2: '├', 8 + 1 + 2: '┤', 4 + 1 + 2: '┬', 8 + 4 + 1: '┴', 15: '┼',
	4: '╶', 1: '╴', 8: '╵', 2: '╷',
}

// asciiFlowchart renders a mermaid flowchart as an ASCII box diagram.
func asciiFlowchart(g mermaidGraph, maxW int) string {
	if len(g.nodeOrder) == 0 {
		return "(empty diagram)"
	}

	labelOf := func(id string) string {
		if l := g.nodes[id]; l != "" {
			return l
		}
		return id
	}
	nodeBoxW := func(id string) int { return len([]rune(labelOf(id))) + 4 }

	// Build adjacency and in-degree
	succMap := make(map[string][]string)
	tmpIn := make(map[string]int)
	for id := range g.nodes {
		tmpIn[id] = 0
	}
	for _, e := range g.edges {
		if e.from.id == e.to.id {
			continue
		}
		succMap[e.from.id] = append(succMap[e.from.id], e.to.id)
		tmpIn[e.to.id]++
	}

	// Longest-path ranking via Kahn's algorithm
	rank := make(map[string]int)
	q := []string{}
	for id := range g.nodes {
		if tmpIn[id] == 0 {
			q = append(q, id)
		}
	}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, next := range succMap[cur] {
			if rank[cur]+1 > rank[next] {
				rank[next] = rank[cur] + 1
			}
			tmpIn[next]--
			if tmpIn[next] == 0 {
				q = append(q, next)
			}
		}
	}

	maxRank := 0
	for _, r := range rank {
		if r > maxRank {
			maxRank = r
		}
	}

	// Group nodes by rank, preserving nodeOrder within each level
	levels := make([][]string, maxRank+1)
	for _, id := range g.nodeOrder {
		r := rank[id]
		levels[r] = append(levels[r], id)
	}

	// Compute x positions within each level
	const hGap = 3
	nodeX := make(map[string]int)
	levelW := make([]int, maxRank+1)
	for r, nodes := range levels {
		x := 0
		for i, id := range nodes {
			nodeX[id] = x
			x += nodeBoxW(id)
			if i < len(nodes)-1 {
				x += hGap
			}
		}
		levelW[r] = x
	}

	totalW := 1
	for _, lw := range levelW {
		if lw > totalW {
			totalW = lw
		}
	}
	if maxW > 0 && totalW > maxW {
		totalW = maxW
	}

	// Center each level within totalW
	for r, lw := range levelW {
		off := (totalW - lw) / 2
		if off < 0 {
			off = 0
		}
		for _, id := range levels[r] {
			nodeX[id] += off
		}
	}

	// Y positions: 3 rows per box + 2 connector rows between levels
	nodeY := make(map[string]int)
	y := 0
	for r, nodes := range levels {
		for _, id := range nodes {
			nodeY[id] = y
		}
		y += 3
		if r < maxRank {
			y += 2
		}
	}
	totalH := y

	// Grid
	grid := make([][]rune, totalH)
	for i := range grid {
		grid[i] = make([]rune, totalW)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	setRaw := func(x, y int, r rune) {
		if x >= 0 && x < totalW && y >= 0 && y < totalH {
			grid[y][x] = r
		}
	}
	// setBox merges box-drawing characters so branching connectors combine cleanly.
	setBox := func(x, y int, r rune) {
		if x < 0 || x >= totalW || y < 0 || y >= totalH {
			return
		}
		existing := grid[y][x]
		if existing == ' ' {
			grid[y][x] = r
			return
		}
		em, ok1 := boxDrawMask[existing]
		nm, ok2 := boxDrawMask[r]
		if ok1 && ok2 {
			if merged, ok3 := maskBoxDraw[em|nm]; ok3 {
				grid[y][x] = merged
				return
			}
		}
		grid[y][x] = r
	}
	writeStr := func(x, y int, s string) {
		for i, r := range []rune(s) {
			setRaw(x+i, y, r)
		}
	}

	// Draw boxes
	for id := range g.nodes {
		label := labelOf(id)
		x, yy, w := nodeX[id], nodeY[id], nodeBoxW(id)
		setRaw(x, yy, '┌')
		for i := 1; i < w-1; i++ {
			setRaw(x+i, yy, '─')
		}
		setRaw(x+w-1, yy, '┐')
		setRaw(x, yy+1, '│')
		writeStr(x+2, yy+1, label)
		setRaw(x+w-1, yy+1, '│')
		setRaw(x, yy+2, '└')
		for i := 1; i < w-1; i++ {
			setRaw(x+i, yy+2, '─')
		}
		setRaw(x+w-1, yy+2, '┘')
	}

	// Draw edges between adjacent-rank nodes
	for _, e := range g.edges {
		fid, tid := e.from.id, e.to.id
		if fid == tid || rank[fid]+1 != rank[tid] {
			continue
		}
		fw := nodeBoxW(fid)
		fcx := nodeX[fid] + fw/2
		row1 := nodeY[fid] + 3 // first connector row

		tw := nodeBoxW(tid)
		tcx := nodeX[tid] + tw/2

		switch {
		case fcx == tcx:
			// Straight down: │ then ▼
			setBox(fcx, row1, '│')
			setRaw(tcx, row1+1, '▼')
		case fcx < tcx:
			// Go right: └────┐ then ▼
			setBox(fcx, row1, '└')
			for x := fcx + 1; x < tcx; x++ {
				setBox(x, row1, '─')
			}
			setBox(tcx, row1, '┐')
			setRaw(tcx, row1+1, '▼')
		default:
			// Go left: ┌────┘ then ▼
			setBox(tcx, row1, '┌')
			for x := tcx + 1; x < fcx; x++ {
				setBox(x, row1, '─')
			}
			setBox(fcx, row1, '┘')
			setRaw(tcx, row1+1, '▼')
		}
	}

	var sb strings.Builder
	for _, row := range grid {
		sb.WriteString(strings.TrimRight(string(row), " "))
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ── ASCII sequence diagram renderer ──────────────────────────────────────────

type seqMsg struct {
	from, to, label string
	dashed          bool
}

func parseSequenceDiagram(code string) (participants []string, msgs []seqMsg) {
	seen := make(map[string]bool)
	add := func(name string) {
		if name != "" && !seen[name] {
			participants = append(participants, name)
			seen[name] = true
		}
	}
	for _, line := range strings.Split(code, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		lower := strings.ToLower(t)
		if strings.HasPrefix(lower, "participant ") || strings.HasPrefix(lower, "actor ") {
			fields := strings.Fields(t)
			if len(fields) < 2 {
				continue
			}
			name := fields[1]
			for i, f := range fields {
				if strings.EqualFold(f, "as") && i+1 < len(fields) {
					name = strings.Join(fields[i+1:], " ")
					break
				}
			}
			add(name)
			continue
		}
		// Message arrows – check in order of decreasing length to avoid mis-matching
		for _, op := range []struct {
			s      string
			dashed bool
		}{
			{"-->>", true}, {"-->", true}, {"->>", false}, {"->", false},
			{"--x", true}, {"-x", false}, {"--)", true}, {"-)", false},
		} {
			idx := strings.Index(t, op.s)
			if idx < 0 {
				continue
			}
			from := strings.TrimSpace(t[:idx])
			rest := strings.TrimSpace(t[idx+len(op.s):])
			to, label := rest, ""
			if ci := strings.Index(rest, ":"); ci >= 0 {
				to = strings.TrimSpace(rest[:ci])
				label = strings.TrimSpace(rest[ci+1:])
			}
			if from == "" || to == "" {
				break
			}
			add(from)
			add(to)
			msgs = append(msgs, seqMsg{from: from, to: to, label: label, dashed: op.dashed})
			break
		}
	}
	return
}

func asciiSequenceDiagram(participants []string, msgs []seqMsg, maxW int) string {
	if len(participants) == 0 {
		return "(no participants)"
	}

	// Column width: participant name + 2 spaces padding, minimum 14, even
	colW := 14
	for _, p := range participants {
		if w := len(p) + 4; w > colW {
			colW = w
		}
	}
	if colW%2 != 0 {
		colW++
	}

	colIdx := make(map[string]int)
	for i, p := range participants {
		colIdx[p] = i
	}
	n := len(participants)
	totalW := n * colW
	if maxW > 0 && totalW > maxW {
		totalW = maxW
	}

	centerOf := func(i int) int { return i*colW + colW/2 }

	var sb strings.Builder

	// Participant header row
	for _, p := range participants {
		name := p
		if len(name) > colW-2 {
			name = name[:colW-2]
		}
		pad := (colW - len(name)) / 2
		sb.WriteString(strings.Repeat(" ", pad))
		sb.WriteString(name)
		sb.WriteString(strings.Repeat(" ", colW-pad-len(name)))
	}
	sb.WriteByte('\n')

	// Lifeline header
	lifeline := func() []rune {
		row := make([]rune, totalW)
		for i := range row {
			row[i] = ' '
		}
		for i := range participants {
			if cx := centerOf(i); cx < totalW {
				row[cx] = '│'
			}
		}
		return row
	}
	sb.WriteString(strings.TrimRight(string(lifeline()), " ") + "\n")

	lineChar := func(dashed bool) rune {
		if dashed {
			return '╌'
		}
		return '─'
	}

	for _, msg := range msgs {
		fi, ok1 := colIdx[msg.from]
		ti, ok2 := colIdx[msg.to]
		if !ok1 || !ok2 {
			continue
		}

		row := lifeline()
		fcx := centerOf(fi)
		tcx := centerOf(ti)

		if fi == ti {
			// Self-arrow
			lx := fcx + 1
			label := "↩"
			if msg.label != "" {
				label += " " + msg.label
			}
			for i, r := range []rune(label) {
				if lx+i < totalW {
					row[lx+i] = r
				}
			}
		} else {
			goRight := fi < ti
			lx, rx := fcx, tcx
			if !goRight {
				lx, rx = tcx, fcx
			}
			lc := lineChar(msg.dashed)
			for x := lx + 1; x < rx; x++ {
				if x < totalW {
					row[x] = lc
				}
			}
			if goRight {
				if rx < totalW {
					row[rx] = '►'
				}
			} else {
				if lx < totalW {
					row[lx] = '◄'
				}
			}
			// Place label centred on the arrow
			if msg.label != "" {
				label := " " + msg.label + " "
				lrunes := []rune(label)
				lw := len(lrunes)
				mid := lx + (rx-lx-lw)/2 + 1
				if mid < lx+1 {
					mid = lx + 1
				}
				for i, r := range lrunes {
					if mid+i > lx && mid+i < rx && mid+i < totalW {
						row[mid+i] = r
					}
				}
			}
		}

		sb.WriteString(strings.TrimRight(string(row), " ") + "\n")
		sb.WriteString(strings.TrimRight(string(lifeline()), " ") + "\n")
	}

	return strings.TrimRight(sb.String(), "\n")
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

	// Fixed iteration order so parsing is deterministic regardless of map randomisation.
	for _, pair := range [3][2]byte{{'[', ']'}, {'(', ')'}, {'{', '}'}} {
		open, close := pair[0], pair[1]
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

var mermaidTextReplacer = strings.NewReplacer("\"", "", "'", "", "|", " ", "`", "")

func cleanMermaidText(in string) string {
	in = strings.TrimSpace(in)
	in = mermaidTextReplacer.Replace(in)
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

func moveToTrash(path string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	trashPath := filepath.Join(homeDir, ".Trash")
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	baseName := filepath.Base(path)
	destPath := filepath.Join(trashPath, baseName)
	if info.IsDir() {
		for i := 1; ; i++ {
			testPath := filepath.Join(trashPath, fmt.Sprintf("%s %d", baseName, i))
			if _, err := os.Stat(testPath); os.IsNotExist(err) {
				destPath = testPath
				break
			}
		}
	} else {
		ext := filepath.Ext(baseName)
		stem := strings.TrimSuffix(baseName, ext)
		for i := 1; ; i++ {
			testName := fmt.Sprintf("%s %d%s", stem, i, ext)
			testPath := filepath.Join(trashPath, testName)
			if _, err := os.Stat(testPath); os.IsNotExist(err) {
				destPath = testPath
				break
			}
		}
	}
	return os.Rename(path, destPath)
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
