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
      <Text color={colors.dim}>{"─".repeat(Math.max(1, width - 2))}</Text>
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

  if (offset > 0) {
    rows.push(
      <Box key={`slot-${slot++}`}>
        <Text color={colors.scrollbar}>  ↑ line {offset + 1}</Text>
      </Box>,
    );
  }

  const contentH = offset > 0 ? bodyH - 1 : bodyH;
  const visibleLines = allLines.slice(offset, offset + contentH);

  // Render preview body. Strictly limit to contentH lines and truncate
  // each line to prevent overflow that breaks the layout.
  const truncatedLines = visibleLines.slice(0, contentH);
  const bodyText = truncatedLines.join("\n");
  rows.push(
    <Box key={`slot-${slot++}`} width={width - 2} height={contentH} overflow="hidden">
      <Text wrap="truncate-end">{bodyText || " "}</Text>
    </Box>,
  );

  return (
    <Box
      flexDirection="column"
      width={width}
      height={height}
      borderStyle="round"
      borderColor={state.focusedPane === "preview" ? colors.borderStrong : colors.border}
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
