import chalk from "chalk";
import { colors } from "../theme.js";
import { humanSize } from "../utils/humanSize.js";

export function buildHexPreview(data: Buffer, name: string, size: number, modTime: Date): string {
  const addrStyle = chalk.ansi256(110);
  const hexStyle = chalk.ansi256(189);
  const asciiStyle = chalk.ansi256(245);
  const dimStyle = chalk.ansi256(240);
  const mutedStyle = chalk.ansi256(Number(colors.muted));
  const labelStyle = chalk.ansi256(Number(colors.binary)).bold;

  const lines: string[] = [];

  // Header
  lines.push(labelStyle(`binary file: ${name}`));
  lines.push(
    mutedStyle(
      `size: ${humanSize(size)}  modified: ${modTime.toLocaleDateString("en-GB", { day: "2-digit", month: "short", year: "numeric" })} ${modTime.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZoneName: "short" })}`,
    ),
  );
  lines.push(dimStyle("─".repeat(50)));
  lines.push("");

  // Hex dump – max 512 bytes, 16 bytes per row
  const limit = Math.min(data.length, 512);

  for (let offset = 0; offset < limit; offset += 16) {
    const end = Math.min(offset + 16, limit);
    const row = data.subarray(offset, end);

    // Address
    const addr = addrStyle(offset.toString(16).padStart(8, "0"));

    // Hex bytes
    const hexParts: string[] = [];
    for (let i = 0; i < 16; i++) {
      if (i === 8) hexParts.push(" ");
      if (i < row.length) {
        hexParts.push(row[i].toString(16).padStart(2, "0"));
      } else {
        hexParts.push("  ");
      }
    }
    const hex = hexStyle(hexParts.join(" "));

    // ASCII
    let ascii = "";
    for (let i = 0; i < row.length; i++) {
      const b = row[i];
      ascii += b >= 0x20 && b < 0x7f ? String.fromCharCode(b) : ".";
    }

    const pipe = dimStyle("|");
    lines.push(`${addr}  ${hex}  ${pipe}${asciiStyle(ascii)}${pipe}`);
  }

  if (data.length > 512) {
    lines.push("");
    lines.push(mutedStyle(`… showing 512 of ${data.length} bytes`));
  }

  return lines.join("\n");
}
