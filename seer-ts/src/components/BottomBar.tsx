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
  // ── status line ────────────────────────────────────────────────────
  let statusText: string;

  if (state.searching) {
    statusText = [
      chalk.hex(colors.accent).bold("/"),
      chalk.hex(colors.accentFg)(state.searchQuery),
      chalk.hex(colors.accent)("▎"),
      chalk.hex(colors.dim)(" fuzzy"),
    ].join("");
  } else if (state.inputMode !== InputMode.None) {
    const title = state.inputMode === InputMode.Rename ? "Rename"
      : state.inputMode === InputMode.NewFile ? "New File" : "New Directory";
    statusText = chalk.hex(colors.accent).bold(`${title}`);
  } else {
    const isReady = state.status === "ready";
    const dot = isReady
      ? chalk.hex(colors.success)("●")
      : chalk.hex(colors.accent)("●");
    statusText = `${dot} ${chalk.hex(colors.status)(state.status)}`;
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

  // Build hint string with budget checking
  const key = chalk.hex(colors.hintKey).bold;
  const desc = chalk.hex(colors.hintText);
  const sep = chalk.hex(colors.dim)(" │ ");

  const parts: string[] = [];
  let used = 2;
  for (let i = 0; i < hints.length; i++) {
    const h = hints[i];
    const segLen = h.key.length + 1 + h.desc.length;
    const sepLen = i > 0 ? 3 : 0;
    if (used + sepLen + segLen > width - 1) break;
    if (i > 0) { parts.push(sep); used += 3; }
    parts.push(`${key(h.key)} ${desc(h.desc)}`);
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
