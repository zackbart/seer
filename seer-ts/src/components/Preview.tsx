import { Box, Text } from "ink";
import path from "path";
import { AppState } from "../types.js";
import { categorise, fileIconExt, entryColor, symlinkIcon, colors } from "../theme.js";
import { humanSize } from "../utils/humanSize.js";

interface Props {
  state: AppState;
  width: number;
  height: number;
}

export function Preview({ state, width, height }: Props) {
  const innerH = Math.max(3, height - 2);

  // Header
  let headerLeft = "";
  let headerRight = "";

  if (state.entries.length > 0 && state.selected < state.entries.length) {
    const e = state.entries[state.selected];
    const cat = categorise(e);
    const icon = e.isSymlink ? symlinkIcon() : fileIconExt(cat, path.extname(e.name));
    let name = icon + e.name;
    if (e.isDir) name += "/";
    if (e.isSymlink && e.symlinkTarget) name += ` → ${e.symlinkTarget}`;
    headerLeft = name;

    if (!e.isDir) {
      headerRight = `${humanSize(e.size)}  ${e.modTime.toLocaleDateString("en-US", { month: "short", day: "2-digit", hour: "2-digit", minute: "2-digit" })}`;
    } else {
      headerRight = e.modTime.toLocaleDateString("en-US", { month: "short", day: "2-digit", hour: "2-digit", minute: "2-digit" });
    }
    if (state.loading) headerRight = "loading…";
  } else {
    headerLeft = "no selection";
  }

  // Preview body with scroll offset
  let previewBody = state.preview;
  if (!previewBody && !state.loading) previewBody = "  (no preview available)";
  if (state.loading && !previewBody) previewBody = "  loading preview…";

  const previewH = Math.max(1, innerH - 2);
  const previewLines = previewBody.split("\n");

  // Clamp offset
  const maxOffset = Math.max(0, previewLines.length - previewH);
  const offset = Math.min(state.previewOffset, maxOffset);

  let displayLines: string[];
  let scrollIndicator = "";

  if (offset > 0) {
    scrollIndicator = `  ↑ line ${offset + 1}`;
    const contentH = Math.max(1, previewH - 1);
    displayLines = previewLines.slice(offset, offset + contentH);
  } else {
    displayLines = previewLines.slice(0, previewH);
  }

  const headerColor = state.entries.length > 0 && state.selected < state.entries.length
    ? entryColor(state.entries[state.selected])
    : colors.muted;

  return (
    <Box flexDirection="column" width={width} height={height}>
      {/* Header */}
      <Box width={width}>
        <Text color={headerColor} bold>{headerLeft}</Text>
        <Box flexGrow={1} />
        <Text color={colors.muted}>{headerRight}</Text>
      </Box>

      {/* Divider */}
      <Box width={width}>
        <Text color={colors.dim}>{"─".repeat(Math.max(1, width))}</Text>
      </Box>

      {/* Scroll indicator */}
      {scrollIndicator && (
        <Box width={width}>
          <Text color={colors.scrollbar}>{scrollIndicator}</Text>
        </Box>
      )}

      {/* Preview content — render as raw ANSI text */}
      <Box flexDirection="column" width={width} flexGrow={1}>
        {displayLines.map((line, i) => (
          <Box key={i} width={width}>
            <Text>{line || " "}</Text>
          </Box>
        ))}
      </Box>
    </Box>
  );
}
