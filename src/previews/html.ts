import TurndownService from "turndown";
// @ts-expect-error — turndown-plugin-gfm ships no types
import * as turndownPluginGfm from "turndown-plugin-gfm";
import { renderMarkdown } from "./markdown.js";

// turndown ships as CJS; Bun's ESM interop may surface it as `.default` or as
// the constructor directly. Unwrap defensively. Same for the gfm plugin.
const Ctor: typeof TurndownService =
  (TurndownService as unknown as { default?: typeof TurndownService }).default ??
  TurndownService;

interface GfmPluginModule {
  gfm: (service: TurndownService) => void;
  default?: GfmPluginModule;
}
const gfmMod = turndownPluginGfm as unknown as GfmPluginModule;
const gfm = gfmMod.gfm ?? gfmMod.default?.gfm;

let service: TurndownService | null = null;

function getService(): TurndownService {
  if (service) return service;
  const s = new Ctor({
    headingStyle: "atx",
    codeBlockStyle: "fenced",
    bulletListMarker: "-",
    emDelimiter: "_",
  });
  // Strip non-content blocks so their text doesn't leak into the markdown.
  s.remove(["script", "style", "noscript", "title"]);
  // GFM plugin adds markdown table syntax (| col | col |) that marked-terminal
  // renders as a formatted grid; without it turndown dumps cells line-by-line.
  if (gfm) gfm(s);
  service = s;
  return s;
}

export function renderHtml(text: string, width: number, truncated: boolean): string {
  try {
    const md = getService().turndown(text);
    return renderMarkdown(md, width, truncated);
  } catch {
    const out = truncated ? text + "\n\n... preview truncated ..." : text;
    return out;
  }
}
