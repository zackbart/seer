import React from "react";
import { Box, Text } from "ink";
import path from "path";
import { AppState, SortMode, sortModeLabel } from "../types.js";
import { colors } from "../theme.js";
import { visualWidth } from "../utils/ansiText.js";

interface Props {
  state: AppState;
  width: number;
}

// A flat text segment — one color/style run of plain text. We build the bar
// content as segment arrays first so we can measure total width via
// `visualWidth` and emit an explicit padding Text between left and right
// sides. This sidesteps Ink's missing `Box backgroundColor` — every Text we
// render sets `backgroundColor={colors.surface}` explicitly, so the bar is
// uniformly colored edge to edge.
interface Seg {
  text: string;
  color?: string;
  dimColor?: boolean;
  bold?: boolean;
}

function segWidth(segs: Seg[]): number {
  let w = 0;
  for (const s of segs) w += visualWidth(s.text);
  return w;
}

function renderSegs(segs: Seg[], keyPrefix: string): React.ReactNode[] {
  return segs.map((s, i) => (
    <Text
      key={`${keyPrefix}-${i}`}
      color={s.color}
      dimColor={s.dimColor}
      bold={s.bold}
      backgroundColor={colors.surface}
    >
      {s.text}
    </Text>
  ));
}

export function TopBar({ state, width }: Props) {
  // ── left: breadcrumb ───────────────────────────────────────────────────
  const pathParts = state.cwd.split(path.sep).filter(Boolean);
  const maxParts = Math.min(pathParts.length, 4);
  const shown = pathParts.slice(-maxParts);
  const truncated = pathParts.length > maxParts;

  const leftSegs: Seg[] = [];
  // Leading pad
  leftSegs.push({ text: " " });
  if (truncated) {
    leftSegs.push({ text: "…", color: colors.dim });
  }
  for (const p of shown) {
    leftSegs.push({ text: " / ", color: colors.dim });
    leftSegs.push({ text: p, color: colors.breadcrumb });
  }

  // ── right: badges ──────────────────────────────────────────────────────
  const rightSegs: Seg[] = [];
  rightSegs.push({ text: `${state.entries.length} items`, color: colors.muted });
  if (state.showHidden) {
    rightSegs.push({ text: " · hidden", color: colors.accent, dimColor: true });
  }
  if (state.sortBy !== SortMode.NameAsc) {
    rightSegs.push({ text: ` · ${sortModeLabel[state.sortBy]}`, color: colors.accent, dimColor: true });
  }
  rightSegs.push({ text: " " });

  // ── layout: pad left-to-right, truncate right segments on overflow ────
  let leftW = segWidth(leftSegs);
  let rightW = segWidth(rightSegs);
  let pad = width - leftW - rightW;

  // On very narrow terminals, drop right segments from the right until the
  // row fits (keep the trailing space).
  while (pad < 0 && rightSegs.length > 1) {
    // Remove the second-to-last segment (keep the trailing " ").
    rightSegs.splice(rightSegs.length - 2, 1);
    rightW = segWidth(rightSegs);
    pad = width - leftW - rightW;
  }
  // If still overflowing (single long breadcrumb), just clamp pad to 0 — Ink
  // will clip the rightmost content to fit the Box width.
  if (pad < 0) pad = 0;

  return (
    <Box width={width} height={1}>
      {renderSegs(leftSegs, "l")}
      <Text backgroundColor={colors.surface}>{" ".repeat(pad)}</Text>
      {renderSegs(rightSegs, "r")}
    </Box>
  );
}
