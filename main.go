package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxPreviewBytes = 256 * 1024
	maxDirPreview   = 40
)

var imageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
	".svg":  true,
}

type entry struct {
	name    string
	path    string
	isDir   bool
	size    int64
	modTime time.Time
}

type previewLoadedMsg struct {
	requestID int
	cacheKey  string
	content   string
	err       error
}

type model struct {
	cwd           string
	entries       []entry
	selected      int
	showHidden    bool
	preview       string
	status        string
	width         int
	height        int
	previewOffset int
	loading       bool
	requestID     int
	cache         map[string]string
}

func initialModel() model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	entries, listErr := listDir(cwd, false)
	status := "ready"
	if listErr != nil {
		status = listErr.Error()
	}

	return model{
		cwd:        cwd,
		entries:    entries,
		selected:   0,
		preview:    "",
		status:     status,
		cache:      make(map[string]string),
		showHidden: false,
	}
}

func (m model) Init() tea.Cmd {
	return m.requestPreview()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampPreviewOffset()
		return m, m.requestPreview()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.selected < len(m.entries)-1 {
				m.selected++
				m.previewOffset = 0
				return m, m.requestPreview()
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.previewOffset = 0
				return m, m.requestPreview()
			}
		case "g", "home":
			m.selected = 0
			m.previewOffset = 0
			return m, m.requestPreview()
		case "G", "end":
			if len(m.entries) > 0 {
				m.selected = len(m.entries) - 1
				m.previewOffset = 0
				return m, m.requestPreview()
			}
		case "l", "right", "enter":
			if len(m.entries) == 0 {
				break
			}
			picked := m.entries[m.selected]
			if picked.isDir {
				if err := m.changeDir(picked.path); err != nil {
					m.status = err.Error()
				}
				return m, m.requestPreview()
			}
			m.status = fmt.Sprintf("previewing %s", picked.name)
			return m, m.requestPreview()
		case "h", "left", "backspace":
			parent := filepath.Dir(m.cwd)
			if parent != m.cwd {
				if err := m.changeDir(parent); err != nil {
					m.status = err.Error()
				}
				return m, m.requestPreview()
			}
		case ".":
			m.showHidden = !m.showHidden
			entries, err := listDir(m.cwd, m.showHidden)
			if err != nil {
				m.status = err.Error()
			} else {
				m.entries = entries
				m.selected = 0
				m.previewOffset = 0
				if m.showHidden {
					m.status = "showing hidden files"
				} else {
					m.status = "hiding hidden files"
				}
			}
			return m, m.requestPreview()
		case "ctrl+d", "pagedown":
			m.previewOffset += previewPageSize(m.height)
			m.clampPreviewOffset()
		case "ctrl+u", "pageup":
			m.previewOffset -= previewPageSize(m.height)
			m.clampPreviewOffset()
		case "r":
			entries, err := listDir(m.cwd, m.showHidden)
			if err != nil {
				m.status = err.Error()
			} else {
				m.entries = entries
				if m.selected >= len(m.entries) {
					m.selected = max(0, len(m.entries)-1)
				}
				m.status = "reloaded"
			}
			return m, m.requestPreview()
		}

	case tea.MouseMsg:
		event := tea.MouseEvent(msg)
		if !m.isInPreviewPane(event.X, event.Y) {
			return m, nil
		}

		switch event.Button {
		case tea.MouseButtonWheelDown:
			m.previewOffset += 3
			m.clampPreviewOffset()
		case tea.MouseButtonWheelUp:
			m.previewOffset -= 3
			m.clampPreviewOffset()
		}

	case previewLoadedMsg:
		if msg.requestID != m.requestID {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.preview = "preview error: " + msg.err.Error()
			return m, nil
		}
		m.cache[msg.cacheKey] = msg.content
		m.preview = msg.content
		m.clampPreviewOffset()
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading..."
	}

	leftW := max(24, m.width/3)
	rightW := m.width - leftW - 1
	bodyH := max(4, m.height-2)

	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	selectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Bold(true)
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("111"))

	var listLines []string
	listLines = append(listLines, headerStyle.Render(trimToWidth(m.cwd, leftW)))
	listLines = append(listLines, mutedStyle.Render(strings.Repeat("-", max(1, leftW-1))))

	if len(m.entries) == 0 {
		listLines = append(listLines, mutedStyle.Render("(empty directory)"))
	} else {
		for i, e := range m.entries {
			name := e.name
			if e.isDir {
				name = dirStyle.Render(name + "/")
			}
			line := trimToWidth(name, leftW)
			if i == m.selected {
				line = selectStyle.Render(trimToWidth(rawName(e), leftW))
			}
			listLines = append(listLines, line)
		}
	}

	list := lipgloss.NewStyle().Width(leftW).Height(bodyH).Render(strings.Join(listLines, "\n"))

	previewTitle := "preview"
	if len(m.entries) > 0 {
		previewTitle = m.entries[m.selected].name
	}
	if m.loading {
		previewTitle += " (loading...)"
	}

	previewStyle := lipgloss.NewStyle().Width(rightW).Height(max(1, bodyH-2))
	previewHeader := headerStyle.Render(trimToWidth(previewTitle, rightW))
	previewDivider := mutedStyle.Render(strings.Repeat("-", max(1, rightW-1)))
	previewBody := m.preview
	if previewBody == "" {
		previewBody = mutedStyle.Render("(no preview)")
	}
	previewBody = previewStyle.Render(m.slicePreview(previewBody, bodyH-2))

	statusLine := mutedStyle.Render(trimToWidth(m.status, m.width))
	controlsLine := mutedStyle.Render(trimToWidth(controlLegend(), m.width))

	body := lipgloss.JoinHorizontal(lipgloss.Top, list, previewHeader+"\n"+previewDivider+"\n"+previewBody)
	return body + "\n" + statusLine + "\n" + controlsLine
}

