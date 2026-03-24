import { Box, Text } from "ink";
import path from "path";
import { AppState } from "../types.js";
import { colors } from "../theme.js";

interface Props {
  state: AppState;
  width: number;
  height: number;
}

export function DeleteDialog({ state, width, height }: Props) {
  const dialogWidth = Math.min(56, Math.max(36, width - 12));
  const targets = state.multiSelected.size > 0
    ? [...state.multiSelected]
    : state.deleteTarget ? [state.deleteTarget] : [];

  const fileName = targets.length === 1
    ? path.basename(targets[0])
    : `${targets.length} items`;
  const meta = targets.length === 1 ? "file" : "multiple files / folders";

  const topPad = Math.max(0, Math.floor((height - 10) / 2));

  return (
    <Box flexDirection="column" width={width} height={height}>
      {Array.from({ length: topPad }).map((_, i) => (
        <Box key={`p${i}`} height={1} />
      ))}
      <Box justifyContent="center" width={width}>
        <Box
          flexDirection="column"
          width={dialogWidth}
          borderStyle="round"
          borderColor={colors.danger}
          paddingX={2}
          paddingY={1}
        >
          <Text color={colors.danger} bold>Move to Trash?</Text>
          <Text> </Text>
          <Text color={colors.accentFg} bold>{fileName}</Text>
          <Text color={colors.muted}>{meta}</Text>
          <Text> </Text>
          <Box>
            <Text color={colors.accentFg} backgroundColor={colors.danger} bold> y  trash </Text>
            <Text>  </Text>
            <Text color={colors.hintText} dimColor> n  cancel </Text>
          </Box>
        </Box>
      </Box>
    </Box>
  );
}
