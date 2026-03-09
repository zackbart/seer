package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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
