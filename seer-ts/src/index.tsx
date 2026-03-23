#!/usr/bin/env bun
import { render } from "ink";
import path from "path";
import { App } from "./App.js";

const version = "0.5.0";

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
  console.log("  alt+←/→     History back/forward");
  console.log("  r           Rename");
  console.log("  R           Reload");
  console.log("  n / N       New file / New directory");
  console.log("  e           Open in $EDITOR");
  console.log("  y / x       Yank (copy) / Cut");
  console.log("  P           Paste");
  console.log("  space       Multi-select toggle");
  console.log("  s           Cycle sort mode");
  console.log("  b           Bookmark current directory");
  console.log("  1-9         Jump to bookmark slot");
  console.log("  < / >       Resize left pane");
  console.log("  /           Fuzzy search");
  console.log("  .           Toggle hidden files");
  console.log("  p           Copy path to clipboard");
  console.log("  backspace   Move to trash");
  console.log("  ctrl+d/u    Scroll preview");
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

// Render the app
const { waitUntilExit } = render(
  <App startDir={resolvedDir} cwdFile={cwdFile} />,
  {
    exitOnCtrlC: false,
  },
);

await waitUntilExit();
