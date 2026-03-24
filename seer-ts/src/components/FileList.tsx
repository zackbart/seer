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
  const innerW = Math.max(8, width - 2);
  const innerH = Math.max(3, height - 2);
  const sizeW = 8;
  const hasGit = state.gitStatus !== null;
  const outputLines: string[] = [];

  // ── title row ──────────────────────────────────────────────────────
  const titleIcon = chalk.hex(colors.accent)("◈");
  const titleText = chalk.hex(colors.title).bold(" Explorer");
  const selInfo = state.multiSelected.size > 0
    ? chalk.hex(colors.success)(` ${state.multiSelected.size}▪`)
    : "";
  const countText = chalk.hex(colors.dim)(`${state.entries.length}`);
  outputLines.push(`${titleIcon}${titleText}${selInfo}  ${countText}`);

  // ── thin separator ─────────────────────────────────────────────────
  outputLines.push(chalk.hex(colors.dim)("─".repeat(innerW)));

  // ── entries ────────────────────────────────────────────────────────
  if (state.entries.length === 0) {
    outputLines.push("");
    outputLines.push(chalk.hex(colors.dim)("    no files"));
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
      outputLines.push(chalk.hex(colors.scrollbar).dim(`  ↑ ${start} more`));
    }

    for (let i = start; i < end; i++) {
      const e = state.entries[i];
      const cat = categorise(e);
      const icon = e.isSymlink ? symlinkIcon() : fileIconExt(cat, path.extname(e.name));
      const clr = entryColor(e);
      const bold = entryBold(e);
      const isSel = i === state.selected;
      const isMulti = state.multiSelected.has(e.path);

      let name = e.name;
      if (e.isDir) name += "/";
      if (e.isSymlink && !e.isDir) name += " →";

      const sizeStr = e.isDir ? "" : humanSize(e.size);

      // Git badge
      let git = "";
      if (hasGit && state.gitStatus) {
        const code = state.gitStatus.get(e.name) ?? "";
        if (code.includes("M")) git = chalk.hex("#e0af68").bold(" M");
        else if (code.includes("A")) git = chalk.hex("#9ece6a").bold(" A");
        else if (code.includes("D")) git = chalk.hex("#f7768e").bold(" D");
        else if (code.includes("R")) git = chalk.hex("#7aa2f7").bold(" R");
        else if (code.includes("?")) git = chalk.hex("#565f89")(" ?");
      }

      if (isSel) {
        // Selected row — accent marker + elevated background
        const marker = isMulti
          ? chalk.hex(colors.success)("┃")
          : chalk.hex(colors.accent)("┃");
        const bg = chalk.bgHex(colors.surfaceElevated);
        const nameStyled = bg.hex(colors.accentFg).bold(` ${icon}${name}`);
        const gitStyled = git ? bg(git) : "";
        const sizeStyled = bg.hex(colors.muted)(sizeStr.padStart(sizeW));
        // Pad the row to fill width
        const contentLen = 1 + icon.length + name.length + (git ? 2 : 0) + sizeW + 1;
        const pad = Math.max(0, innerW - contentLen);
        outputLines.push(`${marker}${nameStyled}${bg(" ".repeat(pad))}${gitStyled}${sizeStyled}${bg(" ")}`);
      } else {
        // Normal row
        const prefix = isMulti
          ? chalk.hex(colors.success)("◆")
          : " ";
        const nameStyled = bold
          ? chalk.hex(clr).bold(`${icon}${name}`)
          : chalk.hex(clr)(`${icon}${name}`);
        const sizeStyled = chalk.hex(colors.dim)(sizeStr.padStart(sizeW));
        outputLines.push(`${prefix}${nameStyled}${git} ${sizeStyled}`);
      }
    }

    if (needBot) {
      outputLines.push(chalk.hex(colors.scrollbar).dim(`  ↓ ${state.entries.length - end} more`));
    }
  }

  // Pad to fixed height
  while (outputLines.length < innerH) {
    outputLines.push("");
  }

  return (
    <Box
      flexDirection="column"
      width={width}
      height={height}
      borderStyle="round"
      borderColor={colors.border}
    >
      <Text>{outputLines.join("\n")}</Text>
    </Box>
  );
}
