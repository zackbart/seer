package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
	width := max(40, rightW-2) // -2 for pane border; preview renders at innerW
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
	return max(1, bodyH-4)
}
