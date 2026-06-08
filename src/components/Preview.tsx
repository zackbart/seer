import React from "react";
import { Box, Text } from "ink";
import path from "path";
import { AppState } from "../types.js";
import { categorise, fileIconExt, entryColor, symlinkIcon, colors } from "../theme.js";
import { humanSize } from "../utils/humanSize.js";
import {
  ansiSlice,
  computeWrappedBodyFromLines,
  visualWidth,
  wrapAnsiText,
} from "../utils/ansiText.js";
import { layoutDimensions } from "../utils/layout.js";

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

  // Out-of-band Kitty placeholder emit. Ink's ansi-tokenize splits every
  // codepoint into its own cell, so placeholder + diacritics don't survive
  // the text pipeline. Write the grid directly to stdout after each commit,
  // positioned into the preview body via CUP. Ink paints blanks in the
  // image region on the next frame; this effect re-emits after Ink's 32ms
  // throttled flush settles, restoring the placeholders over the blanks.
  React.useEffect(() => {
    const img = state.previewImage;
    if (!img) return;
    const { leftW } = layoutDimensions(state.width, state.height, state.paneOffset);
    // Preview body top-left in 1-based CUP coordinates:
    //   row = topbar(1) + paneBorder(1) + header(1) + divider(1) + 1 = 5
    //   col = leftW + sep(1) + paneBorder(1) + 1 = leftW + 3
    const topRow = 5;
    const leftCol = leftW + 3;
    const bodyRows = Math.max(0, height - 4);
    const rowsToEmit = Math.min(img.gridRows.length, bodyRows);
    const emit = () => {
      let out = "";
      for (let r = 0; r < rowsToEmit; r++) {
        out += `\x1b[${topRow + r};${leftCol}H${img.gridRows[r]}`;
      }
      // Park cursor at bottom-left so it doesn't sit in the middle of the
      // image region (some terminals show a cursor blink overlay).
      out += `\x1b[${state.height};1H`;
      try { process.stdout.write(out); } catch {}
    };
    // Ink throttles stdout at 32ms with a trailing-edge flush that overwrites
    // our placeholders after React's commit. Aggressive schedule: several
    // emits in the first 100ms to catch Ink's flush within one throttle
    // window, plus safety emits at 200/400ms for coalesced renders.
    emit();
    const timers = [15, 40, 80, 150, 250, 500].map((ms) => setTimeout(emit, ms));
    return () => { for (const t of timers) clearTimeout(t); };
  });

  // ── header row ─────────────────────────────────────────────────────
  if (state.entries.length > 0 && state.selected < state.entries.length) {
    const e = state.entries[state.selected];
    const cat = categorise(e);
    const icon = e.isSymlink ? symlinkIcon() : fileIconExt(cat, path.extname(e.name));
    const clr = entryColor(e);

    let suffix = "";
    if (e.isDir) suffix = "/";
    const symlinkSuffix = e.isSymlink && e.symlinkTarget ? ` → ${e.symlinkTarget}` : "";

    // Build metadata pieces. For files with a computed line count we show
    // a richer breakdown (size · lines · tokens); directories and opaque
    // previews (hex/archive/image) fall back to just the date.
    const meta = buildHeaderMeta(state, e, innerW, icon, suffix, symlinkSuffix);

    rows.push(
      <Box key={`slot-${slot++}`}>
        <Text color={clr} bold>  {icon}{e.name}{suffix}</Text>
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

  // Compute the wrapped text body unconditionally. For image previews the
  // result is unused but the hook call must stay — React's rules-of-hooks
  // requires a consistent call order across renders.
  let previewBody = state.preview;
  if (!previewBody && !state.loading) previewBody = "  no preview";
  if (state.loading && !previewBody) previewBody = "  loading…";

  const wrappedLines = React.useMemo(
    () => wrapAnsiText(previewBody, innerW),
    [previewBody, innerW],
  );
  const body = React.useMemo(
    () => computeWrappedBodyFromLines(wrappedLines, bodyH, state.previewOffset),
    [wrappedLines, bodyH, state.previewOffset],
  );

  // Kitty-placeholder image path: reserve blank space in Ink's layout; the
  // actual placeholder grid is written out-of-band by the useEffect above.
  // No scroll indicators, no selection.
  if (state.previewImage) {
    for (let i = 0; i < bodyH; i++) {
      rows.push(
        <Box key={`slot-${slot++}`} width={innerW} height={1} />,
      );
    }
    while (slot < innerH) {
      rows.push(<Box key={`slot-${slot++}`} width={innerW} height={1} />);
    }
    return (
      <Box
        flexDirection="column"
        width={width}
        height={height}
        borderStyle="round"
        borderColor={colors.border}
        overflow="hidden"
      >
        {rows}
      </Box>
    );
  }

  // Top scroll indicator — mirrors FileList's vocabulary ("N more" rather
  // than a line number, which after wrapping would refer to wrapped rows,
  // not source lines).
  if (body.scrollRow > 0) {
    rows.push(
      <Box key={`slot-${slot++}`}>
        <Text color={colors.scrollbar}>  ↑ {body.offset} more</Text>
      </Box>,
    );
  }

  // Normalize selection coordinates
  const selecting = state.previewSelecting;
  let sel: { start: { x: number; y: number }; end: { x: number; y: number } } | null = null;
  if (selecting || (state.previewSelStart.x !== state.previewSelEnd.x || state.previewSelStart.y !== state.previewSelEnd.y)) {
    let s0 = state.previewSelStart;
    let s1 = state.previewSelEnd;
    if (s0.y > s1.y || (s0.y === s1.y && s0.x > s1.x)) {
      [s0, s1] = [s1, s0];
    }
    if (selecting) {
      sel = { start: s0, end: s1 };
    }
  }

  // Render each line individually for selection highlighting. Lines are
  // already wrapped to ≤ innerW by computeWrappedBody, so truncate-end only
  // fires as a safety net for wide-char edge cases.
  for (let i = 0; i < body.contentH; i++) {
    const line = i < body.visibleLines.length ? body.visibleLines[i] : "";
    // The row index in selection coordinates accounts for scroll indicator
    const selRow = body.scrollRow + i;

    if (sel && selRow >= sel.start.y && selRow <= sel.end.y) {
      const colStart = selRow === sel.start.y ? sel.start.x : 0;
      const colEnd = selRow === sel.end.y ? sel.end.x : innerW;

      const before = ansiSlice(line, 0, colStart);
      const selectedText = stripAnsiPlain(ansiSlice(line, colStart, colEnd));
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

  // Bottom scroll indicator — shown when there's wrapped content below the
  // visible window. Mirrors FileList's `↓ N more` pattern.
  if (body.scrollRowBot > 0) {
    const hidden = Math.max(0, body.wrappedLines.length - body.offset - body.contentH);
    rows.push(
      <Box key={`slot-${slot++}`}>
        <Text color={colors.scrollbar}>  ↓ {hidden} more</Text>
      </Box>,
    );
  }

  while (slot < innerH) {
    rows.push(
      <Box key={`slot-${slot++}`} width={innerW} height={1} overflow="hidden">
        <Text> </Text>
      </Box>,
    );
  }

  return (
    <Box
      flexDirection="column"
      width={width}
      height={height}
      borderStyle="round"
      borderColor={colors.border}
      overflow="hidden"
    >
      {rows}
    </Box>
  );
}

// ── header metadata ──────────────────────────────────────────────────────────

function buildHeaderMeta(
  state: AppState,
  e: import("../types.js").Entry,
  innerW: number,
  icon: string,
  suffix: string,
  symlinkSuffix: string,
): React.ReactNode {
  if (state.loading) {
    return <Text color={colors.loading}>loading…</Text>;
  }
  if (e.isDir) {
    return <Text color={colors.dim}>{formatDate(e.modTime)}</Text>;
  }

  const date = formatDate(e.modTime);

  // Metric-less preview (hex/archive/image): fall back to size + date only.
  if (state.previewLineCount <= 0) {
    return (
      <Text color={colors.muted}>{humanSize(e.size)}  {date}</Text>
    );
  }

  // Budget check: how much room is left after the filename on the left and
  // the date on the right? Drop metrics from least important to most.
  const nameWidth = visualWidth(` ${icon}${e.name}${suffix}${symlinkSuffix}`);
  const dateWidth = visualWidth(date);
  const trailingSpace = 1;
  const spacerMin = 2;
  // Reserve a gap between metrics and date.
  const gap = 2;
  const budget = innerW - nameWidth - dateWidth - trailingSpace - spacerMin - gap;

  const size = humanSize(e.size);
  const linesText = formatLines(state.previewLineCount, state.previewTruncated);
  const tokensText = formatTokens(state.previewTokenEstimate, state.previewTruncated);

  const sep = " · ";
  const fullChunks: string[] = [size, linesText, tokensText];
  let chunks = fullChunks.slice();

  // Drop from the right (tokens, then lines) if we can't fit.
  const measure = (parts: string[]) => visualWidth(parts.join(sep));
  while (chunks.length > 1 && measure(chunks) > budget) {
    chunks.pop();
  }
  // If even size alone doesn't fit, just show size + date.
  if (measure(chunks) > budget) {
    chunks = [size];
  }

  return (
    <Text color={colors.muted}>{chunks.join(sep)}  {date}</Text>
  );
}

function formatLines(n: number, truncated: boolean): string {
  const suffix = truncated ? "+" : "";
  if (n < 1000) return `${n}${suffix} lines`;
  if (n < 10000) return `${(n / 1000).toFixed(1)}k${suffix} lines`;
  return `${Math.round(n / 1000)}k${suffix} lines`;
}

function formatTokens(n: number, truncated: boolean): string {
  const suffix = truncated ? "+" : "";
  if (n < 1000) return `~${n}${suffix} tok`;
  if (n < 10000) return `~${(n / 1000).toFixed(1)}k${suffix} tok`;
  if (n < 1_000_000) return `~${Math.round(n / 1000)}k${suffix} tok`;
  return `~${(n / 1_000_000).toFixed(1)}M${suffix} tok`;
}

function formatDate(d: Date): string {
  const month = d.toLocaleDateString("en-US", { month: "short" });
  const day = d.getDate().toString().padStart(2, "0");
  const h = d.getHours().toString().padStart(2, "0");
  const m = d.getMinutes().toString().padStart(2, "0");
  return `${month} ${day} ${h}:${m}`;
}

// Strip ANSI from a selected-text slice for the highlight overlay. Kept
// local (not using the util's stripAnsi) to avoid accidentally widening
// the background highlight when the original line had color codes.
function stripAnsiPlain(s: string): string {
  return s.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, "");
}
