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
  const fileName = state.deleteTarget ? path.basename(state.deleteTarget) : "";

  // Compact mode for tiny terminals — drops the blank spacers and meta row
  // so the dialog fits in ~5 rows instead of 10.
  const compact = height < 12 || width < 44;
  const dialogWidth = Math.min(56, Math.max(30, width - 8));

  if (compact) {
    const topPad = Math.max(0, Math.floor((height - 5) / 2));
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
            paddingX={1}
          >
            <Text color={colors.danger} bold>Trash?</Text>
            <Text color={colors.accentFg} bold wrap="truncate-end">{fileName}</Text>
            <Box>
              <Text color={colors.accentFg} backgroundColor={colors.danger} bold> y </Text>
              <Text>  </Text>
              <Text color={colors.hintText} dimColor> n </Text>
            </Box>
          </Box>
        </Box>
      </Box>
    );
  }

  const meta = "file";
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
