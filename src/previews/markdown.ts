import chalk from "chalk";
import { Marked, RendererThis, Tokens } from "marked";
import { markedTerminal } from "marked-terminal";
import { colors, currentThemeIndex } from "../theme.js";
import { renderRowsAsTable } from "./csv.js";

let renderer: Marked | null = null;
let rendererWidth = 0;
let rendererThemeIdx = -1;

function plainTableCell(parser: RendererThis["parser"], cell: Tokens.TableCell): string {
  return parser.parseInline(cell.tokens, parser.textRenderer).replace(/\s+/g, " ").trim();
}

function getRenderer(width: number): Marked {
  const themeIdx = currentThemeIndex();
  if (renderer && rendererWidth === width && rendererThemeIdx === themeIdx) return renderer;
  rendererWidth = width;
  rendererThemeIdx = themeIdx;

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
  renderer.use({
    extensions: [{
      name: "table",
      renderer(this: RendererThis, token: Tokens.Generic): string {
        const table = token as Tokens.Table;
        const rows = [
          table.header.map((cell) => plainTableCell(this.parser, cell)),
          ...table.rows.map((row) => row.map((cell) => plainTableCell(this.parser, cell))),
        ];
        return renderRowsAsTable(rows, width, false) + "\n\n";
      },
    }],
  });

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
