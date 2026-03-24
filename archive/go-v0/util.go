package main

import "fmt"

func humanSize(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(n)
	idx := 0
	for v >= 1024 && idx < len(units)-1 {
		v /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%d %s", n, units[idx])
	}
	return fmt.Sprintf("%.1f %s", v, units[idx])
}

func previewPageSize(h int) int {
	return max(3, h/3)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