func controlLegend() string {
	return "j/k or up/down move  g/G home/end  enter/l open  h/backspace up  . hidden  ctrl+d/u scroll preview  trackpad/mouse wheel scroll preview  r reload  q quit"
}

func (m model) isInPreviewPane(x, y int) bool {
	leftW := max(24, m.width/3)
	bodyH := max(4, m.height-2)
	previewStartX := leftW
	previewStartY := 2
	previewEndY := bodyH - 1

	return x >= previewStartX && y >= previewStartY && y <= previewEndY
}

func (m *model) changeDir(path string) error {
	entries, err := listDir(path, m.showHidden)
	if err != nil {
		return err
	}
	m.cwd = path
	m.entries = entries
	m.selected = 0
	m.previewOffset = 0
	m.status = path
	return nil
}

func (m *model) requestPreview() tea.Cmd {
	if len(m.entries) == 0 {
		m.preview = ""
		m.loading = false
		return nil
	}

	picked := m.entries[m.selected]
	cacheKey := previewKey(picked.path, picked.modTime, picked.size, m.width, m.height)
	if val, ok := m.cache[cacheKey]; ok {
		m.preview = val
		m.loading = false
		return nil
	}

	m.requestID++
	requestID := m.requestID
	m.loading = true
	path := picked.path
	width := max(40, m.width-m.width/3-2)
	height := max(8, m.height-4)

	return func() tea.Msg {
		content, err := buildPreview(path, width, height)
		return previewLoadedMsg{
			requestID: requestID,
			cacheKey:  cacheKey,
			content:   content,
			err:       err,
		}
	}
}

func (m *model) slicePreview(in string, h int) string {
	if h <= 0 {
		return ""
	}
	lines := strings.Split(in, "\n")
	maxStart := max(0, len(lines)-h)
	if m.previewOffset > maxStart {
		m.previewOffset = maxStart
	}
	if m.previewOffset < 0 {
		m.previewOffset = 0
	}
	start := m.previewOffset
	end := min(len(lines), start+h)
	return strings.Join(lines[start:end], "\n")
}

