import fsp from "fs/promises";
import path from "path";
import chalk from "chalk";
import { Entry, FileCategory, MAX_DIR_PREVIEW } from "../types.js";
import { categorise, fileIconExt, entryColor, colors } from "../theme.js";

export async function buildDirPreview(dirPath: string): Promise<string> {
  const items = await fsp.readdir(dirPath, { withFileTypes: true });

  // Match the in-folder listing order: dirs first, then alphabetical.
  items.sort((a, b) => {
    const aDir = a.isDirectory();
    const bDir = b.isDirectory();
    if (aDir !== bDir) return aDir ? -1 : 1;
    return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
  });

  const lines: string[] = [];
  const dirIcon = fileIconExt(FileCategory.Dir, "");
  lines.push(chalk.hex(colors.dir).bold(`${dirIcon}${path.basename(dirPath)}/`));
  lines.push(chalk.hex(colors.muted)(`  ${items.length} items`));
  lines.push(chalk.hex(colors.dim)(`  ${"─".repeat(30)}`));
  lines.push("");

  const limit = Math.min(items.length, MAX_DIR_PREVIEW);
  for (let i = 0; i < limit; i++) {
    const item = items[i];
    const name = item.name;
    const isDir = item.isDirectory();
    const fakeEntry: Entry = {
      name,
      path: path.join(dirPath, name),
      isDir,
      size: 0,
      modTime: new Date(),
      isSymlink: false,
      symlinkTarget: "",
    };
    const cat = categorise(fakeEntry);
    const color = entryColor(fakeEntry);
    const icon = fileIconExt(cat, path.extname(name));

    if (isDir) {
      lines.push(chalk.hex(color).bold(`  ${icon}${name}/`));
    } else {
      lines.push(chalk.hex(color)(`  ${icon}${name}`));
    }
  }

  if (items.length > limit) {
    lines.push("");
    lines.push(chalk.hex(colors.muted)(`  … and ${items.length - limit} more`));
  }

  return lines.join("\n");
}
