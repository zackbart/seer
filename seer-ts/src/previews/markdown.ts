import { Marked } from "marked";
import { markedTerminal } from "marked-terminal";

let renderer: Marked | null = null;

function getRenderer(width: number): Marked {
  // Recreate when width changes for proper wrapping
  renderer = new Marked();
  renderer.use(
    markedTerminal({
      width: Math.max(40, width),
      reflowText: true,
      showSectionPrefix: false,
      tab: 2,
    }) as any,
  );
  return renderer;
}

export function renderMarkdown(text: string, width: number, truncated: boolean): string {
  try {
    const md = getRenderer(width);
    let result = md.parse(text) as string;

    // Clean up leading/trailing whitespace
    result = result.replace(/^\n+/, "").replace(/\s+$/, "");

    if (truncated) {
      result += "\n\n... preview truncated ...";
    }
    return result;
  } catch {
    return text;
  }
}
