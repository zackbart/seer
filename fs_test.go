package main

import (
	"bytes"
	"testing"
	"time"
)

// ── isLikelyBinary ────────────────────────────────────────────────────────────

func TestIsLikelyBinary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", []byte{}, false},
		{"plain text", []byte("hello world\n"), false},
		{"go source", []byte("package main\n\nfunc main() {}\n"), false},
		{"null byte at start", []byte{0x00, 0x01, 0x02}, true},
		{"null byte in middle", []byte("hello\x00world"), true},
		{"binary data", []byte{0xff, 0xfe, 0x00, 0x00}, true},
		{"high bytes no null", []byte{0xc3, 0xa9, 0xc3, 0xa0}, false}, // UTF-8 é à
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLikelyBinary(tc.data); got != tc.want {
				t.Errorf("isLikelyBinary(%q) = %v, want %v", tc.data, got, tc.want)
			}
		})
	}
}

func TestIsLikelyBinaryLargeFile(t *testing.T) {
	// Null byte beyond 8192 bytes should NOT be detected
	buf := bytes.Repeat([]byte("a"), 8193)
	buf[8192] = 0x00
	if isLikelyBinary(buf) {
		t.Error("null byte beyond 8192 bytes should not trigger binary detection")
	}

	// Null byte at position 8191 (within scan window) SHOULD be detected
	buf2 := bytes.Repeat([]byte("a"), 8193)
	buf2[8191] = 0x00
	if !isLikelyBinary(buf2) {
		t.Error("null byte at position 8191 should trigger binary detection")
	}
}

// ── previewKey ────────────────────────────────────────────────────────────────

func TestPreviewKey(t *testing.T) {
	t0 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	k1 := previewKey("/foo/bar.txt", t0, 1024, 80, 40)
	k2 := previewKey("/foo/bar.txt", t0, 1024, 80, 40)
	if k1 != k2 {
		t.Error("same inputs must produce same key")
	}

	// Different path
	k3 := previewKey("/foo/baz.txt", t0, 1024, 80, 40)
	if k1 == k3 {
		t.Error("different path must produce different key")
	}

	// Different size
	k4 := previewKey("/foo/bar.txt", t0, 2048, 80, 40)
	if k1 == k4 {
		t.Error("different size must produce different key")
	}

	// Different width
	k5 := previewKey("/foo/bar.txt", t0, 1024, 100, 40)
	if k1 == k5 {
		t.Error("different width must produce different key")
	}

	// Different height
	k6 := previewKey("/foo/bar.txt", t0, 1024, 80, 50)
	if k1 == k6 {
		t.Error("different height must produce different key")
	}

	// Different modTime
	t1 := t0.Add(time.Second)
	k7 := previewKey("/foo/bar.txt", t1, 1024, 80, 40)
	if k1 == k7 {
		t.Error("different modTime must produce different key")
	}
}

func TestPreviewKeyFormat(t *testing.T) {
	t0 := time.Unix(1705320000, 500)
	key := previewKey("/a/b", t0, 42, 80, 24)
	if key == "" {
		t.Fatal("key must not be empty")
	}
	// Key must contain the path
	if key[:4] != "/a/b" {
		t.Errorf("key %q should start with path /a/b", key)
	}
}
