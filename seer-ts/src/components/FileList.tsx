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
  const innerH = Math.max(3, height - 2);
  const sizeW = 9;
  const hasGit = state.gitStatus !== null;

  // Header
  const lines: React.ReactNode[] = [];

  // Title row
  lines.push(
    <Box key="title" width={width}>
      <Text bold color={colors.title}>Explorer</Text>
      {state.multiSelected.size > 0 && (
        <Text color={colors.muted}> · {state.multiSelected.size} sel</Text>
      )}
      <Box flexGrow={1} />
      <Text color={colors.muted}>{state.entries.length}</Text>
    </Box>,
  );

  // Divider
  lines.push(
    <Box key="div" width={width}>
      <Text color={colors.dim}>{"─".repeat(Math.max(1, width))}</Text>
    </Box>,
  );

  if (state.entries.length === 0) {
    lines.push(
      <Box key="empty">
        <Text color={colors.muted}>  (empty directory)</Text>
      </Box>,
    );
  } else {
    const listH = Math.max(1, innerH - 2);

    // Calculate visible window with scroll indicators
    let [start, end] = visibleWindow(state.selected, state.entries.length, listH);
    let needTop = start > 0;
    let needBot = end < state.entries.length;

    // Recalculate with scroll indicators taking space
    for (let iter = 0; iter < 3; iter++) {
      let capacity = listH;
      if (needTop) capacity--;
      if (needBot) capacity--;
      if (capacity < 1) capacity = 1;
      [start, end] = visibleWindow(state.selected, state.entries.length, capacity);
      const newTop = start > 0;
      const newBot = end < state.entries.length;
      if (newTop === needTop && newBot === needBot) break;
      needTop = newTop;
      needBot = newBot;
    }

    if (needTop) {
      lines.push(
        <Box key="scroll-top">
          <Text color={colors.scrollbar}>  ↑ {start} more</Text>
        </Box>,
      );
    }

    for (let i = start; i < end; i++) {
      const e = state.entries[i];
      const cat = categorise(e);
      const icon = e.isSymlink ? symlinkIcon() : fileIconExt(cat, path.extname(e.name));
      const color = entryColor(e);
      const bold = entryBold(e);
      const isSelected = i === state.selected;
      const isMultiSel = state.multiSelected.has(e.path);

      let displayName = e.name;
      if (e.isDir) displayName += "/";
      if (e.isSymlink && !e.isDir) displayName += " →";

      const sizeStr = e.isDir ? "" : humanSize(e.size);

      // Git badge
      let gitBadge = "";
      let gitColor: string = colors.muted;
      if (hasGit && state.gitStatus) {
        const code = state.gitStatus.get(e.name) ?? "";
        if (code.includes("M")) { gitBadge = "M"; gitColor = "221"; }
        else if (code.includes("A")) { gitBadge = "A"; gitColor = "114"; }
        else if (code.includes("D")) { gitBadge = "D"; gitColor = "203"; }
        else if (code.includes("R")) { gitBadge = "R"; gitColor = "111"; }
        else if (code.includes("?")) { gitBadge = "?"; gitColor = "244"; }
      }

      if (isSelected) {
        const marker = isMultiSel ? "▌" : "▌";
        const markerColor = isMultiSel ? colors.exec : colors.accent;
        lines.push(
          <Box key={`entry-${i}`} width={width}>
            <Text color={markerColor}>{marker}</Text>
            <Text color={colors.accentFg} bold backgroundColor={colors.surfaceElevated}>
              {icon}{displayName}
            </Text>
            <Box flexGrow={1}>
              <Text backgroundColor={colors.surfaceElevated}> </Text>
            </Box>
            {gitBadge ? <Text color={gitColor} bold backgroundColor={colors.surfaceElevated}>{gitBadge} </Text> : null}
            <Text color={colors.size} backgroundColor={colors.surfaceElevated}>
              {sizeStr.padStart(sizeW)}
            </Text>
          </Box>,
        );
      } else {
        const prefix = isMultiSel ? "◆" : " ";
        const prefixColor = isMultiSel ? colors.exec : undefined;
        lines.push(
          <Box key={`entry-${i}`} width={width}>
            <Text color={prefixColor}>{prefix}</Text>
            <Text color={color} bold={bold}>{icon}{displayName}</Text>
            <Box flexGrow={1} />
            {gitBadge ? <Text color={gitColor} bold>{gitBadge} </Text> : null}
            <Text color={colors.size}>{sizeStr.padStart(sizeW)}</Text>
          </Box>,
        );
      }
    }

    if (needBot) {
      lines.push(
        <Box key="scroll-bot">
          <Text color={colors.scrollbar}>  ↓ {state.entries.length - end} more</Text>
        </Box>,
      );
    }
  }

  return (
    <Box flexDirection="column" width={width} height={height}>
      {lines}
    </Box>
  );
}
