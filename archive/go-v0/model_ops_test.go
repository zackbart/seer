package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── fuzzyMatch ────────────────────────────────────────────────────────────────

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name, query string
		want        bool
	}{
		{"main.go", "mg", true},
		{"main.go", "main", true},
		{"main.go", "mango", true}, // m-a-n from "main", g-o from ".go"
		{"README.md", "rm", true},
		{"README.md", "rmd", true},
		{"README.md", "xyz", false},
		{"", "a", false},
		{"abc", "", true},
		// Multi-byte UTF-8: accented characters.
		{"café.txt", "caf", true},
		{"café.txt", "café", true},
		{"café.txt", "ct", true},
		{"日本語.go", "日語", true},
		{"日本語.go", "語本", false}, // wrong order
	}
	for _, tc := range tests {
		got := fuzzyMatch(tc.name, tc.query)
		if got != tc.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tc.name, tc.query, got, tc.want)
		}
	}
}

// ── applySearch ───────────────────────────────────────────────────────────────

func TestApplySearchEmpty(t *testing.T) {
	m := model{searchQuery: ""}
	entries := []entry{{name: "foo"}, {name: "bar"}}
	got := m.applySearch(entries)
	if len(got) != len(entries) {
		t.Errorf("empty query should return all entries, got %d want %d", len(got), len(entries))
	}
}

func TestApplySearchMatch(t *testing.T) {
	m := model{searchQuery: "foo"}
	entries := []entry{
		{name: "foo.go"},
		{name: "bar.go"},
		{name: "foobar.txt"},
		{name: "baz"},
	}
	got := m.applySearch(entries)
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d: %v", len(got), got)
	}
	if got[0].name != "foo.go" {
		t.Errorf("expected foo.go, got %q", got[0].name)
	}
	if got[1].name != "foobar.txt" {
		t.Errorf("expected foobar.txt, got %q", got[1].name)
	}
}

func TestApplySearchCaseInsensitive(t *testing.T) {
	m := model{searchQuery: "FOO"}
	entries := []entry{{name: "foo.go"}, {name: "bar.go"}}
	got := m.applySearch(entries)
	if len(got) != 1 || got[0].name != "foo.go" {
		t.Errorf("search should be case-insensitive, got %v", got)
	}
}

func TestApplySearchNoMatch(t *testing.T) {
	m := model{searchQuery: "xyz"}
	entries := []entry{{name: "foo"}, {name: "bar"}}
	got := m.applySearch(entries)
	if len(got) != 0 {
		t.Errorf("expected no matches, got %v", got)
	}
}

func TestApplySearchEmptyList(t *testing.T) {
	m := model{searchQuery: "foo"}
	got := m.applySearch(nil)
	if got != nil && len(got) != 0 {
		t.Errorf("search on nil list should return nil/empty, got %v", got)
	}
}

// ── applySort ─────────────────────────────────────────────────────────────────

func TestApplySort(t *testing.T) {
	now := time.Now()
	entries := []entry{
		{name: "beta", isDir: false, size: 200, modTime: now.Add(-2 * time.Hour)},
		{name: "alpha", isDir: false, size: 100, modTime: now.Add(-1 * time.Hour)},
		{name: "gamma", isDir: true, size: 0, modTime: now},
		{name: "delta", isDir: true, size: 0, modTime: now.Add(-3 * time.Hour)},
	}

	// Name ascending — returns entries unchanged (listDir already sorted).
	sorted := applySort(entries, sortNameAsc)
	if len(sorted) != 4 || sorted[0].name != "beta" {
		t.Errorf("sortNameAsc: expected original order, got %q first", sorted[0].name)
	}

	// Name descending — dirs first (reverse alpha), then files (reverse alpha).
	sorted = applySort(entries, sortNameDesc)
	if !sorted[0].isDir || !sorted[1].isDir {
		t.Errorf("sortNameDesc: dirs should be first")
	}

	// Size descending — largest file first among files.
	sorted = applySort(entries, sortSizeDesc)
	for _, e := range sorted {
		if !e.isDir {
			if e.name != "beta" {
				t.Errorf("sortSizeDesc: expected 'beta' (200) first among files, got %q", e.name)
			}
			break
		}
	}

	// Size ascending — smallest file first among files.
	sorted = applySort(entries, sortSizeAsc)
	for _, e := range sorted {
		if !e.isDir {
			if e.name != "alpha" {
				t.Errorf("sortSizeAsc: expected 'alpha' (100) first among files, got %q", e.name)
			}
			break
		}
	}

	// Modified descending — most recent dir first.
	sorted = applySort(entries, sortModifiedDesc)
	if sorted[0].name != "gamma" {
		t.Errorf("sortModifiedDesc: expected 'gamma' first, got %q", sorted[0].name)
	}
}

