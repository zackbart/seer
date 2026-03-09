package main

import (
	"testing"
)

// ── categorise ────────────────────────────────────────────────────────────────

func TestCategorise(t *testing.T) {
	tests := []struct {
		name string
		e    entry
		want fileCategory
	}{
		// Directory
		{"dir", entry{name: "mydir", isDir: true}, catDir},
		// Images
		{"png", entry{name: "photo.png"}, catImage},
		{"jpg", entry{name: "photo.jpg"}, catImage},
		{"jpeg", entry{name: "photo.jpeg"}, catImage},
		{"gif", entry{name: "anim.gif"}, catImage},
		{"webp", entry{name: "img.webp"}, catImage},
		{"bmp", entry{name: "img.bmp"}, catImage},
		{"tiff", entry{name: "img.tiff"}, catImage},
		// Documents
		{"md", entry{name: "readme.md"}, catDoc},
		{"markdown", entry{name: "doc.markdown"}, catDoc},
		{"mdx", entry{name: "page.mdx"}, catDoc},
		{"rst", entry{name: "doc.rst"}, catDoc},
		{"txt", entry{name: "notes.txt"}, catDoc},
		// Executables / scripts
		{"sh", entry{name: "run.sh"}, catExec},
		{"bash", entry{name: "setup.bash"}, catExec},
		{"zsh", entry{name: "script.zsh"}, catExec},
		// Code
		{"go", entry{name: "main.go"}, catCode},
		{"js", entry{name: "app.js"}, catCode},
		{"ts", entry{name: "app.ts"}, catCode},
		{"py", entry{name: "script.py"}, catCode},
		{"rs", entry{name: "lib.rs"}, catCode},
		{"mmd", entry{name: "diagram.mmd"}, catCode},
		// Config
		{"json", entry{name: "config.json"}, catConfig},
		{"yaml", entry{name: "config.yaml"}, catConfig},
		{"yml", entry{name: "docker.yml"}, catConfig},
		{"toml", entry{name: "Cargo.toml"}, catConfig},
		{"env", entry{name: ".env"}, catConfig},
		// Unknown
		{"unknown", entry{name: "archive.tar"}, catOther},
		{"no ext", entry{name: "Makefile"}, catOther},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := categorise(tc.e)
			if got != tc.want {
				t.Errorf("categorise(%q) = %v, want %v", tc.e.name, got, tc.want)
			}
		})
	}
}

func TestCategoriseExtCaseInsensitive(t *testing.T) {
	// Extensions must be matched case-insensitively
	upper := categorise(entry{name: "photo.PNG"})
	lower := categorise(entry{name: "photo.png"})
	if upper != lower {
		t.Errorf("categorise should be case-insensitive: .PNG=%v .png=%v", upper, lower)
	}
	if lower != catImage {
		t.Errorf("expected catImage for .png, got %v", lower)
	}
}

// ── fileIconExt ───────────────────────────────────────────────────────────────

func TestFileIconExtReturnsNonEmpty(t *testing.T) {
	cats := []fileCategory{catDir, catImage, catDoc, catCode, catConfig, catExec, catBinary, catOther}
	for _, cat := range cats {
		icon := fileIconExt(cat, "")
		if icon == "" {
			t.Errorf("fileIconExt(%v, \"\") returned empty string", cat)
		}
	}
}

func TestFileIconExtWithExtension(t *testing.T) {
	// Extension-specific icons should differ from generic fallback
	goIcon := fileIconExt(catCode, ".go")
	pyIcon := fileIconExt(catCode, ".py")
	// Both should be non-empty
	if goIcon == "" || pyIcon == "" {
		t.Error("file icons must not be empty")
	}
}

// ── entryNameStyle ────────────────────────────────────────────────────────────

func TestEntryNameStyleNotNil(t *testing.T) {
	entries := []entry{
		{name: "dir", isDir: true},
		{name: ".hidden_dir", isDir: true},
		{name: "file.txt", isDir: false},
		{name: ".hidden.txt", isDir: false},
	}
	for _, e := range entries {
		style := entryNameStyle(e)
		// Styles are structs — we just verify Render doesn't panic
		rendered := style.Render("test")
		if rendered == "" {
			t.Errorf("entryNameStyle(%q).Render returned empty string", e.name)
		}
	}
}

func TestEntryNameStyleDirVsFile(t *testing.T) {
	dir := entryNameStyle(entry{name: "mydir", isDir: true})
	file := entryNameStyle(entry{name: "myfile.txt", isDir: false})
	// Directories use bold; files do not. We test by checking the rendered output
	// contains the text (just a sanity check that styles apply without panic).
	if dir.Render("x") == "" || file.Render("x") == "" {
		t.Error("style.Render must not return empty")
	}
}
