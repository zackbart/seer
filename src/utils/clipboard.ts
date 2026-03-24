export async function copyToClipboard(text: string): Promise<void> {
  if (!text) return;

  const platform = process.platform;

  if (platform === "darwin") {
    await runClipboard(text, "pbcopy");
    return;
  }

  if (platform === "win32") {
    await runClipboard(text, "cmd", "/c", "clip");
    return;
  }

  // Linux: try wl-copy, xclip, xsel
  const candidates = [
    ["wl-copy"],
    ["xclip", "-selection", "clipboard"],
    ["xsel", "--clipboard", "--input"],
  ];

  let lastErr: Error | null = null;
  for (const [cmd, ...args] of candidates) {
    try {
      const which = Bun.spawnSync(["which", cmd]);
      if (which.exitCode !== 0) continue;
      await runClipboard(text, cmd, ...args);
      return;
    } catch (e) {
      lastErr = e as Error;
    }
  }

  throw lastErr ?? new Error("no clipboard utility found");
}

async function runClipboard(text: string, cmd: string, ...args: string[]): Promise<void> {
  const proc = Bun.spawn([cmd, ...args], {
    stdin: "pipe",
  });
  proc.stdin.write(text);
  proc.stdin.end();
  await proc.exited;
}
