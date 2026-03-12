package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// archiveExts lists file extensions handled by buildArchivePreview.
var archiveExts = map[string]bool{
	".zip":  true,
	".jar":  true,
	".war":  true,
	".tar":  true,
	".gz":   true,
	".tgz":  true,
	".bz2":  true,
	".tbz2": true,
	".xz":   true,
	".txz":  true,
	".zst":  true,
	".7z":   true,
	".rar":  true,
}

// buildArchivePreview lists the contents of an archive using an external tool.
// Falls back to a simple info block if no suitable tool is available.
func buildArchivePreview(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	lines, err := archiveListings(path, ext)
	if err != nil || len(lines) == 0 {
		// Graceful fallback: just show file info.
		return fmt.Sprintf("archive: %s\n\n(install unzip / tar / 7z to preview contents)",
			filepath.Base(path)), nil
	}

	labelStyle := lipgloss.NewStyle().Foreground(clrConfig).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(clrMuted)
	dimStyle := lipgloss.NewStyle().Foreground(clrDim)
	fileStyle := lipgloss.NewStyle().Foreground(clrFile)
	dirStyle := lipgloss.NewStyle().Foreground(clrDir).Bold(true)

	var sb strings.Builder
	sb.WriteString(labelStyle.Render("archive: "+filepath.Base(path)) + "\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("  %d entries", len(lines))) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", 30)) + "\n\n")

	maxShow := 200
	for i, line := range lines {
		if i >= maxShow {
			sb.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("  … and %d more", len(lines)-maxShow)) + "\n")
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, "/") {
			sb.WriteString(dirStyle.Render("  "+fileIconExt(catDir, "")+line) + "\n")
		} else {
			sb.WriteString(fileStyle.Render("  "+line) + "\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}

// archiveListings runs the appropriate tool and returns one entry per line.
func archiveListings(path, ext string) ([]string, error) {
	var cmd *exec.Cmd

	switch ext {
	case ".zip", ".jar", ".war":
		if _, err := exec.LookPath("unzip"); err == nil {
			cmd = exec.Command("unzip", "-Z1", path)
		} else if _, err := exec.LookPath("7z"); err == nil {
			cmd = exec.Command("7z", "l", "-ba", "-slt", path)
		}
	case ".tar":
		if _, err := exec.LookPath("tar"); err == nil {
			cmd = exec.Command("tar", "tf", path)
		}
	case ".gz", ".tgz":
		if _, err := exec.LookPath("tar"); err == nil {
			cmd = exec.Command("tar", "tzf", path)
		}
	case ".bz2", ".tbz2":
		if _, err := exec.LookPath("tar"); err == nil {
			cmd = exec.Command("tar", "tjf", path)
		}
	case ".xz", ".txz":
		if _, err := exec.LookPath("tar"); err == nil {
			cmd = exec.Command("tar", "tJf", path)
		}
	case ".zst":
		if _, err := exec.LookPath("tar"); err == nil {
			cmd = exec.Command("tar", "--use-compress-program=zstd", "-tf", path)
		}
	case ".7z", ".rar":
		if _, err := exec.LookPath("7z"); err == nil {
			cmd = exec.Command("7z", "l", "-ba", path)
		}
	}

	if cmd == nil {
		return nil, fmt.Errorf("no tool available")
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	raw := strings.Split(strings.TrimSpace(string(out)), "\n")
	// For 7z's "-slt" output, extract just the Path fields.
	if len(raw) > 0 && (ext == ".zip" || ext == ".jar" || ext == ".war" || ext == ".7z" || ext == ".rar") {
		if _, err := exec.LookPath("unzip"); err != nil && ext != ".tar" {
			var paths []string
			for _, line := range raw {
				if strings.HasPrefix(line, "Path = ") {
					p := strings.TrimPrefix(line, "Path = ")
					paths = append(paths, p)
				}
			}
			if len(paths) > 0 {
				return paths, nil
			}
		}
	}

	return raw, nil
}
