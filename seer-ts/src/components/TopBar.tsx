import React from "react";
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

  // Style individual breadcrumb segments with dimmed separators
  const pathParts = state.cwd.split(path.sep).filter(Boolean);
  const breadcrumbElements: React.ReactNode[] = [];
  if (pathParts.length > 0) {
    breadcrumbElements.push(<Text key="root" color={colors.pathSep}>/ </Text>);
    pathParts.forEach((p, i) => {
      if (i > 0) {
        breadcrumbElements.push(<Text key={`sep-${i}`} color={colors.pathSep}> › </Text>);
      }
      breadcrumbElements.push(<Text key={`seg-${i}`} color={colors.breadcrumb}>{p}</Text>);
    });
  } else {
    breadcrumbElements.push(<Text key="root" color={colors.breadcrumb}>/</Text>);
  }

  return (
    <Box width={width} height={1}>
      <Text backgroundColor={colors.surface}> </Text>
      <Box flexGrow={1}>
        <Text backgroundColor={colors.surface}>{breadcrumbElements}</Text>
      </Box>
      <Box>
        <Text color={colors.muted} backgroundColor={colors.surface}> {count} </Text>
      </Box>
    </Box>
  );
}