func (m *model) clampPreviewOffset() {
	if m.previewOffset < 0 {
		m.previewOffset = 0
	}
	if m.preview == "" {
		m.previewOffset = 0
		return
	}
	lines := strings.Split(m.preview, "\n")
	viewport := m.previewViewportHeight()
	maxStart := max(0, len(lines)-viewport)
	if m.previewOffset > maxStart {
		m.previewOffset = maxStart
	}
}

func (m model) previewViewportHeight() int {
	bodyH := max(4, m.height-2)
	return max(1, bodyH-2)
}

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
		return fmt.Sprintf("binary file: %s\nsize: %s\nmodified: %s", filepath.Base(path), humanSize(info.Size()), info.ModTime().Format(time.RFC822)), nil
	}

	text := string(buf)
	if !utf8.ValidString(text) {
		return fmt.Sprintf("non-utf8 text file: %s\nsize: %s", filepath.Base(path), humanSize(info.Size())), nil
	}

	switch ext {
	case ".md", ".markdown", ".mdx":
		return renderMarkdownPreview(text, width, n == maxPreviewBytes), nil
	case ".mmd", ".mermaid":
		return renderMermaidNative(text), nil
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

	items := make([]string, 0, maxDirPreview+3)
	items = append(items, fmt.Sprintf("directory: %s", path))
	items = append(items, fmt.Sprintf("items: %d", len(entries)))
	items = append(items, "")

	limit := min(len(entries), maxDirPreview)
	for i := 0; i < limit; i++ {
		name := entries[i].Name()
		if entries[i].IsDir() {
			name += "/"
		}
		items = append(items, name)
	}
	if len(entries) > limit {
		items = append(items, fmt.Sprintf("... and %d more", len(entries)-limit))
	}

	return strings.Join(items, "\n"), nil
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
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(24, width-3)),
		glamour.WithTableWrap(true),
		glamour.WithPreservedNewLines(),
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

func renderImageASCII(img image.Image, width, height int) string {
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return ""
	}

	chars := []rune(" .:-=+*#%@")
	outW := max(16, width-2)
	outH := max(8, height-3)

	var sb strings.Builder
	for y := 0; y < outH; y++ {
		sy := b.Min.Y + (y*(b.Dy()-1))/max(1, outH-1)
		for x := 0; x < outW; x++ {
			sx := b.Min.X + (x*(b.Dx()-1))/max(1, outW-1)
			lum := luminance(img.At(sx, sy))
			idx := int(lum * float64(len(chars)-1) / 255.0)
			if idx < 0 {
				idx = 0
			}
			if idx >= len(chars) {
				idx = len(chars) - 1
			}
			sb.WriteRune(chars[idx])
		}
		if y < outH-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	rf := float64(r>>8) * 0.299
	gf := float64(g>>8) * 0.587
	bf := float64(b>>8) * 0.114
	return rf + gf + bf
}

func replaceMermaidFences(markdown string) string {
	lines := strings.Split(markdown, "\n")
	inMermaidFence := false
	var block strings.Builder
	transformed := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inMermaidFence {
			if strings.HasPrefix(trimmed, "```") {
				lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				if strings.EqualFold(lang, "mermaid") {
					inMermaidFence = true
					block.Reset()
					continue
				}
			}
			transformed = append(transformed, line)
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			content := strings.TrimSpace(block.String())
			if content != "" {
				transformed = append(transformed, "")
				transformed = append(transformed, renderMermaidMarkdownPreview(content))
				transformed = append(transformed, "")
			}
			inMermaidFence = false
			continue
		}
		block.WriteString(line)
		block.WriteString("\n")
	}

	if inMermaidFence {
		transformed = append(transformed, "```mermaid")
		transformed = append(transformed, strings.TrimRight(block.String(), "\n"))
	}

	return strings.Join(transformed, "\n")
}

type mermaidNode struct {
	id    string
	label string
}

type mermaidEdge struct {
	from      mermaidNode
	to        mermaidNode
	edgeLabel string
}

type mermaidGraph struct {
	chartType string
	nodeOrder []string
	nodes     map[string]string
	edges     []mermaidEdge
}

