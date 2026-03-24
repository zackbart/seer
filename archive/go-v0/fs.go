package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ── directory listing ─────────────────────────────────────────────────────────

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

		// item.Info() uses Lstat — detects symlinks.
		linfo, err := item.Info()
		if err != nil {
			continue
		}

		isSymlink := linfo.Mode()&os.ModeSymlink != 0
		var symlinkTarget string
		isDir := linfo.IsDir()
		sz := linfo.Size()
		modTime := linfo.ModTime()

		if isSymlink {
			if target, err := os.Readlink(full); err == nil {
				symlinkTarget = target
			}
			// Follow the symlink to get real size/type info.
			if sinfo, err := os.Stat(full); err == nil {
				isDir = sinfo.IsDir()
				sz = sinfo.Size()
				modTime = sinfo.ModTime()
			}
		}

		entries = append(entries, entry{
			name:          name,
			path:          full,
			isDir:         isDir,
			size:          sz,
			modTime:       modTime,
			isSymlink:     isSymlink,
			symlinkTarget: symlinkTarget,
		})
	}

	// Default sort: dirs first, then alphabetical.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})

	return entries, nil
}

// applySort re-sorts entries by the requested mode.  Directories always stay
// above files regardless of mode.
func applySort(entries []entry, mode sortMode) []entry {
	if mode == sortNameAsc {
		return entries // listDir already returns name-asc
	}
	result := make([]entry, len(entries))
	copy(result, entries)
	sort.SliceStable(result, func(i, j int) bool {
		// Dirs always first.
		if result[i].isDir != result[j].isDir {
			return result[i].isDir
		}
		switch mode {
		case sortNameDesc:
			return strings.ToLower(result[i].name) > strings.ToLower(result[j].name)
		case sortSizeDesc:
			if result[i].size != result[j].size {
				return result[i].size > result[j].size
			}
		case sortSizeAsc:
			if result[i].size != result[j].size {
				return result[i].size < result[j].size
			}
		case sortModifiedDesc:
			if !result[i].modTime.Equal(result[j].modTime) {
				return result[i].modTime.After(result[j].modTime)
			}
		case sortModifiedAsc:
			if !result[i].modTime.Equal(result[j].modTime) {
				return result[i].modTime.Before(result[j].modTime)
			}
		}
		return strings.ToLower(result[i].name) < strings.ToLower(result[j].name)
	})
	return result
}

// ── trash ─────────────────────────────────────────────────────────────────────

func trashDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var dir string
	if runtime.GOOS == "linux" {
		dir = filepath.Join(homeDir, ".local", "share", "Trash", "files")
	} else {
		dir = filepath.Join(homeDir, ".Trash")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func moveToTrash(path string) error {
	trashPath, err := trashDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	baseName := filepath.Base(path)
	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)

	// Try the bare name first, then add a numeric suffix on collision.
	destPath := filepath.Join(trashPath, baseName)
	for i := 1; ; i++ {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		destPath = filepath.Join(trashPath, fmt.Sprintf("%s %d%s", stem, i, ext))
	}
	return os.Rename(path, destPath)
}

// ── copy / paste ──────────────────────────────────────────────────────────────

// uniqueDstPath returns a non-conflicting destination path inside dir for a
// file/dir named name.  Appends " (1)", " (2)", … on collision.
// Caps attempts at 10000 to avoid infinite loops.
func uniqueDstPath(dir, name string) string {
	dst := filepath.Join(dir, name)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return dst
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i <= 10000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	// Exhausted — use a timestamp suffix as last resort.
	return filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, time.Now().UnixNano(), ext))
}

// copyEntry copies src to dst recursively.
func copyEntry(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, info.Mode())
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	items, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := copyEntry(
			filepath.Join(src, item.Name()),
			filepath.Join(dst, item.Name()),
		); err != nil {
			return err
		}
	}
	return nil
}

// pasteEntries pastes paths into dstDir according to op (copy or cut).
// Returns the number of entries pasted and any error.
func pasteEntries(paths []string, dstDir string, op yankMode) (int, error) {
	count := 0
	for _, src := range paths {
		name := filepath.Base(src)
		dst := uniqueDstPath(dstDir, name)

		if op == yankCut {
			// Try a fast rename first; fall back to copy+delete if cross-device.
			if err := os.Rename(src, dst); err != nil {
				if err2 := copyEntry(src, dst); err2 != nil {
					// Copy failed — clean up partial destination.
					os.RemoveAll(dst)
					return count, err2
				}
				// Copy succeeded — now safe to remove source.
				if err2 := os.RemoveAll(src); err2 != nil {
					return count, err2
				}
			}
		} else {
			if err := copyEntry(src, dst); err != nil {
				return count, err
			}
		}
		count++
	}
	return count, nil
}

// ── git status ────────────────────────────────────────────────────────────────

// loadGitStatus returns a tea.Cmd that runs `git status --porcelain` in cwd
// and sends a gitStatusMsg with the results.
func loadGitStatus(cwd string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("git", "-C", cwd, "status", "--porcelain").Output()
		if err != nil {
			// Not a git repo or git not available — send empty map.
			return gitStatusMsg{cwd: cwd, status: nil}
		}
		status := make(map[string]string)
		for _, line := range strings.Split(string(out), "\n") {
			if len(line) < 4 {
				continue
			}
			xy := strings.TrimSpace(line[:2])
			file := strings.TrimSpace(line[3:])
			// Renames: "old -> new"
			if idx := strings.Index(file, " -> "); idx >= 0 {
				file = file[idx+4:]
			}
			file = strings.Trim(file, "\"")
			// Only show status for files directly in this directory,
			// not in subdirectories (which would cause basename collisions).
			dir := filepath.Dir(file)
			if dir != "." && dir != "" {
				continue
			}
			base := filepath.Base(file)
			if xy != "" && base != "" {
				status[base] = xy
			}
		}
		return gitStatusMsg{cwd: cwd, status: status}
	}
}

// ── preview cache key ─────────────────────────────────────────────────────────

func previewKey(path string, modTime time.Time, size int64, width, height int) string {
	return fmt.Sprintf("%s|%d|%d|%d|%d", path, modTime.UnixNano(), size, width, height)
}

// ── binary detection ──────────────────────────────────────────────────────────

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
