package main

import "strings"

// ── Mermaid types ─────────────────────────────────────────────────────────────

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

type seqMsg struct {
	from, to, label string
	dashed          bool
}

// ── Mermaid dispatch ──────────────────────────────────────────────────────────

func mermaidChartType(code string) string {
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		if parts := strings.Fields(trimmed); len(parts) > 0 {
			return parts[0]
		}
	}
	return "diagram"
}

func renderMermaidNative(code string) string {
	ct := mermaidChartType(code)
	switch ct {
	case "sequenceDiagram":
		parts, msgs := parseSequenceDiagram(code)
		if len(msgs) > 0 {
			return asciiSequenceDiagram(parts, msgs, 0)
		}
	default:
		g := parseMermaidGraph(code)
		if len(g.nodeOrder) > 0 {
			return asciiFlowchart(g, 0)
		}
	}
	return code
}

func renderMermaidMarkdownPreview(code string) string {
	ct := mermaidChartType(code)
	var art string
	switch ct {
	case "sequenceDiagram":
		parts, msgs := parseSequenceDiagram(code)
		if len(msgs) > 0 {
			art = asciiSequenceDiagram(parts, msgs, 80)
		}
	default:
		g := parseMermaidGraph(code)
		if len(g.nodeOrder) > 0 {
			art = asciiFlowchart(g, 80)
		}
	}
	if art == "" {
		return "_Mermaid block: no diagram content parsed._"
	}
	return "```text\n" + art + "\n```\n"
}

// replaceMermaidFences replaces ```mermaid fences in Markdown with rendered ASCII art.
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

// ── ASCII flowchart renderer ──────────────────────────────────────────────────

// boxDrawMask maps box-drawing runes to a 4-bit NESW connectivity mask.
// N=8 E=4 S=2 W=1 — combining masks via OR produces the correct merged character.
var boxDrawMask = map[rune]int{
	'│': 8 + 2, '─': 4 + 1,
	'┌': 4 + 2, '┐': 1 + 2, '└': 8 + 4, '┘': 8 + 1,
	'├': 8 + 4 + 2, '┤': 8 + 1 + 2, '┬': 4 + 1 + 2, '┴': 8 + 4 + 1, '┼': 15,
}

// maskBoxDraw is the reverse of boxDrawMask.
var maskBoxDraw = map[int]rune{
	4 + 2: '┌', 1 + 2: '┐', 8 + 4: '└', 8 + 1: '┘',
	4 + 1: '─', 8 + 2: '│',
	8 + 4 + 2: '├', 8 + 1 + 2: '┤', 4 + 1 + 2: '┬', 8 + 4 + 1: '┴', 15: '┼',
	4: '╶', 1: '╴', 8: '╵', 2: '╷',
}

