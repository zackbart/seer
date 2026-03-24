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
  // Right side: count info
  let count = `${state.entries.length} items`;
  if (state.showHidden) count += " (hidden)";
  if (state.sortBy !== SortMode.NameAsc) count += `  ${sortModeLabel[state.sortBy]}`;
  if (state.multiSelected.size > 0) count += `  ${state.multiSelected.size} selected`;
  if (state.yankPaths.length > 0 && state.yankOp !== YankMode.None) {
    const op = state.yankOp === YankMode.Cut ? "cut" : "copy";
    count += `  [${op}: ${state.yankPaths.length}]`;
  }

  // Breadcrumb with styled separators
  const pathParts = state.cwd.split(path.sep).filter(Boolean);
  const sepStyle = chalk.ansi256(Number(colors.pathSep));
  const segStyle = chalk.ansi256(Number(colors.breadcrumb));
  let breadcrumb: string;
  if (pathParts.length > 0) {
    breadcrumb = sepStyle("/ ") + pathParts.map((p) => segStyle(p)).join(sepStyle(" › "));
  } else {
    breadcrumb = segStyle("/");
  }

  const countStyled = chalk.ansi256(Number(colors.muted))(count);

  return (
    <Box width={width} height={1}>
      <Text backgroundColor={colors.surface}> {breadcrumb}</Text>
      <Box flexGrow={1}>
        <Text backgroundColor={colors.surface}> </Text>
      </Box>
      <Text backgroundColor={colors.surface}>{countStyled} </Text>
    </Box>
  );
}
