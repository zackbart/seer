// ── open file in a new tab of the host terminal ───────────────────────────
//
// Detects the terminal seer is running in via $TERM_PROGRAM / env vars and
// spawns a new tab with `<editor> <filePath>`. Ghostty is the primary target
// but we support the common macOS terminals too.

export type TerminalKind =
  | "ghostty"
  | "iterm2"
  | "terminal"
  | "wezterm"
  | "kitty"
  | "vscode"
  | "unknown";

export function detectTerminal(): TerminalKind {
  const tp = process.env.TERM_PROGRAM ?? "";
  if (tp === "ghostty") return "ghostty";
  if (tp === "iTerm.app") return "iterm2";
  if (tp === "Apple_Terminal") return "terminal";
  if (tp === "WezTerm") return "wezterm";
  if (tp === "vscode") return "vscode";
  if (process.env.KITTY_WINDOW_ID) return "kitty";
  return "unknown";
}

// Shell-quote: wrap in single quotes, escape any embedded single quote.
function shellQuote(s: string): string {
  return "'" + s.replace(/'/g, `'\\''`) + "'";
}

// Escape for an AppleScript double-quoted string literal.
function osaEscape(s: string): string {
  return s.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

async function runOsascript(lines: string[]): Promise<void> {
  const args: string[] = [];
  for (const l of lines) args.push("-e", l);
  const proc = Bun.spawn(["osascript", ...args], { stdout: "ignore", stderr: "pipe" });
  const code = await proc.exited;
  if (code !== 0) {
    const err = await new Response(proc.stderr).text();
    throw new Error(err.trim() || `osascript exited ${code}`);
  }
}

async function spawnOnly(cmd: string[]): Promise<void> {
  const proc = Bun.spawn(cmd, { stdout: "ignore", stderr: "pipe" });
  const code = await proc.exited;
  if (code !== 0) {
    const err = await new Response(proc.stderr).text();
    throw new Error(err.trim() || `${cmd[0]} exited ${code}`);
  }
}

export async function openInNewTab(
  filePath: string,
  editor: string = "nano",
): Promise<TerminalKind> {
  const kind = detectTerminal();
  const shellCmd = `${editor} ${shellQuote(filePath)}`;

  switch (kind) {
    case "ghostty": {
      // Ghostty exposes limited AppleScript; drive it via System Events.
      // Requires macOS Accessibility permission for the terminal app
      // (typically already granted).
      await runOsascript([
        'tell application "Ghostty" to activate',
        "delay 0.1",
        'tell application "System Events" to keystroke "t" using command down',
        "delay 0.2",
        `tell application "System Events" to keystroke "${osaEscape(shellCmd)}"`,
        'tell application "System Events" to key code 36',
      ]);
      return kind;
    }

    case "iterm2": {
      await runOsascript([
        'tell application "iTerm"',
        "  activate",
        "  tell current window",
        "    create tab with default profile",
        `    tell current session of current tab to write text "${osaEscape(shellCmd)}"`,
        "  end tell",
        "end tell",
      ]);
      return kind;
    }

    case "terminal": {
      await runOsascript([
        'tell application "Terminal" to activate',
        'tell application "System Events" to tell process "Terminal" to keystroke "t" using command down',
        "delay 0.2",
        `tell application "Terminal" to do script "${osaEscape(shellCmd)}" in selected tab of front window`,
      ]);
      return kind;
    }

    case "wezterm": {
      await spawnOnly(["wezterm", "cli", "spawn", "--new-tab", "--", editor, filePath]);
      return kind;
    }

    case "kitty": {
      await spawnOnly(["kitty", "@", "launch", "--type=tab", editor, filePath]);
      return kind;
    }

    case "vscode": {
      await spawnOnly(["code", "-r", filePath]);
      return kind;
    }

    default:
      throw new Error(
        `unsupported terminal: ${process.env.TERM_PROGRAM ?? "unknown"}`,
      );
  }
}
