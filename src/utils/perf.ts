import fs from "fs";
import os from "os";
import path from "path";

const enabled = process.env.SEER_DEBUG_PERF === "1";
const logPath = process.env.SEER_DEBUG_PERF_FILE
  ?? path.join(os.tmpdir(), "seer-perf.log");

export function perfNow(): number {
  return enabled ? performance.now() : 0;
}

export function logPerf(event: string, fields: Record<string, unknown> = {}): void {
  if (!enabled) return;
  const line = JSON.stringify({
    ts: new Date().toISOString(),
    event,
    ...fields,
  });
  try {
    fs.appendFileSync(logPath, line + "\n");
  } catch {
    // Perf logging must never affect the TUI.
  }
}

export function perfLogPath(): string {
  return logPath;
}
