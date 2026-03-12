package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// ── initialisation ────────────────────────────────────────────────────────────

func initialModel(startDir string) model {
	cwd := startDir
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "."
		}
	}

	entries, listErr := listDir(cwd, false)
	status := "ready"
	if listErr != nil {
		status = listErr.Error()
	}

	return model{
		cwd:           cwd,
		allEntries:    entries,
		entries:       entries,
		selected:      0,
		preview:       "",
		status:        status,
		cache:         make(map[string]string),
		showHidden:    false,
		multiSelected: make(map[string]bool),
		dirHistory:    []string{cwd},
		historyPos:    0,
		bookmarks:     []string{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.requestPreview(), loadGitStatus(m.cwd))
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampPreviewOffset()
		return m, m.requestPreview()

	case gitStatusMsg:
		if msg.cwd == m.cwd {
			m.gitStatus = msg.status
			m.gitLoadCwd = msg.cwd
		}
		return m, nil

	case editorDoneMsg:
		if msg.err != nil {
			m.status = "editor error: " + msg.err.Error()
		} else {
			m.status = "returned from editor"
		}
		// Reload in case the file was modified.
		if err := m.reloadDir(); err != nil {
			m.status = err.Error()
		}
		return m, m.requestPreview()

	case tea.KeyMsg:
		// Priority 1: text input mode (rename / new file / new dir).
		if m.inputMode != inputNone {
			return m.handleInputKey(msg)
		}
		// Priority 2: delete confirmation.
		if m.confirmingDelete {
			return m.handleDeleteKey(msg)
		}
		// Normal / search mode.
		return m.handleNormalKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

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

// ── input mode ────────────────────────────────────────────────────────────────

func (m model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.inputMode = inputNone
		m.inputValue = ""
		m.status = "cancelled"
		return m, nil

	case "enter":
		return m.confirmInput()

	case "backspace":
		if len(m.inputValue) > 0 {
			runes := []rune(m.inputValue)
			m.inputValue = string(runes[:len(runes)-1])
		}
		return m, nil
	}

	// Printable character → append to input.
	if len(msg.Runes) == 1 {
		m.inputValue += string(msg.Runes)
	}
	return m, nil
}

func (m model) confirmInput() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.inputValue)
	if name == "" {
		m.inputMode = inputNone
		m.inputValue = ""
		m.status = "cancelled"
		return m, nil
	}

	mode := m.inputMode
	m.inputMode = inputNone
	m.inputValue = ""

	switch mode {
	case inputRename:
		if m.selected < len(m.entries) {
			old := m.entries[m.selected].path
			newPath := filepath.Join(m.cwd, name)
			if err := os.Rename(old, newPath); err != nil {
				m.status = "rename failed: " + err.Error()
			} else {
				m.status = "renamed → " + name
				if err := m.reloadDir(); err != nil {
					m.status = err.Error()
				}
				// Try to re-select the renamed file.
				for i, e := range m.entries {
					if e.name == name {
						m.selected = i
						break
					}
				}
			}
		}

	case inputNewFile:
		newPath := filepath.Join(m.cwd, name)
		f, err := os.Create(newPath)
		if err != nil {
			m.status = "create failed: " + err.Error()
		} else {
			f.Close()
			m.status = "created " + name
			if err := m.reloadDir(); err != nil {
				m.status = err.Error()
			}
			for i, e := range m.entries {
				if e.name == name {
					m.selected = i
					break
				}
			}
		}

	case inputNewDir:
		newPath := filepath.Join(m.cwd, name)
		if err := os.MkdirAll(newPath, 0755); err != nil {
			m.status = "mkdir failed: " + err.Error()
		} else {
			m.status = "created " + name + "/"
			if err := m.reloadDir(); err != nil {
				m.status = err.Error()
			}
			for i, e := range m.entries {
				if e.name == name {
					m.selected = i
					break
				}
			}
		}
	}

	return m, m.requestPreview()
}

// ── delete confirmation ───────────────────────────────────────────────────────

