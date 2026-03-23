import { Marked } from "marked";
import markedTerminal from "marked-terminal";

let renderer: Marked | null = null;

function getRenderer(): Marked {
  if (!renderer) {
    renderer = new Marked(
      markedTerminal({
        width: 80,
        reflowText: true,
        showSectionPrefix: false,
        tab: 2,
      }) as any,
    );
  }
  return renderer;
}

export function renderMarkdown(text: string, _width: number, truncated: boolean): string {
  try {
    const md = getRenderer();
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
