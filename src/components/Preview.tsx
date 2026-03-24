import React from "react";
import { Box, Text } from "ink";
import path from "path";
import { AppState } from "../types.js";
import { categorise, fileIconExt, entryColor, symlinkIcon, colors } from "../theme.js";
import { humanSize } from "../utils/humanSize.js";

interface Props {
  state: AppState;
  width: number;
  height: number;
}

export function Preview({ state, width, height }: Props) {
  const innerH = Math.max(3, height - 2);
  const innerW = Math.max(1, width - 2);
  const rows: React.ReactNode[] = [];
  let slot = 0;

  // ── header row ─────────────────────────────────────────────────────
  if (state.entries.length > 0 && state.selected < state.entries.length) {
    const e = state.entries[state.selected];
    const cat = categorise(e);
    const icon = e.isSymlink ? symlinkIcon() : fileIconExt(cat, path.extname(e.name));
    const clr = entryColor(e);

    let suffix = "";
    if (e.isDir) suffix = "/";
    const symlinkSuffix = e.isSymlink && e.symlinkTarget ? ` → ${e.symlinkTarget}` : "";

    let meta: React.ReactNode;
    if (state.loading) {
      meta = <Text color={colors.loading}>loading…</Text>;
    } else if (!e.isDir) {
      meta = <Text color={colors.muted}>{humanSize(e.size)}  {formatDate(e.modTime)}</Text>;
    } else {
      meta = <Text color={colors.dim}>{formatDate(e.modTime)}</Text>;
    }

    rows.push(
      <Box key={`slot-${slot++}`}>
        <Text color={clr} bold> {icon}{e.name}{suffix}</Text>
        {symlinkSuffix && <Text color={colors.dim}>{symlinkSuffix}</Text>}
        <Box flexGrow={1} />
        {meta}
        <Text> </Text>
      </Box>,
    );
  } else {
    rows.push(
      <Box key={`slot-${slot++}`}>
        <Text color={colors.dim}> no selection</Text>
      </Box>,
    );
  }

  // ── divider ────────────────────────────────────────────────────────
  rows.push(
    <Box key={`slot-${slot++}`}>
      <Text color={colors.dim}>{"─".repeat(Math.max(1, innerW))}</Text>
    </Box>,
  );

  // ── body ───────────────────────────────────────────────────────────
  const bodyH = Math.max(1, innerH - 2);
  let previewBody = state.preview;
  if (!previewBody && !state.loading) previewBody = "  no preview";
  if (state.loading && !previewBody) previewBody = "  loading…";

  const allLines = previewBody.split("\n");
  const maxOffset = Math.max(0, allLines.length - bodyH);
  const offset = Math.min(state.previewOffset, maxOffset);

  // Scroll indicator row
  let scrollRow = 0; // how many rows the scroll indicator takes
  if (offset > 0) {
    rows.push(
      <Box key={`slot-${slot++}`}>
        <Text color={colors.scrollbar}>  ↑ line {offset + 1}</Text>
      </Box>,
    );
    scrollRow = 1;
  }

  const contentH = bodyH - scrollRow;
  const visibleLines = allLines.slice(offset, offset + contentH);

  // Normalize selection coordinates
  const selecting = state.previewSelecting;
  let sel: { start: { x: number; y: number }; end: { x: number; y: number } } | null = null;
  if (selecting || (state.previewSelStart.x !== state.previewSelEnd.x || state.previewSelStart.y !== state.previewSelEnd.y)) {
    let s0 = state.previewSelStart;
    let s1 = state.previewSelEnd;
    if (s0.y > s1.y || (s0.y === s1.y && s0.x > s1.x)) {
      [s0, s1] = [s1, s0];
    }
    // Only show highlight while actively selecting
    if (selecting) {
      sel = { start: s0, end: s1 };
    }
  }

  // Render each line individually for selection highlighting
  for (let i = 0; i < contentH; i++) {
    const line = i < visibleLines.length ? visibleLines[i] : "";
    // The row index in selection coordinates accounts for scroll indicator
    const selRow = scrollRow + i;

    if (sel && selRow >= sel.start.y && selRow <= sel.end.y) {
      // This line intersects the selection
      const colStart = selRow === sel.start.y ? sel.start.x : 0;
      const colEnd = selRow === sel.end.y ? sel.end.x : innerW;

      const before = ansiSlice(line, 0, colStart);
      const selectedText = stripAnsi(ansiSlice(line, colStart, colEnd));
      const after = ansiSlice(line, colEnd, innerW);

      rows.push(
        <Box key={`slot-${slot++}`} width={innerW} height={1} overflow="hidden">
          {before ? <Text wrap="truncate-end">{before}</Text> : null}
          <Text wrap="truncate-end" backgroundColor={colors.accentDim} color={colors.accentFg}>{selectedText || " "}</Text>
          {after ? <Text wrap="truncate-end">{after}</Text> : null}
        </Box>,
      );
    } else {
      rows.push(
        <Box key={`slot-${slot++}`} width={innerW} height={1} overflow="hidden">
          <Text wrap="truncate-end">{line || " "}</Text>
        </Box>,
      );
    }
  }

  return (
    <Box
      flexDirection="column"
      width={width}
      height={height}
      borderStyle="round"
      borderColor={colors.border}
    >
      {rows}
    </Box>
  );
}

function formatDate(d: Date): string {
  const month = d.toLocaleDateString("en-US", { month: "short" });
  const day = d.getDate().toString().padStart(2, "0");
  const h = d.getHours().toString().padStart(2, "0");
  const m = d.getMinutes().toString().padStart(2, "0");
  return `${month} ${day} ${h}:${m}`;
}

// ── ANSI-aware string slicing ────────────────────────────────────────────────

// Strip ANSI escape codes from a string.
function stripAnsi(s: string): string {
  return s.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, "");
}

// Slice a string containing ANSI escape codes by visible column range [startCol, endCol).
// Preserves ANSI color state across the slice boundary.
function ansiSlice(str: string, startCol: number, endCol: number): string {
  if (startCol >= endCol) return "";
  if (!str) return "";

  // Fast path: no ANSI codes at all
  if (!str.includes("\x1b[")) {
    return str.slice(startCol, endCol);
  }

  // Walk the string, tracking visible column and ANSI state
  const ansiRe = /\x1b\[[0-9;]*[a-zA-Z]/g;
  let result = "";
  let col = 0;
  let i = 0;
  let pendingAnsi = ""; // accumulated ANSI state before our range
  let emitting = false;

  while (i < str.length && col < endCol) {
    // Try to match an ANSI escape at current position
    ansiRe.lastIndex = i;
    const match = ansiRe.exec(str);

    if (match && match.index === i) {
      const seq = match[0];
      if (col >= startCol) {
        // In range — emit directly
        result += seq;
      } else {
        // Before range — track state for later
        if (seq === "\x1b[0m") {
          pendingAnsi = "";
        } else {
          pendingAnsi = seq;
        }
      }
      i += seq.length;
      continue;
    }

    // Visible character
    if (col >= startCol) {
      if (!emitting) {
        // First char in range — prepend any pending ANSI state
        if (pendingAnsi) result += pendingAnsi;
        emitting = true;
      }
      result += str[i];
    }
    col++;
    i++;
  }

  // Append reset if we emitted any ANSI codes
  if (result && (pendingAnsi || result.includes("\x1b["))) {
    result += "\x1b[0m";
  }

  return result;
}
