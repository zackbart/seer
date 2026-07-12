#!/usr/bin/env bun
import chalk from "chalk";
import { render } from "ink";
import path from "path";
import { App } from "./App.js";
import { buildKittyDeleteAll, hasTransmittedAny } from "./utils/termGraphics.js";

// Ensure true-color support for chalk (used by preview renderers)
if (chalk.level < 3) chalk.level = 3;

const version = "1.0.23";

function printHelp() {
  console.log(`seer ${version}`);
  console.log();
  console.log("A TUI file browser with live preview.");
  console.log();
  console.log("Usage: seer [options] [directory]");
  console.log();
  console.log("Options:");
  console.log("  -h, --help            Show this help message");
  console.log("  -v, --version         Show version");
  console.log("  --cwd-file=FILE       Write final directory to FILE on exit");
  console.log("                        (for shell cd integration)");
  console.log();
  console.log("Shell cd integration — add to ~/.bashrc or ~/.zshrc:");
  console.log();
  console.log("  function s() {");
  console.log('    local tmp=$(mktemp)');
  console.log('    command seer --cwd-file="$tmp" "$@"');
  console.log('    local dir=$(cat "$tmp" 2>/dev/null)');
  console.log('    rm -f "$tmp"');
  console.log('    [[ -n "$dir" && -d "$dir" ]] && cd "$dir"');
  console.log("  }");
  console.log();
  console.log("Keybindings:");
  console.log("  j / k         Move selection up / down");
  console.log("  g / G         Jump to top / bottom");
  console.log("  l / ↵         Enter directory");
  console.log("  h             Go to parent directory");
  console.log("  /             Fuzzy search within current directory");
  console.log("  .             Toggle hidden files");
  console.log("  s             Cycle sort mode (name / size / modified)");
  console.log("  t             Cycle theme");
  console.log("  p             Copy selected path to clipboard");
  console.log("  o             Open selected item with the default app");
  console.log("  < / >         Resize left / right pane");
  console.log("  R             Reload current directory");
  console.log("  ⌫ (backspace) Move selected file to trash");
  console.log("  q / ctrl-c    Quit");
  console.log();
  console.log("Preview:");
  console.log("  Mouse wheel   Scroll preview body (over preview pane)");
  console.log("  Click + drag  Select text in preview body → clipboard on release");
  console.log("  Header shows  size · lines · token estimate · modified date");
  console.log();
  console.log("Environment:");
  console.log("  SEER_NO_NERD_FONT=1   Use plain Unicode instead of Nerd Font glyphs");
  console.log("  SEER_DEBUG_PERF=1     Write perf events to $TMPDIR/seer-perf.log");
}

// Parse CLI args
let cwdFile: string | undefined;
let startDir: string | undefined;

for (const arg of process.argv.slice(2)) {
  if (arg === "--version" || arg === "-v") {
    console.log(`seer ${version}`);
    process.exit(0);
  }
  if (arg === "--help" || arg === "-h") {
    printHelp();
    process.exit(0);
  }
  if (arg.startsWith("--cwd-file=")) {
    cwdFile = arg.slice("--cwd-file=".length);
    continue;
  }
  if (!startDir) {
    startDir = arg;
  }
}

// Resolve start directory
const resolvedDir = startDir ? path.resolve(startDir) : process.cwd();

// Enter alt-screen and enable mouse reporting
process.stdout.write("\x1b[?1049h"); // enter alt screen
process.stdout.write("\x1b[?25l");   // hide cursor
process.stdout.write("\x1b[?1000h"); // enable mouse button reporting
process.stdout.write("\x1b[?1002h"); // enable mouse cell-motion reporting
process.stdout.write("\x1b[?1006h"); // enable SGR extended mouse mode

// Clean up on exit
function cleanup() {
  // Free any Kitty graphics we uploaded this session. Guarded so we don't
  // emit a bogus APC on terminals that never saw one.
  if (hasTransmittedAny()) {
    process.stdout.write(buildKittyDeleteAll());
  }
  process.stdout.write("\x1b[?1006l"); // disable SGR mouse
  process.stdout.write("\x1b[?1002l"); // disable cell-motion
  process.stdout.write("\x1b[?1000l"); // disable mouse reporting
  process.stdout.write("\x1b[?25h");   // show cursor
  process.stdout.write("\x1b[?1049l"); // leave alt screen
}
process.on("exit", cleanup);
process.on("SIGINT", () => { cleanup(); process.exit(0); });
process.on("SIGTERM", () => { cleanup(); process.exit(0); });

// Render the app
const { waitUntilExit } = render(
  <App startDir={resolvedDir} cwdFile={cwdFile} />,
  {
    exitOnCtrlC: false,
  },
);

await waitUntilExit();
cleanup();
