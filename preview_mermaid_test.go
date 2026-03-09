package main

import (
	"strings"
	"testing"
)

// ── cleanMermaidText / cleanMermaidID ─────────────────────────────────────────

func TestCleanMermaidText(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"hello", "hello"},
		{`"hello"`, "hello"},
		{"hello world", "hello world"},
		{"  spaces  ", "spaces"},
		{`foo|bar`, "foo bar"},
		{"a  b  c", "a b c"}, // collapsed whitespace
		{"`code`", "code"},
		{"", ""},
	}
	for _, tc := range tests {
		got := cleanMermaidText(tc.in)
		if got != tc.want {
			t.Errorf("cleanMermaidText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCleanMermaidID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"node1", "node1"},
		{"[node1]", "node1"},
		{"(node1)", "node1"},
		{"{node1}", "node1"},
		{`"node1"`, "node1"},
		{"node1;", "node1"},
		{"node1 extra", "node1"}, // first field only
		{"", ""},
	}
	for _, tc := range tests {
		got := cleanMermaidID(tc.in)
		if got != tc.want {
			t.Errorf("cleanMermaidID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── fitMermaidLabel ───────────────────────────────────────────────────────────

func TestFitMermaidLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "node"},
		{"short", "short"},
		{"exactly twenty four!", "exactly twenty four!"},        // 20 chars < 24
		{"this is a very long label that exceeds limit", "this is a very long lab~"}, // truncated at byte 23
	}
	for _, tc := range tests {
		got := fitMermaidLabel(tc.in)
		if got != tc.want {
			t.Errorf("fitMermaidLabel(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── escapeMarkdownTableCell ───────────────────────────────────────────────────

func TestEscapeMarkdownTableCell(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"hello", "hello"},
		{"foo|bar", `foo\|bar`},
		{"  spaces  ", "spaces"},
		{"a|b|c", `a\|b\|c`},
		{"", ""},
	}
	for _, tc := range tests {
		got := escapeMarkdownTableCell(tc.in)
		if got != tc.want {
			t.Errorf("escapeMarkdownTableCell(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── mermaidChartType ──────────────────────────────────────────────────────────

func TestMermaidChartType(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"graph TD\nA --> B", "graph"},
		{"flowchart LR\nA --> B", "flowchart"},
		{"sequenceDiagram\nA->>B: hello", "sequenceDiagram"},
		{"%% comment\ngraph TD", "graph"},
		{"", "diagram"},
		{"%% only comments\n%% more", "diagram"},
	}
	for _, tc := range tests {
		got := mermaidChartType(tc.code)
		if got != tc.want {
			t.Errorf("mermaidChartType(%q) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

// ── parseMermaidNode ──────────────────────────────────────────────────────────

func TestParseMermaidNode(t *testing.T) {
	tests := []struct {
		raw       string
		wantID    string
		wantLabel string
	}{
		{"A", "A", "A"},
		{"A[Label]", "A", "Label"},
		{"A(Label)", "A", "Label"},
		{"A{Label}", "A", "Label"},
		{`A["Quoted"]`, "A", "Quoted"},
		{"", "", ""},
		{"A:::style", "A", "A"},
	}
	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			got := parseMermaidNode(tc.raw)
			if got.id != tc.wantID {
				t.Errorf("parseMermaidNode(%q).id = %q, want %q", tc.raw, got.id, tc.wantID)
			}
			if got.label != tc.wantLabel {
				t.Errorf("parseMermaidNode(%q).label = %q, want %q", tc.raw, got.label, tc.wantLabel)
			}
		})
	}
}

// ── parseMermaidEdge ─────────────────────────────────────────────────────────

func TestParseMermaidEdge(t *testing.T) {
	tests := []struct {
		line      string
		wantFrom  string
		wantTo    string
		wantLabel string
		wantOk    bool
	}{
		{"A --> B", "A", "B", "", true},
		{"A --> B", "A", "B", "", true},
		{"A --> |label| B", "A", "B", "label", true},
		{"A ==> B", "A", "B", "", true},
		{"no arrow here", "", "", "", false},
		{"", "", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			edge, ok := parseMermaidEdge(tc.line)
			if ok != tc.wantOk {
				t.Errorf("parseMermaidEdge(%q) ok=%v, want %v", tc.line, ok, tc.wantOk)
				return
			}
			if !ok {
				return
			}
			if edge.from.id != tc.wantFrom {
				t.Errorf("from.id = %q, want %q", edge.from.id, tc.wantFrom)
			}
			if edge.to.id != tc.wantTo {
				t.Errorf("to.id = %q, want %q", edge.to.id, tc.wantTo)
			}
			if edge.edgeLabel != tc.wantLabel {
				t.Errorf("edgeLabel = %q, want %q", edge.edgeLabel, tc.wantLabel)
			}
		})
	}
}

// ── parseMermaidGraph ─────────────────────────────────────────────────────────

func TestParseMermaidGraph(t *testing.T) {
	code := `graph TD
A[Start] --> B[Process]
B --> C[End]`
	g := parseMermaidGraph(code)

	if len(g.nodeOrder) != 3 {
		t.Errorf("expected 3 nodes, got %d: %v", len(g.nodeOrder), g.nodeOrder)
	}
	if len(g.edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(g.edges))
	}
	if g.nodes["A"] != "Start" {
		t.Errorf("node A label = %q, want %q", g.nodes["A"], "Start")
	}
}

func TestParseMermaidGraphEmpty(t *testing.T) {
	g := parseMermaidGraph("")
	if len(g.nodeOrder) != 0 {
		t.Errorf("empty code should produce no nodes, got %v", g.nodeOrder)
	}
}

func TestParseMermaidGraphSkipsComments(t *testing.T) {
	code := `graph LR
%% this is a comment
A --> B`
	g := parseMermaidGraph(code)
	if len(g.nodeOrder) != 2 {
		t.Errorf("expected 2 nodes after skipping comment, got %d", len(g.nodeOrder))
	}
}

// ── parseSequenceDiagram ──────────────────────────────────────────────────────

func TestParseSequenceDiagram(t *testing.T) {
	code := `sequenceDiagram
participant Alice
participant Bob
Alice->>Bob: Hello
Bob-->>Alice: World`
	parts, msgs := parseSequenceDiagram(code)

	if len(parts) != 2 {
		t.Errorf("expected 2 participants, got %d: %v", len(parts), parts)
	}
	if parts[0] != "Alice" || parts[1] != "Bob" {
		t.Errorf("unexpected participants: %v", parts)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].from != "Alice" || msgs[0].to != "Bob" {
		t.Errorf("first message: from=%q to=%q, want Alice→Bob", msgs[0].from, msgs[0].to)
	}
	if msgs[0].label != "Hello" {
		t.Errorf("first message label = %q, want %q", msgs[0].label, "Hello")
	}
	if !msgs[1].dashed {
		t.Error("-->> should be dashed")
	}
}

func TestParseSequenceDiagramNoDuplicateParticipants(t *testing.T) {
	code := `sequenceDiagram
A->>B: one
A->>B: two`
	parts, _ := parseSequenceDiagram(code)
	if len(parts) != 2 {
		t.Errorf("expected 2 unique participants, got %d: %v", len(parts), parts)
	}
}

// ── replaceMermaidFences ──────────────────────────────────────────────────────

func TestReplaceMermaidFencesPassthrough(t *testing.T) {
	// Non-mermaid fences should pass through unchanged
	md := "```go\nfmt.Println(\"hello\")\n```"
	got := replaceMermaidFences(md)
	if !strings.Contains(got, "fmt.Println") {
		t.Errorf("non-mermaid fence should be preserved, got: %s", got)
	}
}

func TestReplaceMermaidFencesReplaces(t *testing.T) {
	md := "# Title\n\n```mermaid\ngraph TD\nA --> B\n```\n\nAfter"
	got := replaceMermaidFences(md)
	// The mermaid fence should be gone, replaced by rendered art
	if strings.Contains(got, "```mermaid") {
		t.Error("mermaid fence should have been replaced")
	}
	// Surrounding text should be preserved
	if !strings.Contains(got, "# Title") {
		t.Error("text before mermaid fence should be preserved")
	}
	if !strings.Contains(got, "After") {
		t.Error("text after mermaid fence should be preserved")
	}
}

func TestReplaceMermaidFencesUnclosed(t *testing.T) {
	// Unclosed fence should be left as a mermaid block
	md := "```mermaid\ngraph TD\nA --> B"
	got := replaceMermaidFences(md)
	if !strings.Contains(got, "mermaid") {
		t.Error("unclosed mermaid fence should remain in output")
	}
}
