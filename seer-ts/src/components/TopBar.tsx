import { Box, Text } from "ink";
import path from "path";
import chalk from "chalk";
import { AppState, SortMode, YankMode, sortModeLabel } from "../types.js";
import { colors } from "../theme.js";

interface Props {
  state: AppState;
  width: number;
}

export function TopBar({ state, width }: Props) {
  // Right side info badges
  const badges: string[] = [];
  badges.push(chalk.hex(colors.muted)(`${state.entries.length} items`));
  if (state.showHidden) badges.push(chalk.hex(colors.accent).dim("hidden"));
  if (state.sortBy !== SortMode.NameAsc) {
    badges.push(chalk.hex(colors.accent)(sortModeLabel[state.sortBy]));
  }
  if (state.multiSelected.size > 0) {
    badges.push(chalk.hex(colors.success).bold(`${state.multiSelected.size} sel`));
  }
  if (state.yankPaths.length > 0 && state.yankOp !== YankMode.None) {
    const op = state.yankOp === YankMode.Cut ? "cut" : "yank";
    badges.push(chalk.hex(colors.media)(`${op}:${state.yankPaths.length}`));
  }
  const badgeStr = badges.join(chalk.hex(colors.dim)(" · "));

  // Breadcrumb — last 3 segments with ellipsis
  const pathParts = state.cwd.split(path.sep).filter(Boolean);
  const seg = chalk.hex(colors.breadcrumb);
  const sep = chalk.hex(colors.dim)(" / ");
  const maxParts = Math.min(pathParts.length, 4);
  const shownParts = pathParts.slice(-maxParts);
  let breadcrumb: string;
  if (pathParts.length > maxParts) {
    breadcrumb = chalk.hex(colors.dim)("… ") + sep + shownParts.map(p => seg(p)).join(sep);
  } else {
    breadcrumb = sep.trimStart() + shownParts.map(p => seg(p)).join(sep);
  }

  return (
    <Box width={width} height={1}>
      <Text backgroundColor={colors.surface}> {breadcrumb}</Text>
      <Box flexGrow={1}>
        <Text backgroundColor={colors.surface}> </Text>
      </Box>
      <Text backgroundColor={colors.surface}>{badgeStr} </Text>
    </Box>
  );
}