func (m model) handleDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "y" || key == "Y" || key == "enter" {
		targets := m.deleteTargets()
		var lastErr error
		deleted := 0
		for _, t := range targets {
			if err := moveToTrash(t); err != nil {
				lastErr = err
			} else {
				deleted++
			}
		}
		if lastErr != nil {
			m.status = fmt.Sprintf("trash failed: %s", lastErr.Error())
		} else {
			m.status = fmt.Sprintf("moved %d to trash", deleted)
		}
		m.confirmingDelete = false
		m.deleteTarget = ""
		m.multiSelected = make(map[string]bool)
		m.preview = ""
		if err := m.reloadDir(); err != nil {
			m.status = err.Error()
		}
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

// deleteTargets returns the paths to be deleted: multi-selected if any,
// otherwise just deleteTarget.
func (m model) deleteTargets() []string {
	if len(m.multiSelected) > 0 {
		paths := make([]string, 0, len(m.multiSelected))
		for p := range m.multiSelected {
			paths = append(paths, p)
		}
		return paths
	}
	if m.deleteTarget != "" {
		return []string{m.deleteTarget}
	}
	return nil
}

// ── normal key handling ───────────────────────────────────────────────────────

func (m model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In search mode, printable characters extend the query.
	if m.searching && len(msg.Runes) == 1 {
		m.searchQuery += string(msg.Runes)
		m.entries = m.applySearch(m.allEntries)
		m.selected = 0
		return m, m.requestPreview()
	}

	switch msg.String() {

	// ── quit ──────────────────────────────────────────────────────────────
	case "q", "ctrl+c":
		return m, tea.Quit

	// ── navigation ────────────────────────────────────────────────────────
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
				return m, nil
			}
			return m, tea.Batch(m.requestPreview(), loadGitStatus(m.cwd))
		}
		return m, m.requestPreview()

	case "h", "left":
		if m.searching {
			break
		}
		parent := filepath.Dir(m.cwd)
		if parent != m.cwd {
			childName := filepath.Base(m.cwd)
			if err := m.changeDir(parent); err != nil {
				m.status = err.Error()
				return m, nil
			}
			// Re-select the directory we came from.
			for i, e := range m.entries {
				if e.name == childName {
					m.selected = i
					break
				}
			}
			return m, tea.Batch(m.requestPreview(), loadGitStatus(m.cwd))
		}

	// ── history ───────────────────────────────────────────────────────────
	case "alt+left":
		if m.historyPos > 0 {
			m.historyPos--
			dest := m.dirHistory[m.historyPos]
			if err := m.changeDirNoHistory(dest); err != nil {
				m.historyPos++ // revert
				m.status = err.Error()
				return m, nil
			}
			return m, tea.Batch(m.requestPreview(), loadGitStatus(m.cwd))
		}
	case "alt+right":
		if m.historyPos < len(m.dirHistory)-1 {
			m.historyPos++
			dest := m.dirHistory[m.historyPos]
			if err := m.changeDirNoHistory(dest); err != nil {
				m.historyPos-- // revert
				m.status = err.Error()
				return m, nil
			}
			return m, tea.Batch(m.requestPreview(), loadGitStatus(m.cwd))
		}

	// ── preview scroll ────────────────────────────────────────────────────
	case "ctrl+d", "pagedown":
		m.previewOffset += previewPageSize(m.height)
		m.clampPreviewOffset()
	case "ctrl+u", "pageup":
		m.previewOffset -= previewPageSize(m.height)
		m.clampPreviewOffset()

	// ── search ────────────────────────────────────────────────────────────
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

	// ── hidden files ──────────────────────────────────────────────────────
	case ".":
		var prevName string
		if m.selected < len(m.entries) {
			prevName = m.entries[m.selected].name
		}
		m.showHidden = !m.showHidden
		entries, err := listDir(m.cwd, m.showHidden)
		if err != nil {
			m.status = err.Error()
		} else {
			m.allEntries = applySort(entries, m.sortBy)
			m.entries = m.applySearch(m.allEntries)
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

	// ── sort ──────────────────────────────────────────────────────────────
	case "s":
		m.sortBy = (m.sortBy + 1) % sortModeCount
		m.allEntries = applySort(m.allEntries, m.sortBy)
		m.entries = m.applySearch(m.allEntries)
		if m.selected >= len(m.entries) {
			m.selected = max(0, len(m.entries)-1)
		}
		m.status = "sort: " + m.sortBy.Label()
		return m, m.requestPreview()

	// ── pane width ────────────────────────────────────────────────────────
	case "<":
		m.paneOffset -= 2
		if m.paneOffset < -(m.width/3 - 16) {
			m.paneOffset = -(m.width/3 - 16)
		}
		return m, m.requestPreview()
	case ">":
		m.paneOffset += 2
		if m.paneOffset > m.width/4 {
			m.paneOffset = m.width / 4
		}
		return m, m.requestPreview()

	// ── multi-select ──────────────────────────────────────────────────────
	case " ":
		if len(m.entries) > 0 && m.selected < len(m.entries) {
			p := m.entries[m.selected].path
			if m.multiSelected[p] {
				delete(m.multiSelected, p)
			} else {
				m.multiSelected[p] = true
			}
			// Advance selection.
			if m.selected < len(m.entries)-1 {
				return m, m.navigate(m.selected + 1)
			}
		}

	// ── yank / paste ──────────────────────────────────────────────────────
	case "y":
		paths := m.selectedPaths()
		if len(paths) > 0 {
			m.yankPaths = paths
			m.yankOp = yankCopy
			m.multiSelected = make(map[string]bool)
			if len(paths) == 1 {
				m.status = fmt.Sprintf("yanked: %s", filepath.Base(paths[0]))
			} else {
				m.status = fmt.Sprintf("yanked %d files", len(paths))
			}
		}

	case "x":
		paths := m.selectedPaths()
		if len(paths) > 0 {
			m.yankPaths = paths
			m.yankOp = yankCut
			m.multiSelected = make(map[string]bool)
			if len(paths) == 1 {
				m.status = fmt.Sprintf("cut: %s", filepath.Base(paths[0]))
			} else {
				m.status = fmt.Sprintf("cut %d files", len(paths))
			}
		}

	case "P":
		if len(m.yankPaths) == 0 || m.yankOp == yankNone {
			m.status = "nothing to paste (yank with y or x)"
			return m, nil
		}
		n, err := pasteEntries(m.yankPaths, m.cwd, m.yankOp)
		if err != nil {
			m.status = "paste failed: " + err.Error()
		} else {
			if m.yankOp == yankCut {
				m.yankPaths = nil
				m.yankOp = yankNone
			}
			m.status = fmt.Sprintf("pasted %d item(s)", n)
		}
		if err := m.reloadDir(); err != nil {
			m.status = err.Error()
		}
		return m, m.requestPreview()

	// ── rename ────────────────────────────────────────────────────────────
	case "r":
		if len(m.entries) > 0 && m.selected < len(m.entries) {
			m.inputMode = inputRename
			m.inputValue = m.entries[m.selected].name
			return m, nil
		}

	// ── reload ────────────────────────────────────────────────────────────
	case "R":
		if err := m.reloadDir(); err != nil {
			m.status = err.Error()
		} else {
			m.status = "reloaded"
		}
		return m, tea.Batch(m.requestPreview(), loadGitStatus(m.cwd))

	// ── new file / directory ──────────────────────────────────────────────
	case "n":
		m.inputMode = inputNewFile
		m.inputValue = ""
		return m, nil

	case "N":
		m.inputMode = inputNewDir
		m.inputValue = ""
		return m, nil

	// ── open in editor ────────────────────────────────────────────────────
	case "e":
		if len(m.entries) > 0 && m.selected < len(m.entries) {
			e := m.entries[m.selected]
			if e.isDir {
				m.status = "can't edit a directory"
				return m, nil
			}
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				m.status = "$EDITOR not set"
				return m, nil
			}
			cmd := exec.Command(editor, e.path)
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return editorDoneMsg{err: err}
			})
		}

	// ── copy path ─────────────────────────────────────────────────────────
	case "p":
		if len(m.entries) > 0 && m.selected < len(m.entries) {
			p := m.entries[m.selected].path
			if err := copyToClipboard(p); err != nil {
				m.status = "copy failed: " + err.Error()
			} else {
				m.status = fmt.Sprintf("copied path: %s", p)
			}
		}

	// ── bookmarks ─────────────────────────────────────────────────────────
	case "b":
		slot, added := m.toggleBookmark()
		if added {
			m.status = fmt.Sprintf("bookmarked at #%d", slot)
		} else {
			m.status = fmt.Sprintf("bookmark #%d removed", slot)
		}

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		n := int(msg.Runes[0] - '0')
		if n >= 1 && n <= len(m.bookmarks) {
			dest := m.bookmarks[n-1]
			if err := m.changeDir(dest); err != nil {
				m.status = err.Error()
				return m, nil
			}
			return m, tea.Batch(m.requestPreview(), loadGitStatus(m.cwd))
		} else {
			m.status = fmt.Sprintf("no bookmark #%d", n)
		}
	}

	return m, nil
}

