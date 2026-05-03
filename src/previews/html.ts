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

// Turndown's `s.remove([...])` is a DOM-walk rule — the parser still walks the
// full body of every <script>/<style> element before pruning. SPA shells with
// hundreds of KB of inline JS pay that walk cost on every preview. We strip
// the bodies from raw text first so turndown never sees them. Two passes per
// tag: paired (open + close found) and open-only (close cut off by the 256KB
// preview cap, leaving an unclosed `<script>` we still need to drop).
function stripNoiseTags(text: string): string {
  let out = text;
  for (const tag of ["script", "style", "noscript"]) {
    const paired = new RegExp(`<${tag}\\b[^>]*>[\\s\\S]*?<\\/${tag}\\s*>`, "gi");
    out = out.replace(paired, "");
    const openOnly = new RegExp(`<${tag}\\b[^>]*>[\\s\\S]*$`, "i");
    out = out.replace(openOnly, "");
  }
  return out;
}

export function renderHtml(
  text: string,
  width: number,
  truncated: boolean,
  signal?: AbortSignal,
): string {
  if (signal?.aborted) return "";
  try {
    const stripped = stripNoiseTags(text);
    if (signal?.aborted) return "";
    const md = getService().turndown(stripped);
    if (signal?.aborted) return "";
    return renderMarkdown(md, width, truncated);
  } catch {
    const out = truncated ? text + "\n\n... preview truncated ..." : text;
    return out;
  }
}
