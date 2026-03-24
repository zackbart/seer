import chalk from "chalk";
import path from "path";
import { FileCategory } from "../types.js";
import { colors, fileIconExt } from "../theme.js";

const archiveExtsSet = new Set([
  ".zip", ".jar", ".war", ".tar", ".gz", ".tgz", ".bz2", ".tbz2",
  ".xz", ".txz", ".zst", ".7z", ".rar",
]);

export function isArchiveExt(ext: string): boolean {
  return archiveExtsSet.has(ext);
}

export async function buildArchivePreview(filePath: string): Promise<string> {
  const ext = path.extname(filePath).toLowerCase();
  const lines = await archiveListings(filePath, ext);

  if (!lines || lines.length === 0) {
    return `archive: ${path.basename(filePath)}\n\n(install unzip / tar / 7z to preview contents)`;
  }

  const labelStyle = chalk.hex(colors.config).bold;
  const mutedStyle = chalk.hex(colors.muted);
  const dimStyle = chalk.hex(colors.dim);
  const fileStyle = chalk.hex(colors.file);
  const dirStyle = chalk.hex(colors.dir).bold;

  const output: string[] = [];
  output.push(labelStyle(`archive: ${path.basename(filePath)}`));
  output.push(mutedStyle(`  ${lines.length} entries`));
  output.push(dimStyle(`  ${"─".repeat(30)}`));
  output.push("");

  const maxShow = 200;
  for (let i = 0; i < lines.length; i++) {
    if (i >= maxShow) {
      output.push("");
      output.push(mutedStyle(`  … and ${lines.length - maxShow} more`));
      break;
    }
    const line = lines[i].trim();
    if (!line) continue;
    if (line.endsWith("/")) {
      const icon = fileIconExt(FileCategory.Dir, "");
      output.push(dirStyle(`  ${icon}${line}`));
    } else {
      output.push(fileStyle(`  ${line}`));
    }
  }

  return output.join("\n");
}

async function archiveListings(filePath: string, ext: string): Promise<string[] | null> {
  let args: string[] | null = null;

  const hasCmd = async (cmd: string) => {
    try {
      return Bun.spawnSync(["which", cmd]).exitCode === 0;
    } catch {
      return false;
    }
  };

  switch (ext) {
    case ".zip": case ".jar": case ".war":
      if (await hasCmd("unzip")) {
        args = ["unzip", "-Z1", filePath];
      } else if (await hasCmd("7z")) {
        args = ["7z", "l", "-ba", "-slt", filePath];
      }
      break;
    case ".tar":
      if (await hasCmd("tar")) args = ["tar", "tf", filePath];
      break;
    case ".gz": case ".tgz":
      if (await hasCmd("tar")) args = ["tar", "tzf", filePath];
      break;
    case ".bz2": case ".tbz2":
      if (await hasCmd("tar")) args = ["tar", "tjf", filePath];
      break;
    case ".xz": case ".txz":
      if (await hasCmd("tar")) args = ["tar", "tJf", filePath];
      break;
    case ".zst":
      if (await hasCmd("tar")) args = ["tar", "--use-compress-program=zstd", "-tf", filePath];
      break;
    case ".7z": case ".rar":
      if (await hasCmd("7z")) args = ["7z", "l", "-ba", filePath];
      break;
  }

  if (!args) return null;

  try {
    const proc = Bun.spawn(args, { stdout: "pipe", stderr: "ignore" });
    const text = await new Response(proc.stdout).text();
    const exitCode = await proc.exited;
    if (exitCode !== 0) return null;

    const raw = text.trim().split("\n");

    // Detect 7z -slt output
    const used7z = ext === ".7z" || ext === ".rar" ||
      ((ext === ".zip" || ext === ".jar" || ext === ".war") && !(await hasCmd("unzip")));
    if (used7z && raw.length > 0) {
      const paths = raw
        .filter((l) => l.startsWith("Path = "))
        .map((l) => l.slice(7));
      if (paths.length > 0) return paths;
    }

    return raw;
  } catch {
    return null;
  }
}
