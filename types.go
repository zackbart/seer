package main

import "time"

const (
	maxPreviewBytes = 256 * 1024
	maxDirPreview   = 40
	previewCacheMax = 50
)

// fileCategory is a broad category for an entry, used to pick colour/icon.
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

// ── sort ─────────────────────────────────────────────────────────────────────

type sortMode int

const (
	sortNameAsc sortMode = iota
	sortNameDesc
	sortSizeDesc
	sortSizeAsc
	sortModifiedDesc
	sortModifiedAsc
	sortModeCount // sentinel
)

func (s sortMode) Label() string {
	switch s {
	case sortNameAsc:
		return "name ↑"
	case sortNameDesc:
		return "name ↓"
	case sortSizeDesc:
		return "size ↓"
	case sortSizeAsc:
		return "size ↑"
	case sortModifiedDesc:
		return "date ↓"
	case sortModifiedAsc:
		return "date ↑"
	}
	return "name ↑"
}

// ── input mode ───────────────────────────────────────────────────────────────

type inputModeType int

const (
	inputNone inputModeType = iota
	inputRename
	inputNewFile
	inputNewDir
)

func (m inputModeType) Title() string {
	switch m {
	case inputRename:
		return "Rename"
	case inputNewFile:
		return "New File"
	case inputNewDir:
		return "New Directory"
	}
	return ""
}

// ── yank ─────────────────────────────────────────────────────────────────────

type yankMode int

const (
	yankNone yankMode = iota
	yankCopy
	yankCut
)

// ── entry ─────────────────────────────────────────────────────────────────────

type entry struct {
	name          string
	path          string
	isDir         bool
	size          int64
	modTime       time.Time
	isSymlink     bool
	symlinkTarget string
}

// ── messages ──────────────────────────────────────────────────────────────────

type previewLoadedMsg struct {
	requestID int
	cacheKey  string
	content   string
	err       error
}

type gitStatusMsg struct {
	cwd    string
	status map[string]string // filename → git status code
}

type editorDoneMsg struct {
	err error
}

type selectionPoint struct {
	x int
	y int
}

// ── model ─────────────────────────────────────────────────────────────────────

type model struct {
	cwd        string
	allEntries []entry // full unfiltered listing
	entries    []entry // visible (filtered + sorted) listing
	selected   int
	showHidden bool
	preview    string
	status     string
	width      int
	height     int

	previewOffset int
	loading       bool
	requestID     int
	cache         map[string]string
	cacheOrder    []string // LRU insertion order

	// Search / filter
	searching   bool
	searchQuery string

	// Delete confirmation dialog
	confirmingDelete bool
	deleteTarget     string

	// Preview mouse selection for auto-copy on release.
	previewSelecting bool
	previewSelStart  selectionPoint
	previewSelEnd    selectionPoint

	// Text input mode (rename, new file, new dir)
	inputMode  inputModeType
	inputValue string

	// Yank / paste
	yankPaths []string
	yankOp    yankMode

	// Multi-select: set of entry paths
	multiSelected map[string]bool

	// Sort
	sortBy sortMode

	// Directory history (browser-style back/forward)
	dirHistory []string
	historyPos int

	// Bookmarks (list of directory paths, 1-indexed)
	bookmarks []string

	// Pane divider offset (+= wider left pane, -= narrower)
	paneOffset int

	// Git status for current directory
	gitStatus  map[string]string // filename → status code ("M","A","D","??",…)
	gitLoadCwd string            // cwd for which gitStatus was loaded
}