// ── uniqueDstPath ─────────────────────────────────────────────────────────────

func TestUniqueDstPath(t *testing.T) {
	dir := t.TempDir()

	// No collision — returns bare path.
	got := uniqueDstPath(dir, "newfile.txt")
	want := filepath.Join(dir, "newfile.txt")
	if got != want {
		t.Errorf("no collision: got %q, want %q", got, want)
	}

	// Create the file, should return " (1)".
	os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("x"), 0644)
	got = uniqueDstPath(dir, "newfile.txt")
	want = filepath.Join(dir, "newfile (1).txt")
	if got != want {
		t.Errorf("first collision: got %q, want %q", got, want)
	}

	// Create (1) too, should return " (2)".
	os.WriteFile(filepath.Join(dir, "newfile (1).txt"), []byte("x"), 0644)
	got = uniqueDstPath(dir, "newfile.txt")
	want = filepath.Join(dir, "newfile (2).txt")
	if got != want {
		t.Errorf("second collision: got %q, want %q", got, want)
	}

	// No-extension file.
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte("x"), 0644)
	got = uniqueDstPath(dir, "Makefile")
	want = filepath.Join(dir, "Makefile (1)")
	if got != want {
		t.Errorf("no ext: got %q, want %q", got, want)
	}
}

// ── cacheSet ──────────────────────────────────────────────────────────────────

func TestCacheSet(t *testing.T) {
	m := model{
		cache:      make(map[string]string),
		cacheOrder: nil,
	}

	m.cacheSet("key1", "value1")
	if v, ok := m.cache["key1"]; !ok || v != "value1" {
		t.Errorf("cacheSet did not store value, got %q", v)
	}
	if len(m.cacheOrder) != 1 || m.cacheOrder[0] != "key1" {
		t.Errorf("cacheOrder should contain key1, got %v", m.cacheOrder)
	}
}

func TestCacheSetUpdate(t *testing.T) {
	m := model{
		cache:      make(map[string]string),
		cacheOrder: nil,
	}
	m.cacheSet("key1", "v1")
	m.cacheSet("key1", "v2") // update existing key
	if m.cache["key1"] != "v2" {
		t.Errorf("expected updated value v2, got %q", m.cache["key1"])
	}
	if len(m.cacheOrder) != 1 {
		t.Errorf("updating existing key should not add duplicate to order, got %v", m.cacheOrder)
	}
}

func TestCacheSetEviction(t *testing.T) {
	m := model{
		cache:      make(map[string]string),
		cacheOrder: nil,
	}
	// Fill cache beyond capacity
	for i := 0; i <= previewCacheMax; i++ {
		key := string(rune('a' + i))
		m.cacheSet(key, "val")
	}
	if len(m.cache) > previewCacheMax {
		t.Errorf("cache size %d exceeds max %d", len(m.cache), previewCacheMax)
	}
	if len(m.cacheOrder) > previewCacheMax {
		t.Errorf("cacheOrder length %d exceeds max %d", len(m.cacheOrder), previewCacheMax)
	}
	// The first key "a" should have been evicted
	if _, ok := m.cache["a"]; ok {
		t.Error("first inserted key 'a' should have been evicted")
	}
}
