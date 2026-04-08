import React from "react";
import { Box, Text } from "ink";
import { AppState } from "../types.js";
import { colors } from "../theme.js";
import { visualWidth } from "../utils/ansiText.js";

interface Props {
  state: AppState;
  width: number;
}

interface Hint {
  key: string;
  desc: string;
}

// Same flat-Text pattern as TopBar. Every Text we emit sets
// `backgroundColor={colors.surface}` so both rows of the bar are uniformly
// colored — no reliance on `<Box flexGrow={1}>` which doesn't paint bg.
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

function layoutRow(
  leftSegs: Seg[],
  rightSegs: Seg[],
  width: number,
  keyPrefix: string,
): React.ReactNode {
  let leftW = segWidth(leftSegs);
  let rightW = segWidth(rightSegs);
  let pad = width - leftW - rightW;

  // Overflow — drop right segments until the row fits. Keep at least the
  // trailing space if one exists.
  while (pad < 0 && rightSegs.length > 0) {
    rightSegs.pop();
    rightW = segWidth(rightSegs);
    pad = width - leftW - rightW;
  }
  if (pad < 0) pad = 0;

  return (
    <Box width={width} height={1} key={keyPrefix}>
      {renderSegs(leftSegs, `${keyPrefix}-l`)}
      <Text backgroundColor={colors.surface}>{" ".repeat(pad)}</Text>
      {renderSegs(rightSegs, `${keyPrefix}-r`)}
    </Box>
  );
}

export function BottomBar({ state, width }: Props) {
  // ── status row ─────────────────────────────────────────────────────────
  const statusLeft: Seg[] = [];
  statusLeft.push({ text: " " });

  if (state.searching) {
    statusLeft.push({ text: "/ ", color: colors.accent, bold: true });
    statusLeft.push({ text: state.searchQuery, color: colors.accentFg });
    statusLeft.push({ text: "▎", color: colors.accent });
    statusLeft.push({ text: "  fuzzy", color: colors.dim });
  } else {
    const isReady = state.status === "ready";
    statusLeft.push({ text: "● ", color: isReady ? colors.success : colors.accent });
    statusLeft.push({ text: state.status, color: colors.status });
  }

  // ── hints row ──────────────────────────────────────────────────────────
  let hints: Hint[];
  if (state.searching) {
    hints = [
      { key: "esc", desc: "cancel" },
      { key: "⌫", desc: "delete" },
      { key: "↵", desc: "open" },
    ];
  } else {
    hints = [
      { key: "j/k", desc: "nav" },
      { key: "↵", desc: "open" },
      { key: "h", desc: "back" },
      { key: "/", desc: "find" },
      { key: "s", desc: "sort" },
      { key: ".", desc: "hidden" },
      { key: "p", desc: "path" },
      { key: "e", desc: "edit" },
      { key: "t", desc: "theme" },
      { key: "⌫", desc: "trash" },
      { key: "q", desc: "quit" },
    ];
  }

  // Budget-check hints against width. Build as segments; the row layout
  // below pads with surface bg and keeps the whole row uniformly colored.
  const hintSegs: Seg[] = [];
  hintSegs.push({ text: " " }); // leading pad
  let used = 1;
  for (let i = 0; i < hints.length; i++) {
    const h = hints[i];
    const segLen = visualWidth(h.key) + 1 + visualWidth(h.desc);
    const sepLen = i > 0 ? 3 : 0;
    if (used + sepLen + segLen > width - 1) break;
    if (i > 0) {
      hintSegs.push({ text: " │ ", color: colors.dim });
      used += 3;
    }
    hintSegs.push({ text: h.key, color: colors.hintKey, bold: true });
    hintSegs.push({ text: " " + h.desc, color: colors.hintText });
    used += segLen;
  }

  return (
    <Box flexDirection="column" width={width} height={2}>
      {layoutRow(statusLeft, [], width, "status")}
      {layoutRow(hintSegs, [], width, "hints")}
    </Box>
  );
}
