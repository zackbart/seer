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
  // Breadcrumb
  const pathParts = state.cwd.split(path.sep).filter(Boolean);
  const maxParts = Math.min(pathParts.length, 4);
  const shown = pathParts.slice(-maxParts);
  const truncated = pathParts.length > maxParts;

  // Right side badges
  const badges: React.ReactNode[] = [];
  badges.push(<Text key="count" color={colors.muted}>{state.entries.length} items</Text>);
  if (state.showHidden) {
    badges.push(<Text key="hidden" color={colors.accent} dimColor> · hidden</Text>);
  }
  if (state.sortBy !== SortMode.NameAsc) {
    badges.push(<Text key="sort" color={colors.accent} dimColor> · {sortModeLabel[state.sortBy]}</Text>);
  }
  if (state.multiSelected.size > 0) {
    badges.push(<Text key="sel" color={colors.success} bold> · {state.multiSelected.size} sel</Text>);
  }
  if (state.yankPaths.length > 0 && state.yankOp !== YankMode.None) {
    const op = state.yankOp === YankMode.Cut ? "cut" : "yank";
    badges.push(<Text key="yank" color={colors.media}> · {op}:{state.yankPaths.length}</Text>);
  }

  return (
    <Box width={width} height={1}>
      <Text backgroundColor={colors.surface}> </Text>
      {truncated && <Text color={colors.dim} backgroundColor={colors.surface}>…</Text>}
      {shown.map((p, i) => (
        <React.Fragment key={i}>
          <Text color={colors.dim} backgroundColor={colors.surface}> / </Text>
          <Text color={colors.breadcrumb} backgroundColor={colors.surface}>{p}</Text>
        </React.Fragment>
      ))}
      <Box flexGrow={1}>
        <Text backgroundColor={colors.surface}> </Text>
      </Box>
      <Text backgroundColor={colors.surface}>{badges} </Text>
    </Box>
  );
}
