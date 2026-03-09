package main

import "testing"

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
