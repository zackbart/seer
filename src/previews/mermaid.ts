import chalk from "chalk";
import { colors } from "../theme.js";

// Memoize the `which mmdc` lookup — the sync subprocess used to run on every
// mermaid preview, blocking the event loop. One hit per session now.
let mmdcAvailable: boolean | null = null;

function hasMmdc(): boolean {
  if (mmdcAvailable === null) {
    try {
      mmdcAvailable = Bun.spawnSync(["which", "mmdc"]).exitCode === 0;
    } catch {
      mmdcAvailable = false;
    }
  }
  return mmdcAvailable;
}

export async function renderMermaid(text: string): Promise<string> {
  try {
    if (hasMmdc()) {
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
  const mutedStyle = chalk.hex(colors.muted);
  return mutedStyle("(install @mermaid-js/mermaid-cli for rendered preview)") + "\n\n" + text;
}
