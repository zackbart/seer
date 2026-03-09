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
