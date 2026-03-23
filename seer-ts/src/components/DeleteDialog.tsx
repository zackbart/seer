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
  const dialogWidth = Math.min(72, Math.max(42, width - 8));

  const targets = state.multiSelected.size > 0
    ? [...state.multiSelected]
    : state.deleteTarget ? [state.deleteTarget] : [];

  let fileName: string;
  let meta: string;

  if (targets.length === 1) {
    fileName = path.basename(targets[0]);
    meta = "file";
  } else {
    fileName = `${targets.length} items`;
    meta = "multiple files / folders";
  }

  const topPad = Math.max(0, Math.floor((height - 10) / 2));

  return (
    <Box flexDirection="column" width={width} height={height}>
      {Array.from({ length: topPad }).map((_, i) => (
        <Box key={`pad-${i}`} height={1} />
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
          <Text color={colors.hintText}>Enter or y confirms. Esc or n cancels.</Text>
          <Text> </Text>
          <Box>
            <Text color={colors.accentFg} backgroundColor={colors.danger} bold> enter / y  move </Text>
            <Text>  </Text>
            <Text color={colors.hintText}> esc / n  cancel </Text>
          </Box>
        </Box>
      </Box>
    </Box>
  );
}
