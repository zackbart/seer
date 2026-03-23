import chalk from "chalk";

// JSON color tokens (256-color)
const jsonKey = chalk.ansi256(147);     // periwinkle
const jsonStr = chalk.ansi256(114);     // sage green
const jsonNum = chalk.ansi256(222);     // pale gold
const jsonBool = chalk.ansi256(215).bold; // amber
const jsonNull = chalk.ansi256(240).bold; // dim
const jsonBracket = chalk.ansi256(244); // grey
const jsonMuted = chalk.ansi256(240);   // dim

export function renderJSONPreview(text: string, truncated: boolean): string {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text.trim());
  } catch (e) {
    return chalk.ansi256(203)(`  invalid JSON: ${(e as Error).message}`) + "\n\n" + text;
  }

  const lines: string[] = [];
  writeJSON(lines, parsed, 0);
  let out = lines.join("");

  if (truncated) {
    out += "\n" + jsonMuted("  … file truncated, showing partial parse");
  }
  return out;
}

function writeJSON(out: string[], v: unknown, depth: number): void {
  const indent = "  ".repeat(depth);
  const childIndent = "  ".repeat(depth + 1);

  if (v === null) {
    out.push(jsonNull("null"));
    return;
  }

  if (typeof v === "boolean") {
    out.push(jsonBool(String(v)));
    return;
  }

  if (typeof v === "number") {
    if (Number.isInteger(v)) {
      out.push(jsonNum(String(v)));
    } else {
      out.push(jsonNum(String(v)));
    }
    return;
  }

  if (typeof v === "string") {
    const escaped = v.replace(/"/g, '\\"');
    out.push(jsonStr(`"${escaped}"`));
    return;
  }

  if (Array.isArray(v)) {
    if (v.length === 0) {
      out.push(jsonBracket("[]"));
      return;
    }
    out.push(jsonBracket("[") + "\n");
    const limit = Math.min(v.length, 100);
    const capped = v.length > 100;
    for (let i = 0; i < limit; i++) {
      out.push(childIndent);
      writeJSON(out, v[i], depth + 1);
      if (i < v.length - 1) out.push(jsonMuted(","));
      out.push("\n");
    }
    if (capped) {
      out.push(childIndent + jsonMuted(`… ${v.length - limit} more items`) + "\n");
    }
    out.push(indent + jsonBracket("]"));
    return;
  }

  if (typeof v === "object") {
    const obj = v as Record<string, unknown>;
    const keys = Object.keys(obj).sort();
    if (keys.length === 0) {
      out.push(jsonBracket("{}"));
      return;
    }
    out.push(jsonBracket("{") + "\n");
    for (let i = 0; i < keys.length; i++) {
      const k = keys[i];
      out.push(childIndent);
      out.push(jsonKey(`"${k}"`));
      out.push(jsonMuted(": "));
      writeJSON(out, obj[k], depth + 1);
      if (i < keys.length - 1) out.push(jsonMuted(","));
      out.push("\n");
    }
    out.push(indent + jsonBracket("}"));
    return;
  }

  out.push(String(v));
}
