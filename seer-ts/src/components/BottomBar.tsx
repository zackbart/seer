import React from "react";
import { Box, Text } from "ink";
import { AppState, InputMode } from "../types.js";
import { colors } from "../theme.js";

interface Props {
  state: AppState;
  width: number;
}

interface Hint {
  key: string;
  desc: string;
}

export function BottomBar({ state, width }: Props) {
  // ── status line ────────────────────────────────────────────────────
  let statusContent: React.ReactNode;

  if (state.searching) {
    statusContent = (
      <>
        <Text color={colors.accent} bold>/ </Text>
        <Text color={colors.accentFg}>{state.searchQuery}</Text>
        <Text color={colors.accent}>▎</Text>
        <Text color={colors.dim}>  fuzzy</Text>
      </>
    );
  } else if (state.inputMode !== InputMode.None) {
    const title = state.inputMode === InputMode.Rename ? "Rename"
      : state.inputMode === InputMode.NewFile ? "New File" : "New Directory";
    statusContent = <Text color={colors.accent} bold>{title}</Text>;
  } else {
    const isReady = state.status === "ready";
    statusContent = (
      <>
        <Text color={isReady ? colors.success : colors.accent}>● </Text>
        <Text color={colors.status}>{state.status}</Text>
      </>
    );
  }

  // ── key hints ──────────────────────────────────────────────────────
  let hints: Hint[];
  if (state.searching) {
    hints = [
      { key: "esc", desc: "cancel" },
      { key: "⌫", desc: "delete" },
      { key: "↵", desc: "open" },
    ];
  } else if (state.inputMode !== InputMode.None) {
    hints = [
      { key: "↵", desc: "confirm" },
      { key: "esc", desc: "cancel" },
    ];
  } else {
    hints = [
      { key: "j/k", desc: "nav" },
      { key: "↵", desc: "open" },
      { key: "h", desc: "back" },
      { key: "tab", desc: "pane" },
      { key: "^d/u", desc: "scroll" },
      { key: "/", desc: "find" },
      { key: "r", desc: "rename" },
      { key: "n", desc: "new" },
      { key: "e", desc: "edit" },
      { key: "y/x", desc: "yank" },
      { key: "P", desc: "paste" },
      { key: "⌫", desc: "trash" },
      { key: "s", desc: "sort" },
      { key: ".", desc: "hidden" },
      { key: "q", desc: "quit" },
    ];
  }

  // Budget-check hints against width
  const hintElements: React.ReactNode[] = [];
  let used = 2;
  for (let i = 0; i < hints.length; i++) {
    const h = hints[i];
    const segLen = h.key.length + 1 + h.desc.length;
    const sepLen = i > 0 ? 3 : 0;
    if (used + sepLen + segLen > width - 1) break;
    if (i > 0) {
      hintElements.push(<Text key={`sep-${i}`} color={colors.dim}> │ </Text>);
      used += 3;
    }
    hintElements.push(
      <React.Fragment key={`hint-${i}`}>
        <Text color={colors.hintKey} bold>{h.key}</Text>
        <Text color={colors.hintText}> {h.desc}</Text>
      </React.Fragment>,
    );
    used += segLen;
  }

  return (
    <Box flexDirection="column" width={width} height={2}>
      <Box width={width} height={1}>
        <Text backgroundColor={colors.surface}> </Text>
        {statusContent}
        <Box flexGrow={1}>
          <Text backgroundColor={colors.surface}> </Text>
        </Box>
      </Box>
      <Box width={width} height={1}>
        <Text backgroundColor={colors.surface}> </Text>
        {hintElements}
        <Box flexGrow={1}>
          <Text backgroundColor={colors.surface}> </Text>
        </Box>
      </Box>
    </Box>
  );
}
