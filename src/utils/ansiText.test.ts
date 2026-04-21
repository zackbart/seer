import { describe, expect, test } from "bun:test";
import { sanitizeTerminalText, stripAnsi } from "./ansiText.js";

describe("sanitizeTerminalText", () => {
  test("preserves SGR styling but removes cursor-control escapes", () => {
    const input = "\x1b[31mred\x1b[0m\x1b[2K\x1b[10Cafter";
    expect(sanitizeTerminalText(input)).toBe("\x1b[31mred\x1b[0mafter");
  });

  test("normalizes control characters that commonly appear in extracted docs", () => {
    const input = "a\rb\t\f\x08c\n";
    expect(sanitizeTerminalText(input)).toBe("a\nb    \nc\n");
    expect(sanitizeTerminalText(input, { preserveTabs: true })).toBe("a\nb\t\nc\n");
  });
});

describe("stripAnsi", () => {
  test("removes non-SGR CSI escapes as well as styling", () => {
    const input = "\x1b[31mred\x1b[0m\x1b[2K\x1b[10Cdone";
    expect(stripAnsi(input)).toBe("reddone");
  });
});
