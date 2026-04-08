import chalk from "chalk";
import { colors } from "../theme.js";

export function renderCSVPreview(
  text: string,
  separator: string,
  width: number,
  truncated: boolean,
): string {
  const rows = parseCSV(text, separator);
  if (rows.length === 0) {
    return truncated ? text + "\n\n... preview truncated ..." : text;
  }
  return renderRowsAsTable(rows, width, truncated);
}

// Shared table renderer — used by CSV previews and the xlsx preview so
// spreadsheet cells with commas/quotes/newlines don't have to round-trip
// through CSV serialization.
export function renderRowsAsTable(
  rows: string[][],
  width: number,
  truncated: boolean,
): string {
  if (rows.length === 0) {
    return truncated ? "... preview truncated ..." : "";
  }

  // Cap rows
  const maxRows = 200;
  let showMore = false;
  let displayRows = rows;
  if (rows.length > maxRows + 1) {
    displayRows = rows.slice(0, maxRows + 1);
    showMore = true;
  }

  // Sanitize cells: collapse embedded whitespace (newlines, tabs) so column
  // alignment is preserved. Spreadsheet cells commonly contain newlines that
  // would otherwise break the rendered table.
  displayRows = displayRows.map((row) =>
    row.map((cell) => (cell ?? "").replace(/\s+/g, " ").trim()),
  );

  // Calculate column widths
  let cols = 0;
  for (const row of displayRows) {
    if (row.length > cols) cols = row.length;
  }
  const colW: number[] = new Array(cols).fill(0);
  for (const row of displayRows) {
    for (let c = 0; c < row.length; c++) {
      if (row[c].length > colW[c]) colW[c] = row[c].length;
    }
  }

  // Cap column widths
  const available = width - 2;
  for (let i = 0; i < colW.length; i++) {
    let maxCellW = Math.floor(available / Math.max(1, cols));
    if (maxCellW < 6) maxCellW = 6;
    if (colW[i] > maxCellW) colW[i] = maxCellW;
  }

  const headerStyle = chalk.hex(colors.accentFg).bold;
  const cellStyle = chalk.hex(colors.file);
  const dimStyle = chalk.hex(colors.dim);
  const mutedStyle = chalk.hex(colors.muted);

  const lines: string[] = [];

  for (let rowIdx = 0; rowIdx < displayRows.length; rowIdx++) {
    const row = displayRows[rowIdx];
    const parts: string[] = [];
    for (let c = 0; c < cols; c++) {
      let cell = c < row.length ? row[c] : "";
      // Truncate cell
      if (cell.length > colW[c]) {
        cell = colW[c] > 1 ? cell.slice(0, colW[c] - 1) + "…" : cell.slice(0, colW[c]);
      }
      const padded = cell.padEnd(colW[c]);
      parts.push(rowIdx === 0 ? headerStyle(padded) : cellStyle(padded));
    }
    lines.push("  " + parts.join(dimStyle("  │  ")));

    // Divider after header
    if (rowIdx === 0) {
      const divParts = colW.map((w) => "─".repeat(w));
      lines.push("  " + dimStyle(divParts.join("──┼──")));
    }
  }

  if (showMore) {
    lines.push("");
    lines.push(mutedStyle(`  … and more rows (showing ${maxRows})`));
  }
  if (truncated) {
    lines.push("");
    lines.push(mutedStyle("... preview truncated ..."));
  }

  return lines.join("\n");
}

function parseCSV(text: string, sep: string): string[][] {
  // Simple CSV parser with quote support
  const rows: string[][] = [];
  const lines = text.split("\n");

  for (const line of lines) {
    if (!line.trim()) continue;
    const cells: string[] = [];
    let current = "";
    let inQuotes = false;

    for (let i = 0; i < line.length; i++) {
      const ch = line[i];
      if (inQuotes) {
        if (ch === '"' && line[i + 1] === '"') {
          current += '"';
          i++;
        } else if (ch === '"') {
          inQuotes = false;
        } else {
          current += ch;
        }
      } else if (ch === '"') {
        inQuotes = true;
      } else if (ch === sep) {
        cells.push(current.trim());
        current = "";
      } else {
        current += ch;
      }
    }
    cells.push(current.trim());
    rows.push(cells);
  }

  return rows;
}