// ── mouse handling ────────────────────────────────────────────────────────────

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	event := tea.MouseEvent(msg)
	inPreviewPane := m.isInPreviewPane(event.X, event.Y)
	inPreviewBody := m.isInPreviewBody(event.X, event.Y)
	inFileList := m.isInFileList(event.X, event.Y)

	if event.IsWheel() {
		scroll := previewPageSize(m.height) / 3
		if scroll < 1 {
			scroll = 1
		}
		if inPreviewPane {
			switch event.Button {
			case tea.MouseButtonWheelDown:
				m.previewOffset += scroll
				m.clampPreviewOffset()
			case tea.MouseButtonWheelUp:
				m.previewOffset -= scroll
				m.clampPreviewOffset()
			}
		} else if inFileList {
			switch event.Button {
			case tea.MouseButtonWheelDown:
				if m.selected < len(m.entries)-1 {
					return m, m.navigate(m.selected + 1)
				}
			case tea.MouseButtonWheelUp:
				if m.selected > 0 {
					return m, m.navigate(m.selected - 1)
				}
			}
		}
		return m, nil
	}

	// Click in file list → select / open.
	if event.Action == tea.MouseActionPress && event.Button == tea.MouseButtonLeft && inFileList {
		idx, ok := m.fileListClickIndex(event.X, event.Y)
		if ok {
			if idx == m.selected {
				// Double-click effect: open/enter.
				if m.selected < len(m.entries) {
					picked := m.entries[m.selected]
					if picked.isDir {
						if err := m.changeDir(picked.path); err != nil {
							m.status = err.Error()
							return m, nil
						}
						return m, tea.Batch(m.requestPreview(), loadGitStatus(m.cwd))
					}
				}
			}
			return m, m.navigate(idx)
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

	return m, nil
}
