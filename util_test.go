package main

import "testing"

func TestMin(t *testing.T) {
	tests := []struct{ a, b, want int }{
		{1, 2, 1},
		{2, 1, 1},
		{0, 0, 0},
		{-5, 3, -5},
		{3, -5, -5},
	}
	for _, tc := range tests {
		if got := min(tc.a, tc.b); got != tc.want {
			t.Errorf("min(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct{ a, b, want int }{
		{1, 2, 2},
		{2, 1, 2},
		{0, 0, 0},
		{-5, 3, 3},
		{3, -5, 3},
	}
	for _, tc := range tests {
		if got := max(tc.a, tc.b); got != tc.want {
			t.Errorf("max(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{int64(1.5 * 1024 * 1024 * 1024), "1.5 GB"},
	}
	for _, tc := range tests {
		if got := humanSize(tc.n); got != tc.want {
			t.Errorf("humanSize(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestPreviewPageSize(t *testing.T) {
	tests := []struct{ h, want int }{
		{0, 3},  // max(3, 0/3)
		{6, 3},  // max(3, 6/3) = max(3, 2)
		{9, 3},  // max(3, 9/3) = max(3, 3)
		{12, 4}, // max(3, 12/3) = max(3, 4)
		{30, 10},
	}
	for _, tc := range tests {
		if got := previewPageSize(tc.h); got != tc.want {
			t.Errorf("previewPageSize(%d) = %d, want %d", tc.h, got, tc.want)
		}
	}
}
