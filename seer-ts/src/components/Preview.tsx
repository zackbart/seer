import { Box, Text } from "ink";
import path from "path";
import chalk from "chalk";
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
  const headerLines: string[] = [];

  // ── header ─────────────────────────────────────────────────────────
  if (state.entries.length > 0 && state.selected < state.entries.length) {
    const e = state.entries[state.selected];
    const cat = categorise(e);
    const icon = e.isSymlink ? symlinkIcon() : fileIconExt(cat, path.extname(e.name));
    const clr = entryColor(e);

    let name = icon + e.name;
    if (e.isDir) name += "/";
    if (e.isSymlink && e.symlinkTarget) name += chalk.hex(colors.dim)(` → ${e.symlinkTarget}`);

    const styledName = chalk.hex(clr).bold(name);

    let meta: string;
    if (state.loading) {
      meta = chalk.hex(colors.loading)("loading…");
    } else if (!e.isDir) {
      const size = chalk.hex(colors.muted)(humanSize(e.size));
      const date = chalk.hex(colors.dim)(
        e.modTime.toLocaleDateString("en-US", { month: "short", day: "2-digit" }) +
        " " +
        e.modTime.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit", hour12: false })
      );
      meta = `${size}  ${date}`;
    } else {
      meta = chalk.hex(colors.dim)(
        e.modTime.toLocaleDateString("en-US", { month: "short", day: "2-digit" })
      );
    }

    headerLines.push(` ${styledName}  ${meta}`);
  } else {
    headerLines.push(chalk.hex(colors.dim)(" no selection"));
  }

  // Thin separator
  headerLines.push(chalk.hex(colors.dim)("─".repeat(Math.max(1, width - 2))));

  // ── body ───────────────────────────────────────────────────────────
  let previewBody = state.preview;
  if (!previewBody && !state.loading) previewBody = chalk.hex(colors.dim)("  no preview");
  if (state.loading && !previewBody) previewBody = chalk.hex(colors.loading)("  loading…");

  const previewH = Math.max(1, innerH - 2);
  const previewLines = previewBody.split("\n");

  const maxOffset = Math.max(0, previewLines.length - previewH);
  const offset = Math.min(state.previewOffset, maxOffset);

  let displayLines: string[];
  const bodyLines: string[] = [];

  if (offset > 0) {
    bodyLines.push(chalk.hex(colors.scrollbar)(`  ↑ line ${offset + 1}`));
    const contentH = Math.max(1, previewH - 1);
    displayLines = previewLines.slice(offset, offset + contentH);
  } else {
    displayLines = previewLines.slice(0, previewH);
  }

  bodyLines.push(...displayLines);

  // Pad to fixed height
  while (bodyLines.length < previewH) {
    bodyLines.push("");
  }

  const allLines = [...headerLines, ...bodyLines];
  while (allLines.length < innerH) {
    allLines.push("");
  }

  return (
    <Box
      flexDirection="column"
      width={width}
      height={height}
      borderStyle="round"
      borderColor={colors.borderStrong}
    >
      <Text>{allLines.join("\n")}</Text>
    </Box>
  );
}
