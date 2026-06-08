// ── open with OS default app ────────────────────────────────────────────────

async function spawnDetached(cmd: string[]): Promise<void> {
  const proc = Bun.spawn(cmd, {
    stdout: "ignore",
    stderr: "pipe",
  });
  const code = await proc.exited;
  if (code !== 0) {
    const err = await new Response(proc.stderr).text();
    throw new Error(err.trim() || `${cmd[0]} exited ${code}`);
  }
}

export async function openWithDefaultApp(filePath: string): Promise<void> {
  if (process.platform === "darwin") {
    await spawnDetached(["open", filePath]);
    return;
  }

  if (process.platform === "win32") {
    await spawnDetached(["cmd", "/c", "start", "", filePath]);
    return;
  }

  const candidates = [
    ["xdg-open", filePath],
    ["gio", "open", filePath],
    ["kde-open", filePath],
    ["gnome-open", filePath],
  ];

  let lastErr: Error | null = null;
  for (const cmd of candidates) {
    try {
      const which = Bun.spawnSync(["which", cmd[0]]);
      if (which.exitCode !== 0) continue;
      await spawnDetached(cmd);
      return;
    } catch (e) {
      lastErr = e as Error;
    }
  }

  throw lastErr ?? new Error("no default-open utility found");
}
