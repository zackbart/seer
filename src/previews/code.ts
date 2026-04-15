import { createHighlighterCore, type HighlighterCore } from "shiki/core";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";
import nordTheme from "@shikijs/themes/nord";

// ── language loaders ────────────────────────────────────────────────────────
// Each entry is a function that dynamic-imports the grammar only when the
// user actually opens a file of that type. The loader module itself does not
// execute until called, so startup pays zero cost for languages that are
// never used.
const LANG_LOADERS: Record<string, () => Promise<unknown>> = {
  javascript: () => import("@shikijs/langs/javascript"),
  jsx: () => import("@shikijs/langs/jsx"),
  typescript: () => import("@shikijs/langs/typescript"),
  tsx: () => import("@shikijs/langs/tsx"),
  python: () => import("@shikijs/langs/python"),
  ruby: () => import("@shikijs/langs/ruby"),
  rust: () => import("@shikijs/langs/rust"),
  go: () => import("@shikijs/langs/go"),
  c: () => import("@shikijs/langs/c"),
  cpp: () => import("@shikijs/langs/cpp"),
  java: () => import("@shikijs/langs/java"),
  csharp: () => import("@shikijs/langs/csharp"),
  php: () => import("@shikijs/langs/php"),
  swift: () => import("@shikijs/langs/swift"),
  kotlin: () => import("@shikijs/langs/kotlin"),
  lua: () => import("@shikijs/langs/lua"),
  haskell: () => import("@shikijs/langs/haskell"),
  elixir: () => import("@shikijs/langs/elixir"),
  ocaml: () => import("@shikijs/langs/ocaml"),
  clojure: () => import("@shikijs/langs/clojure"),
  scala: () => import("@shikijs/langs/scala"),
  bash: () => import("@shikijs/langs/bash"),
  fish: () => import("@shikijs/langs/fish"),
  powershell: () => import("@shikijs/langs/powershell"),
  markdown: () => import("@shikijs/langs/markdown"),
  mdx: () => import("@shikijs/langs/mdx"),
  json: () => import("@shikijs/langs/json"),
  yaml: () => import("@shikijs/langs/yaml"),
  toml: () => import("@shikijs/langs/toml"),
  xml: () => import("@shikijs/langs/xml"),
  ini: () => import("@shikijs/langs/ini"),
  sql: () => import("@shikijs/langs/sql"),
  graphql: () => import("@shikijs/langs/graphql"),
  css: () => import("@shikijs/langs/css"),
  scss: () => import("@shikijs/langs/scss"),
  less: () => import("@shikijs/langs/less"),
  html: () => import("@shikijs/langs/html"),
  svelte: () => import("@shikijs/langs/svelte"),
  vue: () => import("@shikijs/langs/vue"),
  dockerfile: () => import("@shikijs/langs/docker"),
  r: () => import("@shikijs/langs/r"),
  dart: () => import("@shikijs/langs/dart"),
  zig: () => import("@shikijs/langs/zig"),
  nix: () => import("@shikijs/langs/nix"),
  terraform: () => import("@shikijs/langs/terraform"),
  proto: () => import("@shikijs/langs/proto"),
};

let highlighterPromise: Promise<HighlighterCore> | null = null;
const loadedLangs = new Set<string>();
const loadingLangs = new Map<string, Promise<boolean>>();

async function getHighlighter(): Promise<HighlighterCore> {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighterCore({
      themes: [nordTheme],
      langs: [],
      engine: createJavaScriptRegexEngine(),
    });
  }
  return highlighterPromise;
}

async function ensureLanguage(hl: HighlighterCore, lang: string): Promise<boolean> {
  if (loadedLangs.has(lang)) return true;
  const existing = loadingLangs.get(lang);
  if (existing) return existing;

  const loader = LANG_LOADERS[lang];
  if (!loader) return false;

  const promise = (async () => {
    try {
      const mod = (await loader()) as { default: unknown };
      // @shikijs/langs modules default-export the grammar as a loader array
      // of one or more LanguageRegistration objects.
      await hl.loadLanguage(mod.default as any);
      loadedLangs.add(lang);
      return true;
    } catch {
      return false;
    } finally {
      loadingLangs.delete(lang);
    }
  })();
  loadingLangs.set(lang, promise);
  return promise;
}

export async function highlightCode(
  filePath: string,
  text: string,
  signal?: AbortSignal,
): Promise<string> {
  const ext = filePath.split(".").pop()?.toLowerCase() ?? "";
  const lang = extToLang(ext);
  if (lang === "text") return text;

  try {
    const hl = await getHighlighter();
    if (signal?.aborted) return text;

    const loaded = await ensureLanguage(hl, lang);
    if (!loaded || signal?.aborted) return text;

    return ansiFromShiki(hl, text, lang);
  } catch {
    return text;
  }
}

function ansiFromShiki(hl: HighlighterCore, text: string, lang: string): string {
  try {
    const tokens = hl.codeToTokensBase(text, { lang: lang as any, theme: "nord" });
    const lines: string[] = [];

    for (const line of tokens) {
      let lineStr = "";
      for (const token of line) {
        const color = token.color;
        if (color) {
          lineStr += hexToAnsi(color) + token.content + "\x1b[0m";
        } else {
          lineStr += token.content;
        }
      }
      lines.push(lineStr);
    }
    return lines.join("\n");
  } catch {
    return text;
  }
}

function hexToAnsi(hex: string): string {
  if (!hex.startsWith("#") || hex.length < 7) return "";
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `\x1b[38;2;${r};${g};${b}m`;
}

function extToLang(ext: string): string {
  const map: Record<string, string> = {
    js: "javascript",
    mjs: "javascript",
    cjs: "javascript",
    jsx: "jsx",
    ts: "typescript",
    mts: "typescript",
    cts: "typescript",
    tsx: "tsx",
    py: "python",
    rb: "ruby",
    rs: "rust",
    go: "go",
    c: "c",
    cpp: "cpp",
    cc: "cpp",
    cxx: "cpp",
    h: "c",
    hpp: "cpp",
    java: "java",
    cs: "csharp",
    php: "php",
    swift: "swift",
    kt: "kotlin",
    kts: "kotlin",
    lua: "lua",
    hs: "haskell",
    ex: "elixir",
    exs: "elixir",
    ml: "ocaml",
    mli: "ocaml",
    clj: "clojure",
    cljs: "clojure",
    scala: "scala",
    sh: "bash",
    bash: "bash",
    zsh: "bash",
    fish: "fish",
    ps1: "powershell",
    psm1: "powershell",
    md: "markdown",
    markdown: "markdown",
    mdx: "mdx",
    json: "json",
    jsonc: "json",
    yaml: "yaml",
    yml: "yaml",
    toml: "toml",
    xml: "xml",
    svg: "xml",
    plist: "xml",
    ini: "ini",
    conf: "ini",
    cfg: "ini",
    sql: "sql",
    graphql: "graphql",
    gql: "graphql",
    css: "css",
    scss: "scss",
    sass: "scss",
    less: "less",
    svelte: "svelte",
    vue: "vue",
    dockerfile: "dockerfile",
    r: "r",
    dart: "dart",
    zig: "zig",
    nix: "nix",
    tf: "terraform",
    tfvars: "terraform",
    proto: "proto",
  };
  return map[ext] ?? "text";
}
