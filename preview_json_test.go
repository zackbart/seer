package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// stripANSI removes ANSI escape codes for plain-text assertions.
func stripANSI(s string) string {
	return ansi.Strip(s)
}

// ── renderJSONPreview ─────────────────────────────────────────────────────────

func TestRenderJSONPreviewObject(t *testing.T) {
	input := `{"name":"Alice","age":30}`
	out := stripANSI(renderJSONPreview(input, false))
	if !strings.Contains(out, "name") {
		t.Error("expected 'name' in output")
	}
	if !strings.Contains(out, "Alice") {
		t.Error("expected 'Alice' in output")
	}
	if !strings.Contains(out, "age") {
		t.Error("expected 'age' in output")
	}
	if !strings.Contains(out, "30") {
		t.Error("expected '30' in output")
	}
}

func TestRenderJSONPreviewArray(t *testing.T) {
	input := `[1, 2, 3]`
	out := stripANSI(renderJSONPreview(input, false))
	if !strings.Contains(out, "1") || !strings.Contains(out, "2") || !strings.Contains(out, "3") {
		t.Errorf("expected array elements in output, got: %s", out)
	}
}

func TestRenderJSONPreviewNested(t *testing.T) {
	input := `{"user":{"name":"Bob","active":true}}`
	out := stripANSI(renderJSONPreview(input, false))
	if !strings.Contains(out, "user") {
		t.Error("expected 'user' key in output")
	}
	if !strings.Contains(out, "Bob") {
		t.Error("expected 'Bob' in output")
	}
	if !strings.Contains(out, "true") {
		t.Error("expected 'true' in output")
	}
}

func TestRenderJSONPreviewNull(t *testing.T) {
	input := `{"key":null}`
	out := stripANSI(renderJSONPreview(input, false))
	if !strings.Contains(out, "null") {
		t.Errorf("expected 'null' in output, got: %s", out)
	}
}

func TestRenderJSONPreviewInvalid(t *testing.T) {
	input := `{not valid json`
	out := stripANSI(renderJSONPreview(input, false))
	if !strings.Contains(out, "invalid JSON") {
		t.Errorf("expected 'invalid JSON' error message, got: %s", out)
	}
}

func TestRenderJSONPreviewTruncated(t *testing.T) {
	input := `{"k":"v"}`
	out := stripANSI(renderJSONPreview(input, true))
	if !strings.Contains(out, "truncated") {
		t.Errorf("expected truncation notice when truncated=true, got: %s", out)
	}
}

func TestRenderJSONPreviewNotTruncated(t *testing.T) {
	input := `{"k":"v"}`
	out := stripANSI(renderJSONPreview(input, false))
	if strings.Contains(out, "truncated") {
		t.Errorf("unexpected truncation notice when truncated=false, got: %s", out)
	}
}

func TestRenderJSONPreviewEmptyObject(t *testing.T) {
	out := stripANSI(renderJSONPreview(`{}`, false))
	if !strings.Contains(out, "{}") {
		t.Errorf("expected '{}' for empty object, got: %s", out)
	}
}

func TestRenderJSONPreviewEmptyArray(t *testing.T) {
	out := stripANSI(renderJSONPreview(`[]`, false))
	if !strings.Contains(out, "[]") {
		t.Errorf("expected '[]' for empty array, got: %s", out)
	}
}

func TestRenderJSONPreviewSortedKeys(t *testing.T) {
	input := `{"zebra":1,"apple":2,"mango":3}`
	out := stripANSI(renderJSONPreview(input, false))
	appleIdx := strings.Index(out, "apple")
	mangoIdx := strings.Index(out, "mango")
	zebraIdx := strings.Index(out, "zebra")
	if appleIdx == -1 || mangoIdx == -1 || zebraIdx == -1 {
		t.Fatal("all keys must appear in output")
	}
	if !(appleIdx < mangoIdx && mangoIdx < zebraIdx) {
		t.Errorf("keys should be sorted: apple(%d) < mango(%d) < zebra(%d)",
			appleIdx, mangoIdx, zebraIdx)
	}
}

func TestRenderJSONPreviewLargeArray(t *testing.T) {
	// Arrays > 100 items should be capped
	var items []string
	for i := 0; i < 150; i++ {
		items = append(items, "1")
	}
	input := "[" + strings.Join(items, ",") + "]"
	out := stripANSI(renderJSONPreview(input, false))
	if !strings.Contains(out, "more items") {
		t.Error("expected '…N more items' message for large array")
	}
}
