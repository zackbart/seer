import { createHighlighter, type Highlighter } from "shiki";

let highlighter: Highlighter | null = null;
const loadedLangs = new Set<string>();

async function getHighlighter(): Promise<Highlighter> {
  if (!highlighter) {
    highlighter = await createHighlighter({
      themes: ["nord"],
      langs: [],
    });
  }
  return highlighter;
}

export async function highlightCode(filePath: string, text: string): Promise<string> {
  const ext = filePath.split(".").pop()?.toLowerCase() ?? "";
  const lang = extToLang(ext);

  try {
    const hl = await getHighlighter();

    // Dynamically load language if not already loaded
    if (!loadedLangs.has(lang)) {
      try {
        await hl.loadLanguage(lang as any);
        loadedLangs.add(lang);
      } catch {
        // Language not supported, return raw text
        return text;
      }
    }

    return ansiFromShiki(hl, text, lang);
  } catch {
    return text;
  }
}

function ansiFromShiki(hl: Highlighter, text: string, lang: string): string {
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
    jsx: "jsx",
    ts: "typescript",
    tsx: "tsx",
    py: "python",
    rb: "ruby",
    rs: "rust",
    go: "go",
    c: "c",
    cpp: "cpp",
    h: "c",
    java: "java",
    cs: "csharp",
    php: "php",
    swift: "swift",
    kt: "kotlin",
    lua: "lua",
    hs: "haskell",
    ex: "elixir",
    exs: "elixir",
    ml: "ocaml",
    clj: "clojure",
    scala: "scala",
    vim: "viml",
    sh: "bash",
    bash: "bash",
    zsh: "bash",
    fish: "fish",
    ps1: "powershell",
    bat: "bat",
    cmd: "bat",
    md: "markdown",
    markdown: "markdown",
    mdx: "mdx",
    json: "json",
    yaml: "yaml",
    yml: "yaml",
    toml: "toml",
    xml: "xml",
    ini: "ini",
    conf: "ini",
    env: "dotenv",
    sql: "sql",
    graphql: "graphql",
    css: "css",
    scss: "scss",
    less: "less",
    html: "html",
    svelte: "svelte",
    vue: "vue",
    dockerfile: "dockerfile",
    makefile: "makefile",
    cmake: "cmake",
    r: "r",
    dart: "dart",
    zig: "zig",
    nix: "nix",
    tf: "terraform",
    proto: "protobuf",
    prisma: "prisma",
  };
  return map[ext] ?? "text";
}
