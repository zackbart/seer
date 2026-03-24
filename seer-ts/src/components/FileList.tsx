import { Box, Text } from "ink";
import path from "path";
import chalk from "chalk";
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
  const innerW = Math.max(8, width - 2); // account for border
  const innerH = Math.max(3, height - 2);
  const sizeW = 9;
  const hasGit = state.gitStatus !== null;

  // Build entire file list as a single string to avoid layout thrashing
  // from variable numbers of React nodes.
  const outputLines: string[] = [];

  // Title row
  const titleLeft = chalk.ansi256(Number(colors.title)).bold("Explorer");
  const selCountStr = state.multiSelected.size > 0
    ? chalk.ansi256(Number(colors.muted))(` · ${state.multiSelected.size} sel`)
    : "";
  const countStr = chalk.ansi256(Number(colors.muted))(`${state.entries.length}`);
  outputLines.push(`${titleLeft}${selCountStr}  ${countStr}`);

  // Divider
  outputLines.push(chalk.ansi256(Number(colors.dim))("─".repeat(Math.max(1, innerW))));

  if (state.entries.length === 0) {
    outputLines.push(chalk.ansi256(Number(colors.muted))("  (empty)"));
  } else {
    const listH = Math.max(1, innerH - 2);

    // Calculate visible window with scroll indicators
    let [start, end] = visibleWindow(state.selected, state.entries.length, listH);
    let needTop = start > 0;
    let needBot = end < state.entries.length;

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
      outputLines.push(chalk.ansi256(Number(colors.scrollbar))(`  ↑ ${start} more`));
    }

    for (let i = start; i < end; i++) {
      const e = state.entries[i];
      const cat = categorise(e);
      const icon = e.isSymlink ? symlinkIcon() : fileIconExt(cat, path.extname(e.name));
      const clr = entryColor(e);
      const bold = entryBold(e);
      const isSelected = i === state.selected;
      const isMultiSel = state.multiSelected.has(e.path);

      let displayName = e.name;
      if (e.isDir) displayName += "/";
      if (e.isSymlink && !e.isDir) displayName += " →";

      const sizeStr = e.isDir ? "" : humanSize(e.size);

      // Git badge
      let gitBadge = "";
      if (hasGit && state.gitStatus) {
        const code = state.gitStatus.get(e.name) ?? "";
        if (code.includes("M")) gitBadge = chalk.ansi256(221).bold("M ");
        else if (code.includes("A")) gitBadge = chalk.ansi256(114).bold("A ");
        else if (code.includes("D")) gitBadge = chalk.ansi256(203).bold("D ");
        else if (code.includes("R")) gitBadge = chalk.ansi256(111).bold("R ");
        else if (code.includes("?")) gitBadge = chalk.ansi256(244).bold("? ");
      }

      const nameText = `${icon}${displayName}`;
      const sizeText = sizeStr.padStart(sizeW);

      if (isSelected) {
        const markerColor = isMultiSel ? Number(colors.exec) : Number(colors.accent);
        const marker = chalk.ansi256(markerColor)("▌");
        const content = chalk.bgAnsi256(Number(colors.surfaceElevated)).ansi256(Number(colors.accentFg)).bold(
          `${nameText}`
        );
        const git = gitBadge ? chalk.bgAnsi256(Number(colors.surfaceElevated))(gitBadge) : "";
        const size = chalk.bgAnsi256(Number(colors.surfaceElevated)).ansi256(Number(colors.size))(sizeText);
        outputLines.push(`${marker}${content} ${git}${size}`);
      } else {
        const prefix = isMultiSel
          ? chalk.ansi256(Number(colors.exec))("◆")
          : " ";
        let coloredName: string;
        if (bold) {
          coloredName = chalk.ansi256(Number(clr)).bold(nameText);
        } else {
          coloredName = chalk.ansi256(Number(clr))(nameText);
        }
        const size = chalk.ansi256(Number(colors.size))(sizeText);
        outputLines.push(`${prefix}${coloredName} ${gitBadge}${size}`);
      }
    }

    if (needBot) {
      outputLines.push(chalk.ansi256(Number(colors.scrollbar))(`  ↓ ${state.entries.length - end} more`));
    }
  }

  // Pad to fixed height to prevent layout shifts
  while (outputLines.length < innerH) {
    outputLines.push("");
  }

  const bodyText = outputLines.join("\n");

  return (
    <Box
      flexDirection="column"
      width={width}
      height={height}
      borderStyle="round"
      borderColor={colors.border}
    >
      <Text>{bodyText}</Text>
    </Box>
  );
}
