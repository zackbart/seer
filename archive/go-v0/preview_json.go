package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── JSON renderer ─────────────────────────────────────────────────────────────

// JSON color tokens
var (
	jsonKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("147"))            // periwinkle – keys
	jsonStr     = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))            // sage green – string values
	jsonNum     = lipgloss.NewStyle().Foreground(lipgloss.Color("222"))            // pale gold – numbers
	jsonBool    = lipgloss.NewStyle().Foreground(lipgloss.Color("215")).Bold(true) // amber – booleans
	jsonNull    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true) // dim – null
	jsonBracket = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))            // grey – brackets
	jsonMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))            // dim – punctuation / ellipsis
)

func renderJSONPreview(text string, truncated bool) string {
	// Parse into a generic value
	var v interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &v); err != nil {
		// Not valid JSON — show the error and fall back to raw text
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		return errStyle.Render("  invalid JSON: "+err.Error()) + "\n\n" + text
	}

	var sb strings.Builder
	writeJSON(&sb, v, 0)
	out := sb.String()

	if truncated {
		out += "\n" + jsonMuted.Render("  … file truncated, showing partial parse")
	}
	return out
}

// writeJSON recursively pretty-prints a JSON value with colour.
func writeJSON(sb *strings.Builder, v interface{}, depth int) {
	indent := strings.Repeat("  ", depth)
	childIndent := strings.Repeat("  ", depth+1)

	switch val := v.(type) {
	case map[string]interface{}:
		if len(val) == 0 {
			sb.WriteString(jsonBracket.Render("{}"))
			return
		}
		sb.WriteString(jsonBracket.Render("{") + "\n")
		// Sort keys for deterministic output
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			sb.WriteString(childIndent)
			sb.WriteString(jsonKey.Render(`"` + k + `"`))
			sb.WriteString(jsonMuted.Render(": "))
			writeJSON(sb, val[k], depth+1)
			if i < len(keys)-1 {
				sb.WriteString(jsonMuted.Render(","))
			}
			sb.WriteString("\n")
		}
		sb.WriteString(indent + jsonBracket.Render("}"))

	case []interface{}:
		if len(val) == 0 {
			sb.WriteString(jsonBracket.Render("[]"))
			return
		}
		sb.WriteString(jsonBracket.Render("[") + "\n")
		// Cap array preview at 100 items to avoid enormous output
		limit := len(val)
		capped := false
		if limit > 100 {
			limit = 100
			capped = true
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(childIndent)
			writeJSON(sb, val[i], depth+1)
			if i < len(val)-1 {
				sb.WriteString(jsonMuted.Render(","))
			}
			sb.WriteString("\n")
		}
		if capped {
			sb.WriteString(childIndent + jsonMuted.Render(fmt.Sprintf("… %d more items", len(val)-limit)) + "\n")
		}
		sb.WriteString(indent + jsonBracket.Render("]"))

	case string:
		// Escape double quotes inside the string for display
		escaped := strings.ReplaceAll(val, `"`, `\"`)
		sb.WriteString(jsonStr.Render(`"` + escaped + `"`))

	case float64:
		// Render as integer when there's no fractional part
		if val == float64(int64(val)) {
			sb.WriteString(jsonNum.Render(fmt.Sprintf("%d", int64(val))))
		} else {
			sb.WriteString(jsonNum.Render(fmt.Sprintf("%g", val)))
		}

	case bool:
		if val {
			sb.WriteString(jsonBool.Render("true"))
		} else {
			sb.WriteString(jsonBool.Render("false"))
		}

	case nil:
		sb.WriteString(jsonNull.Render("null"))

	default:
		sb.WriteString(fmt.Sprintf("%v", val))
	}
}
