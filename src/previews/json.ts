import chalk, { ChalkInstance } from "chalk";
import { colors } from "../theme.js";

interface JsonStyles {
  key: ChalkInstance;
  str: ChalkInstance;
  num: ChalkInstance;
  bool: ChalkInstance;
  nul: ChalkInstance;
  bracket: ChalkInstance;
  muted: ChalkInstance;
}

function buildStyles(): JsonStyles {
  return {
    key: chalk.hex(colors.doc),
    str: chalk.hex(colors.exec),
    num: chalk.hex(colors.media),
    bool: chalk.hex(colors.config).bold,
    nul: chalk.hex(colors.muted).bold,
    bracket: chalk.hex(colors.size),
    muted: chalk.hex(colors.muted),
  };
}

export function renderJSONPreview(text: string, truncated: boolean): string {
  const styles = buildStyles();
  let parsed: unknown;
  try {
    parsed = JSON.parse(text.trim());
  } catch (e) {
    return chalk.hex(colors.danger)(`  invalid JSON: ${(e as Error).message}`) + "\n\n" + text;
  }

  const lines: string[] = [];
  writeJSON(lines, parsed, 0, styles);
  let out = lines.join("");

  if (truncated) {
    out += "\n" + styles.muted("  … file truncated, showing partial parse");
  }
  return out;
}

function writeJSON(out: string[], v: unknown, depth: number, s: JsonStyles): void {
  const indent = "  ".repeat(depth);
  const childIndent = "  ".repeat(depth + 1);

  if (v === null) {
    out.push(s.nul("null"));
    return;
  }

  if (typeof v === "boolean") {
    out.push(s.bool(String(v)));
    return;
  }

  if (typeof v === "number") {
    out.push(s.num(String(v)));
    return;
  }

  if (typeof v === "string") {
    const escaped = v.replace(/"/g, '\\"');
    out.push(s.str(`"${escaped}"`));
    return;
  }

  if (Array.isArray(v)) {
    if (v.length === 0) {
      out.push(s.bracket("[]"));
      return;
    }
    out.push(s.bracket("[") + "\n");
    const limit = Math.min(v.length, 100);
    const capped = v.length > 100;
    for (let i = 0; i < limit; i++) {
      out.push(childIndent);
      writeJSON(out, v[i], depth + 1, s);
      if (i < v.length - 1) out.push(s.muted(","));
      out.push("\n");
    }
    if (capped) {
      out.push(childIndent + s.muted(`… ${v.length - limit} more items`) + "\n");
    }
    out.push(indent + s.bracket("]"));
    return;
  }

  if (typeof v === "object") {
    const obj = v as Record<string, unknown>;
    const keys = Object.keys(obj).sort();
    if (keys.length === 0) {
      out.push(s.bracket("{}"));
      return;
    }
    out.push(s.bracket("{") + "\n");
    for (let i = 0; i < keys.length; i++) {
      const k = keys[i];
      out.push(childIndent);
      out.push(s.key(`"${k}"`));
      out.push(s.muted(": "));
      writeJSON(out, obj[k], depth + 1, s);
      if (i < keys.length - 1) out.push(s.muted(","));
      out.push("\n");
    }
    out.push(indent + s.bracket("}"));
    return;
  }

  out.push(String(v));
}