func renderMermaidNative(code string) string {
	g := parseMermaidGraph(code)
	if len(g.edges) == 0 {
		return "Mermaid (native preview)\n\n" + code
	}

	var sb strings.Builder
	sb.WriteString("Mermaid (native preview)\n")
	sb.WriteString("type: ")
	sb.WriteString(g.chartType)
	sb.WriteString("\n")
	sb.WriteString("nodes: ")
	sb.WriteString(fmt.Sprintf("%d", len(g.nodeOrder)))
	sb.WriteString("  edges: ")
	sb.WriteString(fmt.Sprintf("%d", len(g.edges)))
	sb.WriteString("\n\n")

	maxEdges := min(len(g.edges), 30)
	for i := 0; i < maxEdges; i++ {
		e := g.edges[i]
		sb.WriteString(fmt.Sprintf("%2d. %s -> %s", i+1, fitMermaidLabel(e.from.label), fitMermaidLabel(e.to.label)))
		if e.edgeLabel != "" {
			sb.WriteString("  [")
			sb.WriteString(fitMermaidLabel(e.edgeLabel))
			sb.WriteString("]")
		}
		sb.WriteByte('\n')
	}
	if len(g.edges) > maxEdges {
		sb.WriteString("... and ")
		sb.WriteString(fmt.Sprintf("%d", len(g.edges)-maxEdges))
		sb.WriteString(" more edges")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func renderMermaidMarkdownPreview(code string) string {
	g := parseMermaidGraph(code)
	if len(g.edges) == 0 {
		return "_Mermaid block found, but no flow edges were parsed._"
	}

	var sb strings.Builder
	sb.WriteString("#### Mermaid Preview\n\n")
	sb.WriteString("**Type:** `")
	sb.WriteString(g.chartType)
	sb.WriteString("`  \\n")
	sb.WriteString("**Nodes:** `")
	sb.WriteString(fmt.Sprintf("%d", len(g.nodeOrder)))
	sb.WriteString("`  \\n")
	sb.WriteString("**Edges:** `")
	sb.WriteString(fmt.Sprintf("%d", len(g.edges)))
	sb.WriteString("`\n\n")

	sb.WriteString("| # | From | To | Label |\n")
	sb.WriteString("|---:|------|----|-------|\n")
	maxEdges := min(len(g.edges), 20)
	for i := 0; i < maxEdges; i++ {
		e := g.edges[i]
		label := e.edgeLabel
		if label == "" {
			label = "-"
		}
		sb.WriteString("| ")
		sb.WriteString(fmt.Sprintf("%d", i+1))
		sb.WriteString(" | ")
		sb.WriteString(escapeMarkdownTableCell(fitMermaidLabel(e.from.label)))
		sb.WriteString(" | ")
		sb.WriteString(escapeMarkdownTableCell(fitMermaidLabel(e.to.label)))
		sb.WriteString(" | ")
		sb.WriteString(escapeMarkdownTableCell(fitMermaidLabel(label)))
		sb.WriteString(" |\n")
	}
	if len(g.edges) > maxEdges {
		sb.WriteString("\n_...and ")
		sb.WriteString(fmt.Sprintf("%d", len(g.edges)-maxEdges))
		sb.WriteString(" more edges._\n")
	}

	return sb.String()
}

func parseMermaidGraph(code string) mermaidGraph {
	lines := strings.Split(code, "\n")
	edges := make([]mermaidEdge, 0)
	chartType := "diagram"

	nodeOrder := make([]string, 0)
	nodes := make(map[string]string)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		if chartType == "diagram" {
			parts := strings.Fields(trimmed)
			if len(parts) > 0 {
				chartType = parts[0]
			}
		}

		edge, ok := parseMermaidEdge(trimmed)
		if !ok {
			continue
		}
		edges = append(edges, edge)
		registerMermaidNode(nodes, &nodeOrder, edge.from)
		registerMermaidNode(nodes, &nodeOrder, edge.to)
	}

	return mermaidGraph{
		chartType: chartType,
		nodeOrder: nodeOrder,
		nodes:     nodes,
		edges:     edges,
	}
}

func parseMermaidEdge(line string) (mermaidEdge, bool) {
	operators := []string{"-->", "==>", "-.->"}
	op := ""
	idx := -1
	for _, candidate := range operators {
		i := strings.Index(line, candidate)
		if i >= 0 && (idx == -1 || i < idx) {
			idx = i
			op = candidate
		}
	}
	if idx < 0 {
		return mermaidEdge{}, false
	}

	left := strings.TrimSpace(line[:idx])
	right := strings.TrimSpace(line[idx+len(op):])
	edgeLabel := ""
	if strings.HasPrefix(right, "|") {
		if end := strings.Index(right[1:], "|"); end >= 0 {
			edgeLabel = strings.TrimSpace(right[1 : end+1])
			right = strings.TrimSpace(right[end+2:])
		}
	}

	from := parseMermaidNode(left)
	to := parseMermaidNode(right)
	if from.id == "" || to.id == "" {
		return mermaidEdge{}, false
	}
	return mermaidEdge{from: from, to: to, edgeLabel: edgeLabel}, true
}

func parseMermaidNode(raw string) mermaidNode {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, ";"))
	if raw == "" {
		return mermaidNode{}
	}
	raw = strings.Split(raw, ":::")[0]

	openToClose := map[byte]byte{'[': ']', '(': ')', '{': '}'}
	for open, close := range openToClose {
		if i := strings.IndexByte(raw, open); i > 0 {
			id := strings.TrimSpace(raw[:i])
			if j := strings.LastIndexByte(raw, close); j > i {
				label := cleanMermaidText(raw[i+1 : j])
				if label == "" {
					label = cleanMermaidText(id)
				}
				return mermaidNode{id: cleanMermaidID(id), label: label}
			}
		}
	}

	id := cleanMermaidID(raw)
	if id == "" {
		return mermaidNode{}
	}
	return mermaidNode{id: id, label: cleanMermaidText(id)}
}

