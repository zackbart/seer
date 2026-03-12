package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ── directory navigation ──────────────────────────────────────────────────────

// changeDir navigates to path, pushing it onto the history stack.
func (m *model) changeDir(path string) error {
	return m.changeDirInternal(path, true)
}

// changeDirNoHistory navigates to path without modifying history (used for
// back/forward history traversal).
func (m *model) changeDirNoHistory(path string) error {
	return m.changeDirInternal(path, false)
}

func (m *model) changeDirInternal(path string, pushHistory bool) error {
	entries, err := listDir(path, m.showHidden)
	if err != nil {
		return err
	}
	m.cwd = path
	m.allEntries = applySort(entries, m.sortBy)
	m.entries = m.applySearch(m.allEntries)
	m.selected = 0
	m.previewOffset = 0
	m.searchQuery = ""
	m.searching = false
	m.status = path
	m.multiSelected = make(map[string]bool)

	if pushHistory {
		// Truncate any forward history, then append.
		if m.historyPos < len(m.dirHistory)-1 {
			m.dirHistory = m.dirHistory[:m.historyPos+1]
		}
		// Avoid duplicate consecutive entries.
		if len(m.dirHistory) == 0 || m.dirHistory[len(m.dirHistory)-1] != path {
			m.dirHistory = append(m.dirHistory, path)
		}
		m.historyPos = len(m.dirHistory) - 1
	}
	return nil
}

// navigate sets the selected index, resets the preview scroll, and returns a
// requestPreview command.  It is the single canonical way to change selection.
func (m *model) navigate(idx int) tea.Cmd {
	m.selected = idx
	m.previewOffset = 0
	return m.requestPreview()
}

// reloadDir refreshes the current directory listing without changing cwd or history.
func (m *model) reloadDir() error {
	entries, err := listDir(m.cwd, m.showHidden)
	if err != nil {
		return err
	}

	// Preserve selection by name.
	var prevName string
	if m.selected < len(m.entries) {
		prevName = m.entries[m.selected].name
	}

	m.allEntries = applySort(entries, m.sortBy)
	m.entries = m.applySearch(m.allEntries)

	// Restore selection.
	m.selected = 0
	for i, e := range m.entries {
		if e.name == prevName {
			m.selected = i
			break
		}
	}
	if m.selected >= len(m.entries) {
		m.selected = max(0, len(m.entries)-1)
	}
	return nil
}

// ── search / filter ───────────────────────────────────────────────────────────

// applySearch filters entries by the current searchQuery using fuzzy matching.
// Returns all entries unchanged when the query is empty.
func (m model) applySearch(entries []entry) []entry {
	if m.searchQuery == "" {
		return entries
	}
	var out []entry
	for _, e := range entries {
		if fuzzyMatch(e.name, m.searchQuery) {
			out = append(out, e)
		}
	}
	return out
}

// fuzzyMatch returns true when all runes of query appear in name in order
// (case-insensitive).  This is the same algorithm used by fzf / telescope.
func fuzzyMatch(name, query string) bool {
	nameRunes := []rune(strings.ToLower(name))
	queryRunes := []rune(strings.ToLower(query))
	qi := 0
	for i := 0; i < len(nameRunes) && qi < len(queryRunes); i++ {
		if nameRunes[i] == queryRunes[qi] {
			qi++
		}
	}
	return qi == len(queryRunes)
}

// ── pane offset ───────────────────────────────────────────────────────────

// clampPaneOffset ensures the pane divider offset stays within valid bounds
// after a terminal resize.
func (m *model) clampPaneOffset() {
	minOffset := -(m.width/3 - 16)
	maxOffset := m.width / 4
	if m.paneOffset < minOffset {
		m.paneOffset = minOffset
	}
	if m.paneOffset > maxOffset {
		m.paneOffset = maxOffset
	}
}

// ── preview cache ─────────────────────────────────────────────────────────────

// cacheSet stores a preview result, evicting the oldest entry when the cache
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

// ── preview request ───────────────────────────────────────────────────────────

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
	width := max(40, rightW-2)
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

// ── preview viewport ──────────────────────────────────────────────────────────

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
	return max(1, bodyH-4)
}

// ── bookmarks ────────────────────────────────────────────────────────────────

// toggleBookmark adds or removes the current cwd from the bookmarks list.
// Returns the slot number (1-based) and whether it was added (true) or removed (false).
func (m *model) toggleBookmark() (slot int, added bool) {
	for i, bm := range m.bookmarks {
		if bm == m.cwd {
			// Remove it.
			m.bookmarks = append(m.bookmarks[:i], m.bookmarks[i+1:]...)
			return i + 1, false
		}
	}
	// Add to end (up to 9).
	if len(m.bookmarks) >= 9 {
		// Replace oldest (slot 1) by shifting.
		m.bookmarks = append(m.bookmarks[1:], m.cwd)
		return 9, true
	}
	m.bookmarks = append(m.bookmarks, m.cwd)
	return len(m.bookmarks), true
}

// ── multi-select helpers ──────────────────────────────────────────────────────

// selectedPaths returns the paths to operate on: multi-selected if any,
// otherwise the current cursor entry.
func (m model) selectedPaths() []string {
	if len(m.multiSelected) > 0 {
		paths := make([]string, 0, len(m.multiSelected))
		for p := range m.multiSelected {
			paths = append(paths, p)
		}
		return paths
	}
	if len(m.entries) > 0 && m.selected < len(m.entries) {
		return []string{m.entries[m.selected].path}
	}
	return nil
}
