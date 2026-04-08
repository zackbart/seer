import chalk from "chalk";
import mammoth from "mammoth";
import { colors } from "../theme.js";

const MAX_CHARS = 20_000;

export interface DocxPreview {
  rendered: string;
  extracted: string;
  truncated: boolean;
}

export async function renderDocxPreview(buffer: Buffer): Promise<DocxPreview> {
  try {
    const result = await mammoth.extractRawText({ buffer });
    let extracted = (result.value ?? "").replace(/\r\n/g, "\n").replace(/\r/g, "\n");

    let truncated = false;
    if (extracted.length > MAX_CHARS) {
      extracted = extracted.slice(0, MAX_CHARS);
      truncated = true;
    }

    if (extracted.trim().length === 0) {
      return {
        rendered: chalk.hex(colors.muted)("(empty document)"),
        extracted: "",
        truncated: false,
      };
    }

    const mutedStyle = chalk.hex(colors.muted);
    const rendered = truncated
      ? extracted + "\n\n" + mutedStyle("... preview truncated ...")
      : extracted;

    return { rendered, extracted, truncated };
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    return {
      rendered: chalk.hex(colors.muted)(`(unable to read docx: ${msg})`),
      extracted: "",
      truncated: false,
    };
  }
}
