import { Box, Text } from "ink";
import { AppState, inputModeTitle } from "../types.js";
import { colors } from "../theme.js";

interface Props {
  state: AppState;
  width: number;
  height: number;
}

export function InputDialog({ state, width, height }: Props) {
  const dialogWidth = Math.min(60, Math.max(36, width - 12));
  const title = inputModeTitle[state.inputMode];
  const topPad = Math.max(0, Math.floor((height - 7) / 2));

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
          borderColor={colors.accent}
          paddingX={2}
          paddingY={1}
        >
          <Text color={colors.accent} bold>{title}</Text>
          <Text> </Text>
          <Box>
            <Text color={colors.dim}>❯ </Text>
            <Text color={colors.accentFg} bold>{state.inputValue}</Text>
            <Text color={colors.accent}>▎</Text>
          </Box>
          <Text> </Text>
          <Text color={colors.hintText} dimColor>↵ confirm  ·  esc cancel</Text>
        </Box>
      </Box>
    </Box>
  );
}
