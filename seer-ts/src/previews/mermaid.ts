import chalk from "chalk";
import { colors } from "../theme.js";

export async function renderMermaid(text: string): Promise<string> {
  // Try mmdc CLI if available
  try {
    const which = Bun.spawnSync(["which", "mmdc"]);
    if (which.exitCode === 0) {
      const proc = Bun.spawn(
        ["mmdc", "-i", "/dev/stdin", "-o", "/dev/stdout", "-e", "txt"],
        { stdin: "pipe", stdout: "pipe", stderr: "ignore" },
      );
      proc.stdin.write(text);
      proc.stdin.end();
      const output = await new Response(proc.stdout).text();
      const exitCode = await proc.exited;
      if (exitCode === 0 && output.trim()) {
        return output.trim();
      }
    }
  } catch {}

  // Fallback: show raw text with a note
  const mutedStyle = chalk.ansi256(Number(colors.muted));
  return mutedStyle("(install @mermaid-js/mermaid-cli for rendered preview)") + "\n\n" + text;
}
