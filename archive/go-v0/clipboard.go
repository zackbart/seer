package main

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

func copyToClipboard(text string) error {
	if text == "" {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		return runClipboardCommand(text, "pbcopy")
	case "windows":
		return runClipboardCommand(text, "cmd", "/c", "clip")
	default:
		candidates := [][]string{
			{"wl-copy"},
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
		}
		var lastErr error
		for _, c := range candidates {
			if _, err := exec.LookPath(c[0]); err != nil {
				continue
			}
			if err := runClipboardCommand(text, c[0], c[1:]...); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		if lastErr != nil {
			return lastErr
		}
		return errors.New("no clipboard utility found (tried wl-copy, xclip, xsel)")
	}
}

func runClipboardCommand(text, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
