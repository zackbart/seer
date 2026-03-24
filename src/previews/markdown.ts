import chalk from "chalk";
import { Marked } from "marked";
import { markedTerminal } from "marked-terminal";
import { colors } from "../theme.js";

let renderer: Marked | null = null;
let rendererWidth = 0;

function getRenderer(width: number): Marked {
  if (renderer && rendererWidth === width) return renderer;
  rendererWidth = width;

  // Force chalk color support for the markdown renderer
  const level = chalk.level;
  if (level === 0) chalk.level = 3;

  renderer = new Marked();
  renderer.use(
    markedTerminal({
      // Custom styles using our theme colors
      firstHeading: chalk.hex(colors.accent).bold.underline,
      heading: chalk.hex(colors.doc).bold,
      strong: chalk.hex(colors.accentFg).bold,
      em: chalk.italic,
      codespan: chalk.hex(colors.media),
      code: chalk.hex(colors.media),
      blockquote: chalk.hex(colors.muted).italic,
      link: chalk.hex(colors.symlink).underline,
      href: chalk.hex(colors.symlink).underline,
      del: chalk.hex(colors.muted).strikethrough,
      listitem: chalk.hex(colors.file),
      // Layout
      width: Math.max(40, width),
      reflowText: true,
      showSectionPrefix: true,
      tab: 2,
      unescape: true,
      emoji: true,
    }) as any,
  );

  chalk.level = level;
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
