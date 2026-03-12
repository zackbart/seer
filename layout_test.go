package main

import (
	"strings"
	"testing"
)

// ── visibleWindow ─────────────────────────────────────────────────────────────

func TestVisibleWindow(t *testing.T) {
	tests := []struct {
		name                    string
		selected, total, height int
		wantStart, wantEnd      int
	}{
		{"all fit", 2, 5, 10, 0, 5},
		{"all fit exact", 0, 5, 5, 0, 5},
		{"selected at top", 0, 20, 5, 0, 5},
		{"selected at bottom", 19, 20, 5, 15, 20},
		{"selected in middle", 10, 20, 5, 8, 13},
		{"height 1 at start", 0, 10, 1, 0, 1},
		{"height 1 at end", 9, 10, 1, 9, 10},
		{"selected centered", 5, 10, 5, 3, 8},
		{"empty list", 0, 0, 5, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := visibleWindow(tc.selected, tc.total, tc.height)
			if start != tc.wantStart || end != tc.wantEnd {
				t.Errorf("visibleWindow(%d, %d, %d) = [%d, %d), want [%d, %d)",
					tc.selected, tc.total, tc.height, start, end, tc.wantStart, tc.wantEnd)
			}
			// Selected must always be within the window (when list is non-empty)
			if tc.total > 0 && (tc.selected < start || tc.selected >= end) {
				t.Errorf("selected %d not in window [%d, %d)", tc.selected, start, end)
			}
		})
	}
}

func TestVisibleWindowWindowSize(t *testing.T) {
	// Window should never exceed height
	for _, h := range []int{1, 3, 5, 10} {
		start, end := visibleWindow(0, 100, h)
		if end-start > h {
			t.Errorf("window size %d exceeds height %d", end-start, h)
		}
	}
}

// ── trimVisual ────────────────────────────────────────────────────────────────

func TestTrimVisual(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 0, ""},
		{"hello", 1, "…"},
		{"abcde", 3, "ab…"},
		{"", 5, ""},
	}
	for _, tc := range tests {
		t.Run(tc.s+"/"+strings.Repeat("x", tc.n), func(t *testing.T) {
			got := trimVisual(tc.s, tc.n)
			if got != tc.want {
				t.Errorf("trimVisual(%q, %d) = %q, want %q", tc.s, tc.n, got, tc.want)
			}
		})
	}
}

func TestTrimVisualNoTruncation(t *testing.T) {
	s := "short"
	got := trimVisual(s, 100)
	if got != s {
		t.Errorf("trimVisual should return original when within budget, got %q", got)
	}
}

// ── padRight ──────────────────────────────────────────────────────────────────

func TestPadRight(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"hi", 5, "hi   "},
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"", 3, "   "},
	}
	for _, tc := range tests {
		got := padRight(tc.s, tc.n)
		if got != tc.want {
			t.Errorf("padRight(%q, %d) = %q, want %q", tc.s, tc.n, got, tc.want)
		}
	}
}

// ── trimToWidth ───────────────────────────────────────────────────────────────

func TestTrimToWidth(t *testing.T) {
	tests := []struct {
		s     string
		width int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 0, ""},
		{"", 5, ""},
	}
	for _, tc := range tests {
		got := trimToWidth(tc.s, tc.width)
		if got != tc.want {
			t.Errorf("trimToWidth(%q, %d) = %q, want %q", tc.s, tc.width, got, tc.want)
		}
	}
}

// ── byteIndexForColumn / sliceByColumns ──────────────────────────────────────

func TestByteIndexForColumn(t *testing.T) {
	tests := []struct {
		s    string
		col  int
		want int
	}{
		{"hello", 0, 0},
		{"hello", 3, 3},
		{"hello", 5, 5},
		{"hello", 10, 5}, // clamped to string length
		{"", 3, 0},
	}
	for _, tc := range tests {
		got := byteIndexForColumn(tc.s, tc.col)
		if got != tc.want {
			t.Errorf("byteIndexForColumn(%q, %d) = %d, want %d", tc.s, tc.col, got, tc.want)
		}
	}
}

func TestSliceByColumns(t *testing.T) {
	tests := []struct {
		s          string
		start, end int
		want       string
	}{
		{"hello", 0, 5, "hello"},
		{"hello", 1, 4, "ell"},
		{"hello", 0, 3, "hel"},
		{"hello", 3, 5, "lo"},
		{"hello", 2, 2, ""},
		{"hello", 5, 3, ""}, // end < start → empty
		{"", 0, 3, ""},
	}
	for _, tc := range tests {
		got := sliceByColumns(tc.s, tc.start, tc.end)
		if got != tc.want {
			t.Errorf("sliceByColumns(%q, %d, %d) = %q, want %q", tc.s, tc.start, tc.end, got, tc.want)
		}
	}
}
