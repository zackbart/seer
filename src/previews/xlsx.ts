import chalk from "chalk";
import ExcelJS from "exceljs";
import { colors } from "../theme.js";
import { renderRowsAsTable } from "./csv.js";

const MAX_ROWS = 200;

export interface XlsxPreview {
  rendered: string;
  extracted: string;
  truncated: boolean;
}

function stringifyCell(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (value instanceof Date) return value.toISOString().slice(0, 10);
  // ExcelJS cell values can be rich text, formulas, hyperlinks, etc.
  if (typeof value === "object") {
    const v = value as Record<string, unknown>;
    if (typeof v.text === "string") return v.text;
    if (Array.isArray(v.richText)) {
      return v.richText.map((r: { text?: string }) => r.text ?? "").join("");
    }
    if (typeof v.result !== "undefined") return stringifyCell(v.result);
    if (typeof v.hyperlink === "string") return v.hyperlink;
  }
  try {
    return String(value);
  } catch {
    return "";
  }
}

export async function renderXlsxPreview(
  buffer: Buffer,
  width: number,
): Promise<XlsxPreview> {
  try {
    const wb = new ExcelJS.Workbook();
    // exceljs accepts a Node Buffer for xlsx.load; cast the type so TS accepts it.
    await wb.xlsx.load(buffer as unknown as ArrayBuffer);

    const sheets = wb.worksheets;
    if (sheets.length === 0) {
      return {
        rendered: chalk.hex(colors.muted)("(empty workbook)"),
        extracted: "",
        truncated: false,
      };
    }

    const ws = sheets[0];
    const rows: string[][] = [];
    let truncated = false;

    // Manual row walk instead of ws.eachRow(): eachRow has no early-break
    // primitive, so on huge sheets it would visit every row even after the cap.
    const lastRow = ws.actualRowCount > 0 ? ws.rowCount : 0;
    for (let r = 1; r <= lastRow; r++) {
      if (rows.length >= MAX_ROWS + 1) {
        truncated = true;
        break;
      }
      const row = ws.getRow(r);
      if (!row || row.actualCellCount === 0) continue;
      const cells: string[] = [];
      const last = row.cellCount;
      for (let c = 1; c <= last; c++) {
        cells.push(stringifyCell(row.getCell(c).value));
      }
      rows.push(cells);
    }

    if (rows.length === 0) {
      return {
        rendered: chalk.hex(colors.muted)(`(empty sheet: ${ws.name})`),
        extracted: "",
        truncated: false,
      };
    }

    const table = renderRowsAsTable(rows, width, truncated);

    const mutedStyle = chalk.hex(colors.muted);
    const header =
      sheets.length > 1
        ? mutedStyle(`  sheet 1 of ${sheets.length}: ${ws.name}`) + "\n\n"
        : mutedStyle(`  sheet: ${ws.name}`) + "\n\n";

    const rendered = header + table;
    // Metric source is the sheet rendered as plain text rows so line/token
    // counts reflect real data, not ANSI-styled output.
    const extracted = rows.map((r) => r.join("\t")).join("\n");

    return { rendered, extracted, truncated };
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    return {
      rendered: chalk.hex(colors.muted)(`(unable to read xlsx: ${msg})`),
      extracted: "",
      truncated: false,
    };
  }
}
