import chalk from "chalk";
import { extractText, getDocumentProxy } from "unpdf";
import { colors } from "../theme.js";

const MAX_CHARS = 20_000;

export interface PdfPreview {
  rendered: string;
  extracted: string;
  truncated: boolean;
}

export async function renderPdfPreview(buffer: Buffer): Promise<PdfPreview> {
  try {
    const pdf = await getDocumentProxy(new Uint8Array(buffer));
    const { text } = await extractText(pdf, { mergePages: true });

    let extracted = (typeof text === "string" ? text : (text as string[]).join("\n"))
      .replace(/\r\n/g, "\n")
      .replace(/\r/g, "\n");

    let truncated = false;
    if (extracted.length > MAX_CHARS) {
      extracted = extracted.slice(0, MAX_CHARS);
      truncated = true;
    }

    if (extracted.trim().length === 0) {
      return {
        rendered: chalk.hex(colors.muted)(
          "(no extractable text — this PDF may be scanned/image-based)",
        ),
        extracted: "",
        truncated: false,
      };
    }

    const mutedStyle = chalk.hex(colors.muted);
    const header = mutedStyle(`  pdf: ${pdf.numPages} page${pdf.numPages === 1 ? "" : "s"}`) + "\n\n";
    const body = truncated
      ? extracted + "\n\n" + mutedStyle("... preview truncated ...")
      : extracted;

    return { rendered: header + body, extracted, truncated };
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    return {
      rendered: chalk.hex(colors.muted)(`(unable to read pdf: ${msg})`),
      extracted: "",
      truncated: false,
    };
  }
}
