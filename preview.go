package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// ── preview builders ──────────────────────────────────────────────────────────

func buildPreview(path string, width, height int) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return buildDirPreview(path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if imageExts[ext] {
		if img, ok := imagePreview(path, width, height); ok {
			return img, nil
		}
		return fmt.Sprintf("image file: %s\nsize: %s\n\npreview unavailable for this format", filepath.Base(path), humanSize(info.Size())), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, maxPreviewBytes)
	n, readErr := f.Read(buf)
	if readErr != nil && readErr != io.EOF {
		return "", readErr
	}
	buf = buf[:n]

	if isLikelyBinary(buf) {
		return fmt.Sprintf("binary file: %s\nsize: %s\nmodified: %s", filepath.Base(path), humanSize(info.Size()), info.ModTime().Format("02 Jan 2006 15:04 MST")), nil
	}

	text := string(buf)
	if !utf8.ValidString(text) {
		return fmt.Sprintf("non-utf8 text file: %s\nsize: %s", filepath.Base(path), humanSize(info.Size())), nil
	}
	// Normalize Windows-style line endings so \r doesn't corrupt terminal rendering.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	switch ext {
	case ".md", ".markdown", ".mdx":
		return renderMarkdownPreview(text, width, n == maxPreviewBytes), nil
	case ".mmd", ".mermaid":
		return renderMermaidNative(text), nil
	case ".json":
		return renderJSONPreview(text, n == maxPreviewBytes), nil
	}

	if highlighted := highlight(path, text); highlighted != "" {
		if n == maxPreviewBytes {
			highlighted += "\n\n... preview truncated ..."
		}
		return highlighted, nil
	}

	if n == maxPreviewBytes {
		text += "\n\n... preview truncated ..."
	}
	return text, nil
}

func buildDirPreview(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	// Styled directory preview
	dirStyle := lipgloss.NewStyle().Foreground(clrDir).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)
	dimStyle := lipgloss.NewStyle().Foreground(clrDim)

	var sb strings.Builder
	sb.WriteString(dirStyle.Render(fileIconExt(catDir, "")+filepath.Base(path)+"/") + "\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("  %d items", len(entries))) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", 30)) + "\n\n")

	limit := min(len(entries), maxDirPreview)
	for i := 0; i < limit; i++ {
		e := entries[i]
		name := e.Name()
		fakeEntry := entry{name: name, isDir: e.IsDir()}
		var line string
		if e.IsDir() {
			line = entryNameStyle(fakeEntry).Render("  " + fileIconExt(catDir, "") + name + "/")
		} else {
			// Categorise by name only (no stat for speed).
			cat := categorise(fakeEntry)
			col := entryNameStyle(fakeEntry)
			line = col.Render("  " + fileIconExt(cat, filepath.Ext(name)) + name)
		}
		sb.WriteString(line + "\n")
	}
	if len(entries) > limit {
		sb.WriteString(mutedStyle.Render(fmt.Sprintf("\n  … and %d more", len(entries)-limit)) + "\n")
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}

func imagePreview(path string, width, height int) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", false
	}

	rendered := renderImageASCII(img, width, height)
	if rendered == "" {
		return "", false
	}
	return rendered, true
}

func renderMarkdownPreview(markdown string, width int, truncated bool) string {
	prepared := replaceMermaidFences(markdown)
	rendered := prepared
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("tokyo-night"),
		glamour.WithWordWrap(max(24, width-2)),
		glamour.WithTableWrap(true),
		glamour.WithEmoji(),
	)
	if err == nil {
		if out, renderErr := r.Render(prepared); renderErr == nil {
			rendered = out
		}
	}

	if truncated {
		rendered += "\n\n... preview truncated ..."
	}

	return rendered
}

func highlight(path, text string) string {
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Analyse(text)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("nord")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return ""
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return ""
	}
	return buf.String()
}
