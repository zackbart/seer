import { Box, Text } from "ink";
import path from "path";
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

  // Breadcrumb path
  const parts = state.cwd.split(path.sep).filter(Boolean);
  const breadcrumb = parts.length > 0 ? "/ " + parts.join(" › ") : "/";

  return (
    <Box width={width} height={1}>
      <Box flexGrow={1}>
        <Text color={colors.breadcrumb}>{breadcrumb}</Text>
      </Box>
      <Box>
        <Text color={colors.muted}>{count}</Text>
      </Box>
    </Box>
  );
}