func registerMermaidNode(nodes map[string]string, order *[]string, n mermaidNode) {
	if n.id == "" {
		return
	}
	if _, exists := nodes[n.id]; !exists {
		nodes[n.id] = n.label
		*order = append(*order, n.id)
	}
}

func cleanMermaidText(in string) string {
	in = strings.TrimSpace(in)
	replacer := strings.NewReplacer("\"", "", "'", "", "|", " ", "`", "")
	in = replacer.Replace(in)
	in = strings.Join(strings.Fields(in), " ")
	return in
}

func cleanMermaidID(in string) string {
	in = strings.TrimSpace(in)
	in = strings.TrimPrefix(in, "(")
	in = strings.TrimPrefix(in, "[")
	in = strings.TrimPrefix(in, "{")
	in = strings.TrimSuffix(in, ")")
	in = strings.TrimSuffix(in, "]")
	in = strings.TrimSuffix(in, "}")
	in = strings.TrimSuffix(in, ";")
	fields := strings.Fields(in)
	if len(fields) == 0 {
		return ""
	}
	in = fields[0]
	in = strings.TrimSpace(in)
	return strings.Trim(in, "\"")
}

func fitMermaidLabel(in string) string {
	in = cleanMermaidText(in)
	if in == "" {
		return "node"
	}
	if len(in) <= 24 {
		return in
	}
	return in[:23] + "~"
}

func escapeMarkdownTableCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.TrimSpace(s)
}

func listDir(path string, showHidden bool) ([]entry, error) {
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, len(items))
	for _, item := range items {
		name := item.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		info, err := item.Info()
		if err != nil {
			continue
		}
		entries = append(entries, entry{
			name:    name,
			path:    full,
			isDir:   item.IsDir(),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})

	return entries, nil
}

func previewKey(path string, modTime time.Time, size int64, width, height int) string {
	return fmt.Sprintf("%s|%d|%d|%d|%d", path, modTime.UnixNano(), size, width, height)
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

func isLikelyBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	for i := 0; i < len(data) && i < 8192; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

func trimToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return string(runes[:1])
	}
	return string(runes[:width-1]) + "…"
}

func rawName(e entry) string {
	if e.isDir {
		return e.name + "/"
	}
	return e.name
}

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

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