// asciiFlowchart renders a mermaid flowchart as an ASCII box diagram.
func asciiFlowchart(g mermaidGraph, maxW int) string {
	if len(g.nodeOrder) == 0 {
		return "(empty diagram)"
	}

	labelOf := func(id string) string {
		if l := g.nodes[id]; l != "" {
			return l
		}
		return id
	}
	nodeBoxW := func(id string) int { return len([]rune(labelOf(id))) + 4 }

	// Build adjacency and in-degree
	succMap := make(map[string][]string)
	tmpIn := make(map[string]int)
	for id := range g.nodes {
		tmpIn[id] = 0
	}
	for _, e := range g.edges {
		if e.from.id == e.to.id {
			continue
		}
		succMap[e.from.id] = append(succMap[e.from.id], e.to.id)
		tmpIn[e.to.id]++
	}

	// Longest-path ranking via Kahn's algorithm
	rank := make(map[string]int)
	q := []string{}
	for id := range g.nodes {
		if tmpIn[id] == 0 {
			q = append(q, id)
		}
	}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, next := range succMap[cur] {
			if rank[cur]+1 > rank[next] {
				rank[next] = rank[cur] + 1
			}
			tmpIn[next]--
			if tmpIn[next] == 0 {
				q = append(q, next)
			}
		}
	}

	maxRank := 0
	for _, r := range rank {
		if r > maxRank {
			maxRank = r
		}
	}

	// Group nodes by rank, preserving nodeOrder within each level
	levels := make([][]string, maxRank+1)
	for _, id := range g.nodeOrder {
		r := rank[id]
		levels[r] = append(levels[r], id)
	}

	// Compute x positions within each level
	const hGap = 3
	nodeX := make(map[string]int)
	levelW := make([]int, maxRank+1)
	for r, nodes := range levels {
		x := 0
		for i, id := range nodes {
			nodeX[id] = x
			x += nodeBoxW(id)
			if i < len(nodes)-1 {
				x += hGap
			}
		}
		levelW[r] = x
	}

	totalW := 1
	for _, lw := range levelW {
		if lw > totalW {
			totalW = lw
		}
	}
	if maxW > 0 && totalW > maxW {
		totalW = maxW
	}

	// Center each level within totalW
	for r, lw := range levelW {
		off := (totalW - lw) / 2
		if off < 0 {
			off = 0
		}
		for _, id := range levels[r] {
			nodeX[id] += off
		}
	}

	// Y positions: 3 rows per box + 2 connector rows between levels
	nodeY := make(map[string]int)
	y := 0
	for r, nodes := range levels {
		for _, id := range nodes {
			nodeY[id] = y
		}
		y += 3
		if r < maxRank {
			y += 2
		}
	}
	totalH := y

	// Grid
	grid := make([][]rune, totalH)
	for i := range grid {
		grid[i] = make([]rune, totalW)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	setRaw := func(x, y int, r rune) {
		if x >= 0 && x < totalW && y >= 0 && y < totalH {
			grid[y][x] = r
		}
	}
	// setBox merges box-drawing characters so branching connectors combine cleanly.
	setBox := func(x, y int, r rune) {
		if x < 0 || x >= totalW || y < 0 || y >= totalH {
			return
		}
		existing := grid[y][x]
		if existing == ' ' {
			grid[y][x] = r
			return
		}
		em, ok1 := boxDrawMask[existing]
		nm, ok2 := boxDrawMask[r]
		if ok1 && ok2 {
			if merged, ok3 := maskBoxDraw[em|nm]; ok3 {
				grid[y][x] = merged
				return
			}
		}
		grid[y][x] = r
	}
	writeStr := func(x, y int, s string) {
		for i, r := range []rune(s) {
			setRaw(x+i, y, r)
		}
	}

	// Draw boxes
	for id := range g.nodes {
		label := labelOf(id)
		x, yy, w := nodeX[id], nodeY[id], nodeBoxW(id)
		setRaw(x, yy, '┌')
		for i := 1; i < w-1; i++ {
			setRaw(x+i, yy, '─')
		}
		setRaw(x+w-1, yy, '┐')
		setRaw(x, yy+1, '│')
		writeStr(x+2, yy+1, label)
		setRaw(x+w-1, yy+1, '│')
		setRaw(x, yy+2, '└')
		for i := 1; i < w-1; i++ {
			setRaw(x+i, yy+2, '─')
		}
		setRaw(x+w-1, yy+2, '┘')
	}

	// Draw edges between adjacent-rank nodes
	for _, e := range g.edges {
		fid, tid := e.from.id, e.to.id
		if fid == tid || rank[fid]+1 != rank[tid] {
			continue
		}
		fw := nodeBoxW(fid)
		fcx := nodeX[fid] + fw/2
		row1 := nodeY[fid] + 3 // first connector row

		tw := nodeBoxW(tid)
		tcx := nodeX[tid] + tw/2

		switch {
		case fcx == tcx:
			// Straight down: │ then ▼
			setBox(fcx, row1, '│')
			setRaw(tcx, row1+1, '▼')
		case fcx < tcx:
			// Go right: └────┐ then ▼
			setBox(fcx, row1, '└')
			for x := fcx + 1; x < tcx; x++ {
				setBox(x, row1, '─')
			}
			setBox(tcx, row1, '┐')
			setRaw(tcx, row1+1, '▼')
		default:
			// Go left: ┌────┘ then ▼
			setBox(tcx, row1, '┌')
			for x := tcx + 1; x < fcx; x++ {
				setBox(x, row1, '─')
			}
			setBox(fcx, row1, '┘')
			setRaw(tcx, row1+1, '▼')
		}
	}

	var sb strings.Builder
	for _, row := range grid {
		sb.WriteString(strings.TrimRight(string(row), " "))
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ── ASCII sequence diagram renderer ──────────────────────────────────────────

func parseSequenceDiagram(code string) (participants []string, msgs []seqMsg) {
	seen := make(map[string]bool)
	add := func(name string) {
		if name != "" && !seen[name] {
			participants = append(participants, name)
			seen[name] = true
		}
	}
	for _, line := range strings.Split(code, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		lower := strings.ToLower(t)
		if strings.HasPrefix(lower, "participant ") || strings.HasPrefix(lower, "actor ") {
			fields := strings.Fields(t)
			if len(fields) < 2 {
				continue
			}
			name := fields[1]
			for i, f := range fields {
				if strings.EqualFold(f, "as") && i+1 < len(fields) {
					name = strings.Join(fields[i+1:], " ")
					break
				}
			}
			add(name)
			continue
		}
		// Message arrows – check in order of decreasing length to avoid mis-matching
		for _, op := range []struct {
			s      string
			dashed bool
		}{
			{"-->>", true}, {"-->", true}, {"->>", false}, {"->", false},
			{"--x", true}, {"-x", false}, {"--)", true}, {"-)", false},
		} {
			idx := strings.Index(t, op.s)
			if idx < 0 {
				continue
			}
			from := strings.TrimSpace(t[:idx])
			rest := strings.TrimSpace(t[idx+len(op.s):])
			to, label := rest, ""
			if ci := strings.Index(rest, ":"); ci >= 0 {
				to = strings.TrimSpace(rest[:ci])
				label = strings.TrimSpace(rest[ci+1:])
			}
			if from == "" || to == "" {
				break
			}
			add(from)
			add(to)
			msgs = append(msgs, seqMsg{from: from, to: to, label: label, dashed: op.dashed})
			break
		}
	}
	return
}

func asciiSequenceDiagram(participants []string, msgs []seqMsg, maxW int) string {
	if len(participants) == 0 {
		return "(no participants)"
	}

	// Column width: participant name + 2 spaces padding, minimum 14, even
	colW := 14
	for _, p := range participants {
		if w := len(p) + 4; w > colW {
			colW = w
		}
	}
	if colW%2 != 0 {
		colW++
	}

	colIdx := make(map[string]int)
	for i, p := range participants {
		colIdx[p] = i
	}
	n := len(participants)
	totalW := n * colW
	if maxW > 0 && totalW > maxW {
		totalW = maxW
	}

	centerOf := func(i int) int { return i*colW + colW/2 }

	var sb strings.Builder

	// Participant header row
	for _, p := range participants {
		name := p
		if len(name) > colW-2 {
			name = name[:colW-2]
		}
		pad := (colW - len(name)) / 2
		sb.WriteString(strings.Repeat(" ", pad))
		sb.WriteString(name)
		sb.WriteString(strings.Repeat(" ", colW-pad-len(name)))
	}
	sb.WriteByte('\n')

	// Lifeline header
	lifeline := func() []rune {
		row := make([]rune, totalW)
		for i := range row {
			row[i] = ' '
		}
		for i := range participants {
			if cx := centerOf(i); cx < totalW {
				row[cx] = '│'
			}
		}
		return row
	}
	sb.WriteString(strings.TrimRight(string(lifeline()), " ") + "\n")

	lineChar := func(dashed bool) rune {
		if dashed {
			return '╌'
		}
		return '─'
	}

	for _, msg := range msgs {
		fi, ok1 := colIdx[msg.from]
		ti, ok2 := colIdx[msg.to]
		if !ok1 || !ok2 {
			continue
		}

		row := lifeline()
		fcx := centerOf(fi)
		tcx := centerOf(ti)

		if fi == ti {
			// Self-arrow
			lx := fcx + 1
			label := "↩"
			if msg.label != "" {
				label += " " + msg.label
			}
			for i, r := range []rune(label) {
				if lx+i < totalW {
					row[lx+i] = r
				}
			}
		} else {
			goRight := fi < ti
			lx, rx := fcx, tcx
			if !goRight {
				lx, rx = tcx, fcx
			}
			lc := lineChar(msg.dashed)
			for x := lx + 1; x < rx; x++ {
				if x < totalW {
					row[x] = lc
				}
			}
			if goRight {
				if rx < totalW {
					row[rx] = '►'
				}
			} else {
				if lx < totalW {
					row[lx] = '◄'
				}
			}
			// Place label centred on the arrow
			if msg.label != "" {
				label := " " + msg.label + " "
				lrunes := []rune(label)
				lw := len(lrunes)
				mid := lx + (rx-lx-lw)/2 + 1
				if mid < lx+1 {
					mid = lx + 1
				}
				for i, r := range lrunes {
					if mid+i > lx && mid+i < rx && mid+i < totalW {
						row[mid+i] = r
					}
				}
			}
		}

		sb.WriteString(strings.TrimRight(string(row), " ") + "\n")
		sb.WriteString(strings.TrimRight(string(lifeline()), " ") + "\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

// ── Mermaid graph parsing ─────────────────────────────────────────────────────

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

	// Fixed iteration order so parsing is deterministic regardless of map randomisation.
	for _, pair := range [3][2]byte{{'[', ']'}, {'(', ')'}, {'{', '}'}} {
		open, close := pair[0], pair[1]
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

var mermaidTextReplacer = strings.NewReplacer("\"", "", "'", "", "|", " ", "`", "")

func cleanMermaidText(in string) string {
	in = strings.TrimSpace(in)
	in = mermaidTextReplacer.Replace(in)
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
