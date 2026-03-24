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
  // Status line
  let statusContent: React.ReactNode;

  if (state.searching) {
    statusContent = (
      <Box>
        <Text color={colors.accent} bold>/ </Text>
        <Text color="white">{state.searchQuery}</Text>
        <Text color={colors.accent}>▌</Text>
        <Text color={colors.dim}>  fuzzy</Text>
      </Box>
    );
  } else if (state.inputMode !== InputMode.None) {
    statusContent = <Text color={colors.accent} bold>{state.inputMode === InputMode.Rename ? "Rename" : state.inputMode === InputMode.NewFile ? "New File" : "New Directory"}</Text>;
  } else {
    const icon = state.status === "ready" ? "◆" : "●";
    const iconColor = state.status === "ready" ? colors.exec : colors.status;
    statusContent = <Text color={iconColor}>{icon} {state.status}</Text>;
  }

  // Key hints
  let hints: Hint[];
  if (state.searching) {
    hints = [
      { key: "esc", desc: "cancel" },
      { key: "backspace", desc: "delete" },
      { key: "enter/l", desc: "open" },
    ];
  } else if (state.inputMode !== InputMode.None) {
    hints = [
      { key: "enter", desc: "confirm" },
      { key: "esc", desc: "cancel" },
      { key: "backspace", desc: "delete char" },
    ];
  } else {
    hints = [
      { key: "j/k", desc: "move" },
      { key: "enter/l", desc: "open" },
      { key: "h", desc: "up" },
      { key: "r", desc: "rename" },
      { key: "n/N", desc: "new file/dir" },
      { key: "e", desc: "edit" },
      { key: "y/x", desc: "yank/cut" },
      { key: "P", desc: "paste" },
      { key: "space", desc: "select" },
      { key: "s", desc: "sort" },
      { key: "b", desc: "bookmark" },
      { key: "1-9", desc: "jump" },
      { key: "backspace", desc: "trash" },
      { key: "p", desc: "copy path" },
      { key: "/", desc: "search" },
      { key: ".", desc: "hidden" },
      { key: "</>", desc: "pane" },
      { key: "R", desc: "reload" },
      { key: "^d/u", desc: "scroll" },
      { key: "q", desc: "quit" },
    ];
  }

  return (
    <Box flexDirection="column" width={width}>
      <Box width={width} height={1}>
        <Text backgroundColor={colors.surface}> </Text>
        <Box flexGrow={1}>
          <Text backgroundColor={colors.surface}>{statusContent}</Text>
        </Box>
        <Text backgroundColor={colors.surface}> </Text>
      </Box>
      <Box width={width} height={1}>
        <Text backgroundColor={colors.surface}> </Text>
        {hints.map((h, i) => (
          <React.Fragment key={h.key}>
            {i > 0 && <Text color={colors.dim} backgroundColor={colors.surface}> · </Text>}
            <Text color={colors.hintKey} bold backgroundColor={colors.surface}>{h.key}</Text>
            <Text color={colors.hintText} backgroundColor={colors.surface}> {h.desc}</Text>
          </React.Fragment>
        ))}
        <Box flexGrow={1}>
          <Text backgroundColor={colors.surface}> </Text>
        </Box>
      </Box>
    </Box>
  );
}
