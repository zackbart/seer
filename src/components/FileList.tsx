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

// ── git badge helpers ─────────────────────────────────────────────────────────

interface GitInfo {
  code: string;
  color: string;
  bold: boolean;
}

function getGitInfo(name: string, gitStatus: Map<string, string> | null): GitInfo | null {
  if (!gitStatus) return null;
  const code = gitStatus.get(name) ?? "";
  if (code.includes("M")) return { code: "M", color: "#e0af68", bold: true };
  if (code.includes("A")) return { code: "A", color: "#9ece6a", bold: true };
  if (code.includes("D")) return { code: "D", color: "#f7768e", bold: true };
  if (code.includes("R")) return { code: "R", color: "#7aa2f7", bold: true };
  if (code.includes("?")) return { code: "?", color: colors.muted, bold: false };
  return null;
}

// ── component ─────────────────────────────────────────────────────────────────

export function FileList({ state, width, height }: Props) {
  const innerH = Math.max(3, height - 2); // border takes 2 rows
  const innerW = Math.max(10, width - 2); // border takes 2 cols
  const sizeW = 8;
  const hasGit = state.gitStatus !== null;
  const iconW = 2; // all icons render as 2 display columns

  // Layout budget per row (inside border):
  //   prefix(2) + icon(2) + name(variable) + git(3 if repo) + size(8) + rightPad(1)
  const fixedChars = 2 + iconW + (hasGit ? 3 : 0) + sizeW + 1;
  const maxNameLen = Math.max(1, innerW - fixedChars);

  // We render exactly `innerH` row slots, always. Each slot has a stable key.
  // This prevents React from churning the DOM tree.
  const rows: React.ReactNode[] = [];
  let slot = 0;

  // Row 0: title
  rows.push(
    <Box key={`slot-${slot++}`}>
      <Text color={colors.accent}>◈</Text>
      <Text color={colors.title} bold> Explorer</Text>
      <Box flexGrow={1} />
      <Text color={colors.dim}>{state.entries.length}</Text>
    </Box>,
  );

  // Row 1: divider
  rows.push(
    <Box key={`slot-${slot++}`}>
      <Text color={colors.dim}>{"─".repeat(Math.max(1, innerW))}</Text>
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

      let displayName = e.name;
      if (e.isDir) displayName += "/";
      if (e.isSymlink && !e.isDir) displayName += " →";

      // Truncate name to fit
      let truncName = displayName;
      if (truncName.length > maxNameLen) {
        truncName = truncName.slice(0, maxNameLen - 1) + "…";
      }

      const sizeStr = e.isDir ? "" : humanSize(e.size);
      const git = getGitInfo(e.name, state.gitStatus);

      // Right-side content: git badge + size + padding
      const rightLen = (hasGit && git ? 2 : (hasGit ? 2 : 0)) + sizeW + 1;
      // Left-side content: prefix(2) + icon(iconW) + name
      const nameLen = truncName.length;
      // Fill gap between name and right side
      const fillLen = Math.max(0, innerW - 2 - iconW - nameLen - rightLen);

      if (isSel) {
        // Selected: single Text wrapper ensures continuous background
        rows.push(
          <Box key={`slot-${slot++}`}>
            <Text color={colors.accent}>┃</Text>
            <Text backgroundColor={colors.surfaceElevated}>
              <Text color={colors.accentFg} bold>
                {" "}{icon}{truncName}
              </Text>
              {" ".repeat(fillLen)}
              {hasGit && git ? <Text color={git.color} bold={git.bold}> {git.code}</Text> : (hasGit ? "  " : "")}
              <Text color={colors.muted}>
                {sizeStr.padStart(sizeW)}{" "}
              </Text>
            </Text>
          </Box>,
        );
      } else {
        rows.push(
          <Box key={`slot-${slot++}`}>
            <Text>
              <Text color={colors.dim}> </Text>
              <Text color={clr} bold={bold}>{" "}{icon}{truncName}</Text>
              {" ".repeat(fillLen)}
              {hasGit && git ? <Text color={git.color} bold={git.bold}> {git.code}</Text> : (hasGit ? "  " : "")}
              <Text color={colors.dim}>{sizeStr.padStart(sizeW)}{" "}</Text>
            </Text>
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
      borderColor={colors.borderStrong}
      overflow="hidden"
    >
      {rows}
    </Box>
  );
}
