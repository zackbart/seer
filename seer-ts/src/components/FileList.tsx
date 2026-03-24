import React from "react";
import { Box, Text } from "ink";
import path from "path";
import { AppState } from "../types.js";
import { categorise, fileIconExt, entryColor, entryBold, symlinkIcon, colors } from "../theme.js";
import { humanSize } from "../utils/humanSize.js";

interface Props {
  state: AppState;
  width: number;
  height: number;
}

function visibleWindow(selected: number, total: number, height: number): [number, number] {
  if (total <= height) return [0, total];
  const half = Math.floor(height / 2);
  let start = selected - half;
  if (start < 0) start = 0;
  let end = start + height;
  if (end > total) {
    end = total;
    start = Math.max(0, end - height);
  }
  return [start, end];
}

export function FileList({ state, width, height }: Props) {
  const innerH = Math.max(3, height - 2); // border takes 2
  const sizeW = 8;
  const hasGit = state.gitStatus !== null;

  // We render exactly `innerH` row slots, always. Each slot has a stable key.
  // This prevents React from churning the DOM tree.
  const rows: React.ReactNode[] = [];
  let slot = 0;

  // Row 0: title
  rows.push(
    <Box key={`slot-${slot++}`}>
      <Text color={colors.accent}>◈</Text>
      <Text color={colors.title} bold> Explorer</Text>
      {state.multiSelected.size > 0 && (
        <Text color={colors.success}> {state.multiSelected.size}sel</Text>
      )}
      <Box flexGrow={1} />
      <Text color={colors.dim}>{state.entries.length}</Text>
    </Box>,
  );

  // Row 1: divider
  rows.push(
    <Box key={`slot-${slot++}`}>
      <Text color={colors.dim}>{"─".repeat(Math.max(1, width - 2))}</Text>
    </Box>,
  );

  if (state.entries.length === 0) {
    rows.push(
      <Box key={`slot-${slot++}`}>
        <Text color={colors.dim}>   no files</Text>
      </Box>,
    );
  } else {
    const listH = Math.max(1, innerH - 2);
    let [start, end] = visibleWindow(state.selected, state.entries.length, listH);
    let needTop = start > 0;
    let needBot = end < state.entries.length;

    for (let iter = 0; iter < 3; iter++) {
      let capacity = listH;
      if (needTop) capacity--;
      if (needBot) capacity--;
      if (capacity < 1) capacity = 1;
      [start, end] = visibleWindow(state.selected, state.entries.length, capacity);
      const nt = start > 0;
      const nb = end < state.entries.length;
      if (nt === needTop && nb === needBot) break;
      needTop = nt;
      needBot = nb;
    }

    if (needTop) {
      rows.push(
        <Box key={`slot-${slot++}`}>
          <Text color={colors.scrollbar} dimColor>  ↑ {start} more</Text>
        </Box>,
      );
    }

    for (let i = start; i < end; i++) {
      const e = state.entries[i];
      const cat = categorise(e);
      const icon = e.isSymlink ? symlinkIcon() : fileIconExt(cat, path.extname(e.name));
      const clr = entryColor(e);
      const bold = entryBold(e);
      const isSel = i === state.selected;
      const isMulti = state.multiSelected.has(e.path);

      let displayName = e.name;
      if (e.isDir) displayName += "/";
      if (e.isSymlink && !e.isDir) displayName += " →";

      const sizeStr = e.isDir ? "" : humanSize(e.size);

      // Git badge
      let gitBadge: React.ReactNode = null;
      if (hasGit && state.gitStatus) {
        const code = state.gitStatus.get(e.name) ?? "";
        if (code.includes("M")) gitBadge = <Text color="#e0af68" bold> M</Text>;
        else if (code.includes("A")) gitBadge = <Text color="#9ece6a" bold> A</Text>;
        else if (code.includes("D")) gitBadge = <Text color="#f7768e" bold> D</Text>;
        else if (code.includes("R")) gitBadge = <Text color="#7aa2f7" bold> R</Text>;
        else if (code.includes("?")) gitBadge = <Text color={colors.muted}> ?</Text>;
      }

      if (isSel) {
        const markerClr = isMulti ? colors.success : colors.accent;
        rows.push(
          <Box key={`slot-${slot++}`}>
            <Text color={markerClr}>┃</Text>
            <Text color={colors.accentFg} bold backgroundColor={colors.surfaceElevated}>
              {" "}{icon}{displayName}
            </Text>
            <Box flexGrow={1}>
              <Text backgroundColor={colors.surfaceElevated}> </Text>
            </Box>
            {gitBadge ? <Text backgroundColor={colors.surfaceElevated}>{gitBadge}</Text> : null}
            <Text color={colors.muted} backgroundColor={colors.surfaceElevated}>
              {sizeStr.padStart(sizeW)}{" "}
            </Text>
          </Box>,
        );
      } else {
        const prefix = isMulti ? "◆" : " ";
        const prefixClr = isMulti ? colors.success : undefined;
        rows.push(
          <Box key={`slot-${slot++}`}>
            <Text color={prefixClr}>{prefix}</Text>
            <Text color={clr} bold={bold}>{icon}{displayName}</Text>
            <Box flexGrow={1} />
            {gitBadge}
            <Text color={colors.dim}>{sizeStr.padStart(sizeW)}</Text>
          </Box>,
        );
      }
    }

    if (needBot) {
      rows.push(
        <Box key={`slot-${slot++}`}>
          <Text color={colors.scrollbar} dimColor>  ↓ {state.entries.length - end} more</Text>
        </Box>,
      );
    }
  }

  // Pad remaining slots with empty rows for stable height
  while (slot < innerH) {
    rows.push(<Box key={`slot-${slot++}`}><Text> </Text></Box>);
  }

  return (
    <Box
      flexDirection="column"
      width={width}
      height={height}
      borderStyle="round"
      borderColor={state.focusedPane === "files" ? colors.borderStrong : colors.border}
      overflow="hidden"
    >
      {rows}
    </Box>
  );
}
