#!/usr/bin/env bun
import chalk from "chalk";
import { render } from "ink";
import path from "path";
import { App } from "./App.js";

// Ensure true-color support for chalk (used by preview renderers)
if (chalk.level < 3) chalk.level = 3;

const version = "1.0.2";

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
  console.log("  j/k         Move up/down");
  console.log("  l/enter     Open directory");
  console.log("  h           Go to parent");
  console.log("  g/G         Top/bottom");
  console.log("  /           Fuzzy search");
  console.log("  .           Toggle hidden files");
  console.log("  s           Cycle sort mode");
  console.log("  p           Copy path to clipboard");
  console.log("  < / >       Resize panes");
  console.log("  t           Cycle theme");
  console.log("  R           Reload");
  console.log("  backspace   Move to trash");
  console.log("  q           Quit");
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
