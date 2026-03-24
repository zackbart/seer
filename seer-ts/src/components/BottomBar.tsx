import { Box, Text } from "ink";
import chalk from "chalk";
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
  // Status line — build as styled string
  let statusText: string;

  if (state.searching) {
    const accent = chalk.ansi256(Number(colors.accent)).bold;
    const query = chalk.white(state.searchQuery);
    const cursor = chalk.ansi256(Number(colors.accent))("▌");
    const mode = chalk.ansi256(Number(colors.dim))("  fuzzy");
    statusText = `${accent("/ ")}${query}${cursor}${mode}`;
  } else if (state.inputMode !== InputMode.None) {
    const title = state.inputMode === InputMode.Rename ? "Rename"
      : state.inputMode === InputMode.NewFile ? "New File" : "New Directory";
    statusText = chalk.ansi256(Number(colors.accent)).bold(title);
  } else {
    const icon = state.status === "ready" ? "◆" : "●";
    const iconColor = state.status === "ready" ? Number(colors.exec) : Number(colors.status);
    statusText = chalk.ansi256(iconColor)(`${icon} ${state.status}`);
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
      { key: "n/N", desc: "new" },
      { key: "e", desc: "edit" },
      { key: "y/x", desc: "yank/cut" },
      { key: "P", desc: "paste" },
      { key: "space", desc: "sel" },
      { key: "s", desc: "sort" },
      { key: "b", desc: "bkm" },
      { key: "/", desc: "search" },
      { key: ".", desc: "hidden" },
      { key: "q", desc: "quit" },
    ];
  }

  const keyStyle = chalk.ansi256(Number(colors.hintKey)).bold;
  const descStyle = chalk.ansi256(Number(colors.hintText));
  const sepStyle = chalk.ansi256(Number(colors.dim));

  // Build hint string, budget-checked against width
  const parts: string[] = [];
  let used = 2; // padding
  for (let i = 0; i < hints.length; i++) {
    const h = hints[i];
    const seg = `${keyStyle(h.key)}${descStyle(` ${h.desc}`)}`;
    const segLen = h.key.length + 1 + h.desc.length;
    const sepLen = i > 0 ? 3 : 0; // " · "
    if (used + sepLen + segLen > width) break;
    if (i > 0) {
      parts.push(sepStyle(" · "));
      used += 3;
    }
    parts.push(seg);
    used += segLen;
  }
  const hintsText = parts.join("");

  return (
    <Box flexDirection="column" width={width} height={2}>
      <Box width={width} height={1}>
        <Text backgroundColor={colors.surface}> {statusText}</Text>
        <Box flexGrow={1}>
          <Text backgroundColor={colors.surface}> </Text>
        </Box>
      </Box>
      <Box width={width} height={1}>
        <Text backgroundColor={colors.surface}> {hintsText}</Text>
        <Box flexGrow={1}>
          <Text backgroundColor={colors.surface}> </Text>
        </Box>
      </Box>
    </Box>
  );
}
