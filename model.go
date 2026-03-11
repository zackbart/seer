package main

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

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
				m.preview = ""
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
		case "h", "left":
			if m.searching {
				break
			}
			parent := filepath.Dir(m.cwd)
			if parent != m.cwd {
				if err := m.changeDir(parent); err != nil {
					m.status = err.Error()
				}
				return m, m.requestPreview()
			}
		case "backspace":
			if m.searching {
				if len(m.searchQuery) > 0 {
					runes := []rune(m.searchQuery)
					m.searchQuery = string(runes[:len(runes)-1])
					m.entries = m.applySearch(m.allEntries)
					m.selected = 0
					return m, m.requestPreview()
				}
				break
			}
			fallthrough
		case "delete":
			if len(m.entries) > 0 && m.selected < len(m.entries) {
				m.confirmingDelete = true
				m.deleteTarget = m.entries[m.selected].path
				m.status = "confirm move to trash"
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
		case "p":
			if len(m.entries) > 0 && m.selected < len(m.entries) {
				p := m.entries[m.selected].path
				if err := copyToClipboard(p); err != nil {
					m.status = "copy failed: " + err.Error()
				} else {
					m.status = fmt.Sprintf("copied path: %s", p)
				}
			}
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
